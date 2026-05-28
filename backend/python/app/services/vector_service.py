# -*- coding: utf-8 -*-
"""
炼丹炉 - 向量操作业务逻辑层 (Vector Service)
负责向量化入库、搜索、删除等核心炼丹流程

犹如炼丹房总管，掌控金丹炼制的全流程
"""
import logging
from typing import List, Dict, Any

from app.core.config import settings
from app.core.embedding.embedder import Embedder
from app.core.vectorstore.qdrant_store import QdrantStore
from app.core.retrieval.retriever import Retriever

logger = logging.getLogger(__name__)


class VectorService:
    """
    向量操作服务 - 炼丹房总管
    
    管理向量的完整生命周期：
    - 炼丹（入库）: 文本 → Embedding → Qdrant
    - 寻丹（搜索）: Query → Embedding → 检索
    - 毁丹（删除）: 按金丹ID删除向量
    """
    
    def __init__(
        self,
        embedder: Embedder = None,
        vector_store: QdrantStore = None,
        retriever: Retriever = None,
    ) -> None:
        """
        初始化向量服务
        
        Args:
            embedder: 向量化器实例
            vector_store: 向量存储实例
            retriever: 检索器实例
        """
        self.embedder = embedder or Embedder()
        self.vector_store = vector_store or QdrantStore()
        self.retriever = retriever or Retriever(
            embedder=self.embedder,
            vector_store=self.vector_store,
        )
        logger.info("炼丹房总管就位")
    
    def ingest_vectors(
        self,
        pill_id: int,
        recipe_id: int,
        chunks: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """
        向量化入库 - 炼丹入炉
        
        将文本块列表向量化后存入 Qdrant：
        1. 提取所有块的 content
        2. 调用 Embedding 模型批量向量化
        3. 将向量连同 payload 存入 Qdrant
        
        Args:
            pill_id: 金丹ID
            recipe_id: 丹方ID
            chunks: 文本块列表，每项含 content 和 metadata
        
        Returns:
            入库结果: {pill_id, recipe_id, vector_count, message}
        
        Raises:
            ValueError: 参数校验失败
            RuntimeError: 入库失败
        """
        if not chunks:
            raise ValueError("丹材列表为空，无法入炉")
        
        try:
            logger.info(
                f"开始炼丹入炉 - 金丹{pill_id}, 丹方{recipe_id}, "
                f"丹材{len(chunks)}份"
            )
            
            # 步骤 1: 提取文本内容
            texts = [chunk["content"] for chunk in chunks]
            
            # 步骤 2: 向量化 - 提炼真元
            logger.info(f"开始提炼真元: {len(texts)} 条文本")
            vectors = self.embedder.embed(texts)
            logger.info(f"真元提炼完毕: {len(vectors)} 个向量")
            
            # 步骤 3: 存入 Qdrant - 入炉
            count = self.vector_store.upsert(
                pill_id=pill_id,
                recipe_id=recipe_id,
                chunks=chunks,
                vectors=vectors,
            )
            
            message = f"炼丹成功 - 金丹{pill_id} 入库 {count} 个向量"
            logger.info(message)
            
            return {
                "pill_id": pill_id,
                "recipe_id": recipe_id,
                "vector_count": count,
                "message": message,
            }
            
        except Exception as e:
            logger.error(f"炼丹入炉失败: {e}")
            raise RuntimeError(f"向量化入库失败: {e}")
    
    async def aingest_vectors(
        self,
        pill_id: int,
        recipe_id: int,
        chunks: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """
        异步向量化入库 - 异步炼丹入炉
        
        Args:
            pill_id: 金丹ID
            recipe_id: 丹方ID
            chunks: 文本块列表
        
        Returns:
            入库结果
        """
        if not chunks:
            raise ValueError("丹材列表为空，无法入炉")
        
        try:
            logger.info(f"开始异步炼丹入炉 - 金丹{pill_id}, 丹方{recipe_id}")
            
            texts = [chunk["content"] for chunk in chunks]
            vectors = await self.embedder.aembed(texts)
            
            count = self.vector_store.upsert(
                pill_id=pill_id,
                recipe_id=recipe_id,
                chunks=chunks,
                vectors=vectors,
            )
            
            message = f"异步炼丹成功 - 金丹{pill_id} 入库 {count} 个向量"
            logger.info(message)
            
            return {
                "pill_id": pill_id,
                "recipe_id": recipe_id,
                "vector_count": count,
                "message": message,
            }
            
        except Exception as e:
            logger.error(f"异步炼丹入炉失败: {e}")
            raise RuntimeError(f"异步向量化入库失败: {e}")
    
    def search_vectors(
        self,
        pill_ids: List[int],
        query: str,
        top_k: int = None,
    ) -> Dict[str, Any]:
        """
        向量搜索 - 寻丹之术
        
        使用检索器在指定金丹中搜索相关内容。
        
        Args:
            pill_ids: 金丹ID列表
            query: 查询文本
            top_k: 返回结果数量
        
        Returns:
            搜索结果: {results, total, query}
        """
        k = top_k or settings.top_k
        
        try:
            logger.info(f"寻丹请求 - 查询: '{query[:50]}', 金丹: {pill_ids}")
            
            results = self.retriever.retrieve(
                pill_ids=pill_ids,
                query=query,
                top_k=k,
            )
            
            # 格式化结果
            formatted_results = [
                {
                    "content": r["content"],
                    "score": round(r["score"], 4),
                    "metadata": r["metadata"],
                    "pill_id": r["pill_id"],
                    "recipe_id": r["recipe_id"],
                }
                for r in results
            ]
            
            logger.info(f"寻丹完毕 - 找到 {len(formatted_results)} 枚")
            
            return {
                "results": formatted_results,
                "total": len(formatted_results),
                "query": query,
            }
            
        except Exception as e:
            logger.error(f"寻丹失败: {e}")
            raise RuntimeError(f"向量搜索失败: {e}")
    
    def delete_by_pill(self, pill_id: int) -> Dict[str, Any]:
        """
        删除金丹的所有向量 - 毁丹
        
        Args:
            pill_id: 金丹ID
        
        Returns:
            删除结果: {pill_id, deleted_count, message}
        """
        try:
            logger.info(f"毁丹请求 - 金丹{pill_id}")
            
            count = self.vector_store.delete_by_pill(pill_id)
            
            message = f"毁丹完毕 - 金丹{pill_id} 删除 {count} 个向量"
            logger.info(message)
            
            return {
                "pill_id": pill_id,
                "deleted_count": count,
                "message": message,
            }
            
        except Exception as e:
            logger.error(f"毁丹失败 - 金丹{pill_id}: {e}")
            raise RuntimeError(f"删除金丹向量失败: {e}")
    
    def delete_by_recipe(self, recipe_id: int) -> Dict[str, Any]:
        """
        删除丹方的所有向量 - 毁丹方
        
        Args:
            recipe_id: 丹方ID
        
        Returns:
            删除结果
        """
        try:
            logger.info(f"毁丹方请求 - 丹方{recipe_id}")
            
            count = self.vector_store.delete_by_recipe(recipe_id)
            
            return {
                "recipe_id": recipe_id,
                "deleted_count": count,
                "message": f"毁丹方完毕 - 删除 {count} 个向量",
            }
            
        except Exception as e:
            logger.error(f"毁丹方失败 - 丹方{recipe_id}: {e}")
            raise RuntimeError(f"删除丹方向量失败: {e}")
    
    def get_store_info(self) -> Dict[str, Any]:
        """
        获取向量存储信息 - 查看金丹阁状况
        
        Returns:
            存储信息
        """
        return self.vector_store.get_collection_info()
