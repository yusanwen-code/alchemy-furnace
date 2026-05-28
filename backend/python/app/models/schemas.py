# -*- coding: utf-8 -*-
"""
炼丹炉 - Pydantic 数据模型 (Data Schemas)
定义所有请求与响应的数据契约，犹如炼丹之丹方模板
"""
from typing import List, Dict, Any, Optional, Literal
from pydantic import BaseModel, Field, field_validator


# ==================== 通用响应模型 ====================

class BaseResponse(BaseModel):
    """基础响应模型 - 万法归宗"""
    code: int = Field(default=0, description="状态码：0为成功，非0为失败")
    message: str = Field(default="success", description="响应消息")
    data: Optional[Any] = Field(default=None, description="响应数据")


class PaginationParams(BaseModel):
    """分页参数 - 分批炼制"""
    page: int = Field(default=1, ge=1, description="页码")
    page_size: int = Field(default=20, ge=1, le=100, description="每页数量")


# ==================== 文档处理相关模型 ====================

class DocumentExtractRequest(BaseModel):
    """
    文档提取请求 - 呈上丹方，请求解读
    
    Attributes:
        file_path: 丹方文件在服务器上的存储路径
        file_type: 丹方类型 (doc/docx/xls/xlsx/md/txt/pdf)
    """
    file_path: str = Field(..., description="文件路径", examples=["/app/uploads/test.docx"])
    file_type: str = Field(..., description="文件类型", examples=["docx"])

    @field_validator("file_type")
    @classmethod
    def validate_file_type(cls, v: str) -> str:
        """验证文件类型是否受支持 - 检验丹方是否可炼"""
        allowed = {"doc", "docx", "xls", "xlsx", "md", "txt", "pdf"}
        if v.lower() not in allowed:
            raise ValueError(f"不支持的丹方类型「{v}」，仅支持: {', '.join(allowed)}")
        return v.lower()


class ChunkInfo(BaseModel):
    """
    文本块信息 - 丹材切片
    
    Attributes:
        content: 切片内容
        metadata: 元数据（如页码、sheet名等）
    """
    content: str = Field(..., description="文本块内容")
    metadata: Dict[str, Any] = Field(default_factory=dict, description="元数据")


class DocumentExtractResponse(BaseModel):
    """
    文档提取响应 - 丹方解读完毕
    
    Attributes:
        text: 提取的完整文本
        chunks: 切分后的文本块列表
        chunk_count: 块数量
        file_type: 文件类型
    """
    text: str = Field(..., description="提取的完整文本")
    chunks: List[ChunkInfo] = Field(default_factory=list, description="切分后的文本块")
    chunk_count: int = Field(default=0, description="块数量")
    file_type: str = Field(default="", description="文件类型")


class DocumentSplitRequest(BaseModel):
    """
    文档切分请求 - 请求切分丹材
    
    Attributes:
        text: 要切分的文本
        strategy: 切分策略 (fixed/paragraph/semantic)
        chunk_size: 固定长度切分时的块大小
        overlap: 固定长度切分时的重叠大小
    """
    text: str = Field(..., min_length=1, description="要切分的文本")
    strategy: Literal["fixed", "paragraph", "semantic"] = Field(
        default="fixed", description="切分策略"
    )
    chunk_size: int = Field(default=500, ge=100, le=2000, description="块大小")
    overlap: int = Field(default=50, ge=0, le=500, description="重叠大小")


# ==================== 向量相关模型 ====================

class VectorIngestChunk(BaseModel):
    """
    向量化入库的单个块 - 待炼化的丹材
    
    Attributes:
        content: 文本内容
        metadata: 附加元数据
    """
    content: str = Field(..., description="文本内容")
    metadata: Dict[str, Any] = Field(default_factory=dict, description="元数据")


class VectorIngestRequest(BaseModel):
    """
    向量化入库请求 - 开始炼丹
    
    Attributes:
        pill_id: 金丹ID（知识库ID）
        recipe_id: 丹方ID（文档ID）
        chunks: 要入库的文本块列表
    """
    pill_id: int = Field(..., gt=0, description="金丹ID")
    recipe_id: int = Field(..., gt=0, description="丹方ID")
    chunks: List[VectorIngestChunk] = Field(..., min_length=1, description="文本块列表")


class VectorIngestResponse(BaseModel):
    """向量化入库响应 - 炼丹完毕"""
    pill_id: int = Field(..., description="金丹ID")
    recipe_id: int = Field(..., description="丹方ID")
    vector_count: int = Field(..., description="成功入炉的向量数量")
    message: str = Field(default="", description="状态消息")


class VectorSearchRequest(BaseModel):
    """
    向量搜索请求 - 寻丹之术
    
    Attributes:
        pill_ids: 要搜索的金丹ID列表
        query: 查询文本
        top_k: 返回结果数量
    """
    pill_ids: List[int] = Field(..., min_length=1, description="金丹ID列表")
    query: str = Field(..., min_length=1, description="查询文本")
    top_k: int = Field(default=5, ge=1, le=50, description="返回结果数量")


class VectorSearchResult(BaseModel):
    """
    向量搜索结果 - 寻得的丹材
    
    Attributes:
        content: 匹配文本内容
        score: 相似度分数
        metadata: 关联元数据
        pill_id: 来源金丹ID
        recipe_id: 来源丹方ID
    """
    content: str = Field(..., description="匹配文本内容")
    score: float = Field(..., description="相似度分数")
    metadata: Dict[str, Any] = Field(default_factory=dict, description="元数据")
    pill_id: int = Field(..., description="金丹ID")
    recipe_id: int = Field(..., description="丹方ID")


class VectorSearchResponse(BaseModel):
    """向量搜索响应"""
    results: List[VectorSearchResult] = Field(default_factory=list, description="搜索结果")
    total: int = Field(default=0, description="结果数量")
    query: str = Field(default="", description="查询文本")


# ==================== 对话相关模型 ====================

class ChatMessage(BaseModel):
    """
    对话消息 - 道人与修士的问答
    
    Attributes:
        role: 角色 (system/user/assistant)
        content: 消息内容
    """
    role: Literal["system", "user", "assistant"] = Field(..., description="角色")
    content: str = Field(..., description="消息内容")


class ChatCompletionRequest(BaseModel):
    """
    对话请求 - 求道之问
    
    Attributes:
        messages: 消息历史
        pill_ids: 要引用的金丹ID列表
        model: 使用的LLM模型
        temperature: 温度参数
        max_tokens: 最大token数
        stream: 是否流式返回
    """
    messages: List[ChatMessage] = Field(..., min_length=1, description="消息历史")
    pill_ids: List[int] = Field(default_factory=list, description="金丹ID列表")
    model: str = Field(default="gpt-4o", description="LLM模型")
    temperature: float = Field(default=0.7, ge=0.0, le=2.0, description="温度参数")
    max_tokens: int = Field(default=4096, ge=1, le=8192, description="最大token数")
    stream: bool = Field(default=False, description="是否流式返回")


class ChatSource(BaseModel):
    """
    对话引用来源 - 金丹出处
    
    Attributes:
        content: 引用的内容
        score: 相似度分数
        metadata: 来源元数据
    """
    content: str = Field(..., description="引用内容")
    score: float = Field(..., description="相似度分数")
    metadata: Dict[str, Any] = Field(default_factory=dict, description="元数据")


class ChatCompletionResponse(BaseModel):
    """
    对话响应 - 道人应答
    
    Attributes:
        content: AI生成的回答
        sources: 引用的知识来源
        model: 使用的模型
        usage: token用量
    """
    content: str = Field(..., description="AI回答")
    sources: List[ChatSource] = Field(default_factory=list, description="引用来源")
    model: str = Field(default="", description="使用模型")
    usage: Dict[str, int] = Field(default_factory=dict, description="token用量")


# ==================== 媒体处理相关模型 ====================

class TranscribeRequest(BaseModel):
    """
    音频转录请求 - 听音辨道
    
    Attributes:
        file_path: 音频文件路径
        language: 语言代码 (可选)
    """
    file_path: str = Field(..., description="音频文件路径")
    language: Optional[str] = Field(default=None, description="语言代码，如 zh/en")


class TranscribeResponse(BaseModel):
    """音频转录响应"""
    text: str = Field(..., description="转录文本")
    duration: Optional[float] = Field(default=None, description="音频时长（秒）")
    language: Optional[str] = Field(default=None, description="检测到的语言")


class ExtractSubtitlesRequest(BaseModel):
    """
    视频提取字幕请求 - 从影像中提取真言
    
    Attributes:
        file_path: 视频文件路径
        language: 语言代码
        extract_audio_only: 是否仅提取音频
    """
    file_path: str = Field(..., description="视频文件路径")
    language: Optional[str] = Field(default=None, description="语言代码")
    extract_audio_only: bool = Field(default=False, description="是否仅提取音频")


class ExtractSubtitlesResponse(BaseModel):
    """视频提取字幕响应"""
    text: str = Field(..., description="提取的字幕文本")
    audio_path: Optional[str] = Field(default=None, description="提取的音频路径")
    duration: Optional[float] = Field(default=None, description="视频时长（秒）")


# ==================== 系统相关模型 ====================

class HealthResponse(BaseModel):
    """健康检查响应 - 探查炼丹炉状态"""
    status: str = Field(default="ok", description="服务状态")
    version: str = Field(default="1.0.0", description="版本号")
    components: Dict[str, str] = Field(
        default_factory=dict, description="各组件状态"
    )


class ErrorResponse(BaseModel):
    """错误响应 - 炼丹失败之因"""
    code: int = Field(..., description="错误码")
    message: str = Field(..., description="错误消息")
    detail: Optional[str] = Field(default=None, description="详细错误信息")
