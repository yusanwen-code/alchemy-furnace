# -*- coding: utf-8 -*-
"""
炼丹炉 - Qdrant 向量数据库封装 (Vector Store)
金丹之归处，存储所有向量化的丹方片段

使用 qdrant-client 与 Qdrant 向量数据库通信，
提供初始化、插入、删除、搜索等操作。
犹如金丹阁，存放已炼化的金丹碎片。
"""
import logging
from typing import List, Dict, Any, Optional

from qdrant_client import QdrantClient
from qdrant_client.models import (
    Distance,
    VectorParams,
    PointStruct,
    Filter,
    FieldCondition,
    MatchValue,
    MatchAny,
)

from app.core.config import settings

logger = logging.getLogger(__name__)


class QdrantStore:
    """
    Qdrant 向量存储 - 金丹阁
    
    管理 Qdrant 向量数据库的连接和操作，包括：
    - 初始化 Collection（创建索引结构）
    - 批量插入向量（炼丹入炉）
    - 按金丹 ID 删除向量（毁丹）
    - 在指定金丹中搜索相似向量（寻丹）
    
    Attributes:
        client: Qdrant 客户端
        collection_name: 集合名称
        vector_size: 向量维度
    
    Example:
        store = QdrantStore()
        store.init_collection()
        store.upsert(pill_id=1, recipe_id=1, chunks=[...])
        results = store.search(pill_ids=[1], query_vector=[...], top_k=5)
    """
    
    def __init__(
        self,
        host: Optional[str] = None,
        port: Optional[int] = None,
        collection_name: Optional[str] = None,
        api_key: Optional[str] = None,
        vector_size: int = 1536,
    ) -> None:
        """
        初始化 Qdrant 存储
        
        Args:
            host: Qdrant 主机地址
            port: Qdrant 端口
            collection_name: 集合名称
            api_key: Qdrant API 密钥（可选）
            vector_size: 向量维度，默认 1536
        """
        self.host = host or settings.qdrant_host
        self.port = port or settings.qdrant_port
        self.collection_name = collection_name or settings.qdrant_collection
        self.api_key = api_key or settings.qdrant_api_key
        self.vector_size = vector_size
        
        # 初始化 Qdrant 客户端 - 建立与金丹阁的连接
        if self.api_key:
            self.client = QdrantClient(
                host=self.host,
                port=self.port,
                api_key=self.api_key,
            )
        else:
            self.client = QdrantClient(host=self.host, port=self.port)
        
        logger.info(
            f"金丹阁连接建立: {self.host}:{self.port}, "
            f"阁名: {self.collection_name}, 维度: {self.vector_size}"
        )
    
    def init_collection(self) -> None:
        """
        初始化 Collection - 筑建金丹阁
        
        检查集合是否存在，不存在则创建。
        使用 Cosine 距离度量，适合语义搜索。
        """
        try:
            # 检查集合是否已存在
            collections = self.client.get_collections().collections
            collection_names = [c.name for c in collections]
            
            if self.collection_name in collection_names:
                logger.info(f"金丹阁「{self.collection_name}」已存在，无需重建")
                return
            
            # 创建新集合 - 筑建新阁
            logger.info(f"开始筑建金丹阁「{self.collection_name}」...")
            
            self.client.create_collection(
                collection_name=self.collection_name,
                vectors_config=VectorParams(
                    size=self.vector_size,
                    distance=Distance.COSINE,
                ),
            )
            
            logger.info(
                f"金丹阁「{self.collection_name}」筑建完毕 - "
                f"维度: {self.vector_size}, 距离: Cosine"
            )
            
        except Exception as e:
            logger.error(f"金丹阁筑建失败: {e}")
            raise RuntimeError(f"初始化 Qdrant Collection 失败: {e}")
    
    def upsert(
        self,
        pill_id: int,
        recipe_id: int,
        chunks: List[Dict[str, Any]],
        vectors: List[List[float]],
    ) -> int:
        """
        批量插入向量 - 炼丹入炉
        
        将文本块及其向量批量存入 Qdrant，每个块携带金丹ID和丹方ID作为 payload。
        
        Args:
            pill_id: 金丹ID（知识库ID）
            recipe_id: 丹方ID（文档ID）
            chunks: 文本块列表，每项含 content 和 metadata
            vectors: 对应的向量列表
        
        Returns:
            成功插入的向量数量
        
        Raises:
            ValueError: 参数校验失败
            RuntimeError: 插入失败
        """
        if len(chunks) != len(vectors):
            raise ValueError(
                f"文本块数量({len(chunks)})与向量数量({len(vectors)})不匹配"
            )
        
        if not chunks:
            logger.warning("无丹材可入炉，跳过插入")
            return 0
        
        try:
            logger.info(
                f"开始炼丹入炉 - 金丹{pill_id}, 丹方{recipe_id}, "
                f"丹材{len(chunks)}份"
            )
            
            points: List[PointStruct] = []
            for idx, (chunk, vector) in enumerate(zip(chunks, vectors)):
                point_id = f"{pill_id}_{recipe_id}_{idx}"
                
                # 构建 payload - 丹材之印记
                payload = {
                    "pill_id": pill_id,
                    "recipe_id": recipe_id,
                    "content": chunk.get("content", ""),
                    "chunk_index": idx,
                    "metadata": chunk.get("metadata", {}),
                }
                
                # 使用确定性 ID（避免重复入炉）
                # 将字符串ID转换为整数哈希
                point_id_hash = hash(point_id) & 0x7FFFFFFFFFFFFFFF
                
                points.append(
                    PointStruct(
                        id=point_id_hash,
                        vector=vector,
                        payload=payload,
                    )
                )
            
            # 批量插入 - 入炉
            self.client.upsert(
                collection_name=self.collection_name,
                points=points,
            )
            
            logger.info(f"炼丹入炉完毕 - 成功存入 {len(points)} 个向量")
            return len(points)
            
        except Exception as e:
            logger.error(f"炼丹入炉失败: {e}")
            raise RuntimeError(f"向量插入失败: {e}")
    
    def delete_by_pill(self, pill_id: int) -> int:
        """
        删除金丹的所有向量 - 毁丹
        
        当金丹被删除时，级联删除其下所有向量。
        
        Args:
            pill_id: 金丹ID
        
        Returns:
            删除的向量数量
        """
        try:
            logger.info(f"开始毁丹 - 金丹{pill_id}")
            
            # 构建过滤条件 - 筛选指定金丹
            filter_condition = Filter(
                must=[
                    FieldCondition(
                        key="pill_id",
                        match=MatchValue(value=pill_id),
                    )
                ]
            )
            
            # 先统计数量
            count_result = self.client.count(
                collection_name=self.collection_name,
                count_filter=filter_condition,
            )
            count = count_result.count
            
            if count == 0:
                logger.info(f"金丹{pill_id}下无向量，无需毁丹")
                return 0
            
            # 执行删除
            self.client.delete(
                collection_name=self.collection_name,
                points_filter=filter_condition,
            )
            
            logger.info(f"毁丹完毕 - 金丹{pill_id}, 删除 {count} 个向量")
            return count
            
        except Exception as e:
            logger.error(f"毁丹失败 - 金丹{pill_id}: {e}")
            raise RuntimeError(f"删除金丹向量失败: {e}")
    
    def delete_by_recipe(self, recipe_id: int) -> int:
        """
        删除丹方的所有向量 - 毁丹方
        
        当某个丹方被删除时，删除其对应的向量。
        
        Args:
            recipe_id: 丹方ID
        
        Returns:
            删除的向量数量
        """
        try:
            logger.info(f"开始毁丹方 - 丹方{recipe_id}")
            
            filter_condition = Filter(
                must=[
                    FieldCondition(
                        key="recipe_id",
                        match=MatchValue(value=recipe_id),
                    )
                ]
            )
            
            count_result = self.client.count(
                collection_name=self.collection_name,
                count_filter=filter_condition,
            )
            count = count_result.count
            
            if count == 0:
                return 0
            
            self.client.delete(
                collection_name=self.collection_name,
                points_filter=filter_condition,
            )
            
            logger.info(f"毁丹方完毕 - 丹方{recipe_id}, 删除 {count} 个向量")
            return count
            
        except Exception as e:
            logger.error(f"毁丹方失败 - 丹方{recipe_id}: {e}")
            raise RuntimeError(f"删除丹方向量失败: {e}")
    
    def search(
        self,
        pill_ids: List[int],
        query_vector: List[float],
        top_k: int = 5,
    ) -> List[Dict[str, Any]]:
        """
        在指定金丹中搜索相似向量 - 寻丹之术
        
        使用 query_vector 在指定的金丹集合中搜索最相似的向量，
        返回匹配的文本块及其相似度分数。
        
        Args:
            pill_ids: 要搜索的金丹ID列表
            query_vector: 查询向量
            top_k: 返回结果数量，默认 5
        
        Returns:
            搜索结果列表，每项包含 content, score, metadata, pill_id, recipe_id
        
        Example:
            results = store.search(
                pill_ids=[1, 2],
                query_vector=[0.1, -0.2, ...],
                top_k=5
            )
            # 返回: [
            #   {"content": "...", "score": 0.95, "metadata": {...}, ...},
            #   ...
            # ]
        """
        if not pill_ids:
            logger.warning("未指定金丹ID，无法寻丹")
            return []
        
        try:
            logger.info(
                f"开始寻丹 - 金丹列表: {pill_ids}, 取前 {top_k} 枚"
            )
            
            # 构建过滤条件 - 限定金丹范围
            if len(pill_ids) == 1:
                filter_condition = Filter(
                    must=[
                        FieldCondition(
                            key="pill_id",
                            match=MatchValue(value=pill_ids[0]),
                        )
                    ]
                )
            else:
                filter_condition = Filter(
                    must=[
                        FieldCondition(
                            key="pill_id",
                            match=MatchAny(any=pill_ids),
                        )
                    ]
                )
            
            # 执行搜索
            search_result = self.client.search(
                collection_name=self.collection_name,
                query_vector=query_vector,
                query_filter=filter_condition,
                limit=top_k,
                with_payload=True,
            )
            
            # 整理结果
            results: List[Dict[str, Any]] = []
            for scored_point in search_result:
                payload = scored_point.payload or {}
                results.append({
                    "content": payload.get("content", ""),
                    "score": scored_point.score,
                    "metadata": payload.get("metadata", {}),
                    "pill_id": payload.get("pill_id", 0),
                    "recipe_id": payload.get("recipe_id", 0),
                    "chunk_index": payload.get("chunk_index", 0),
                })
            
            logger.info(f"寻丹完毕 - 找到 {len(results)} 枚相关碎片")
            return results
            
        except Exception as e:
            logger.error(f"寻丹失败: {e}")
            raise RuntimeError(f"向量搜索失败: {e}")
    
    def get_collection_info(self) -> Dict[str, Any]:
        """
        获取集合信息 - 查看金丹阁状况
        
        Returns:
            集合的详细信息，包括向量数量、维度等
        """
        try:
            info = self.client.get_collection(self.collection_name)
            return {
                "name": self.collection_name,
                "status": info.status,
                "vector_size": info.config.params.vectors.size,
                "distance": str(info.config.params.vectors.distance),
                "vectors_count": info.vectors_count,
                "indexed_vectors_count": info.indexed_vectors_count,
                "points_count": info.points_count,
            }
        except Exception as e:
            logger.error(f"获取金丹阁信息失败: {e}")
            raise RuntimeError(f"获取集合信息失败: {e}")
