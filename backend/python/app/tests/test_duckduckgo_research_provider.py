# -*- coding: utf-8 -*-
"""DuckDuckGo discovery tests: attribute-order-independent parsing and
challenge classification. All HTTP is faked; no public network access.
"""
import pytest

from app.services.duckduckgo_research_provider import (
    DuckDuckGoDiscovery,
    DuckDuckGoResearchProvider,
    DuckDuckGoResultParser,
    SearchCandidate,
)
from app.services.research_provider import EvidenceLevel, ResearchAttempt, ResearchError


def test_parser_ignores_attribute_order_and_quote_style(load_fixture):
    parser = DuckDuckGoResultParser()
    parser.feed(load_fixture("duckduckgo-results.html"))
    assert [item.url for item in parser.results] == [
        "https://example.com/essay",
        "https://example.org/interview",
    ]


def test_search_classifies_challenge_instead_of_empty_results(fake_http, load_fixture):
    fake_http.add(status=202, text=load_fixture("duckduckgo-challenge.html"))
    with pytest.raises(ResearchError) as captured:
        DuckDuckGoDiscovery(client=fake_http).search("Ada Lovelace")
    assert captured.value.code == "research_search_blocked"
    assert captured.value.retryable is True


def test_search_classifies_valid_empty_page(fake_http):
    fake_http.add(status=200, text="<html><body>No results.</body></html>")
    results, attempt = DuckDuckGoDiscovery(client=fake_http).search("unlikely query")
    assert results == []
    assert attempt.status == "empty"


class _FakeDiscovery:
    """固定返回一组候选的假 discovery，避免真实搜索请求。"""

    def __init__(self, candidates):
        self.candidates = candidates
        self.attempt = ResearchAttempt(
            "duckduckgo", "ok", len(candidates), len(candidates), None
        )

    def search(self, query):
        return self.candidates, self.attempt


def test_single_long_authoritative_document_is_limited_not_insufficient(fake_fetcher):
    """证据判定用组合判断：单篇 2000 字权威资料应为 limited，而非"至少两篇"。"""
    discovery = _FakeDiscovery([SearchCandidate("Single Essay", "https://example.com/essay")])
    fake_fetcher.add("https://example.com/essay", "x" * 2000)
    provider = DuckDuckGoResearchProvider(
        timeout=4, sleep=lambda _: None, discovery=discovery, fetcher=fake_fetcher
    )
    report = provider.collect("某人", "提炼决策方式", "zh-CN")
    assert report.evidence_level == EvidenceLevel.LIMITED
    assert len(report.documents) == 1


def test_provider_honors_max_documents(fake_fetcher):
    candidates = [
        SearchCandidate(f"doc-{i}", f"https://example{i}.com/doc")
        for i in range(5)
    ]
    for item in candidates:
        fake_fetcher.add(item.url, "x" * 800)
    provider = DuckDuckGoResearchProvider(
        max_documents=2,
        sleep=lambda _: None,
        discovery=_FakeDiscovery(candidates),
        fetcher=fake_fetcher,
    )

    report = provider.collect("人物", "提炼决策方式", "zh-CN")

    assert len(report.documents) == 2


class _ExplodingFetcher:
    """只有第一篇可抓；后续任何 URL 被访问都会失败测试。"""

    def __init__(self, ok_url, ok_excerpt):
        self.ok_url = ok_url
        self.ok_excerpt = ok_excerpt

    def fetch(self, url):
        if url == self.ok_url:
            from app.services.web_document_fetcher import FetchResult
            return FetchResult(url, self.ok_excerpt, "ok", "")
        raise AssertionError(f"不应抓取 {url}")


def test_provider_stops_after_limited_evidence():
    first = SearchCandidate("first", "https://example1.com/doc")
    second = SearchCandidate("second", "https://example2.com/doc")
    fetcher = _ExplodingFetcher(first.url, "x" * 2000)
    provider = DuckDuckGoResearchProvider(
        sleep=lambda _: None,
        discovery=_FakeDiscovery([first, second]),
        fetcher=fetcher,
    )

    report = provider.collect("人物", "提炼决策方式", "zh-CN")

    assert len(report.documents) == 1
    assert report.evidence_level == EvidenceLevel.LIMITED
