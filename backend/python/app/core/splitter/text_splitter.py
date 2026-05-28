# -*- coding: utf-8 -*-
"""
炼丹炉 - 文本切分模块 (Text Splitter)
将丹材（文档文本）切分为适合向量化的片段

支持三种切分策略：
1. fixed - 固定长度切分（按字符数）
2. paragraph - 按段落切分（按空行分隔）
3. semantic - 语义切分（按标题层级）
犹如将大块丹材切分为适合入炉炼化的小块
"""
import logging
import re
from typing import List, Dict, Any, Literal

logger = logging.getLogger(__name__)


class TextSplitter:
    """
    文本切分器 - 丹材切割机
    
    将长文本切分为固定大小的块，以便向量化处理。
    支持多种切分策略，可根据丹方类型灵活选择。
    
    Attributes:
        chunk_size: 块大小（字符数）
        overlap: 块间重叠大小
    
    Example:
        splitter = TextSplitter(chunk_size=500, overlap=50)
        chunks = splitter.split(long_text, strategy="fixed")
    """
    
    def __init__(
        self,
        chunk_size: int = 500,
        overlap: int = 50,
    ) -> None:
        """
        初始化文本切分器
        
        Args:
            chunk_size: 每块的最大字符数，默认 500
            overlap: 相邻块的重叠字符数，默认 50
        """
        self.chunk_size = chunk_size
        self.overlap = overlap
        
        logger.debug(
            f"丹材切割机初始化 - 块大小: {chunk_size}, 重叠: {overlap}"
        )
    
    def split(
        self,
        text: str,
        strategy: Literal["fixed", "paragraph", "semantic"] = "fixed",
        chunk_size: int = None,
        overlap: int = None,
    ) -> List[Dict[str, Any]]:
        """
        切分文本 - 将丹材切分为小块
        
        根据指定策略将文本切分为多个块，每块包含内容和元数据。
        
        Args:
            text: 要切分的原始文本
            strategy: 切分策略
                - fixed: 固定长度切分
                - paragraph: 按段落切分
                - semantic: 按语义边界（标题层级）切分
            chunk_size: 覆盖默认块大小
            overlap: 覆盖默认重叠大小
        
        Returns:
            文本块列表，每项为 dict: {"content": str, "metadata": dict}
        
        Raises:
            ValueError: 文本为空或策略不支持
        """
        if not text or not text.strip():
            raise ValueError("待切分文本为空")
        
        cs = chunk_size or self.chunk_size
        ov = overlap or self.overlap
        
        logger.info(f"开始切分丹材 - 策略: {strategy}, 大小: {cs}, 重叠: {ov}")
        
        if strategy == "fixed":
            chunks = self._split_fixed(text, cs, ov)
        elif strategy == "paragraph":
            chunks = self._split_paragraph(text, cs)
        elif strategy == "semantic":
            chunks = self._split_semantic(text, cs)
        else:
            raise ValueError(f"不支持的切分策略: {strategy}")
        
        logger.info(f"丹材切分完毕 - 共 {len(chunks)} 块")
        return chunks
    
    def _split_fixed(
        self, text: str, chunk_size: int, overlap: int
    ) -> List[Dict[str, Any]]:
        """
        固定长度切分 - 等长切割
        
        按固定字符数切分文本，相邻块之间有重叠区域，
        确保语义连贯性不被破坏。
        
        Args:
            text: 原始文本
            chunk_size: 块大小
            overlap: 重叠大小
        
        Returns:
            文本块列表
        """
        chunks: List[Dict[str, Any]] = []
        start = 0
        text_len = len(text)
        
        while start < text_len:
            # 计算当前块的结束位置
            end = min(start + chunk_size, text_len)
            
            # 如果不是最后一块，尝试在句子或单词边界切分
            if end < text_len:
                # 尝试在标点符号后切分
                for sep in ["\n", "。", "；", "!", "?", ".", ";"]:
                    pos = text.rfind(sep, start, end)
                    if pos > start + chunk_size // 2:  # 至少达到半块才切
                        end = pos + 1
                        break
            
            chunk_text = text[start:end].strip()
            if chunk_text:
                chunks.append({
                    "content": chunk_text,
                    "metadata": {
                        "strategy": "fixed",
                        "chunk_index": len(chunks),
                        "char_start": start,
                        "char_end": end,
                    }
                })
            
            # 下一块的起始位置（考虑重叠）
            start = end - overlap if end < text_len else end
            
            # 防止无限循环
            if start <= 0 or (len(chunks) > 0 and start >= text_len):
                break
        
        return chunks
    
    def _split_paragraph(
        self, text: str, chunk_size: int
    ) -> List[Dict[str, Any]]:
        """
        按段落切分 - 自然段落分割
        
        按空行分隔段落，然后将相邻段落合并为不超过 chunk_size 的块。
        
        Args:
            text: 原始文本
            chunk_size: 最大块大小
        
        Returns:
            文本块列表
        """
        # 按空行分段
        paragraphs = [p.strip() for p in re.split(r'\n\s*\n', text) if p.strip()]
        
        chunks: List[Dict[str, Any]] = []
        current_chunk: List[str] = []
        current_size = 0
        start_para_idx = 0
        
        for idx, para in enumerate(paragraphs):
            para_len = len(para)
            
            # 如果当前段落本身就超过 chunk_size，直接用固定长度切分
            if para_len > chunk_size:
                # 先保存当前累积的内容
                if current_chunk:
                    chunks.append({
                        "content": "\n\n".join(current_chunk),
                        "metadata": {
                            "strategy": "paragraph",
                            "chunk_index": len(chunks),
                            "paragraph_range": f"{start_para_idx}-{idx-1}",
                        }
                    })
                    current_chunk = []
                    current_size = 0
                    start_para_idx = idx
                
                # 长段落使用固定长度切分
                sub_chunks = self._split_fixed(para, chunk_size, self.overlap)
                for sc in sub_chunks:
                    sc["metadata"]["strategy"] = "paragraph"
                    sc["metadata"]["from_long_paragraph"] = True
                    sc["metadata"]["paragraph_index"] = idx
                    chunks.append(sc)
                start_para_idx = idx + 1
                continue
            
            # 如果加入当前段落后超过 chunk_size，先保存当前块
            if current_size + para_len + 2 > chunk_size and current_chunk:
                chunks.append({
                    "content": "\n\n".join(current_chunk),
                    "metadata": {
                        "strategy": "paragraph",
                        "chunk_index": len(chunks),
                        "paragraph_range": f"{start_para_idx}-{idx-1}",
                    }
                })
                current_chunk = []
                current_size = 0
                start_para_idx = idx
            
            current_chunk.append(para)
            current_size += para_len + 2  # +2 for \n\n
        
        # 保存最后一块
        if current_chunk:
            chunks.append({
                "content": "\n\n".join(current_chunk),
                "metadata": {
                    "strategy": "paragraph",
                    "chunk_index": len(chunks),
                    "paragraph_range": f"{start_para_idx}-{len(paragraphs)-1}",
                }
            })
        
        return chunks
    
    def _split_semantic(
        self, text: str, chunk_size: int
    ) -> List[Dict[str, Any]]:
        """
        语义切分 - 按标题层级切分
        
        根据 Markdown 风格的标题（#, ##, ###）或常见标题模式切分文本，
        确保每个语义单元完整。
        
        Args:
            text: 原始文本
            chunk_size: 最大块大小
        
        Returns:
            文本块列表
        """
        # 匹配 Markdown 标题或常见标题模式
        # 支持: # 标题, ## 标题, 第X章, 第X节, 一、, （一）, 1. 等
        heading_patterns = [
            r'^(#{1,6}\s+.+)$',           # Markdown 标题
            r'^(第[一二三四五六七八九十\d]+[章节节]\s*.+)$',  # 第X章/节
            r'^([一二三四五六七八九十]、[\s\S]+?)$',        # 一、二、
            r'^(\([一二三四五六七八九十\d]\)\s*.+)$',      # (一) (1)
            r'^(\d+[\.、]\s*[^\d].+)$',                    # 1. 1、
        ]
        
        combined_pattern = "|".join(f"(?:{p})" for p in heading_patterns)
        
        # 按标题切分
        sections = re.split(f'(?m)(?={combined_pattern})', text)
        sections = [s.strip() for s in sections if s.strip()]
        
        chunks: List[Dict[str, Any]] = []
        current_chunk: List[str] = []
        current_size = 0
        current_title = ""
        start_section_idx = 0
        
        for idx, section in enumerate(sections):
            section_len = len(section)
            
            # 提取标题
            title_match = re.match(combined_pattern, section, re.MULTILINE)
            section_title = title_match.group(0) if title_match else f"第{idx+1}段"
            
            # 如果单节超过 chunk_size，内部再切分
            if section_len > chunk_size:
                if current_chunk:
                    chunks.append({
                        "content": "\n\n".join(current_chunk),
                        "metadata": {
                            "strategy": "semantic",
                            "chunk_index": len(chunks),
                            "title": current_title or "无标题",
                            "section_range": f"{start_section_idx}-{idx-1}",
                        }
                    })
                    current_chunk = []
                    current_size = 0
                
                sub_chunks = self._split_fixed(section, chunk_size, self.overlap)
                for sc in sub_chunks:
                    sc["metadata"]["strategy"] = "semantic"
                    sc["metadata"]["title"] = section_title
                    sc["metadata"]["from_long_section"] = True
                    chunks.append(sc)
                
                start_section_idx = idx + 1
                current_title = ""
                continue
            
            # 如果加入后超过 chunk_size，先保存
            if current_size + section_len + 2 > chunk_size and current_chunk:
                chunks.append({
                    "content": "\n\n".join(current_chunk),
                    "metadata": {
                        "strategy": "semantic",
                        "chunk_index": len(chunks),
                        "title": current_title or "无标题",
                        "section_range": f"{start_section_idx}-{idx-1}",
                    }
                })
                current_chunk = []
                current_size = 0
                start_section_idx = idx
                current_title = ""
            
            current_chunk.append(section)
            current_size += section_len + 2
            
            # 记录第一个标题
            if not current_title and title_match:
                current_title = section_title
        
        # 保存最后一块
        if current_chunk:
            chunks.append({
                "content": "\n\n".join(current_chunk),
                "metadata": {
                    "strategy": "semantic",
                    "chunk_index": len(chunks),
                    "title": current_title or "无标题",
                    "section_range": f"{start_section_idx}-{len(sections)-1}",
                }
            })
        
        return chunks
    
    def count_chunks(self, text: str, strategy: str = "fixed") -> int:
        """
        估算切分后的块数量 - 预估丹材份数
        
        Args:
            text: 原始文本
            strategy: 切分策略
        
        Returns:
            预估块数量
        """
        chunks = self.split(text, strategy=strategy)
        return len(chunks)
