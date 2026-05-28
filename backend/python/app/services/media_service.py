# -*- coding: utf-8 -*-
"""
炼丹炉 - 媒体处理业务逻辑层 (Media Service)
处理音频转文字和视频字幕提取

犹如听音辨道之术，从声音影像中提取真言
"""
import logging
import os
import subprocess
import tempfile
from typing import Optional, Dict, Any

import httpx
from openai import OpenAI

from app.core.config import settings

logger = logging.getLogger(__name__)


class MediaService:
    """
    媒体处理服务 - 听音辨道师
    
    负责处理音频和视频的转录工作：
    - 音频转文字（使用 Whisper API）
    - 视频提取字幕（FFmpeg 提取音频 + Whisper 转录）
    
    Attributes:
        client: OpenAI 客户端（用于 Whisper）
    """
    
    def __init__(
        self,
        api_key: str = None,
        base_url: str = None,
    ) -> None:
        """
        初始化媒体处理服务
        
        Args:
            api_key: OpenAI API 密钥
            base_url: API 基础地址
        """
        self.api_key = api_key or settings.openai_api_key
        self.base_url = base_url or settings.openai_base_url
        
        self.client = OpenAI(
            api_key=self.api_key,
            base_url=self.base_url,
            http_client=httpx.Client(timeout=300.0),
        )
        
        logger.info("听音辨道师就位")
    
    def transcribe_audio(
        self,
        file_path: str,
        language: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        音频转文字 - 听音辨道
        
        使用 OpenAI Whisper API 将音频文件转录为文字。
        支持 mp3, wav, m4a, flac 等格式。
        
        Args:
            file_path: 音频文件路径
            language: 语言代码 (如 "zh", "en")，可选
        
        Returns:
            转录结果: {text, duration, language}
        
        Raises:
            FileNotFoundError: 音频文件不存在
            RuntimeError: 转录失败
        """
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"音频文件不存在: {file_path}")
        
        try:
            logger.info(f"开始听音辨道: {file_path}")
            
            # 准备请求参数
            params = {"model": "whisper-1"}
            if language:
                params["language"] = language
            
            with open(file_path, "rb") as audio_file:
                response = self.client.audio.transcriptions.create(
                    file=audio_file,
                    **params
                )
            
            text = response.text
            
            # 获取音频时长
            duration = self._get_audio_duration(file_path)
            
            logger.info(
                f"听音辨道完毕 - 时长: {duration:.1f}s, "
                f"文本: {len(text)} 字"
            )
            
            return {
                "text": text,
                "duration": duration,
                "language": language or "auto",
            }
            
        except Exception as e:
            logger.error(f"听音辨道失败: {e}")
            raise RuntimeError(f"音频转录失败: {e}")
    
    def extract_subtitles(
        self,
        file_path: str,
        language: Optional[str] = None,
        extract_audio_only: bool = False,
    ) -> Dict[str, Any]:
        """
        视频提取字幕 - 从影像中提取真言
        
        流程：
        1. 使用 FFmpeg 从视频中提取音频
        2. 使用 Whisper 将音频转录为文字
        3. 返回字幕文本
        
        Args:
            file_path: 视频文件路径
            language: 语言代码，可选
            extract_audio_only: 是否仅提取音频不转录
        
        Returns:
            字幕结果: {text, audio_path, duration}
        
        Raises:
            FileNotFoundError: 视频文件不存在
            RuntimeError: 处理失败
        """
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"视频文件不存在: {file_path}")
        
        audio_path = None
        
        try:
            logger.info(f"开始从影像中提取真言: {file_path}")
            
            # 步骤 1: FFmpeg 提取音频
            audio_path = self._extract_audio_from_video(file_path)
            
            if extract_audio_only:
                duration = self._get_audio_duration(audio_path)
                return {
                    "text": "",
                    "audio_path": audio_path,
                    "duration": duration,
                }
            
            # 步骤 2: Whisper 转录
            transcribe_result = self.transcribe_audio(audio_path, language)
            
            # 获取视频时长
            duration = self._get_video_duration(file_path)
            
            # 清理临时音频文件
            if audio_path != file_path and os.path.exists(audio_path):
                os.remove(audio_path)
            
            logger.info(
                f"影像真言提取完毕 - 时长: {duration:.1f}s, "
                f"文本: {len(transcribe_result['text'])} 字"
            )
            
            return {
                "text": transcribe_result["text"],
                "audio_path": None,  # 已清理
                "duration": duration,
            }
            
        except Exception as e:
            # 清理临时文件
            if audio_path and audio_path != file_path and os.path.exists(audio_path):
                try:
                    os.remove(audio_path)
                except Exception:
                    pass
            
            logger.error(f"影像真言提取失败: {e}")
            raise RuntimeError(f"视频字幕提取失败: {e}")
    
    def _extract_audio_from_video(self, video_path: str) -> str:
        """
        从视频中提取音频 - 抽音之术
        
        使用 FFmpeg 将视频中的音频提取为 mp3 格式。
        
        Args:
            video_path: 视频文件路径
        
        Returns:
            提取的音频文件路径
        """
        try:
            # 创建临时音频文件
            temp_dir = tempfile.gettempdir()
            base_name = os.path.splitext(os.path.basename(video_path))[0]
            audio_path = os.path.join(temp_dir, f"{base_name}_audio.mp3")
            
            logger.info(f"抽音中: {video_path} -> {audio_path}")
            
            # FFmpeg 提取音频
            cmd = [
                "ffmpeg",
                "-y",  # 覆盖已有文件
                "-i", video_path,
                "-vn",  # 不处理视频
                "-acodec", "libmp3lame",
                "-ar", "16000",  # 采样率 16kHz（Whisper 推荐）
                "-ac", "1",  # 单声道
                "-q:a", "2",  # 音质
                audio_path,
            ]
            
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=300,
            )
            
            if result.returncode != 0:
                logger.error(f"FFmpeg 抽音失败: {result.stderr}")
                raise RuntimeError(f"FFmpeg 错误: {result.stderr[:500]}")
            
            if not os.path.exists(audio_path):
                raise RuntimeError("音频提取失败：输出文件未生成")
            
            logger.info(f"抽音完毕: {audio_path}")
            return audio_path
            
        except FileNotFoundError:
            logger.error("FFmpeg 未安装，无法抽音")
            raise RuntimeError(
                "FFmpeg 未安装，请先安装: "
                "apt-get install ffmpeg 或 brew install ffmpeg"
            )
        except subprocess.TimeoutExpired:
            logger.error("FFmpeg 抽音超时")
            raise RuntimeError("音频提取超时")
        except Exception as e:
            logger.error(f"抽音失败: {e}")
            raise RuntimeError(f"音频提取失败: {e}")
    
    def _get_audio_duration(self, file_path: str) -> Optional[float]:
        """
        获取音频时长 - 计量音长
        
        Args:
            file_path: 音频文件路径
        
        Returns:
            时长（秒），获取失败返回 None
        """
        try:
            cmd = [
                "ffprobe",
                "-v", "error",
                "-show_entries", "format=duration",
                "-of", "default=noprint_wrappers=1:nokey=1",
                file_path,
            ]
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=30,
            )
            if result.returncode == 0:
                return float(result.stdout.strip())
        except Exception:
            pass
        return None
    
    def _get_video_duration(self, file_path: str) -> Optional[float]:
        """
        获取视频时长 - 计量影长
        
        Args:
            file_path: 视频文件路径
        
        Returns:
            时长（秒），获取失败返回 None
        """
        return self._get_audio_duration(file_path)  # ffprobe 同样适用于视频
    
    def is_ffmpeg_available(self) -> bool:
        """
        检查 FFmpeg 是否可用 - 验音器
        
        Returns:
            FFmpeg 是否已安装可用
        """
        try:
            result = subprocess.run(
                ["ffmpeg", "-version"],
                capture_output=True,
                timeout=5,
            )
            return result.returncode == 0
        except Exception:
            return False
