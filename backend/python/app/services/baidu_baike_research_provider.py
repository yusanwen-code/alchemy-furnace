"""百度百科直达页：免密国内基线源。

国际资料不可达时仍需给出有依据的草稿：直接请求 ``baike.baidu.com/item/{subject}``
（URL 编码），只有返回的最终 URL 仍属于 baike.baidu.com、正文达到
``MIN_CHARACTERS`` 且摘录中出现主题词时才接受为有效文档；其余情况一律判为
"empty"（内容不合格），请求失败判为 "blocked"（该源当前不可达，触发熔断）。

安全验证(challenge)页按 HTTP 200 返回但内容为反爬验证页，特征词命中即判为
"blocked"（提供者被拦截，可熔断），绝不能当作"资料不足"或证据。
"""
from __future__ import annotations

import json
import re
from urllib.parse import quote, urlparse

from app.services.research_provider import (
    EvidenceLevel,
    ResearchAttempt,
    ResearchDocument,
    ResearchProvider,
    ResearchReport,
)
from app.services.web_document_fetcher import WebDocumentFetcher

BAIDU_BAIKE_HOST = "baike.baidu.com"
MIN_CHARACTERS = 800
MAX_EXCERPT_CHARACTERS = 4000
# 百度安全验证/反爬页特征：命中即提供者被拦截（熔断），与"内容不足"区分开
CHALLENGE_MARKERS = ("百度安全验证", "安全验证", "验证码", "captcha")


def _mentions_subject(excerpt: str, subject: str) -> bool:
    """主题词整体或分词出现在摘录中（中文无空格分词，按整词 + "·" 分隔片段）。"""
    if subject in excerpt:
        return True
    normalized = " ".join(subject.split())
    if normalized and normalized in excerpt:
        return True
    return any(
        part.strip() and part.strip() in excerpt
        for part in normalized.replace("·", " ").split()
        if len(part.strip()) >= 2
    )


def _looks_like_challenge(excerpt: str) -> bool:
    lowered = excerpt.lower()
    return any(marker in lowered for marker in CHALLENGE_MARKERS)


def _disambiguation_target(raw_html: str, subject: str) -> str | None:
    """从多义词选择页的 PAGE_DATA 中选择第一义项，固定拼接同源 baike 地址。

    只接受正整数 lemmaId（排除字符串 URL、布尔与负数）；不信任页面提供的
    任何 href/域名/脚本 URL。返回 None 表示没有可跟随的义项。
    """
    match = re.search(
        r"window\.PAGE_DATA\s*=\s*(\{.*?\})\s*;?\s*</script>",
        raw_html,
        re.DOTALL,
    )
    if not match:
        return None
    try:
        payload = json.loads(match.group(1))
    except (TypeError, json.JSONDecodeError):
        return None
    lemmas = ((payload.get("navigation") or {}).get("lemmas") or [])
    candidates = [
        item for item in lemmas
        if isinstance(item, dict)
        and isinstance(item.get("lemmaId"), int)
        and not isinstance(item.get("lemmaId"), bool)
        and item["lemmaId"] > 0
    ]
    if not candidates:
        return None

    def numeric_rank(item: dict) -> int:
        value = item.get("rank")
        return value if isinstance(value, int) and not isinstance(value, bool) else 1_000_000

    selected = min(candidates, key=numeric_rank)
    return (
        f"https://{BAIDU_BAIKE_HOST}/item/"
        f"{quote(subject, safe='')}/{selected['lemmaId']}"
    )


class BaiduBaikeResearchProvider(ResearchProvider):
    """Key-free mainland baseline: direct baike page, validated content."""

    provider_id = "baidu_baike"

    def __init__(self, fetcher=None, timeout: float = 3.0) -> None:
        self.fetcher = fetcher or WebDocumentFetcher(timeout=timeout)

    def collect(
        self,
        subject: str,
        brief: str,
        locale: str = "zh-CN",
        credentials=None,
    ) -> ResearchReport:
        url = f"https://{BAIDU_BAIKE_HOST}/item/{quote(subject)}"
        result = self.fetcher.fetch(url)
        # 多义词选择页：正文过短但携带受限原始 HTML 时，只允许通过同源
        # PAGE_DATA 义项元数据选择第一义项，固定拼接 baike 地址后原路再抓
        # 一次；二次结果仍走下方 challenge/host/长度/主题全套校验。
        if (
            result.status != "ok"
            and result.reason == "text_too_short"
            and result.raw_html
        ):
            target = _disambiguation_target(result.raw_html, subject)
            if target is not None:
                result = self.fetcher.fetch(target)
        if result.status != "ok":
            return ResearchReport(
                [],
                [ResearchAttempt(self.provider_id, "blocked", 0, 0, result.reason)],
                EvidenceLevel.INSUFFICIENT,
            )
        excerpt = result.excerpt or ""
        if _looks_like_challenge(excerpt):
            return ResearchReport(
                [],
                [ResearchAttempt(self.provider_id, "blocked", 0, 0, "challenge")],
                EvidenceLevel.INSUFFICIENT,
            )
        final_host = urlparse(result.url).hostname
        if (
            final_host != BAIDU_BAIKE_HOST
            or len(excerpt) < MIN_CHARACTERS
            or not _mentions_subject(excerpt, subject)
        ):
            return ResearchReport(
                [],
                [ResearchAttempt(self.provider_id, "empty", 0, 0, "invalid_content")],
                EvidenceLevel.INSUFFICIENT,
            )
        document = ResearchDocument(subject, result.url, excerpt[:MAX_EXCERPT_CHARACTERS], "reference")
        return ResearchReport(
            [document],
            [ResearchAttempt(self.provider_id, "ok", 1, 1, None)],
            EvidenceLevel.LIMITED,
        )
