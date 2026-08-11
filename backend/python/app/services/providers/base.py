# -*- coding: utf-8 -*-
"""
Provider 协议层(007-demo-mode) - 对话 / 合成服务的抽象

service 层依赖 ChatProvider / SynthesisProvider 协议而非具体实现,
在 runtime.setup_providers() 阶段根据 is_demo() 注入 real/demo 实例。
"""

from typing import Any, AsyncGenerator, Dict, List, Optional, Protocol, runtime_checkable


# ==================== 协议定义 ====================


@runtime_checkable
class ChatProvider(Protocol):
    """对话服务协议: 与 app.services.chat_service.ChatService 同形"""

    def chat_completion(
        self,
        messages: List[Dict[str, str]],
        model: Optional[str] = None,
        temperature: float = 0.7,
        max_tokens: int = 4096,
        api_key: Optional[str] = None,
        base_url: Optional[str] = None,
    ) -> Dict[str, Any]:
        """非流式对话: 返回 {content, model, usage} 字典"""
        ...

    def chat_completion_stream(
        self,
        messages: List[Dict[str, str]],
        model: Optional[str] = None,
        temperature: float = 0.7,
        max_tokens: int = 4096,
        api_key: Optional[str] = None,
        base_url: Optional[str] = None,
    ) -> AsyncGenerator[str, None]:
        """流式对话: 逐块 yield 形如 'data: {"content": "..."}\\n\\n' 的 SSE 字符串"""
        ...


@runtime_checkable
class SynthesisProvider(Protocol):
    """合成服务协议: 与 app.services.language_synthesis_service 行为对齐"""

    def synthesize(
        self,
        personality: str,
        pills: List[Any],
    ) -> Dict[str, Any]:
        """合成系统提示词: 返回 {system_prompt, emergence_rules, inner_tensions, source_fingerprint}"""
        ...


# ==================== 全局 Provider 句柄（运行时注入） ====================


class _Providers:
    """持有当前生效的 ChatProvider / SynthesisProvider 实例"""

    def __init__(self) -> None:
        self._chat: Optional[ChatProvider] = None
        self._synthesis: Optional[SynthesisProvider] = None

    def set(self, chat: ChatProvider, synthesis: SynthesisProvider) -> None:
        self._chat = chat
        self._synthesis = synthesis

    def chat(self) -> ChatProvider:
        if self._chat is None:
            raise RuntimeError("ChatProvider 未初始化,请先调用 runtime.setup_providers()")
        return self._chat

    def synthesis(self) -> SynthesisProvider:
        if self._synthesis is None:
            raise RuntimeError("SynthesisProvider 未初始化,请先调用 runtime.setup_providers()")
        return self._synthesis


providers = _Providers()
