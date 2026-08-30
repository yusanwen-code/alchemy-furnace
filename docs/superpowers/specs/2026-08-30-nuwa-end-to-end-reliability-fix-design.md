# 女娲生成金丹端到端可用性修复设计

日期：2026-08-30

## 目标

修复“使用女娲从公开资料生成金丹从未成功”的问题，使用户已配置 DeepSeek 合成模型时，可以从金丹创建页完成“输入主体和目标 → 收集公开资料 → 模型蒸馏 → 预览草稿 → 应用到表单”。

本轮只生成草稿，不改变“用户显式应用并提交后才保存金丹”的产品边界。

## 已复现证据

排查使用真实网络、真实桌面模型配置和真实 Go → Python 调用链完成。测试时复制桌面 SQLite 数据库到临时目录，没有修改真实数据库；诊断完成后已删除临时副本。

### 1. 模型输出是当前必现失败的直接原因

使用当前桌面配置 `deepseek-v4-flash` 调用：

- Go API 最终返回 HTTP 503；
- 错误码为 `model_invalid_output`；
- 总耗时约 64 秒；
- 研究阶段约 21 秒；
- 模型阶段约 43 秒。

对模型响应进行不记录凭证和完整提示词的边界诊断后发现：

- 上游 HTTP 状态为 200；
- `finish_reason=length`；
- `completion_tokens=4096`；
- `reasoning_tokens=4096`；
- `reasoning_content` 有内容；
- 最终 `content` 为空。

因此当前代码不是“偶尔解析失败”，而是将思考模型的总输出额度全部消耗在推理内容，最终没有 JSON 正文。

在代理层只修改三个请求参数进行单变量验证：

```json
{
  "max_tokens": 8192,
  "response_format": { "type": "json_object" },
  "thinking": { "type": "disabled" }
}
```

同一输入随后返回 HTTP 200：

- `finish_reason=stop`；
- `reasoning_content` 长度为 0；
- `content` 为合法完整 JSON；
- 模型阶段约 15 秒；
- Go API 成功返回金丹草稿和 4 条来源。

### 2. 百度百科多义词页被误判为源不可用

`https://baike.baidu.com/item/保罗·格雷厄姆` 返回的是多义词选择页。页面正文抽取后只有约 342 字符，因此 `WebDocumentFetcher` 返回 `text_too_short`，`BaiduBaikeResearchProvider` 随后把它标记为 `blocked`。

页面中的 `window.PAGE_DATA.navigation.lemmas` 实际包含具体义项。排名第一的义项 ID 为数字，访问固定同源地址 `/item/<subject>/<lemmaId>` 后能取得约 1862 字符的有效正文。

当前提供者没有识别或跟随这一同源义项，因此规划中的“国内免密基线”对常见多义词人物不可用。

### 3. 国际 6 秒预算不是硬预算

`ResearchOrchestrator` 只在调用 provider 前检查 deadline。provider 内部开始后，DuckDuckGo 会继续串行抓取候选网页，`max_documents` 构造参数目前没有被使用。

实测全球资料链路耗时约 21–26 秒。它最终可能取得标准证据，但与文档声明的 6 秒总预算不一致，也放大了用户对“卡死”的感受。

### 4. 联网 smoke 存在假阳性

当前 `network_cn` 与 `network_global` 用例允许“获得证据”或“出现任意明确 provider 状态”二选一。因此百度百科被拦截、Wikipedia 超时或 DuckDuckGo challenge 时，测试仍可通过。

现有自动化结果 `24 passed, 2 skipped` 只能证明模拟协议正确，不能证明知名人物真的获得公开证据。

### 5. 开发启动存在独立阻断

根目录 `.env` 使用 `LOG_FORMAT=json` 时，`app/main.py` 把字符串 `json` 直接传给 `logging.basicConfig(format=...)`，Python 引擎在导入阶段抛出 `ValueError: Invalid format 'json' for '%' style`。

桌面打包运行时不一定加载同一 `.env`，但 Web/开发模式会因此完全无法调用女娲。

## 方案选择

### 方案 A：按能力修复现有链路（采用）

1. 对已知 DeepSeek 官方端点关闭思考模式，启用 JSON 输出，将结构化生成预算设为 8192。
2. 显式识别 `finish_reason=length`、空正文和非法 JSON，返回不同稳定错误原因，不记录推理内容。
3. 让安全抓取器保留受限长度的原始 HTML，仅供百度百科适配器解析同源多义词元数据。
4. 百度百科只从 `window.PAGE_DATA.navigation.lemmas` 读取正整数 `lemmaId`，固定拼接 `baike.baidu.com` 地址；不信任页面中的任意 URL。
5. 真正使用 DuckDuckGo 的 `max_documents`，达到 limited 证据立即停止；全球 lane 在无国内证据且 Wikipedia 已取得 limited 时停止，不再为了 standard 无条件访问 DuckDuckGo。
6. 支持 `LOG_FORMAT=json` 预设。
7. 联网 smoke 对知名主体必须取得至少一篇有效证据。

优点：当前用户配置无需改变；已用真实请求证明能成功；改动局限在蒸馏链路。

### 方案 B：要求用户另配非思考合成模型（不采用）

实现简单，但把内部兼容问题转嫁给用户；模型管理界面也无法准确表达“女娲必须使用非思考模型”的隐藏限制。

### 方案 C：只提高 max_tokens（不采用）

仍会支付大量思考 token，耗时更长，而且复杂 JSON 仍可能在正文阶段被截断，不能保证成功。

## 详细设计

### 模型请求策略

`NuwaDistillationService` 根据 `base_url` 的 hostname 选择已知能力：

- `api.deepseek.com` 及其子域：`max_tokens=8192`、`response_format={"type":"json_object"}`、`extra_body={"thinking":{"type":"disabled"}}`；
- `api.openai.com`：`max_tokens=8192`、`response_format={"type":"json_object"}`；
- 其他 OpenAI-compatible 端点：保持文本 JSON 提示词，只提升到 `max_tokens=8192`，不发送可能不兼容的厂商参数。

模型返回后先检查响应协议，再解析 JSON：

- `finish_reason == "length"` → `model_output_truncated`；
- `content` 为空 → `model_empty_output`；
- 非法 JSON 或缺少必填结构 → `model_invalid_output`。

三个错误均为 `stage=distill`、`retryable=true`。错误详情只包含 `finish_reason` 和非敏感计数，不包含 `reasoning_content`、提示词、网页正文或 API Key。

### 百度百科多义词解析

`FetchResult` 新增默认空字符串字段 `raw_html`。`WebDocumentFetcher` 对 `text/html` 响应保存已经受 `MAX_BYTES=120000` 限制的 HTML；即使可见正文过短，也返回这段受限 HTML 给特定适配器分析。

`BaiduBaikeResearchProvider` 在首次结果为 `text_too_short` 时：

1. 查找 `window.PAGE_DATA = {...}</script>`；
2. JSON 解析失败则保持原失败；
3. 读取 `navigation.lemmas`；
4. 只接受 `lemmaId` 为正整数且不是布尔值的项；
5. 按 `rank` 升序选择第一项；
6. 拼接固定地址 `https://baike.baidu.com/item/<encoded-subject>/<lemmaId>`；
7. 通过同一个 `WebDocumentFetcher` 再次抓取，使 SSRF、重定向、正文长度和 challenge 校验继续生效。

不得直接访问页面提供的 href、域名或脚本 URL。

### 研究耗时收敛

- `DuckDuckGoResearchProvider.max_documents` 默认改为 3，并在抓取循环中真正执行。
- 每取得一篇文档后重新计算该 provider 的证据等级；达到 `limited` 立即停止。
- 全球 lane 开始时记录是否已有国内证据：
  - 已有国内 limited：继续尝试 Wikipedia，目标是合并成 standard；
  - 没有国内证据：Wikipedia 自身达到 limited 后即可停止并生成有限证据草稿；
  - Wikipedia 无结果或不可用：才进入 DuckDuckGo；
  - 任意时刻达到 standard 立即停止。

本轮不把同步 provider 全部改成异步，不使用无法取消的后台线程伪造硬 timeout。文档中的“6 秒总预算”改为“provider 启动预算”；真实上限由 Wikipedia/DDG 的单请求 timeout、候选上限和 limited 早停共同控制。

### 日志配置

新增纯函数化日志配置模块：

- `json`：使用 `JsonLogFormatter`，通过 `json.dumps` 输出 timestamp、level、logger、message；
- `text` 或空值：使用项目默认文本格式；
- 包含合法 `%(` 占位符的值：作为自定义格式；
- 其他字符串：安全回退默认文本格式，不阻止服务启动。

### 测试与验收

所有生产修改先写失败测试并确认按预期失败。

自动化覆盖：

- DeepSeek 请求参数、截断、空正文、合法 JSON；
- 百度多义词页选择第一义项、非法 PAGE_DATA、不信任外部 URL；
- DDG `max_documents` 与 limited 早停；
- 全球 lane 在 Wikipedia limited 后不调用 DDG；
- `LOG_FORMAT=json` 可格式化日志且不会启动崩溃；
- 前端展示新的模型输出错误并允许重试；
- 联网 smoke 对知名主体必须取得真实证据。

最终人工验收必须使用桌面真实模型配置完成一次成功调用，并只记录：HTTP 状态、耗时、模型名、证据等级、来源数量和草稿名称。不得记录密钥、完整提示词、推理内容或网页全文。

## 非目标

- 不新增搜索 API Key 要求。
- 不更换用户当前 DeepSeek 模型。
- 不重构金丹 CRUD、道人、聊天、融合或 Skill 导出。
- 不自动保存女娲草稿。
- 不解析或绕过验证码。
- 不记录完整模型输出或思维链。

## 完成标准

1. `deepseek-v4-flash` 的真实端到端调用返回 HTTP 200 和可预览金丹草稿。
2. 百度百科多义词知名人物至少取得一篇有效国内证据。
3. 已有 Wikipedia limited 证据时不再无条件访问 DuckDuckGo。
4. `LOG_FORMAT=json` 下 Python 引擎可以启动。
5. 联网测试不再接受“明确失败也算通过”。
6. Python、Go、前端专项与全量门禁通过。
