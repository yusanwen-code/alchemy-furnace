# -*- coding: utf-8 -*-
"""
炼丹炉 - 文档处理路由 (Document API)
POST /api/v1/documents/extract - 提取文档内容
犹如呈上丹方，请丹房解读
"""
import logging

from fastapi import APIRouter, HTTPException, status

from app.models.schemas import (
    DocumentExtractRequest,
    DocumentExtractResponse,
    DocumentSplitRequest,
    BaseResponse,
)
from app.services.document_service import DocumentService

logger = logging.getLogger(__name__)

# 创建路由 - 文档处理之门户
router = APIRouter(prefix="/documents", tags=["文档处理 - 丹方解读"])

# 服务实例 - 丹方处理管事
document_service = DocumentService()


@router.post(
    "/extract",
    response_model=BaseResponse,
    summary="提取文档内容",
    description="上传丹方文件路径和类型，系统将自动选择对应解析器提取文本内容并切分",
)
async def extract_document(request: DocumentExtractRequest) -> BaseResponse:
    """
    提取文档内容 - 解读丹方
    
    根据文件类型自动选择解析器：
    - docx: python-docx 解析
    - xlsx: openpyxl 解析
    - md/txt: 直接读取
    - pdf: pdfplumber 解析
    
    返回提取的完整文本和切分后的 chunks。
    """
    try:
        logger.info(
            f"收到丹方解读请求: {request.file_path}, "
            f"类型: {request.file_type}"
        )
        
        # 调用服务层处理
        result = document_service.extract_document(
            file_path=request.file_path,
            file_type=request.file_type,
        )
        
        return BaseResponse(
            code=0,
            message=f"丹方解读完毕 - 共 {result['chunk_count']} 块",
            data=result,
        )
        
    except FileNotFoundError as e:
        logger.error(f"丹方文件不存在: {e}")
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"丹方文件不存在: {e}",
        )
    except ValueError as e:
        logger.error(f"丹方参数错误: {e}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e),
        )
    except Exception as e:
        logger.error(f"丹方解读失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"丹方解读失败: {e}",
        )


@router.post(
    "/split",
    response_model=BaseResponse,
    summary="切分文本",
    description="将长文本按指定策略切分为多个块",
)
async def split_text(request: DocumentSplitRequest) -> BaseResponse:
    """
    切分文本 - 丹材切分
    
    支持三种切分策略：
    - fixed: 固定长度切分
    - paragraph: 按段落切分
    - semantic: 按语义边界切分
    """
    try:
        logger.info(f"收到丹材切分请求 - 策略: {request.strategy}")
        
        chunks = document_service.split_text(
            text=request.text,
            strategy=request.strategy,
            chunk_size=request.chunk_size,
            overlap=request.overlap,
        )
        
        return BaseResponse(
            code=0,
            message=f"丹材切分完毕 - 共 {len(chunks)} 块",
            data={
                "chunks": chunks,
                "chunk_count": len(chunks),
                "strategy": request.strategy,
            },
        )
        
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e),
        )
    except Exception as e:
        logger.error(f"丹材切分失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"文本切分失败: {e}",
        )


@router.get(
    "/supported-types",
    response_model=BaseResponse,
    summary="获取支持的文件类型",
    description="查看系统支持解析的所有文件类型",
)
async def get_supported_types() -> BaseResponse:
    """获取支持的文件类型 - 查看可炼之丹材"""
    types = document_service.get_supported_types()
    return BaseResponse(
        code=0,
        message="可炼之丹材列表",
        data={"supported_types": types},
    )
