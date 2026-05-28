# -*- coding: utf-8 -*-
"""
================================================================================
炼丹炉 RAG 引擎 - FastAPI 入口
================================================================================
金丹 = 知识库，丹方 = 文档文件，道人 = AI Agent，炼丹 = RAG 处理流程

启动命令:
    uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload

Docker 部署:
    docker run -p 8000:8000 alchemy-furnace/python-rag

API 文档:
    启动后访问 http://localhost:8000/docs (Swagger UI)
    或 http://localhost:8000/redoc (ReDoc)
================================================================================
"""
import logging
import sys

from fastapi import FastAPI, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager

from app.core.config import settings
from app.core.vectorstore.qdrant_store import QdrantStore

# ==================== 日志配置 ====================

# 配置根日志器 - 炼丹炉之耳目
logging.basicConfig(
    level=getattr(logging, settings.log_level.upper(), logging.INFO),
    format=settings.log_format,
    handlers=[
        logging.StreamHandler(sys.stdout),
    ],
)
logger = logging.getLogger("alchemy-furnace")


# ==================== 生命周期管理 ====================

@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    应用生命周期管理 - 炼丹炉启停
    
    启动时:
        1. 连接 Qdrant
        2. 初始化 Collection（金丹阁）
    关闭时:
        1. 清理资源
    """
    logger.info("=" * 60)
    logger.info("炼丹炉 RAG 引擎启动中...")
    logger.info(f"版本: {settings.app_version}")
    logger.info(f"环境: {'调试' if settings.debug else '生产'}")
    logger.info("=" * 60)
    
    # 启动时初始化 Qdrant Collection - 筑建金丹阁
    try:
        logger.info("正在连接金丹阁(Qdrant)...")
        qdrant_store = QdrantStore(
            host=settings.qdrant_host,
            port=settings.qdrant_port,
            collection_name=settings.qdrant_collection,
        )
        qdrant_store.init_collection()
        logger.info("金丹阁初始化完毕")
        
        # 将 store 实例存入 app.state 供全局使用
        app.state.qdrant_store = qdrant_store
        
    except Exception as e:
        logger.error(f"金丹阁连接失败: {e}")
        logger.warning("炼丹炉将继续运行，但向量功能可能不可用")
    
    logger.info("炼丹炉启动完毕，开始接客！")
    
    yield  # 应用运行期间
    
    # 关闭时清理
    logger.info("炼丹炉正在关闭...")
    logger.info("炼丹炉已安全关闭")


# ==================== FastAPI 应用实例 ====================

app = FastAPI(
    title=settings.app_name,
    description="""
    炼丹炉 RAG 引擎 - 以道教炼丹为概念的检索增强生成系统。
    
    ## 核心概念
    - **金丹**: 知识库（Pill）
    - **丹方**: 文档文件（Recipe）
    - **道人**: AI Agent
    - **炼丹**: RAG 处理流程
    
    ## 功能模块
    - 文档处理：提取 docx/xlsx/md/txt/pdf 文本
    - 向量管理：向量化入库、搜索、删除
    - 对话：非流式和 SSE 流式对话
    - 媒体处理：音频转录、视频字幕提取
    """,
    version=settings.app_version,
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json",
    lifespan=lifespan,
)

# ==================== CORS 配置 ====================

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# ==================== 全局异常处理 ====================

@app.exception_handler(Exception)
async def global_exception_handler(request, exc):
    """
    全局异常处理 - 炼丹出错之应对
    
    捕获所有未处理的异常，返回统一格式的错误响应。
    """
    logger.error(f"炼丹出错: {exc}", exc_info=True)
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={
            "code": -1,
            "message": "炼丹炉内部出错",
            "detail": str(exc) if settings.debug else "请稍后重试",
        },
    )


# ==================== 路由注册 ====================

from app.api import documents, vectors, chat, media

# 注册所有路由 - 开启各殿之门
app.include_router(documents.router, prefix="/api/v1")
app.include_router(vectors.router, prefix="/api/v1")
app.include_router(chat.router, prefix="/api/v1")
app.include_router(media.router, prefix="/api/v1")

# ==================== 根路由 ====================

@app.get("/", tags=["根路径"])
async def root():
    """根路径 - 炼丹炉总览"""
    return {
        "name": settings.app_name,
        "version": settings.app_version,
        "description": "炼丹炉 RAG 引擎 - 以道教炼丹为概念的检索增强生成系统",
        "docs": "/docs",
        "endpoints": {
            "documents": "/api/v1/documents",
            "vectors": "/api/v1/vectors",
            "chat": "/api/v1/chat",
            "media": "/api/v1/media",
        },
    }


@app.get("/health", tags=["系统"])
async def health_check():
    """
    健康检查 - 探查炼丹炉状态
    
    检查服务本身及各组件（Qdrant）的健康状态。
    """
    components = {"api": "ok"}
    
    # 检查 Qdrant
    try:
        qdrant_store = QdrantStore(
            host=settings.qdrant_host,
            port=settings.qdrant_port,
        )
        qdrant_store.client.get_collections()
        components["qdrant"] = "ok"
    except Exception as e:
        components["qdrant"] = f"error: {e}"
        logger.warning(f"Qdrant 健康检查失败: {e}")
    
    overall = "ok" if all(v == "ok" for v in components.values()) else "degraded"
    
    return {
        "status": overall,
        "version": settings.app_version,
        "components": components,
    }


# ==================== 启动入口 ====================

if __name__ == "__main__":
    import uvicorn
    
    uvicorn.run(
        "app.main:app",
        host=settings.host,
        port=settings.port,
        reload=settings.debug,
        log_level=settings.log_level.lower(),
    )
