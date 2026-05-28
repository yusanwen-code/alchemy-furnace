# -*- coding: utf-8 -*-
"""
炼丹炉 - 配置管理模块 (Configuration)
以 pydantic_settings 管理环境变量，犹如炼丹之天时地利
"""
import os
from typing import Optional
from pydantic_settings import BaseSettings, SettingsConfigDict
from pydantic import Field


class Settings(BaseSettings):
    """
    炼丹炉全局配置 - 天地法则
    
    所有配置项均可通过环境变量覆盖，环境变量名大写。
    例如: OPENAI_API_KEY, QDRANT_HOST 等。
    """
    
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )
    
    # ==================== 服务配置 ====================
    app_name: str = Field(default="炼丹炉 RAG 引擎", description="应用名称")
    app_version: str = Field(default="1.0.0", description="应用版本")
    debug: bool = Field(default=False, description="调试模式")
    
    # ==================== 服务器配置 ====================
    host: str = Field(default="0.0.0.0", description="监听地址")
    port: int = Field(default=8000, description="监听端口")
    
    # ==================== CORS 配置 ====================
    cors_origins: list[str] = Field(
        default=["*"], description="允许的跨域来源"
    )
    
    # ==================== Qdrant 向量数据库配置 ====================
    qdrant_host: str = Field(default="localhost", description="Qdrant 主机")
    qdrant_port: int = Field(default=6333, description="Qdrant 端口")
    qdrant_collection: str = Field(
        default="elixir_pills", description="Qdrant 集合名称（金丹阁）"
    )
    qdrant_api_key: Optional[str] = Field(
        default=None, description="Qdrant API 密钥"
    )
    
    # ==================== LLM 配置 ====================
    openai_api_key: str = Field(default="", description="OpenAI API 密钥")
    openai_base_url: str = Field(
        default="https://api.openai.com/v1", description="OpenAI 兼容接口地址"
    )
    default_model: str = Field(
        default="gpt-4o", description="默认 LLM 模型"
    )
    
    # ==================== Embedding 配置 ====================
    embedding_model: str = Field(
        default="text-embedding-3-small", description="Embedding 模型"
    )
    embedding_dimensions: int = Field(
        default=1536, description="Embedding 向量维度"
    )
    
    # ==================== 文本切分配置 ====================
    chunk_size: int = Field(default=500, description="默认块大小")
    chunk_overlap: int = Field(default=50, description="默认重叠大小")
    top_k: int = Field(default=5, description="默认检索数量")
    
    # ==================== 文件上传配置 ====================
    upload_dir: str = Field(default="/app/uploads", description="上传目录")
    max_file_size: int = Field(
        default=100 * 1024 * 1024, description="最大文件大小（字节）"
    )
    
    # ==================== 日志配置 ====================
    log_level: str = Field(default="INFO", description="日志级别")
    log_format: str = Field(
        default="%(asctime)s [%(levelname)s] %(name)s - %(message)s",
        description="日志格式"
    )

    @property
    def qdrant_url(self) -> str:
        """获取 Qdrant 连接地址"""
        return f"http://{self.qdrant_host}:{self.qdrant_port}"

    @property
    def openai_api_key_valid(self) -> bool:
        """检查 OpenAI API 密钥是否有效配置"""
        return bool(self.openai_api_key and self.openai_api_key.startswith("sk-"))


# ==================== 全局配置实例 ====================
# 犹如炼丹之全局天时，各处皆可用之
settings = Settings()
