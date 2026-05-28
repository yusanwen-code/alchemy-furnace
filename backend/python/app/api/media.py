# -*- coding: utf-8 -*-
"""
炼丹炉 - 媒体处理路由 (Media API)
- POST /api/v1/media/transcribe - 音频转文字 (听音辨道)
- POST /api/v1/media/extract-subtitles - 视频提取字幕 (影像真言)
犹如听音殿，从声影中汲取智慧
"""
import logging

from fastapi import APIRouter, HTTPException, status

from app.models.schemas import (
    TranscribeRequest,
    TranscribeResponse,
    ExtractSubtitlesRequest,
    ExtractSubtitlesResponse,
    BaseResponse,
)
from app.services.media_service import MediaService

logger = logging.getLogger(__name__)

# 创建路由 - 媒体处理之门户
router = APIRouter(prefix="/media", tags=["媒体处理 - 听音辨道"])

# 服务实例 - 听音辨道师
media_service = MediaService()


@router.post(
    "/transcribe",
    response_model=BaseResponse,
    summary="音频转文字",
    description="使用 Whisper API 将音频文件转录为文字 (听音辨道)",
)
async def transcribe_audio(request: TranscribeRequest) -> BaseResponse:
    """
    音频转文字 - 听音辨道
    
    使用 OpenAI Whisper API 将音频文件转录为文字。
    支持 mp3, wav, m4a, flac 等格式。
    
    Request Body:
        - file_path: 音频文件路径
        - language: 语言代码 (如 "zh", "en")，可选
    """
    try:
        logger.info(f"收到听音辨道请求: {request.file_path}")
        
        result = media_service.transcribe_audio(
            file_path=request.file_path,
            language=request.language,
        )
        
        return BaseResponse(
            code=0,
            message="听音辨道完毕",
            data=result,
        )
        
    except FileNotFoundError as e:
        logger.error(f"音频文件不存在: {e}")
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=str(e),
        )
    except Exception as e:
        logger.error(f"听音辨道失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"音频转录失败: {e}",
        )


@router.post(
    "/extract-subtitles",
    response_model=BaseResponse,
    summary="视频提取字幕",
    description="使用 FFmpeg 提取音频 + Whisper 转录，从视频中提取字幕 (影像真言)",
)
async def extract_subtitles(request: ExtractSubtitlesRequest) -> BaseResponse:
    """
    视频提取字幕 - 影像真言
    
    流程：
    1. FFmpeg 从视频中提取音频
    2. Whisper 将音频转录为文字
    3. 返回字幕文本
    
    需要系统已安装 FFmpeg。
    
    Request Body:
        - file_path: 视频文件路径
        - language: 语言代码，可选
        - extract_audio_only: 是否仅提取音频
    """
    try:
        logger.info(f"收到影像真言提取请求: {request.file_path}")
        
        # 检查 FFmpeg 是否可用
        if not media_service.is_ffmpeg_available():
            raise HTTPException(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                detail="FFmpeg 未安装，无法提取音频。"
                       "请先安装: apt-get install ffmpeg",
            )
        
        result = media_service.extract_subtitles(
            file_path=request.file_path,
            language=request.language,
            extract_audio_only=request.extract_audio_only,
        )
        
        return BaseResponse(
            code=0,
            message="影像真言提取完毕",
            data=result,
        )
        
    except FileNotFoundError as e:
        logger.error(f"视频文件不存在: {e}")
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=str(e),
        )
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"影像真言提取失败: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"视频字幕提取失败: {e}",
        )


@router.get(
    "/ffmpeg-status",
    response_model=BaseResponse,
    summary="检查 FFmpeg 状态",
    description="检查 FFmpeg 是否已安装可用",
)
async def check_ffmpeg() -> BaseResponse:
    """检查 FFmpeg 状态 - 验音器"""
    available = media_service.is_ffmpeg_available()
    return BaseResponse(
        code=0,
        message="FFmpeg 已安装" if available else "FFmpeg 未安装",
        data={"available": available},
    )
