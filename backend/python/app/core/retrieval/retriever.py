# -*- coding: utf-8 -*-
"""
炼丹炉 - 检索模块 (Retriever)
寻丹之术：将查询转化为向量，在金丹阁中搜寻最相关的丹材

检索流程：
1. 将查询文本 Embedding 为向量
2. 在指定金丹中搜索相似向量
3. 返回最相关的文本块
犹如以神识探查金丹阁，寻找最契合之丹材
"""
import logging
from typing import List, Dict, Any

from app.core.config import settings
from app.core.embedding.embedder import Embedder
from app.core.vectorstore.qdrant_store import QdrantStore

logger = logging.getLogger(__name__)


class Retriever:
    """
    检索器 - 寻丹师
    
    协调 Embedding 和 VectorStore 完成完整的检索流程。
    负责将用户查询转化为向量，并在指定知识库中搜索相关内容。
    
    Attributes:
        embedder: 向量化器
        vector_store: 向量存储
        top_k: 默认返回结果数
    
    Example:
        retriever = Retriever()
        results = retriever.retrieve(pill_ids=[1, 2], query="炼丹术的基本原理")
    """
    
    def __init__(
        self,
        embedder: Embedder = None,
        vector_store: QdrantStore = None,
        top_k: int = None,
    ) -> None:
        """
        初始化检索器
        
        Args:
            embedder: 向量化器实例，默认创建新实例
            vector_store: 向量存储实例，默认创建新实例
            top_k: 默认检索数量，从配置读取
        """
        self.embedder = embedder or Embedder()
        self.vector_store = vector_store or QdrantStore()
        self.top_k = top_k or settings.top_k
        
        logger.info(f"寻丹师就位 - 默认取前 {self.top_k} 枚")
    
    def retrieve(
        self,
        pill_ids: List[int],
        query: str,
        top_k: int = None,
    ) -> List[Dict[str, Any]]:
        """
        检索 - 寻丹之术（同步版）
        
        完整检索流程：
        1. 将查询文本向量化
        2. 在指定金丹中搜索相似向量
        3. 返回排序后的相关文本块
        
        Args:
            pill_ids: 要检索的金丹ID列表
            query: 查询文本
            top_k: 返回结果数量，默认使用初始化值
        
        Returns:
            检索结果列表，按相似度分数降序排列
            每项: {
                "content": str,      # 文本内容
                "score": float,      # 相似度分数 (0-1)
                "metadata": dict,    # 元数据
                "pill_id": int,      # 金丹ID
                "recipe_id": int,    # 丹方ID
            }
        
        Raises:
            ValueError: 参数校验失败
            RuntimeError: 检索流程出错
        
        Example:
            results = retriever.retrieve(
                pill_ids=[1, 2],
                query="如何炼制九转金丹？"
            )
            for r in results:
                print(f"[相似度: {r['score']:.3f}] {r['content'][:100]}...")
        """
        k = top_k or self.top_k
        
        if not pill_ids:
            logger.warning("未指定金丹，寻丹无果")
            return []
        
        if not query or not query.strip():
            raise ValueError("查询文本为空")
        
        try:
            logger.info(f"开始寻丹 - 查询: '{query[:50]}...', 金丹: {pill_ids}")
            
            # 步骤 1: 将查询向量化 - 凝练神识
            logger.debug("凝练查询神识...")
            query_vector = self.embedder.embed_query(query)
            
            # 步骤 2: 在向量库中搜索 - 探查金丹阁
            logger.debug("探查金丹阁...")
            results = self.vector_store.search(
                pill_ids=pill_ids,
                query_vector=query_vector,
                top_k=k,
            )
            
            logger.info(f"寻丹完毕 - 找到 {len(results)} 枚相关丹材")
            return results
            
        except Exception as e:
            logger.error(f"寻丹失败: {e}")
            raise RuntimeError(f"检索失败: {e}")
    
    async def aretrieve(
        self,
        pill_ids: List[int],
        query: str,
        top_k: int = None,
    ) -> List[Dict[str, Any]]:
        """
        异步检索 - 异步寻丹之术
        
        与 retrieve 功能相同，但使用异步接口，适合高并发场景。
        
        Args:
            pill_ids: 要检索的金丹ID列表
            query: 查询文本
            top_k: 返回结果数量
        
        Returns:
            检索结果列表
        """
        k = top_k or self.top_k
        
        if not pill_ids:
            return []
        
        if not query or not query.strip():
            raise ValueError("查询文本为空")
        
        try:
            logger.info(f"开始异步寻丹 - 查询: '{query[:50]}...'")
            
            # 步骤 1: 异步向量化查询
            query_vector = await self.embedder.aembed_query(query)
            
            # 步骤 2: 搜索（qdrant-client 的 search 本身是同步的，
            # 但在 async 函数中调用不会阻塞事件循环）
            results = self.vector_store.search(
                pill_ids=pill_ids,
                query_vector=query_vector,
                top_k=k,
            )
            
            logger.info(f"异步寻丹完毕 - 找到 {len(results)} 枚相关丹材")
            return results
            
        except Exception as e:
            logger.error(f"异步寻丹失败: {e}")
            raise RuntimeError(f"异步检索失败: {e}")
    
    def retrieve_with_scores(
        self,
        pill_ids: List[int],
        query: str,
        top_k: int = None,
        score_threshold: float = 0.0,
    ) -> List[Dict[str, Any]]:
        """
        带分数阈值的检索 - 精选高品质丹材
        
        只返回相似度分数高于阈值的结果。
        
        Args:
            pill_ids: 金丹ID列表
            query: 查询文本
            top_k: 返回结果数量
            score_threshold: 最低相似度分数 (0-1)，默认 0
        
        Returns:
            过滤后的检索结果
        """
        results = self.retrieve(pill_ids, query, top_k)
        
        if score_threshold > 0:
            filtered = [r for r in results if r["score"] >= score_threshold]
            logger.info(
                f"精选丹材 - 原 {len(results)} 枚, "
                f"阈值 {score_threshold} 过滤后 {len(filtered)} 枚"
            )
            return filtered
        
        return results
    
    def format_context(self, results: List[Dict[str, Any]]) -> str:
        """
        将检索结果格式化为上下文文本 - 整合丹材
        
        将多个检索结果拼接为可供 LLM 使用的上下文字符串。
        
        Args:
            results: 检索结果列表
        
        Returns:
            格式化后的上下文文本
        """
        if not results:
            return ""
        
        context_parts: List[str] = []
        for idx, result in enumerate(results, 1):
            content = result.get("content", "").strip()
            metadata = result.get("metadata", {})
            source = metadata.get("file_name", "未知来源")
            
            context_parts.append(
                f"[丹材 {idx}] 来源: {source}\n{content}"
            )
        
        return "\n\n---\n\n".join(context_parts)
