# 女娲生成金丹端到端可用性修复实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Do not use subagents unless the user explicitly authorizes delegation.

**Goal:** 修复女娲使用真实公开资料和真实 DeepSeek 合成模型时始终无法生成金丹草稿的问题，并建立不会把“明确失败”当成通过的端到端门禁。

**Architecture:** 保留前端 → Go 网关 → Python 引擎的既有接口。Python 蒸馏服务根据已知模型端点选择结构化输出能力；安全抓取器向百度百科适配器提供受限原始 HTML，以处理同源多义词；研究编排通过 limited 早停和候选上限收敛耗时。开发启动日志配置和联网 smoke 同步修复。

**Tech Stack:** Python 3.13、FastAPI、httpx、OpenAI Python SDK、pytest、Go/Gin、Next.js 16、React 19、TypeScript、Vitest。

**Spec:** `docs/superpowers/specs/2026-08-30-nuwa-end-to-end-reliability-fix-design.md`

## Global Constraints

- 修改前先运行 `git status --short`。当前工作区存在聊天、首页和生成产物相关未提交改动；不得覆盖、暂存或格式化这些无关文件。
- 禁止使用 `git add .`、`git add -A`、`git commit -a`。每次只暂存本 Task 明确列出的文件。
- 不修改或提交 `backend/go/internal/webui/out/`、`frontend/out/`、`frontend/.next/`、`backend/go/build/`。
- 每个生产行为必须先写失败测试，运行并确认因缺少该行为而失败，再写最小实现。
- 不记录 API Key、`MODEL_KEY_SECRET`、完整模型提示词、完整模型输出、`reasoning_content`、网页全文或用户数据库内容。
- 真实桌面数据库只允许复制到 `mktemp -d` 目录后诊断；不得对原数据库运行迁移、UPDATE 或 seed。验证结束后删除临时副本。
- 女娲结果仍然只是草稿；不得自动调用金丹创建或更新接口。
- 不绕过 CAPTCHA、登录墙或反爬限制。
- 不重构金丹 CRUD、道人、聊天、融合或 Skill 导出。
- 中英文错误提示必须同时更新。
- Python 测试必须兼容 `backend/python/app/tests/conftest.py` 的轻量依赖环境。
- 完成声明必须附带本次运行的测试命令与结果，不能引用历史测试结果。

---

## 文件结构与职责

```text
backend/python/app/services/nuwa_distillation_service.py
  已知模型端点的结构化输出策略、finish_reason/空正文分类、JSON 解析

backend/python/app/services/web_document_fetcher.py
  SSRF 安全抓取；在 120 KB 上限内向适配器提供 raw_html

backend/python/app/services/baidu_baike_research_provider.py
  百度百科直达页、多义词 PAGE_DATA 解析和固定同源义项地址

backend/python/app/services/duckduckgo_research_provider.py
  候选与文档上限、limited 证据早停

backend/python/app/services/research_orchestrator.py
  国内证据与全球提供者的 early-stop 规则

backend/python/app/logging_config.py
  text/json/自定义日志格式解析，不允许非法 LOG_FORMAT 阻止启动

backend/python/app/main.py
  使用统一日志配置

frontend/components/nuwa-distill-panel.tsx
  新模型输出错误码的可操作提示与重试

backend/python/app/tests/*
  上述行为的失败测试、组合测试和显式联网门禁
```

---

### Task 1: 修复 DeepSeek 结构化输出契约

**Files:**

- Modify: `backend/python/app/services/nuwa_distillation_service.py`
- Modify: `backend/python/app/tests/test_nuwa_distillation_service.py`

**Interfaces:**

- Produces: `_completion_request_options(base_url: str) -> dict[str, Any]`
- Produces stable errors: `model_output_truncated`、`model_empty_output`、现有 `model_invalid_output`
- Preserves: `NuwaDistillationService.distill(...) -> dict[str, Any]`

- [ ] **Step 1: 扩展测试用 OpenAI 记录器**

在 `test_nuwa_distillation_service.py` 增加一个只记录真实调用参数的测试工厂。测试断言消费者可见结果，不断言 mock 自身存在：

```python
def recording_openai(payload, finish_reason="stop", reasoning_content=None):
    captured = {}
    message = SimpleNamespace(
        content=json.dumps(payload, ensure_ascii=False) if isinstance(payload, dict) else payload,
        reasoning_content=reasoning_content,
    )
    completion = SimpleNamespace(
        choices=[SimpleNamespace(message=message, finish_reason=finish_reason)]
    )

    class _RecordingOpenAI:
        def __init__(self, **kwargs):
            captured["client"] = kwargs
            self.chat = SimpleNamespace(
                completions=SimpleNamespace(create=self._create)
            )

        def _create(self, **kwargs):
            captured["create"] = kwargs
            return completion

        def close(self):
            pass

    return _RecordingOpenAI, captured
```

- [ ] **Step 2: 写 DeepSeek 能力策略失败测试**

```python
def test_deepseek_distillation_disables_thinking_and_requests_json(monkeypatch):
    factory, captured = recording_openai(VALID_PAYLOAD)
    monkeypatch.setattr(distillation_module, "OpenAI", factory)
    service = NuwaDistillationService(FixedResearchProvider(standard_report()))

    result = service.distill(
        "人物",
        "提炼决策方式",
        model="deepseek-v4-flash",
        api_key="sk-test",
        base_url="https://api.deepseek.com/v1",
    )

    assert result["name"] == VALID_PAYLOAD["name"]
    assert captured["create"]["max_tokens"] == 8192
    assert captured["create"]["response_format"] == {"type": "json_object"}
    assert captured["create"]["extra_body"] == {
        "thinking": {"type": "disabled"}
    }
```

另写一个 OpenAI 官方端点测试，断言启用 JSON mode，但不发送 DeepSeek 专有 `thinking`：

```python
def test_openai_distillation_requests_json_without_vendor_extra_body(monkeypatch):
    factory, captured = recording_openai(VALID_PAYLOAD)
    monkeypatch.setattr(distillation_module, "OpenAI", factory)
    service = NuwaDistillationService(FixedResearchProvider(standard_report()))

    service.distill(
        "人物", "提炼决策方式", model="gpt-4o-mini",
        api_key="sk-test", base_url="https://api.openai.com/v1",
    )

    assert captured["create"]["response_format"] == {"type": "json_object"}
    assert "extra_body" not in captured["create"]
```

- [ ] **Step 3: 运行测试并确认 RED**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_nuwa_distillation_service.py \
  -k 'deepseek_distillation or openai_distillation'
```

Expected: FAIL；当前请求为 `max_tokens=4096`，没有 `response_format` 和 `extra_body`。

- [ ] **Step 4: 实现最小端点能力策略**

在 `nuwa_distillation_service.py` 导入 `urlparse`，增加：

```python
from urllib.parse import urlparse

DISTILLATION_MAX_TOKENS = 8192


def _completion_request_options(base_url: str) -> dict[str, Any]:
    host = (urlparse(base_url).hostname or "").lower()
    options: dict[str, Any] = {
        "temperature": 0.25,
        "max_tokens": DISTILLATION_MAX_TOKENS,
    }
    if host == "api.openai.com" or host.endswith(".openai.com"):
        options["response_format"] = {"type": "json_object"}
    if host == "api.deepseek.com" or host.endswith(".deepseek.com"):
        options["response_format"] = {"type": "json_object"}
        options["extra_body"] = {"thinking": {"type": "disabled"}}
    return options
```

构造 completion 时删除硬编码的 `temperature` 和 `max_tokens`，改为：

```python
response = client.chat.completions.create(
    model=research_credentials.model,
    messages=[...],
    **_completion_request_options(research_credentials.base_url),
)
```

不要按模型名称猜厂商；只按解析后的 hostname 选择厂商专有参数。

- [ ] **Step 5: 写截断与空正文失败测试**

```python
def test_length_finish_reason_is_reported_as_truncated(monkeypatch):
    factory, _ = recording_openai("", finish_reason="length", reasoning_content="x" * 20)
    monkeypatch.setattr(distillation_module, "OpenAI", factory)
    service = NuwaDistillationService(FixedResearchProvider(standard_report()))

    with pytest.raises(DistillationError) as captured:
        service.distill(
            "人物", "提炼决策方式", model="deepseek-v4-flash",
            api_key="sk-test", base_url="https://api.deepseek.com/v1",
        )

    assert captured.value.code == "model_output_truncated"
    assert captured.value.stage == "distill"
    assert captured.value.retryable is True
    assert captured.value.details == {"finish_reason": "length"}


def test_empty_content_is_reported_without_exposing_reasoning(monkeypatch):
    factory, _ = recording_openai("", finish_reason="stop", reasoning_content="private reasoning")
    monkeypatch.setattr(distillation_module, "OpenAI", factory)
    service = NuwaDistillationService(FixedResearchProvider(standard_report()))

    with pytest.raises(DistillationError) as captured:
        service.distill(
            "人物", "提炼决策方式", model="deepseek-v4-flash",
            api_key="sk-test", base_url="https://api.deepseek.com/v1",
        )

    assert captured.value.code == "model_empty_output"
    assert "private reasoning" not in str(captured.value.details)
```

- [ ] **Step 6: 运行测试并确认 RED**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_nuwa_distillation_service.py \
  -k 'length_finish_reason or empty_content'
```

Expected: FAIL；当前两个场景都落入笼统的 `model_invalid_output`。

- [ ] **Step 7: 在 JSON 解析前分类响应**

读取 `choice = response.choices[0]`，只保存非敏感协议字段：

```python
choice = response.choices[0]
finish_reason = getattr(choice, "finish_reason", None)
content = getattr(choice.message, "content", None) or ""
if finish_reason == "length":
    raise DistillationError(
        "model_output_truncated",
        "distill",
        "模型输出达到长度上限，请重试或更换合成模型",
        True,
        {"finish_reason": "length"},
    )
if not content.strip():
    raise DistillationError(
        "model_empty_output",
        "distill",
        "模型未返回可用的结构化正文，请重试",
        True,
        {"finish_reason": finish_reason or "unknown"},
    )
```

必须保留现有 `finally: client.close()`。不得读取、记录或返回 `reasoning_content`。

- [ ] **Step 8: 运行模型专项测试并提交**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_nuwa_distillation_service.py
```

Expected: PASS。

Commit:

```bash
git add backend/python/app/services/nuwa_distillation_service.py \
  backend/python/app/tests/test_nuwa_distillation_service.py
git commit -m "fix(nuwa): make structured model output reliable"
```

---

### Task 2: 让前端解释模型截断和空正文

**Files:**

- Modify: `frontend/components/nuwa-distill-panel.tsx`
- Modify: `frontend/components/nuwa-distill-panel.test.tsx`
- Modify: `frontend/messages/zh-CN.json`
- Modify: `frontend/messages/en.json`

**Interfaces:**

- Consumes: `model_output_truncated`、`model_empty_output`、`model_invalid_output`
- Preserves: `NuwaDistillPanel({ onApply })`

- [ ] **Step 1: 写两个错误码的失败测试**

在现有 `nuwa-distill-panel.test.tsx` 沿用真实组件和 mock service 的方式，分别让 `distillNuwa` 拒绝：

```typescript
new ApiError('模型输出达到长度上限', 503, {
  error_code: 'model_output_truncated',
  data: { stage: 'distill', retryable: true },
})
```

以及：

```typescript
new ApiError('模型未返回结构化正文', 503, {
  error_code: 'model_empty_output',
  data: { stage: 'distill', retryable: true },
})
```

每个测试断言：错误正文可见、输出问题提示可见、重试按钮可见。不要只断言 mock 调用次数。

- [ ] **Step 2: 运行测试并确认 RED**

Run:

```bash
cd frontend
pnpm test -- components/nuwa-distill-panel.test.tsx
```

Expected: FAIL；新错误码尚未映射到模型输出提示。

- [ ] **Step 3: 统一模型输出错误分类**

在组件中增加：

```typescript
const invalidModelOutput =
  failure?.code === 'model_invalid_output' ||
  failure?.code === 'distill_invalid_output' ||
  failure?.code === 'model_output_truncated' ||
  failure?.code === 'model_empty_output'
```

用 `invalidModelOutput` 控制现有输出提示。为截断增加更具体但简短的中英文文案，例如：

```json
"outputTruncatedHint": "模型的结构化输出被截断。系统已调整生成策略；请重试，持续失败时更换合成模型。"
```

英文保持同义。`model_empty_output` 复用 `invalidOutputHint`。

- [ ] **Step 4: 运行测试并提交**

Run:

```bash
cd frontend
pnpm test -- components/nuwa-distill-panel.test.tsx
```

Expected: PASS。

Commit:

```bash
git add frontend/components/nuwa-distill-panel.tsx \
  frontend/components/nuwa-distill-panel.test.tsx \
  frontend/messages/zh-CN.json frontend/messages/en.json
git commit -m "fix(frontend): explain Nuwa model output failures"
```

---

### Task 3: 支持百度百科多义词页

**Files:**

- Modify: `backend/python/app/services/web_document_fetcher.py`
- Modify: `backend/python/app/services/baidu_baike_research_provider.py`
- Modify: `backend/python/app/tests/conftest.py`
- Modify: `backend/python/app/tests/test_web_document_fetcher.py`
- Modify: `backend/python/app/tests/test_baidu_baike_research_provider.py`
- Create: `backend/python/app/tests/fixtures/baidu-baike-disambiguation.html`

**Interfaces:**

- Changes: `FetchResult(url, excerpt, status, reason, raw_html="")`
- Produces: `_disambiguation_target(raw_html: str, subject: str) -> str | None`

- [ ] **Step 1: 写 raw HTML 保留失败测试**

```python
def test_tiny_html_keeps_bounded_raw_html_for_provider_specific_parsing(fake_http, public_dns):
    html = "<html><script>window.PAGE_DATA={}</script></html>"
    fake_http.add(
        "https://example.com/a",
        status=200,
        headers={"content-type": "text/html"},
        text=html,
    )
    result = WebDocumentFetcher(client=fake_http, resolver=public_dns).fetch(
        "https://example.com/a"
    )
    assert result.reason == "text_too_short"
    assert result.raw_html == html
```

- [ ] **Step 2: 运行测试并确认 RED**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_web_document_fetcher.py \
  -k tiny_html_keeps_bounded_raw_html
```

Expected: FAIL；`FetchResult` 没有 `raw_html`。

- [ ] **Step 3: 扩展受限抓取结果**

给 dataclass 增加末尾默认字段，保证现有四参数构造器兼容：

```python
@dataclass(frozen=True)
class FetchResult:
    url: str
    excerpt: str
    status: str
    reason: str
    raw_html: str = ""
```

`_extract()` 中解码后的 `raw` 已受 `MAX_BYTES` 限制。仅当 content-type 是 HTML 时把它放入 `raw_html`；成功、正文过短两条返回路径都携带。text/plain、HTTP 错误和拒绝路径保持空字符串。

同步扩展测试 `FakeFetcher`：

```python
def add_result(self, status, reason, raw_html=""):
    self.fallback = FetchResult("", "", status, reason, raw_html)
```

- [ ] **Step 4: 添加最小多义词 fixture**

创建 `baidu-baike-disambiguation.html`：

```html
<!doctype html>
<html>
  <head><title>保罗·格雷厄姆_百度百科</title></head>
  <body>
    <p>请选择义项</p>
    <script>
      window.PAGE_DATA = {"navigation":{"lemmas":[
        {"lemmaId":9902788,"rank":1,"lemmaTitle":"保罗·格雷厄姆","lemmaDesc":"美国程序员、风险投资家"},
        {"lemmaId":68446104,"rank":2,"lemmaTitle":"保罗·格雷厄姆","lemmaDesc":"英国摄影家"}
      ]}};
    </script>
  </body>
</html>
```

- [ ] **Step 5: 写多义词跟随和 URL 安全失败测试**

```python
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
```

另加恶意 fixture 内联测试：`lemmaId` 为字符串 URL、负数或零时 `_disambiguation_target` 必须返回 `None`，绝不能访问页面提供的域名。

- [ ] **Step 6: 运行测试并确认 RED**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_baidu_baike_research_provider.py \
  -k 'disambiguation or same_host'
```

Expected: FAIL；当前 provider 不解析多义词元数据。

- [ ] **Step 7: 实现固定同源义项选择**

在百度 provider 内新增纯函数：

```python
def _disambiguation_target(raw_html: str, subject: str) -> str | None:
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
```

首次抓取 `text_too_short` 时只允许调用此函数；拿到 target 后通过原 fetcher 再抓一次。第二次结果仍执行现有 challenge、host、长度和主题校验。

- [ ] **Step 8: 运行抓取专项测试并提交**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_web_document_fetcher.py \
  app/tests/test_baidu_baike_research_provider.py
```

Expected: PASS。

Commit:

```bash
git add backend/python/app/services/web_document_fetcher.py \
  backend/python/app/services/baidu_baike_research_provider.py \
  backend/python/app/tests/conftest.py \
  backend/python/app/tests/test_web_document_fetcher.py \
  backend/python/app/tests/test_baidu_baike_research_provider.py \
  backend/python/app/tests/fixtures/baidu-baike-disambiguation.html
git commit -m "fix(nuwa): resolve Baidu Baike disambiguation pages"
```

---

### Task 4: 收敛全球资料链路耗时

**Files:**

- Modify: `backend/python/app/services/duckduckgo_research_provider.py`
- Modify: `backend/python/app/services/research_orchestrator.py`
- Modify: `backend/python/app/tests/test_duckduckgo_research_provider.py`
- Modify: `backend/python/app/tests/test_research_orchestrator.py`
- Modify: `docs/architecture/nuwa-integration.md`

**Interfaces:**

- `DuckDuckGoResearchProvider(..., max_documents: int = 3)`
- Global lane rule: no starting evidence + first limited provider → stop; existing domestic evidence → allow one global provider to pursue standard

- [ ] **Step 1: 写 DDG 文档上限失败测试**

构造 5 个候选和 5 个均可成功的假页面，使用不足 1500 字符的每篇内容，避免第一篇就触发 limited：

```python
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
```

- [ ] **Step 2: 写 limited 早停失败测试**

让第一候选返回 2000 字、第二候选的 fetcher 在被调用时抛断言。断言第一篇已经形成 limited，后续候选不再抓取。

- [ ] **Step 3: 运行 DDG 测试并确认 RED**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_duckduckgo_research_provider.py \
  -k 'max_documents or stops_after_limited'
```

Expected: FAIL；当前 `max_documents` 没有进入抓取循环，也不会按证据早停。

- [ ] **Step 4: 实现 DDG 边界**

默认值从 10 改为 3。抓取循环每轮开头和成功追加文档后检查：

```python
if len(documents) >= self.max_documents:
    break

# 成功 append 后
if classify_evidence(documents) is not EvidenceLevel.INSUFFICIENT:
    break
```

从 `research_provider` 导入 `EvidenceLevel`。不得降低 SSRF 校验或页面最小长度。

- [ ] **Step 5: 写全球 lane limited 早停失败测试**

```python
def test_global_lane_accepts_wikipedia_limited_without_calling_ddg():
    wikipedia = StubProvider(
        "wikipedia",
        documents=[_document(0, "wikipedia.org", 2200)],
    )
    ddg = FailIfCalled()

    report = ResearchOrchestrator(
        domestic=[],
        global_providers=[wikipedia, ddg],
    ).collect("Paul Graham", "extract startup decisions", "en")

    assert report.evidence_level == EvidenceLevel.LIMITED
```

再保留一个测试证明“已有百度 limited 时仍调用 Wikipedia，两个域合并达到 standard”。

- [ ] **Step 6: 运行 orchestrator 测试并确认 RED**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_research_orchestrator.py \
  -k 'wikipedia_limited or baike_limited'
```

Expected: 第一个测试 FAIL；当前会继续调用 DDG。

- [ ] **Step 7: 实现组合 early-stop**

进入 `_run_global_budgeted()` 时记录：

```python
started_with_evidence = (
    classify_evidence(documents) is not EvidenceLevel.INSUFFICIENT
)
```

每个 provider 完成后：

```python
level = classify_evidence(documents)
if level is EvidenceLevel.STANDARD:
    break
if not started_with_evidence and level is EvidenceLevel.LIMITED:
    break
```

不要使用后台线程和 `future.result(timeout=...)`；同步 HTTP 请求无法被可靠取消，超时返回后留下后台抓取会制造资源泄漏。

- [ ] **Step 8: 更新架构文档的预算语义**

将“国际 6 秒总预算”改为：

- 6 秒是启动下一个 provider 前的预算；
- Wikipedia limited 可直接生成草稿；
- DDG 默认最多采纳 3 篇，达到 limited 立即停止；
- 每个外部请求仍受 provider timeout 约束；
- 不声称同步 provider 已经开始后可以被硬取消。

- [ ] **Step 9: 运行研究专项测试并提交**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_duckduckgo_research_provider.py \
  app/tests/test_research_orchestrator.py \
  app/tests/test_distillation_research_integration.py
```

Expected: PASS，联网标记默认 SKIP。

Commit:

```bash
git add backend/python/app/services/duckduckgo_research_provider.py \
  backend/python/app/services/research_orchestrator.py \
  backend/python/app/tests/test_duckduckgo_research_provider.py \
  backend/python/app/tests/test_research_orchestrator.py \
  docs/architecture/nuwa-integration.md
git commit -m "perf(nuwa): stop research after usable evidence"
```

---

### Task 5: 修复 LOG_FORMAT=json 启动崩溃

**Files:**

- Create: `backend/python/app/logging_config.py`
- Create: `backend/python/app/tests/test_logging_config.py`
- Modify: `backend/python/app/main.py`

**Interfaces:**

- Produces: `configure_logging(level_name: str, format_spec: str, stream=None) -> None`
- Produces: `JsonLogFormatter(logging.Formatter)`

- [ ] **Step 1: 写 JSON formatter 失败测试**

```python
import io
import json
import logging

from app.logging_config import configure_logging


def test_json_log_format_is_valid_json_and_does_not_raise():
    output = io.StringIO()
    configure_logging("INFO", "json", stream=output)
    logging.getLogger("nuwa-test").info("engine ready")

    record = json.loads(output.getvalue().strip())
    assert record["level"] == "INFO"
    assert record["logger"] == "nuwa-test"
    assert record["message"] == "engine ready"
```

再写非法预设回退测试：`format_spec="unknown-preset"` 时不抛异常，输出包含消息。

- [ ] **Step 2: 运行测试并确认 RED**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_logging_config.py
```

Expected: ERROR；模块尚不存在。

- [ ] **Step 3: 实现日志配置模块**

```python
DEFAULT_TEXT_FORMAT = "%(asctime)s [%(levelname)s] %(name)s - %(message)s"


class JsonLogFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        return json.dumps(
            {
                "timestamp": self.formatTime(record, self.datefmt),
                "level": record.levelname,
                "logger": record.name,
                "message": record.getMessage(),
            },
            ensure_ascii=False,
        )


def configure_logging(level_name: str, format_spec: str, stream=None) -> None:
    handler = logging.StreamHandler(stream or sys.stdout)
    normalized = (format_spec or "text").strip()
    if normalized.lower() == "json":
        handler.setFormatter(JsonLogFormatter())
    else:
        selected = (
            normalized
            if "%(" in normalized
            else DEFAULT_TEXT_FORMAT
        )
        handler.setFormatter(logging.Formatter(selected))
    logging.basicConfig(
        level=getattr(logging, level_name.upper(), logging.INFO),
        handlers=[handler],
        force=True,
    )
```

在 `main.py` 删除直接 `logging.basicConfig(format=settings.log_format, ...)`，改为：

```python
from app.logging_config import configure_logging

configure_logging(settings.log_level, settings.log_format)
```

- [ ] **Step 4: 运行专项和导入 smoke**

Run:

```bash
cd backend/python
.venv/bin/pytest -q app/tests/test_logging_config.py
LOG_FORMAT=json .venv/bin/python -c 'import app.main; print("import-ok")'
```

Expected: 测试 PASS；第二条输出 JSON 日志后包含 `import-ok`，退出码 0。

- [ ] **Step 5: 提交**

```bash
git add backend/python/app/logging_config.py \
  backend/python/app/tests/test_logging_config.py \
  backend/python/app/main.py
git commit -m "fix(engine): accept JSON logging preset"
```

---

### Task 6: 消除联网 smoke 假阳性

**Files:**

- Modify: `backend/python/app/tests/test_distillation_research_integration.py`
- Modify: `backend/python/app/tests/conftest.py`
- Modify: `docs/architecture/nuwa-integration.md`

**Interfaces:**

- `network_cn`: 知名中文主体必须取得至少一篇有效证据
- `network_global`: 知名英文主体必须取得至少一篇有效证据
- Both remain opt-in; ordinary CI does not depend on public network

- [ ] **Step 1: 收紧中国大陆联网断言**

将原来的“证据或任意明确状态”替换为：

```python
assert report.evidence_level in {
    EvidenceLevel.STANDARD,
    EvidenceLevel.LIMITED,
}
assert report.documents
assert any(
    attempt.provider == "baidu_baike"
    and attempt.status == "ok"
    and attempt.accepted >= 1
    for attempt in report.attempts
)
```

这条测试使用已确认存在多义词页的“保罗·格雷厄姆”，必须覆盖 Task 3 的真实行为。

- [ ] **Step 2: 收紧全球联网断言**

```python
assert report.evidence_level in {
    EvidenceLevel.STANDARD,
    EvidenceLevel.LIMITED,
}
assert report.documents
assert any(
    attempt.status == "ok" and attempt.accepted >= 1
    for attempt in report.attempts
    if attempt.provider in {"wikipedia", "duckduckgo"}
)
```

- [ ] **Step 3: 先运行并记录当前网络结果**

Run:

```bash
cd backend/python
.venv/bin/pytest -q -m network_cn --run-network \
  app/tests/test_distillation_research_integration.py
.venv/bin/pytest -q -m network_global --run-network \
  app/tests/test_distillation_research_integration.py
```

Expected: 两条都必须 PASS 才能继续。若外部网络真实不可达，保留失败证据，不得放宽断言；记录 provider、status、reason，不记录网页正文。

- [ ] **Step 4: 更新联网测试文档**

明确写出：

- 明确错误状态只用于诊断，不再算成功；
- `network_cn` 是中国大陆发布前门禁；
- `network_global` 在具备国际网络的 runner 上执行；
- 外部站点临时不可用时门禁失败是预期信号，不得改回宽松 OR 断言。

- [ ] **Step 5: 提交**

```bash
git add backend/python/app/tests/test_distillation_research_integration.py \
  backend/python/app/tests/conftest.py docs/architecture/nuwa-integration.md
git commit -m "test(nuwa): require real evidence in network smoke"
```

---

### Task 7: 真实 Go → Python → 公网 → 模型验收

**Files:**

- Modify only after success: `docs/pending-issues.md`
- No production code changes in this Task

**Preconditions:**

- Tasks 1–6 committed。
- Python 与 Go 服务端口空闲。
- 当前桌面配置中存在启用的合成模型。

- [ ] **Step 1: 创建隔离数据库副本**

macOS：

```bash
NUWA_TMP_DIR="$(mktemp -d /tmp/alchemy-nuwa-e2e.XXXXXX)"
cp "$HOME/Library/Application Support/AlchemyFurnace/alchemy.db" \
  "$NUWA_TMP_DIR/alchemy.db"
```

只对副本启动 Go。不得修改原数据库。

- [ ] **Step 2: 启动 Python 引擎并验证健康**

使用非冲突端口，例如 18000。加载项目环境时不得打印环境变量：

```bash
cd backend/python
LOG_FORMAT=json .venv/bin/uvicorn app.main:app --port 18000
```

另一个终端：

```bash
curl --fail --silent http://127.0.0.1:18000/health
```

Expected: HTTP 200；Python 不因日志格式崩溃。

- [ ] **Step 3: 启动指向数据库副本的 Go 网关**

必须使用桌面真实 `secret.key` 解析副本中的凭证，但不得输出其内容：

```bash
cd backend/go
export DB_DRIVER=sqlite
export DB_SQLITE_PATH="$NUWA_TMP_DIR/alchemy.db"
export GO_PORT=18080
export PYTHON_ENGINE_BASE_URL=http://127.0.0.1:18000
export MODEL_KEY_SECRET="$(<"$HOME/Library/Application Support/AlchemyFurnace/secret.key")"
go run cmd/main/main.go serve
```

若不是 macOS 桌面环境，使用对应平台数据目录中的数据库副本与 `secret.key`。禁止把密钥写入命令输出或文档。

- [ ] **Step 4: 调用真实女娲端点**

```bash
curl --silent --show-error --max-time 180 \
  -o "$NUWA_TMP_DIR/response.json" \
  -w 'HTTP %{http_code} elapsed=%{time_total}\n' \
  -H 'Content-Type: application/json' \
  --data '{"subject":"保罗·格雷厄姆","brief":"提炼创业判断方式","locale":"zh-CN"}' \
  http://127.0.0.1:18080/api/v1/distillation/nuwa
```

Expected: HTTP 200。

- [ ] **Step 5: 只输出脱敏成功摘要**

```bash
python3 - "$NUWA_TMP_DIR/response.json" <<'PY'
import json
import sys

data = json.load(open(sys.argv[1], encoding="utf-8"))
assert data.get("code") == 0, data
value = data["data"]
summary = {
    "name": value.get("name"),
    "model": value.get("model"),
    "source_count": len(value.get("sources") or []),
    "evidence_level": (value.get("research") or {}).get("evidence_level"),
    "document_count": (value.get("research") or {}).get("document_count"),
}
assert summary["name"]
assert summary["source_count"] >= 1
assert summary["evidence_level"] in {"limited", "standard"}
print(json.dumps(summary, ensure_ascii=False, indent=2))
PY
```

不得打印完整 `response.json`。

- [ ] **Step 6: 验证日志不泄露敏感内容**

检查运行日志只包含 request id、主体、阶段耗时、provider 状态、计数和 evidence。确认没有：

- API Key；
- `MODEL_KEY_SECRET`；
- `reasoning_content`；
- 完整 prompt；
- 网页正文；
- 完整模型 JSON。

- [ ] **Step 7: 停止服务并清理临时副本**

先正常 `Ctrl+C` 停止 Go 和 Python。列出临时目录并只删除明确文件，然后 `rmdir`；不要对未验证变量执行递归删除：

```bash
find "$NUWA_TMP_DIR" -maxdepth 1 -type f -print
unlink "$NUWA_TMP_DIR/alchemy.db"
unlink "$NUWA_TMP_DIR/response.json"
rmdir "$NUWA_TMP_DIR"
```

- [ ] **Step 8: 更新待办状态**

只有 Step 4–6 全部通过后，才在 `docs/pending-issues.md` 的 PENDING-002 写：

- 2026-08-30 代码修复完成；
- 真实调用的 HTTP 状态和总耗时；
- 模型名、证据等级和来源数量；
- Tasks 1–6 的真实 commit SHA；
- 尚未完成的网络/平台验收；没有则写“无”。

不要删除历史根因说明。

Commit:

```bash
git add docs/pending-issues.md
git commit -m "docs: record Nuwa end-to-end verification"
```

---

### Task 8: 全量回归与交付

**Files:**

- No new feature files
- Update documentation only when verification exposes a real limitation

- [ ] **Step 1: Python 全量测试**

Run:

```bash
cd backend/python
.venv/bin/pytest -q
```

Expected: 全部非联网测试 PASS；联网标记 SKIP。

- [ ] **Step 2: Go 全量测试**

Run:

```bash
cd backend/go
go test ./...
```

Expected: PASS。

- [ ] **Step 3: 前端专项与全量测试**

Run:

```bash
cd frontend
pnpm test -- components/nuwa-distill-panel.test.tsx app/\(main\)/pills/page.test.tsx
pnpm test
pnpm typecheck
pnpm lint
pnpm build
```

Expected: 全部 PASS。构建产物不得提交。

- [ ] **Step 4: 重新运行两条联网 smoke**

Run:

```bash
cd backend/python
.venv/bin/pytest -q -m network_cn --run-network \
  app/tests/test_distillation_research_integration.py
.venv/bin/pytest -q -m network_global --run-network \
  app/tests/test_distillation_research_integration.py
```

Expected: 当前具备对应网络时 PASS。任一失败必须记录 provider/status/reason，不能改松断言。

- [ ] **Step 5: 检查范围和敏感信息**

Run:

```bash
git status --short
git diff --check
git diff --name-only HEAD~8..HEAD
git grep -n -E 'reasoning_content.*(print|log)|api_key.*(print|log)' -- \
  backend/python/app backend/go frontend || true
```

确认：

- 没有提交 `out/`、`.next/`、`build/`；
- 没有提交用户数据库或临时响应；
- 没有覆盖本计划开始前的聊天/首页改动；
- 没有凭证、完整 prompt 或思维链日志。

- [ ] **Step 6: 最终报告**

按以下模板输出，不写“应该通过”：

```text
完成范围：
- 模型输出：<commit SHA>
- 百度多义词：<commit SHA>
- 研究早停：<commit SHA>
- 日志启动：<commit SHA>
- 联网门禁：<commit SHA>

真实 E2E：
- HTTP：200
- 总耗时：<秒>
- 模型：<名称>
- 证据：limited/standard
- 来源：<数量>

验证：
- Python：<通过数>/<跳过数>
- Go：PASS
- Frontend tests/typecheck/lint/build：PASS
- network_cn：PASS/环境失败及原因
- network_global：PASS/环境失败及原因

遗留限制：
- <无，或真实限制>
```

## 完成定义

- 当前桌面 DeepSeek 合成模型不再把全部 token 消耗在思考内容中。
- 同一真实输入至少完成一次 HTTP 200 的端到端金丹草稿生成。
- 百度百科多义词页能安全落到具体同源义项。
- 有 usable evidence 后不继续无界抓取候选网页。
- `LOG_FORMAT=json` 不再阻止 Python 引擎启动。
- 联网 smoke 的成功含义是“取得真实证据”，不是“失败得很明确”。
- 所有专项与全量门禁有本次新鲜输出。
- 未提交任何凭证、数据库、模型思维链、完整提示词或生成产物。
