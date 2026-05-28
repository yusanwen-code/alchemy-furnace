# -*- coding: utf-8 -*-
"""
炼丹炉 - 向量化模块 (Embedding)
将丹方文本转化为真元向量，犹如将丹材提炼为精华

使用 OpenAI text-embedding-3-small 模型，1536 维向量
支持自定义 API Key 和 Base URL（兼容 OpenAI 接口的第三方服务）
"""
import logging
from typing import List, Optional

import httpx
from openai import AsyncOpenAI, OpenAI

from app.core.config import settings

logger = logging.getLogger(__name__)


class Embedder:
    """
    向量化器 - 真元提炼炉
    
    将文本转化为高维向量，供向量数据库存储和检索。
    犹如炼丹师将草药提炼为丹药精华。
    
    Attributes:
        model: 使用的 Embedding 模型名称
        dimensions: 向量维度
        client: OpenAI 同步客户端
        async_client: OpenAI 异步客户端
    
    Example:
        embedder = Embedder()
        vectors = embedder.embed(["这是一段文本", "这是另一段文本"])
    """
    
    def __init__(
        self,
        api_key: Optional[str] = None,
        base_url: Optional[str] = None,
        model: Optional[str] = None,
        dimensions: Optional[int] = None,
    ) -> None:
        """
        初始化向量化器
        
        Args:
            api_key: OpenAI API 密钥，默认从环境变量读取
            base_url: API 基础地址，默认从环境变量读取
            model: Embedding 模型，默认 text-embedding-3-small
            dimensions: 向量维度，默认 1536
        """
        self.api_key = api_key or settings.openai_api_key
        self.base_url = base_url or settings.openai_base_url
        self.model = model or settings.embedding_model
        self.dimensions = dimensions or settings.embedding_dimensions
        
        if not self.api_key:
            logger.warning("OpenAI API 密钥未配置，向量化功能将不可用")
        
        # 初始化同步客户端 -  sync client
        self.client = OpenAI(
            api_key=self.api_key,
            base_url=self.base_url,
            http_client=httpx.Client(timeout=60.0),
        )
        
        # 初始化异步客户端 - async client
        self.async_client = AsyncOpenAI(
            api_key=self.api_key,
            base_url=self.base_url,
            http_client=httpx.AsyncClient(timeout=60.0),
        )
        
        logger.info(
            f"真元提炼炉初始化完毕 - 模型: {self.model}, "
            f"维度: {self.dimensions}, 接口: {self.base_url}"
        )
    
    def embed(self, texts: List[str]) -> List[List[float]]:
        """
        将文本列表转化为向量 - 批量提炼真元
        
        将一批文本送入 Embedding 模型，返回对应的向量列表。
        自动过滤空文本，自动分批（每批最多 2048 条）。
        
        Args:
            texts: 要向量化的文本列表
        
        Returns:
            向量列表，每个向量是 float 列表，长度为 self.dimensions
        
        Raises:
            ValueError: 输入为空或 API 密钥未配置
            RuntimeError: API 调用失败
        
        Example:
            vectors = embedder.embed(["丹道修炼", "炼丹术"])
            # 返回: [[0.01, -0.02, ...], [0.03, -0.01, ...]]
        """
        if not self.api_key:
            raise ValueError("API 密钥未配置，无法进行向量化")
        
        if not texts:
            raise ValueError("输入文本列表为空")
        
        # 过滤空文本 - 去除杂质
        valid_texts = [t for t in texts if t and t.strip()]
        if not valid_texts:
            raise ValueError("所有文本均为空")
        
        logger.info(f"开始提炼真元: {len(valid_texts)} 条文本")
        
        all_vectors: List[List[float]] = []
        batch_size = 128  # OpenAI 推荐批次大小
        
        try:
            for i in range(0, len(valid_texts), batch_size):
                batch = valid_texts[i:i + batch_size]
                logger.debug(f"提炼批次 {i // batch_size + 1}: {len(batch)} 条")
                
                response = self.client.embeddings.create(
                    model=self.model,
                    input=batch,
                    dimensions=self.dimensions,
                )
                
                batch_vectors = [item.embedding for item in response.data]
                all_vectors.extend(batch_vectors)
            
            logger.info(f"真元提炼完毕: {len(all_vectors)} 个向量")
            return all_vectors
            
        except Exception as e:
            logger.error(f"真元提炼失败: {e}")
            raise RuntimeError(f"向量化失败: {e}")
    
    async def aembed(self, texts: List[str]) -> List[List[float]]:
        """
        异步将文本列表转化为向量 - 异步批量提炼真元
        
        与 embed 相同功能，但使用异步客户端，适合高并发场景。
        
        Args:
            texts: 要向量化的文本列表
        
        Returns:
            向量列表
        """
        if not self.api_key:
            raise ValueError("API 密钥未配置，无法进行向量化")
        
        if not texts:
            raise ValueError("输入文本列表为空")
        
        valid_texts = [t for t in texts if t and t.strip()]
        if not valid_texts:
            raise ValueError("所有文本均为空")
        
        logger.info(f"开始异步提炼真元: {len(valid_texts)} 条文本")
        
        all_vectors: List[List[float]] = []
        batch_size = 128
        
        try:
            for i in range(0, len(valid_texts), batch_size):
                batch = valid_texts[i:i + batch_size]
                
                response = await self.async_client.embeddings.create(
                    model=self.model,
                    input=batch,
                    dimensions=self.dimensions,
                )
                
                batch_vectors = [item.embedding for item in response.data]
                all_vectors.extend(batch_vectors)
            
            logger.info(f"异步真元提炼完毕: {len(all_vectors)} 个向量")
            return all_vectors
            
        except Exception as e:
            logger.error(f"异步真元提炼失败: {e}")
            raise RuntimeError(f"异步向量化失败: {e}")
    
    def embed_query(self, query: str) -> List[float]:
        """
        将查询文本转化为单个向量 - 单条提炼
        
        Args:
            query: 查询文本
        
        Returns:
            单个向量
        """
        vectors = self.embed([query])
        return vectors[0]
    
    async def aembed_query(self, query: str) -> List[float]:
        """
        异步将查询文本转化为单个向量
        
        Args:
            query: 查询文本
        
        Returns:
            单个向量
        """
        vectors = await self.aembed([query])
        return vectors[0]
    
    def count_tokens(self, text: str) -> int:
        """
        估算文本的 token 数量 - 计量丹材分量
        
        Args:
            text: 输入文本
        
        Returns:
            token 数量估算值
        """
        try:
            import tiktoken
            encoding = tiktoken.encoding_for_model(self.model)
            return len(encoding.encode(text))
        except Exception:
            # 粗略估算: 中文约 1 字 1 token，英文约 4 字符 1 token
            import re
            chinese_chars = len(re.findall(r'[\u4e00-\u9fff]', text))
            other_chars = len(text) - chinese_chars
            return chinese_chars + other_chars // 4 + 1
