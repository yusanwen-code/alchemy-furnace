# -*- coding: utf-8 -*-
"""
炼丹炉 - 对话业务逻辑层 (Chat Service)
处理对话流程：检索上下文、构建 Prompt、调用 LLM

犹如道人之智慧核心，融合金丹知识，为修士答疑解惑
"""
import logging
from typing import List, Dict, Any, AsyncGenerator

import httpx
from openai import AsyncOpenAI, OpenAI

from app.core.config import settings
from app.core.retrieval.retriever import Retriever

logger = logging.getLogger(__name__)


class ChatService:
    """
    对话服务 - 道人智慧核心
    
    负责完整的对话流程：
    1. 用查询检索相关金丹内容
    2. 构建 System Prompt + Context + User Query
    3. 调用 LLM 生成回答
    4. 支持流式和非流式输出
    
    Attributes:
        retriever: 检索器
        client: OpenAI 同步客户端
        async_client: OpenAI 异步客户端
    """
    
    # 默认系统提示词 - 道人之格
    DEFAULT_SYSTEM_PROMPT = """你是一个基于知识库的智能助手。你会根据提供的参考资料来回答用户的问题。

请遵循以下原则：
1. 基于提供的参考资料回答，如果资料不足，请明确告知
2. 回答要准确、清晰、有条理
3. 可以适当引用参考资料中的内容
4. 如果用户问题与参考资料无关，请基于你的知识回答

参考资料：
{context}
"""
    
    def __init__(
        self,
        retriever: Retriever = None,
        api_key: str = None,
        base_url: str = None,
    ) -> None:
        """
        初始化对话服务
        
        Args:
            retriever: 检索器实例
            api_key: OpenAI API 密钥
            base_url: API 基础地址
        """
        self.retriever = retriever or Retriever()
        self.api_key = api_key or settings.openai_api_key
        self.base_url = base_url or settings.openai_base_url
        
        # 初始化 LLM 客户端
        http_client = httpx.Client(timeout=120.0)
        self.client = OpenAI(
            api_key=self.api_key,
            base_url=self.base_url,
            http_client=http_client,
        )
        
        async_http_client = httpx.AsyncClient(timeout=120.0)
        self.async_client = AsyncOpenAI(
            api_key=self.api_key,
            base_url=self.base_url,
            http_client=async_http_client,
        )
        
        logger.info("道人智慧核心初始化完毕")
    
    def chat_completion(
        self,
        messages: List[Dict[str, str]],
        pill_ids: List[int] = None,
        model: str = None,
        temperature: float = 0.7,
        max_tokens: int = 4096,
    ) -> Dict[str, Any]:
        """
        非流式对话 - 道人一次性回答
        
        流程：
        1. 从 messages 中提取最后一条用户查询
        2. 用 pill_ids + query 检索相关上下文
        3. 构建 System Prompt（含上下文）
        4. 调用 LLM 生成回答
        5. 返回回答和引用来源
        
        Args:
            messages: 消息历史，每项含 role 和 content
            pill_ids: 金丹ID列表
            model: LLM 模型
            temperature: 温度参数
            max_tokens: 最大 token 数
        
        Returns:
            包含 content, sources, model, usage 的字典
        """
        model = model or settings.default_model
        pill_ids = pill_ids or []
        
        try:
            logger.info(f"求道之问 - 模型: {model}, 金丹: {pill_ids}")
            
            # 步骤 1: 提取查询
            query = self._extract_query(messages)
            logger.info(f"用户查询: {query[:100]}...")
            
            # 步骤 2: 检索上下文 - 寻丹
            sources = []
            context = ""
            if pill_ids and query:
                sources = self.retriever.retrieve(pill_ids, query)
                context = self.retriever.format_context(sources)
                logger.info(f"寻得 {len(sources)} 枚相关丹材")
            
            # 步骤 3: 构建 Prompt
            system_prompt = self._build_system_prompt(context)
            chat_messages = self._build_messages(messages, system_prompt)
            
            # 步骤 4: 调用 LLM
            logger.info(f"调用 LLM - 模型: {model}")
            response = self.client.chat.completions.create(
                model=model,
                messages=chat_messages,
                temperature=temperature,
                max_tokens=max_tokens,
            )
            
            content = response.choices[0].message.content
            usage = {
                "prompt_tokens": response.usage.prompt_tokens,
                "completion_tokens": response.usage.completion_tokens,
                "total_tokens": response.usage.total_tokens,
            }
            
            # 格式化引用来源
            formatted_sources = [
                {
                    "content": s["content"][:300],
                    "score": round(s["score"], 4),
                    "metadata": s["metadata"],
                }
                for s in sources
            ]
            
            logger.info(
                f"道人回答完毕 - tokens: {usage['total_tokens']}"
            )
            
            return {
                "content": content,
                "sources": formatted_sources,
                "model": model,
                "usage": usage,
            }
            
        except Exception as e:
            logger.error(f"道人回答失败: {e}")
            raise RuntimeError(f"对话生成失败: {e}")
    
    async def chat_completion_stream(
        self,
        messages: List[Dict[str, str]],
        pill_ids: List[int] = None,
        model: str = None,
        temperature: float = 0.7,
        max_tokens: int = 4096,
    ) -> AsyncGenerator[str, None]:
        """
        流式对话 - 道人缓缓道来 (SSE)
        
        以 Server-Sent Events 格式流式返回 LLM 生成的内容。
        格式: data: {"content": "..."}\n\n
        
        Args:
            messages: 消息历史
            pill_ids: 金丹ID列表
            model: LLM 模型
            temperature: 温度参数
            max_tokens: 最大 token 数
        
        Yields:
            SSE 格式的数据块
        """
        model = model or settings.default_model
        pill_ids = pill_ids or []
        
        try:
            logger.info(f"求道之问(流式) - 模型: {model}, 金丹: {pill_ids}")
            
            # 步骤 1: 提取查询
            query = self._extract_query(messages)
            
            # 步骤 2: 检索上下文 - 寻丹
            sources = []
            context = ""
            if pill_ids and query:
                sources = await self.retriever.aretrieve(pill_ids, query)
                context = self.retriever.format_context(sources)
                logger.info(f"寻得 {len(sources)} 枚相关丹材")
            
            # 步骤 3: 构建 Prompt
            system_prompt = self._build_system_prompt(context)
            chat_messages = self._build_messages(messages, system_prompt)
            
            # 步骤 4: 调用 LLM 流式接口
            logger.info(f"调用 LLM 流式接口 - 模型: {model}")
            
            # 先发送来源信息
            if sources:
                import json
                source_data = {
                    "sources": [
                        {
                            "content": s["content"][:200],
                            "score": round(s["score"], 4),
                            "metadata": s.get("metadata", {}),
                        }
                        for s in sources
                    ]
                }
                yield f"data: {json.dumps(source_data, ensure_ascii=False)}\n\n"
            
            # 流式获取回答
            stream = await self.async_client.chat.completions.create(
                model=model,
                messages=chat_messages,
                temperature=temperature,
                max_tokens=max_tokens,
                stream=True,
            )
            
            async for chunk in stream:
                delta = chunk.choices[0].delta
                if delta.content:
                    import json
                    data = json.dumps(
                        {"content": delta.content},
                        ensure_ascii=False
                    )
                    yield f"data: {data}\n\n"
            
            # 发送结束标记
            yield "data: [DONE]\n\n"
            
            logger.info("道人回答流式传输完毕")
            
        except Exception as e:
            logger.error(f"道人回答(流式)失败: {e}")
            import json
            error_data = json.dumps({"error": str(e)}, ensure_ascii=False)
            yield f"data: {error_data}\n\n"
            yield "data: [DONE]\n\n"
    
    def _extract_query(self, messages: List[Dict[str, str]]) -> str:
        """
        从消息列表中提取最后一条用户查询 - 辨明所求
        
        Args:
            messages: 消息历史
        
        Returns:
            最后一条用户消息的内容
        """
        # 从后向前找最后一条 user 消息
        for msg in reversed(messages):
            if msg.get("role") == "user":
                return msg.get("content", "").strip()
        
        # 如果没有 user 消息，返回最后一条
        if messages:
            return messages[-1].get("content", "").strip()
        
        return ""
    
    def _build_system_prompt(self, context: str) -> str:
        """
        构建系统提示词 - 铸就道人心智
        
        将检索到的上下文整合进 System Prompt。
        
        Args:
            context: 检索到的参考文本
        
        Returns:
            完整的 System Prompt
        """
        if context:
            return self.DEFAULT_SYSTEM_PROMPT.format(context=context)
        
        return (
            "你是一个智能助手。请基于你的知识，尽可能准确、"
            "清晰、有条理地回答用户的问题。"
        )
    
    def _build_messages(
        self,
        messages: List[Dict[str, str]],
        system_prompt: str,
    ) -> List[Dict[str, str]]:
        """
        构建完整的消息列表 - 组装对话
        
        在消息列表最前面插入 System Prompt。
        
        Args:
            messages: 原始消息历史
            system_prompt: 系统提示词
        
        Returns:
            包含 System Prompt 的完整消息列表
        """
        # 过滤掉已有的 system 消息
        user_messages = [m for m in messages if m.get("role") != "system"]
        
        return [
            {"role": "system", "content": system_prompt},
            *user_messages,
        ]
