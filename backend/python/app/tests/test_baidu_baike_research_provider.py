# -*- coding: utf-8 -*-
"""百度百科直达页基线测试：免密国内源、编码 URL、错误页识别。

fixture 测试走真实 ``WebDocumentFetcher`` 提取链路（fake HTTP + fake DNS），
覆盖两种真实响应形态：正常条目页 200 与安全验证(challenge)页 200。
challenge 页必须判为 "blocked"（可熔断），绝不能作为证据或误报"资料不足"。
"""
from app.services.baidu_baike_research_provider import (
    _disambiguation_target,
    BaiduBaikeResearchProvider,
)
from app.services.web_document_fetcher import FetchResult, WebDocumentFetcher


def test_baike_uses_encoded_subject_url_and_returns_domestic_baseline(fake_fetcher):
    fake_fetcher.add(
        "https://baike.baidu.com/item/%E4%BF%9D%E7%BD%97%C2%B7%E6%A0%BC%E9%9B%B7%E5%8E%84%E5%A7%86",
        excerpt="保罗·格雷厄姆" + "公开生平与作品 " * 300,
    )
    report = BaiduBaikeResearchProvider(fetcher=fake_fetcher).collect(
        "保罗·格雷厄姆", "提炼创业判断方式", "zh-CN"
    )
    assert len(report.documents) == 1
    assert report.documents[0].dimension == "reference"
    assert report.attempts[0].provider == "baidu_baike"


def test_baike_captcha_or_missing_item_is_not_accepted(fake_fetcher):
    fake_fetcher.add_result(status="failed", reason="captcha")
    report = BaiduBaikeResearchProvider(fetcher=fake_fetcher).collect("不存在对象", "目标描述", "zh-CN")
    assert report.documents == []
    assert report.attempts[0].status == "blocked"


def test_baike_accepts_real_html_fixture(fake_http, public_dns, load_fixture):
    fake_http.add(
        status=200,
        text=load_fixture("baidu-baike-item.html"),
        headers={"content-type": "text/html"},
    )
    provider = BaiduBaikeResearchProvider(
        fetcher=WebDocumentFetcher(client=fake_http, resolver=public_dns)
    )
    report = provider.collect("保罗·格雷厄姆", "提炼创业判断方式", "zh-CN")
    assert len(report.documents) == 1
    assert report.documents[0].dimension == "reference"
    assert report.attempts[0].status == "ok"


def test_baike_challenge_page_is_blocked_not_evidence(fake_http, public_dns, load_fixture):
    fake_http.add(
        status=200,
        text=load_fixture("baidu-challenge.html"),
        headers={"content-type": "text/html"},
    )
    provider = BaiduBaikeResearchProvider(
        fetcher=WebDocumentFetcher(client=fake_http, resolver=public_dns)
    )
    report = provider.collect("保罗·格雷厄姆", "目标描述", "zh-CN")
    assert report.documents == []
    assert report.attempts[0].status == "blocked"


def test_baike_follows_first_ranked_same_host_disambiguation(fake_fetcher, load_fixture):
    root = "https://baike.baidu.com/item/%E4%BF%9D%E7%BD%97%C2%B7%E6%A0%BC%E9%9B%B7%E5%8E%84%E5%A7%86"
    target = root + "/9902788"
    fake_fetcher.by_url[root] = FetchResult(
        root, "", "failed", "text_too_short",
        load_fixture("baidu-baike-disambiguation.html"),
    )
    fake_fetcher.add(target, "保罗·格雷厄姆" + "公开生平与作品 " * 300)

    report = BaiduBaikeResearchProvider(fetcher=fake_fetcher).collect(
        "保罗·格雷厄姆", "提炼创业判断方式", "zh-CN"
    )

    assert report.attempts[0].status == "ok"
    assert report.documents[0].url == target


def test_disambiguation_rejects_non_positive_or_foreign_lemma_ids():
    # lemmaId 为字符串 URL：绝不能跟随页面提供的域名
    foreign = (
        '<script>window.PAGE_DATA = {"navigation":{"lemmas":['
        '{"lemmaId":"https://evil.example/x","rank":1}]}}</script>'
    )
    assert _disambiguation_target(foreign, "保罗·格雷厄姆") is None
    # 负数与零：不合法义项 ID
    non_positive = (
        '<script>window.PAGE_DATA = {"navigation":{"lemmas":['
        '{"lemmaId":-5,"rank":1},{"lemmaId":0,"rank":2}]}}</script>'
    )
    assert _disambiguation_target(non_positive, "保罗·格雷厄姆") is None
