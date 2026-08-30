# Nuwa 蒸馏集成

本项目参考 [nuwa-skill](https://github.com/alchaincyf/nuwa-skill) 的公开方法论，
把人物资料研究拆为写作、访谈、决策案例、表达习惯、批评与时间线六个维度，再由已配置的
OpenAI 兼容模型提炼心智模型、决策启发式、表达 DNA、价值观、反模式和诚实边界。

上游是 Agent Skill，不是可直接导入的运行时 SDK。因此集成采用端口与适配器结构：

- `ResearchProvider` 是资料收集端口；默认提供无需密钥的公网搜索适配器，后续可增加
  Tavily、Bing 或企业知识库，而不修改金丹与道人业务。
- `NuwaDistillationService` 只消费标准化资料并产生可编辑草稿，不直接保存数据。
- Go 网关解析用户在设置中配置的正式模型凭证，Python 引擎不使用 Mock 模式。
- 网页内容只作为不可信证据文本处理；抓取限制协议、地址、大小、超时与数量。

方法论来源采用 MIT 许可证，许可信息见仓库根目录 `THIRD_PARTY_NOTICES.md`。

## 资料源与失败语义

收集由 `ResearchOrchestrator` 编排：zh locale 先走国内 lane，证据未达 standard 时补全球
lane；非 zh 先国际、subject 含中文时补百度百科。决策只看 locale 与
返回协议，不做 IP/地理/语言归类。

- **百度百科直达页**：中文免密基线。只访问 `baike.baidu.com/item/{subject}`，要求最终 URL
  仍属百度百科、正文 ≥800 字符且出现主题词，否则判 `empty`（防止把错误页当证据）。
  它是"免密基线"，不等于完整互联网证据。百度搜索 HTML 验证码页不可解析，本链路不依赖它。
- **千帆 Web Search**：只在用户已配置对应供应商凭证时启用（无凭证记 `skipped`，不提示缺
  千帆 Key，基本功能不受影响）。401/403 记 `credential_error`。
- **Wikipedia**：国际可达时的免费基线。zh locale 先查中文维基，为空回退英文；超时记
  `unavailable`，不得假设中国大陆网络必然可达。
- **DuckDuckGo HTML**：尽力型补充。202/429 或 anomaly 挑战页记 `blocked`（
  `research_search_blocked`），不会把挑战页当结果。

超时与熔断：国内抓取 3s、国际提供者 4s；6 秒是启动下一个 provider 前的预算（provider
开始后的单个请求由各自 timeout 约束，同步 HTTP 请求无法被可靠硬取消）。任一提供者一次
`unavailable/blocked` 即熔断 600s（测试用 FakeClock 注入）。国内有证据且国际全部不可达时，
草稿仍可生成并追加 warning「国际资料源当前不可达，草稿仅基于国内公开资料」。

全球 lane 早停：进入时无可用证据（insufficient）则第一个 provider 达到 limited 即停止，
Wikipedia limited 可直接生成草稿；已有国内 limited 时允许继续追 standard（跨域合并）。
DDG 默认最多采纳 3 篇文档，达到 limited 立即停止，不再无界抓取候选网页。

证据等级：`standard`（≥4000 字符、≥2 文档、≥2 域名）/ `limited`（≥1500 字符、≥1 文档）/
`insufficient`。`limited` 草稿允许生成，但前端必须提示人工核对，模型也被要求只输出证据
支持的结论并在 `honest_limits` 说明资料空白。

## 稳定错误码与重试

`DistillationError(code, stage, message, retryable, details)` 跨 Python → Go → 前端透传：

| code | 语义 | retryable |
|---|---|---|
| `research_provider_unavailable` | 有技术失败（超时/挑战/熔断）且证据不足，不误报"资料不足" | 是 |
| `research_insufficient_evidence` | 资料确实不足（无技术失败） | 否 |
| `research_search_blocked` | 搜索被 challenge | 是 |
| `model_not_configured` | 未配置模型（400，模型阶段拒绝，不发起远端调用） | 否 |
| `model_timeout` / `model_invalid_output` / `model_request_failed` | 模型侧失败 | 按状态码 |

可重试错误前端给显式「重试」按钮（不做静默自动重试，避免双倍费用）；`insufficient` 给输入
建议；未知错误展示 request id 供日志定位。每次蒸馏记录一条结构化完成日志（request id、
research/model 耗时、provider 状态、候选/采纳数、证据等级、结果状态）；subject 最多记
80 字符并移除换行，brief 与凭证不记录。

## 联网 smoke 测试

`app/tests/test_distillation_research_integration.py` 的组合集成用例零公网依赖（假 HTTP +
假模型），默认全量运行；联网 smoke 默认跳过，需显式选择：

```bash
cd backend/python
.venv/bin/pytest -q -m network_cn app/tests/test_distillation_research_integration.py   # 百度百科/千帆
.venv/bin/pytest -q -m network_global app/tests/test_distillation_research_integration.py # Wikipedia/DDG
```

CI 不要求公网成功；两条 smoke 是发布门禁：

- `network_cn` 是中国大陆发布前门禁，必须取得百度百科有效证据；
- `network_global` 在具备国际网络的 runner 上执行，必须取得 Wikipedia/DDG 有效证据；
- 明确错误状态（`blocked`/`unavailable`/`empty`）只用于诊断，不再算成功；
- 外部站点临时不可用时门禁失败是预期信号，不得改回宽松 OR 断言。
