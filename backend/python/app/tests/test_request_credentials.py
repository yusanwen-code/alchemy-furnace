# -*- coding: utf-8 -*-
"""
调用级凭证（api_key/base_url）单元测试 (T009)

覆盖：
1. 请求携带 api_key/base_url 时，该次调用构造的 OpenAI 客户端使用这些值
2. 未携带时回退到环境变量配置的共享客户端（向后兼容，不新建客户端）
3. api_key 为空但 base_url 指向本地服务（如 ollama）时，以占位符 "none"
   通过 OpenAI SDK 的非空校验
4. 流式路径：取消（GeneratorExit）时显式关闭上游流；错误统一映射为中文 SSE 事件

运行：cd backend/python && .venv/bin/python -m pytest app/tests/test_request_credentials.py -q
"""
import asyncio
import json
from types import SimpleNamespace

import pytest

from app.core.config import settings
from app.services import chat_service as chat_module
from app.services import language_synthesis_service as synthesis_module
from app.services.chat_service import ChatService, map_llm_error, mask_api_key
from app.services.language_synthesis_service import LanguageSynthesisService


# ==================== 测试夹具 ====================


def _fake_llm_response(content="道人应答"):
    """伪造非流式 chat.completions.create 返回值"""
    message = SimpleNamespace(content=content)
    choice = SimpleNamespace(message=message)
    usage = SimpleNamespace(prompt_tokens=10, completion_tokens=5, total_tokens=15)
    return SimpleNamespace(choices=[choice], usage=usage)


class _SyncClientFactory:
    """记录 OpenAI 构造参数，并返回桩客户端"""

    def __init__(self, create):
        self.calls = []
        self._create = create

    def __call__(self, **kwargs):
        self.calls.append(kwargs)
        client = SimpleNamespace()
        client.chat = SimpleNamespace(
            completions=SimpleNamespace(create=self._create)
        )
        client.close = lambda: None
        return client


class _FakeAsyncStream:
    """伪造上游 AsyncStream：异步迭代 + 可关闭"""

    def __init__(self, chunks=("你", "好")):
        self._chunks = chunks
        self.closed = False

    def __aiter__(self):
        async def _gen():
            for text in self._chunks:
                delta = SimpleNamespace(content=text)
                yield SimpleNamespace(choices=[SimpleNamespace(delta=delta)])

        return _gen()

    async def close(self):
        self.closed = True


class _AsyncClientFactory:
    """记录 AsyncOpenAI 构造参数，并返回桩异步客户端"""

    def __init__(self, create):
        self.calls = []
        self._create = create

    def __call__(self, **kwargs):
        self.calls.append(kwargs)
        client = SimpleNamespace()
        client.chat = SimpleNamespace(
            completions=SimpleNamespace(create=self._create)
        )

        async def _close():
            return None

        client.close = _close
        return client


def _collect_stream(gen):
    """同步驱动异步生成器，收集全部 SSE 事件"""

    async def _run():
        return [item async for item in gen]

    return asyncio.run(_run())


# ==================== 1. 非流式对话：调用级凭证 ====================


class TestChatCompletionCredentials:
    def test_request_credentials_used_for_client(self, monkeypatch):
        """请求携带 api_key/base_url -> 该次调用的客户端使用这些值"""
        monkeypatch.setattr(settings, "openai_api_key", "")
        svc = ChatService(api_key="", base_url="")
        factory = _SyncClientFactory(create=lambda **kw: _fake_llm_response())
        monkeypatch.setattr(chat_module, "OpenAI", factory)

        result = svc.chat_completion(
            messages=[{"role": "user", "content": "求道"}],
            model="deepseek-chat",
            api_key="sk-request-key",
            base_url="https://api.deepseek.com/v1",
        )

        assert result["content"] == "道人应答"
        assert len(factory.calls) == 1
        call = factory.calls[0]
        assert call["api_key"] == "sk-request-key"
        assert call["base_url"] == "https://api.deepseek.com/v1"

    def test_env_fallback_reuses_shared_client(self, monkeypatch):
        """未携带凭证 -> 复用环境变量配置的共享客户端，不新建"""
        monkeypatch.setattr(settings, "openai_api_key", "sk-env-key")
        svc = ChatService(api_key="sk-env-key", base_url="http://env-host/v1")
        factory = _SyncClientFactory(create=lambda **kw: _fake_llm_response())
        monkeypatch.setattr(chat_module, "OpenAI", factory)

        captured = {}

        def fake_create(**kwargs):
            captured["kwargs"] = kwargs
            return _fake_llm_response()

        monkeypatch.setattr(svc.client.chat.completions, "create", fake_create)

        result = svc.chat_completion(
            messages=[{"role": "user", "content": "求道"}],
            model="gpt-4o",
        )

        assert result["content"] == "道人应答"
        assert "kwargs" in captured  # 共享客户端确实被调用
        assert factory.calls == []  # 未构造任何新客户端

    def test_local_service_without_api_key_uses_placeholder(self, monkeypatch):
        """api_key 为空 + 本地 base_url（如 ollama）-> 以占位符构造客户端"""
        monkeypatch.setattr(settings, "openai_api_key", "")
        svc = ChatService(api_key="", base_url="")
        factory = _SyncClientFactory(create=lambda **kw: _fake_llm_response())
        monkeypatch.setattr(chat_module, "OpenAI", factory)

        result = svc.chat_completion(
            messages=[{"role": "user", "content": "求道"}],
            model="llama3",
            base_url="http://localhost:11434/v1",
        )

        assert result["content"] == "道人应答"
        assert len(factory.calls) == 1
        call = factory.calls[0]
        assert call["base_url"] == "http://localhost:11434/v1"
        # OpenAI SDK 要求非空 api_key，本地服务用占位符
        assert call["api_key"] == "none"


# ==================== 2. 流式对话：凭证 / 取消 / 错误映射 ====================


class TestChatStreamCredentials:
    def test_stream_uses_request_credentials_and_closes_stream(self, monkeypatch):
        """流式路径：调用级凭证生效，且流正常结束后被关闭"""
        monkeypatch.setattr(settings, "openai_api_key", "")
        svc = ChatService(api_key="", base_url="")
        stream = _FakeAsyncStream(chunks=("你", "好"))

        async def fake_create(**kwargs):
            return stream

        factory = _AsyncClientFactory(create=fake_create)
        monkeypatch.setattr(chat_module, "AsyncOpenAI", factory)

        events = _collect_stream(
            svc.chat_completion_stream(
                messages=[{"role": "user", "content": "求道"}],
                model="deepseek-chat",
                api_key="sk-request-key",
                base_url="https://api.deepseek.com/v1",
            )
        )

        assert len(factory.calls) == 1
        call = factory.calls[0]
        assert call["api_key"] == "sk-request-key"
        assert call["base_url"] == "https://api.deepseek.com/v1"

        contents = []
        for event in events:
            assert event.startswith("data: ")
            if event.strip() == "data: [DONE]":
                continue
            contents.append(json.loads(event[len("data: "):])["content"])
        assert "".join(contents) == "你好"
        assert events[-1].strip() == "data: [DONE]"
        assert stream.closed is True  # finally 释放了上游流

    def test_stream_cancellation_closes_upstream(self, monkeypatch):
        """客户端断开（GeneratorExit）-> 显式关闭上游流，停止 token 消耗"""
        monkeypatch.setattr(settings, "openai_api_key", "")
        svc = ChatService(api_key="", base_url="")
        stream = _FakeAsyncStream(chunks=("一", "二", "三"))

        async def fake_create(**kwargs):
            return stream

        factory = _AsyncClientFactory(create=fake_create)
        monkeypatch.setattr(chat_module, "AsyncOpenAI", factory)

        async def _run():
            gen = svc.chat_completion_stream(
                messages=[{"role": "user", "content": "求道"}],
                model="deepseek-chat",
                api_key="sk-request-key",
                base_url="https://api.deepseek.com/v1",
            )
            first = await gen.__anext__()
            assert "一" in first
            await gen.aclose()  # 模拟客户端断开

        asyncio.run(_run())
        assert stream.closed is True

    def test_stream_timeout_maps_to_chinese_error(self, monkeypatch):
        """LLM 超时 -> SSE 错误事件：语言引擎响应超时，请稍后重试"""
        import httpx

        monkeypatch.setattr(settings, "openai_api_key", "")
        svc = ChatService(api_key="", base_url="")

        async def fake_create(**kwargs):
            raise httpx.TimeoutException("read timeout")

        factory = _AsyncClientFactory(create=fake_create)
        monkeypatch.setattr(chat_module, "AsyncOpenAI", factory)

        events = _collect_stream(
            svc.chat_completion_stream(
                messages=[{"role": "user", "content": "求道"}],
                model="deepseek-chat",
                api_key="sk-request-key",
                base_url="https://api.deepseek.com/v1",
            )
        )

        assert len(events) == 2
        payload = json.loads(events[0][len("data: "):])
        assert payload["error"] == "语言引擎响应超时，请稍后重试"
        assert payload["code"] == "TIMEOUT"
        assert events[1].strip() == "data: [DONE]"

    def test_stream_auth_failure_maps_to_chinese_error(self, monkeypatch):
        """401/403 鉴权失败 -> SSE 错误事件：模型凭证无效..."""

        class FakeAuthError(Exception):
            status_code = 401

        monkeypatch.setattr(settings, "openai_api_key", "")
        svc = ChatService(api_key="", base_url="")

        async def fake_create(**kwargs):
            raise FakeAuthError("invalid api key")

        factory = _AsyncClientFactory(create=fake_create)
        monkeypatch.setattr(chat_module, "AsyncOpenAI", factory)

        events = _collect_stream(
            svc.chat_completion_stream(
                messages=[{"role": "user", "content": "求道"}],
                model="deepseek-chat",
                api_key="sk-bad-key",
                base_url="https://api.deepseek.com/v1",
            )
        )

        payload = json.loads(events[0][len("data: "):])
        assert payload["error"] == "模型凭证无效，请检查模型管理中的 API Key"
        assert payload["code"] == "AUTH_FAILED"
        assert events[1].strip() == "data: [DONE]"

    def test_stream_model_not_found_maps_to_chinese_error(self, monkeypatch):
        """404 模型不存在 -> SSE 错误事件：模型不存在或不可用"""

        class FakeNotFoundError(Exception):
            status_code = 404

        monkeypatch.setattr(settings, "openai_api_key", "")
        svc = ChatService(api_key="", base_url="")

        async def fake_create(**kwargs):
            raise FakeNotFoundError("model not found")

        factory = _AsyncClientFactory(create=fake_create)
        monkeypatch.setattr(chat_module, "AsyncOpenAI", factory)

        events = _collect_stream(
            svc.chat_completion_stream(
                messages=[{"role": "user", "content": "求道"}],
                model="ghost-model",
                api_key="sk-request-key",
                base_url="https://api.deepseek.com/v1",
            )
        )

        payload = json.loads(events[0][len("data: "):])
        assert payload["error"] == "模型不存在或不可用"
        assert payload["code"] == "MODEL_NOT_FOUND"
        assert events[1].strip() == "data: [DONE]"

    def test_stream_env_fallback_no_new_client(self, monkeypatch):
        """流式路径未携带凭证 -> 复用共享异步客户端"""
        monkeypatch.setattr(settings, "openai_api_key", "sk-env-key")
        svc = ChatService(api_key="sk-env-key", base_url="http://env-host/v1")
        factory = _AsyncClientFactory(create=None)
        monkeypatch.setattr(chat_module, "AsyncOpenAI", factory)

        stream = _FakeAsyncStream(chunks=("道",))

        async def fake_create(**kwargs):
            return stream

        monkeypatch.setattr(svc.async_client.chat.completions, "create", fake_create)

        events = _collect_stream(
            svc.chat_completion_stream(
                messages=[{"role": "user", "content": "求道"}],
                model="gpt-4o",
            )
        )

        assert factory.calls == []  # 未构造新客户端
        assert events[-1].strip() == "data: [DONE]"


# ==================== 3. 合成服务：调用级凭证 ====================


class TestSynthesisCredentials:
    def _llm_json_create(self, **kwargs):
        content = json.dumps(
            {"system_prompt": "你是融合后的 AI 道人。", "emergence_rules": []},
            ensure_ascii=False,
        )
        return _fake_llm_response(content=content)

    def test_request_credentials_used_for_client(self, monkeypatch):
        """合成请求携带凭证 -> 涌现推导客户端使用这些值（即使环境未配置密钥）"""
        monkeypatch.setattr(settings, "openai_api_key", "")
        svc = LanguageSynthesisService(api_key="", base_url="")
        factory = _SyncClientFactory(create=self._llm_json_create)
        monkeypatch.setattr(synthesis_module, "OpenAI", factory)

        result = svc.combine(
            personality="沉稳内敛",
            pills=[],
            model="deepseek-chat",
            api_key="sk-request-key",
            base_url="https://api.deepseek.com/v1",
        )

        # LLM 路径被使用（非降级提示词）
        assert result["system_prompt"] == "你是融合后的 AI 道人。"
        assert len(factory.calls) == 1
        call = factory.calls[0]
        assert call["api_key"] == "sk-request-key"
        assert call["base_url"] == "https://api.deepseek.com/v1"

    def test_env_fallback_reuses_shared_client(self, monkeypatch):
        """合成请求未携带凭证 -> 复用共享客户端"""
        monkeypatch.setattr(settings, "openai_api_key", "sk-env-key")
        svc = LanguageSynthesisService(api_key="sk-env-key", base_url="http://env-host/v1")
        factory = _SyncClientFactory(create=self._llm_json_create)
        monkeypatch.setattr(synthesis_module, "OpenAI", factory)
        monkeypatch.setattr(
            svc.client.chat.completions, "create", self._llm_json_create
        )

        result = svc.combine(personality="沉稳内敛", pills=[], model="gpt-4o-mini")

        assert result["system_prompt"] == "你是融合后的 AI 道人。"
        assert factory.calls == []

    def test_no_credentials_anywhere_falls_back(self, monkeypatch):
        """请求与环境均无凭证 -> 走结构化合并降级路径，不构造客户端"""
        monkeypatch.setattr(settings, "openai_api_key", "")
        svc = LanguageSynthesisService(api_key="", base_url="")
        factory = _SyncClientFactory(create=self._llm_json_create)
        monkeypatch.setattr(synthesis_module, "OpenAI", factory)

        result = svc.combine(personality="沉稳内敛", pills=[])

        assert "沉稳内敛" in result["system_prompt"]  # 降级提示词
        assert result["emergence_rules"] == []
        assert factory.calls == []


# ==================== 4. 错误映射与密钥脱敏 ====================


class TestErrorMappingAndMasking:
    def test_map_llm_error_timeout(self):
        import httpx

        message, code = map_llm_error(httpx.TimeoutException("t"))
        assert (message, code) == ("语言引擎响应超时，请稍后重试", "TIMEOUT")

    def test_map_llm_error_status_codes(self):
        class FakeError(Exception):
            def __init__(self, status_code):
                super().__init__("err")
                self.status_code = status_code

        assert map_llm_error(FakeError(401))[1] == "AUTH_FAILED"
        assert map_llm_error(FakeError(403))[1] == "AUTH_FAILED"
        assert map_llm_error(FakeError(404))[0] == "模型不存在或不可用"
        assert map_llm_error(FakeError(500))[1] == "LLM_ERROR"

    def test_map_llm_error_generic(self):
        message, code = map_llm_error(RuntimeError("boom"))
        assert code == "LLM_ERROR"
        assert message  # 可读中文

    def test_mask_api_key_never_plaintext(self):
        assert mask_api_key(None) == "(none)"
        assert mask_api_key("") == "(none)"
        assert mask_api_key("sk-1234567890abcd") == "sk-****abcd"
        assert "1234567890" not in mask_api_key("sk-1234567890abcd")
