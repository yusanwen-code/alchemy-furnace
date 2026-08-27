"""Nuwa distillation API."""
import logging

from fastapi import APIRouter, Header, HTTPException, status
from fastapi.responses import Response

from app.models.schemas import DistillRequest, DistillResponse, SkillExportRequest
from app.services.baidu_baike_research_provider import BaiduBaikeResearchProvider
from app.services.duckduckgo_research_provider import DuckDuckGoResearchProvider
from app.services.nuwa_distillation_service import DistillationError, NuwaDistillationService
from app.services.qianfan_web_search_provider import QianfanWebSearchProvider
from app.services.research_orchestrator import ResearchOrchestrator
from app.services.skill_export import (
    SkillExportError,
    build_exportable,
    build_zip_bytes,
    zip_filename,
)
from app.services.web_document_fetcher import WebDocumentFetcher
from app.services.wikipedia_research_provider import WikipediaResearchProvider

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/distillation", tags=["女娲蒸馏"])

# 国内优先、国际限时(6s 预算)的组合器;各提供者无注入 client 时按请求创建
# 短生命周期 httpx.Client context manager,模块级不持有连接。
distillation_service = NuwaDistillationService(
    ResearchOrchestrator(
        domestic=[
            BaiduBaikeResearchProvider(fetcher=WebDocumentFetcher(timeout=3)),
            QianfanWebSearchProvider(),
        ],
        global_providers=[
            WikipediaResearchProvider(timeout=4),
            DuckDuckGoResearchProvider(fetcher=WebDocumentFetcher(timeout=4)),
        ],
        global_budget_seconds=6,
        circuit_breaker_seconds=600,
    )
)


@router.post("/nuwa", response_model=DistillResponse, summary="从公开资料蒸馏金丹草稿")
def distill_nuwa(
    request: DistillRequest,
    x_request_id: str | None = Header(default=None),
) -> DistillResponse:
    try:
        return DistillResponse(
            **distillation_service.distill(
                **request.model_dump(), request_id=x_request_id
            )
        )
    except DistillationError as exc:
        # 稳定错误协议: detail 为结构化对象,Go 网关按 code/stage/retryable 透传。
        # model_not_configured 属客户端配置缺失(400),其余可重试 503、不可重试 422。
        if exc.code == "model_not_configured":
            http_status = status.HTTP_400_BAD_REQUEST
        else:
            http_status = (
                status.HTTP_503_SERVICE_UNAVAILABLE
                if exc.retryable
                else status.HTTP_422_UNPROCESSABLE_ENTITY
            )
        raise HTTPException(
            status_code=http_status,
            detail={
                "code": exc.code,
                "stage": exc.stage,
                "message": exc.message,
                "retryable": exc.retryable,
                "details": exc.details,
            },
        ) from exc
    except ValueError as exc:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(exc)) from exc
    except Exception as exc:
        logger.exception("女娲蒸馏失败")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail={
                "code": "distillation_internal_error",
                "stage": "unknown",
                "message": "蒸馏服务内部错误",
                "retryable": False,
                "details": {},
            },
        ) from exc


@router.post(
    "/skill-export",
    summary="生成 Skill 导出 ZIP 包(Codex/Claude)",
    response_class=Response,
)
def skill_export(request: SkillExportRequest) -> Response:
    """用已保存金丹的规范化投影字段渲染确定性 ZIP 包。

    只接收结构化投影(Go 网关已完成权限与字段重校验);无状态、不落库、
    不读取任何 API Key。内容校验失败返回 422(skill_export_invalid,不可重试)。
    """
    try:
        skill = build_exportable(
            name=request.name,
            description=request.description,
            skill_schema=request.skill_schema,
            tags=request.tags,
            sources=[
                {"title": s.title, "url": s.url, "dimension": s.dimension}
                for s in request.sources
            ],
            generated_at=request.generated_at,
            evidence_level=request.evidence_level,
        )
        payload = build_zip_bytes(skill, request.format)
        filename = zip_filename(skill, request.format)
    except SkillExportError as exc:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail={
                "code": exc.code,
                "stage": exc.stage,
                "message": exc.message,
                "retryable": exc.retryable,
                "details": exc.details,
            },
        ) from exc
    return Response(
        content=payload,
        media_type="application/zip",
        headers={"Content-Disposition": f'attachment; filename="{filename}"'},
    )
