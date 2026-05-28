# -*- coding: utf-8 -*-
"""
炼丹炉 - 文档处理业务逻辑层 (Document Service)
承上启下：接收 API 请求，调用解析器和切分器，返回处理结果

犹如丹房管事，统筹安排丹方的解读和切分工作
"""
import logging
from typing import List, Dict, Any

from app.core.document.parser import ParserFactory
from app.core.splitter.text_splitter import TextSplitter
from app.core.config import settings

logger = logging.getLogger(__name__)


class DocumentService:
    """
    文档处理服务 - 丹方处理管事
    
    负责处理文档提取和切分的完整流程：
    1. 根据文件类型选择解析器
    2. 提取文档文本
    3. 按策略切分文本
    4. 返回提取结果和块列表
    """
    
    def __init__(self) -> None:
        """初始化文档处理服务"""
        self.splitter = TextSplitter(
            chunk_size=settings.chunk_size,
            overlap=settings.chunk_overlap,
        )
        logger.info("丹方处理管事就位")
    
    def extract_document(
        self,
        file_path: str,
        file_type: str,
        split_strategy: str = "fixed",
    ) -> Dict[str, Any]:
        """
        提取文档内容并切分 - 解读丹方
        
        完整的文档处理流程：
        1. 根据 file_type 获取对应解析器
        2. 解析文档提取完整文本
        3. 按指定策略切分文本
        4. 返回文本和块列表
        
        Args:
            file_path: 文件路径
            file_type: 文件类型 (docx/xlsx/md/txt/pdf)
            split_strategy: 切分策略 (fixed/paragraph/semantic)
        
        Returns:
            包含 text, chunks, chunk_count, file_type 的字典
        
        Raises:
            FileNotFoundError: 文件不存在
            ValueError: 文件类型不支持
            RuntimeError: 解析失败
        """
        logger.info(f"开始处理丹方: {file_path}, 类型: {file_type}")
        
        try:
            # 步骤 1: 获取解析器 - 选取合适丹炉
            parser = ParserFactory.get_parser(file_path, file_type)
            
            # 步骤 2: 提取文本 - 解读丹方
            logger.info(f"开始解读丹方内容...")
            text = parser.parse()
            
            if not text or not text.strip():
                logger.warning(f"丹方内容为空: {file_path}")
                return {
                    "text": "",
                    "chunks": [],
                    "chunk_count": 0,
                    "file_type": file_type,
                }
            
            # 步骤 3: 切分文本 - 将丹材切分
            logger.info(f"开始切分丹材 - 策略: {split_strategy}")
            raw_chunks = self.splitter.split(text, strategy=split_strategy)
            
            # 转换为标准格式
            chunks = [
                {
                    "content": chunk["content"],
                    "metadata": {
                        **chunk.get("metadata", {}),
                        "file_type": file_type,
                    }
                }
                for chunk in raw_chunks
            ]
            
            logger.info(
                f"丹方处理完毕: {file_path} - "
                f"原文 {len(text)} 字, 切分 {len(chunks)} 块"
            )
            
            return {
                "text": text,
                "chunks": chunks,
                "chunk_count": len(chunks),
                "file_type": file_type,
            }
            
        except FileNotFoundError:
            logger.error(f"丹方文件不存在: {file_path}")
            raise
        except ValueError as e:
            logger.error(f"丹方类型错误: {e}")
            raise
        except Exception as e:
            logger.error(f"丹方处理失败: {file_path}, 错误: {e}")
            raise RuntimeError(f"文档处理失败: {e}")
    
    def split_text(
        self,
        text: str,
        strategy: str = "fixed",
        chunk_size: int = None,
        overlap: int = None,
    ) -> List[Dict[str, Any]]:
        """
        切分文本 - 丹材切分服务
        
        对外提供纯文本切分服务，不经过文档解析。
        
        Args:
            text: 要切分的文本
            strategy: 切分策略
            chunk_size: 块大小（覆盖默认）
            overlap: 重叠大小（覆盖默认）
        
        Returns:
            文本块列表
        """
        logger.info(f"收到丹材切分请求 - 策略: {strategy}")
        
        try:
            chunks = self.splitter.split(
                text,
                strategy=strategy,
                chunk_size=chunk_size,
                overlap=overlap,
            )
            
            logger.info(f"丹材切分完毕 - {len(chunks)} 块")
            return chunks
            
        except Exception as e:
            logger.error(f"丹材切分失败: {e}")
            raise RuntimeError(f"文本切分失败: {e}")
    
    def get_supported_types(self) -> List[str]:
        """
        获取支持的文件类型 - 查询可炼之丹材
        
        Returns:
            支持的文件类型列表
        """
        return ParserFactory.supported_types()
