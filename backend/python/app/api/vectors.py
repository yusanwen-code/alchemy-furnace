# -*- coding: utf-8 -*-
"""
炼丹炉 - 向量管理路由 (Vector API)
- POST /api/v1/vectors/ingest - 向量化入库（炼丹入炉）
- DELETE /api/v1/vectors/pill/{pill_id} - 删除金丹所有向量（毁丹）
- POST /api/v1/vectors/search - 相似度搜索（寻丹）
犹如操控金丹阁，炼丹寻丹毁丹皆在一念之间
"""
import logging

from fastapi import APIRouter, HTTPException, status, Path

from app.models.schemas import (
    VectorIngestRequest,
    VectorIngestResponse,
    VectorSearchRequest,
    VectorSearchResponse,
    BaseResponse,
)
from app.services.vector_service import VectorService

logger = logging.getLogger(__name__)

# 创建路由 - 向量管理之门户
router = APIRouter(prefix="/vectors", tags=["向量管理 - 金丹阁"])

# 服务实例 - 炼丹房总管
vector_service = VectorService()


@router.post(
    "/ingest",
    response_model=BaseResponse,
    summary="向量化入库",
    description="将文本块向量化后存入 Qdrant（炼丹入炉）",
)
async def ingest_vectors(request: VectorIngestRequest) -> BaseResponse:
    """
    向量化入库 - 炼丹入炉
    
    将一批文本块经过 Embedding 模型转化为向量，
    存入 Qdrant 向量数据库，payload 中携带金丹ID和丹方ID。
    
    Request Body:
        - pill_id: 金丹ID（知识库ID）
        - recipe_id: 丹方ID（文档ID）
        - chunks: 文本块列表，每项含 content 和 metadata
    """
    try:
        logger.info(
            f"收到炼丹入炉请求 - 金丹{request.pill_id}, "
            f"丹方{request.recipe_id}, 丹材{len(request.chunks)}份"
        )
        
        # 转换为服务层需要的格式
        chunks = [
            {"content": chunk.content, "metadata": chunk.metadata}
            for chunk in request.chunks
        ]
        
        # 调用服务层
        result = vector_service.ingest_vectors(
            pill_id=request.pill_id,
            recipe_id=request.recipe_id,
            chunks=chunks,
        )
        
        return BaseResponse(
            code=0,
            message=result["message"],
            data=result,
        )
        
    except ValueError as e:
        logger.error(f"炼丹入炉参数错误: {e}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e),
        )
    except Exception as e:
        logger.error(f"炼丹入炉失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"向量化入库失败: {e}",
        )


@router.delete(
    "/pill/{pill_id}",
    response_model=BaseResponse,
    summary="删除金丹的所有向量",
    description="删除指定金丹下的所有向量（毁丹）",
)
async def delete_pill_vectors(
    pill_id: int = Path(..., gt=0, description="金丹ID")
) -> BaseResponse:
    """
    删除金丹的所有向量 - 毁丹
    
    当金丹被删除时，级联删除其在 Qdrant 中的所有向量。
    """
    try:
        logger.info(f"收到毁丹请求 - 金丹{pill_id}")
        
        result = vector_service.delete_by_pill(pill_id)
        
        return BaseResponse(
            code=0,
            message=result["message"],
            data=result,
        )
        
    except Exception as e:
        logger.error(f"毁丹失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"删除金丹向量失败: {e}",
        )


@router.delete(
    "/recipe/{recipe_id}",
    response_model=BaseResponse,
    summary="删除丹方的所有向量",
    description="删除指定丹方下的所有向量（毁丹方）",
)
async def delete_recipe_vectors(
    recipe_id: int = Path(..., gt=0, description="丹方ID")
) -> BaseResponse:
    """
    删除丹方的所有向量 - 毁丹方
    
    当某个丹方被删除时，删除其对应的向量。
    """
    try:
        logger.info(f"收到毁丹方请求 - 丹方{recipe_id}")
        
        result = vector_service.delete_by_recipe(recipe_id)
        
        return BaseResponse(
            code=0,
            message=result["message"],
            data=result,
        )
        
    except Exception as e:
        logger.error(f"毁丹方失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"删除丹方向量失败: {e}",
        )


@router.post(
    "/search",
    response_model=BaseResponse,
    summary="相似度搜索",
    description="在指定金丹中搜索与查询最相似的向量（寻丹）",
)
async def search_vectors(request: VectorSearchRequest) -> BaseResponse:
    """
    相似度搜索 - 寻丹之术
    
    将查询文本 Embedding 后，在指定的金丹集合中搜索最相似的向量，
    返回匹配的文本块及其相似度分数。
    
    Request Body:
        - pill_ids: 要搜索的金丹ID列表
        - query: 查询文本
        - top_k: 返回结果数量，默认 5
    """
    try:
        logger.info(
            f"收到寻丹请求 - 查询: '{request.query[:50]}', "
            f"金丹: {request.pill_ids}"
        )
        
        result = vector_service.search_vectors(
            pill_ids=request.pill_ids,
            query=request.query,
            top_k=request.top_k,
        )
        
        return BaseResponse(
            code=0,
            message=f"寻丹完毕 - 找到 {result['total']} 枚相关丹材",
            data=result,
        )
        
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e),
        )
    except Exception as e:
        logger.error(f"寻丹失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"向量搜索失败: {e}",
        )


@router.get(
    "/store-info",
    response_model=BaseResponse,
    summary="获取向量存储信息",
    description="查看金丹阁的当前状况",
)
async def get_store_info() -> BaseResponse:
    """获取向量存储信息 - 查看金丹阁状况"""
    try:
        info = vector_service.get_store_info()
        return BaseResponse(
            code=0,
            message="金丹阁状况",
            data=info,
        )
    except Exception as e:
        logger.error(f"获取金丹阁信息失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"获取存储信息失败: {e}",
        )
