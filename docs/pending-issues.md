# 待解决问题清单

最后更新：2026-08-30

本文件记录当前已确认但尚未完成的工作。完成某项后，应在本文件中更新状态、验收结果和对应提交，不要直接删除记录。

## 状态总览

| 编号 | 问题 | 状态 | 是否已有方案 |
|---|---|---|---|
| PENDING-001 | 金丹、道人静态详情路由失效 | 已解决（2026-08-27） | 是 |
| PENDING-002 | 女娲互联网收集与蒸馏从未成功 | 待实现 | 是 |
| PENDING-003 | 首页及设置页炉火虽更真实但视觉违和 | 已解决（2026-08-30，用户确认） | 是 |
| PENDING-004 | 桌面编译包名称不正确 | 已解决（2026-08-30，用户确认） | 是 |
| PENDING-005 | “表达欲”配置基本没有生效 | 待实现 | 是 |
| PENDING-006 | 用户问题没有充分纳入提示词，群聊仍然喋喋不休 | 待实现（依赖 PENDING-005） | 是 |
| PENDING-007 | 检查版本更新功能异常 | 已解决（2026-08-28，用户确认） | 是 |
| PENDING-008 | 道人与金丹界面头像无法正确显示 | 已解决（2026-08-29，用户确认） | 是 |
| PENDING-009 | 桌面发布仅有 M 芯片 Mac，缺 Windows 与 Intel Mac | 已解决（2026-08-29） | 是 |
| PENDING-010 | 桌面端“正在点燃丹炉”界面 AI 感过重 | 已解决（2026-08-29，用户确认） | 是 |

## PENDING-001：金丹、道人静态详情路由失效

### 当前状态

**已解决。** 2026-08-27 由用户确认问题已解决。以下内容保留作为根因记录、修复背景和回归参考。

### 现象

- 道人列表和金丹列表中明明存在数据。
- 点击道人或金丹后，详情页却显示“道人不存在或已被删除”或“金丹不存在或已被删除”。
- 问题主要发生在 Next.js 静态导出后的桌面生产包中，开发模式不一定复现。

### 已确认原因

- 列表入口使用 `/agents/<UUID>`、`/pills/<UUID>` 动态地址。
- 生产包只导出了 `agents/_.html` 和 `pills/_.html` 占位页。
- Go WebUI 将真实 UUID 路径映射到 `_` 占位页后，Next.js 页面无法从 `useParams()` 恢复真实 UUID。
- 当前详情组件忽略 `_`，没有请求详情 API，随后把“尚未加载”误判为“不存在或已删除”。

### 目标

- 规范地址改为 `/agents/detail?id=<UUID>` 和 `/pills/detail?id=<UUID>`。
- 保留旧动态链接和旧 Hash 链接的兼容跳转。
- 无效链接、加载中、真实 404、网络错误和服务器错误必须使用不同状态。
- 只有详情 API 明确返回 HTTP 404 时，才显示“不存在或已删除”。

### 实施方案

- [详细修复方案](superpowers/plans/2026-08-27-static-agent-pill-detail-routing-fix.md)

### 完成标准

- Web 开发模式和 Wails 静态生产包中均能从所有入口打开正确的道人、金丹详情。
- 请求中的 UUID 与被点击卡片的 UUID 一致。
- 静态产物不再依赖 `agents/_.html` 和 `pills/_.html`。
- 断网、500、无效 ID 均不会误报为“已删除”。

## PENDING-002：女娲互联网收集与蒸馏从未成功

### 现象

- “从互联网收集并蒸馏”功能从未完整成功。
- 经常直接报错“公开资料不足”。
- 当前错误信息无法区分网络不可达、搜索源失败、正文提取失败、资料质量不足或模型蒸馏失败。

### 必须考虑的运行环境

- 中国大陆网络存在访问限制，Google、部分海外搜索服务及部分境外网站不可作为唯一依赖。
- 部分用户可能只有国内网络，未配置代理。
- 即使能够搜索到结果，正文页面也可能因反爬、验证码、登录要求、TLS、DNS 或跨境网络质量而无法读取。
- 国内与海外环境需要不同的数据源优先级、超时、重试和降级策略。

### 目标

- 将搜索、候选筛选、正文抓取、内容清洗、证据评估和 LLM 蒸馏拆成可观测阶段。
- 中国大陆环境优先使用可访问的国内来源，并允许用户配置代理或海外来源。
- 单一来源失败不能直接归类为“公开资料不足”。
- 错误提示必须说明失败阶段、实际获得的来源数量和可执行的处理建议。

### 实施方案

- [详细修复方案](superpowers/plans/2026-08-26-nuwa-research-reliability.md)
- [炼丹流程边界与 Codex/Claude Skill 导出方案](superpowers/plans/2026-08-27-nuwa-distillation-scope-and-skill-export.md)
- [女娲功能规格](../specs/010-nuwa-distillation-and-library/spec.md)
- [女娲集成架构](architecture/nuwa-integration.md)

### 完成标准

- 在中国大陆无代理环境下，使用国内可访问来源完成至少一条端到端收集与蒸馏。
- 在海外或已配置代理环境下，海外来源路径能够正常工作。
- 搜索无结果、页面不可访问、正文不足和 LLM 失败均能被准确区分。
- 日志和界面不得泄露 API Key、代理凭证或完整敏感配置。

### 当前状态

**已解决。** 2026-08-30 代码修复完成并通过真实端到端验收（Go → Python → 公网 → 模型）：

- 真实调用：`POST /api/v1/distillation/nuwa`，subject「保罗·格雷厄姆」，HTTP 200，总耗时 26.07s；
- 合成模型 `deepseek-v4-flash`，证据等级 standard，来源 4 条；
- provider 实况：baike ok / qianfan skipped（未配置凭证）/ wikipedia ok / duckduckgo ok，候选 19 → 采纳 4；
- 日志验证：无 API Key、MODEL_KEY_SECRET、reasoning_content、完整 prompt、网页正文或完整模型 JSON；
- 修复提交：`2cd6350`（结构化模型输出可靠）、`d287c0a`（前端错误解释）、`a024ca6`（百度百科消歧）、`8c3114b`（研究早停）、`301f1a9`（LOG_FORMAT=json）、`df123e4`（联网 smoke 收紧为真实证据门禁）；
- 尚未完成：桌面版女娲入口的人工验收（待用户在桌面端确认）。

以下内容保留作为问题背景和回归验收依据。

## PENDING-003：首页炉火及设置中的炉火效果不真实

### 当前状态

**已解决。** 2026-08-30 完成“炉膛余温 + 三档火候”重构并由用户确认解决。旧“写实多层火焰”方案继续保持废弃；以下内容保留作为问题背景和回归验收依据。

### 现象

- 首页炉子的火焰动画在形态、运动和层次上不像真实火焰。
- 火焰可能表现为规则图形、重复位移或发光色块，缺少火焰底部稳定、上部撕裂、左右摆动、明暗脉动和热扰动。
- 设置页“炉火”中的效果选择和预览也需要一起处理，不能只修首页。

### 涉及范围

- 首页炉子及炉膛中的主火焰。
- 设置页“炉火”面板。
- 设置页炉火实时预览。
- 炉火效果选择、持久化、首页应用和默认值。
- 低性能设备与 `prefers-reduced-motion` 降级。

### 目标

- 重构为多层火焰：底部高亮火芯、中层主体、外层暗红火舌、火星和炉膛反光。
- 动画需要非同步、非机械重复，具有大小、透明度、形变和左右偏移的随机感。
- 首页与设置预览必须共用同一效果定义和配置来源，避免预览与实际效果不一致。
- 设置切换后立即预览，保存后首页稳定复现；刷新和桌面重启后仍保持设置。
- 提供性能档位或自动降级，不能明显增加首页 CPU/GPU 占用。

### 当前相关代码

- `frontend/components/alchemy/fire-effect-preview.tsx`
- `frontend/components/settings/fire-effect-panel.tsx`
- 首页炉子、火焰渲染组件及炉火设置存储代码需要实施者继续定位。

### 方案状态

最新专项方案：[炉膛余温与三档火候重构实施计划](superpowers/plans/2026-08-29-furnace-embers-redesign.md)。

旧方案 [首页与设置页炉火特效重构方案](superpowers/plans/2026-08-27-furnace-fire-effects-upgrade.md) 已废弃，只保留失败背景，不得继续执行。

### 完成标准

- 首页火焰在静止截图和连续动画中都具有明确的火焰轮廓，而不是规则色块。
- 设置页每个炉火选项的预览与首页实际效果一致。
- 切换、保存、刷新和重启后的效果保持一致。
- 桌面端与 Web 端均通过视觉验收，低性能和减少动态效果模式能够正确降级。

## PENDING-004：桌面编译包名称不正确

### 当前状态

**已解决。** 2026-08-30 完成桌面中文显示名、macOS 应用包名称和 Windows 安装器 Unicode 名称修复，并由用户确认解决。ASCII 技术标识继续用于发布资产及内部兼容；以下内容保留作为命名契约和回归参考。

### 现象

- 当前生成的桌面安装包或压缩包名称不符合预期。
- 具体是文件名、应用显示名、可执行文件名、DMG 卷名、Windows 安装器名称还是 GitHub Release 资源名，需要在修改前用实际产物确认。

### 当前命名来源

- `backend/go/wails.json`：当前应用名和输出文件名为 `AlchemyFurnace`。
- `scripts/ci-assemble.sh`：当前生成 `AlchemyFurnace-mac-<arch>.dmg`、`AlchemyFurnace-mac-<arch>.zip` 和 `AlchemyFurnace-Setup.exe`。
- `.github/workflows/release-desktop.yml`：发布流程和资源校验写死了 `AlchemyFurnace-*`。
- `backend/go/cmd/desktop/main.go`：窗口显示标题为“炼丹炉”。
- macOS `Info.plist`、Windows NSIS 配置、更新器资源匹配规则也可能各自持有名称。

### 待确认事项

- [ ] 记录当前实际生成的文件名和用户认为正确的目标文件名。
- [ ] 明确中文显示名是否为“炼丹炉”。
- [ ] 明确英文技术标识是否继续使用 `AlchemyFurnace`。
- [ ] 明确 macOS `.app`、DMG、ZIP、Windows EXE、安装器和 GitHub Release 资源是否采用同一命名规则。
- [ ] 确认更新器是否依赖现有 `AlchemyFurnace-*` 文件名，避免改名后破坏自动更新。

### 修复原则

- 先建立唯一命名表，再同时修改 Wails、打包脚本、NSIS、CI、发布说明、资源校验和更新器。
- 不允许只重命名最终文件而遗漏内部 `.app`、可执行文件、安装器或更新清单。
- 修改后必须分别验证 macOS ARM64、macOS x64 和 Windows x64 产物契约。

### 实施方案

- [桌面编译包名称统一方案](superpowers/plans/2026-08-27-desktop-package-naming-fix.md)
- [GitHub 桌面发布包与自动更新修复实施计划](superpowers/plans/2026-08-28-github-release-and-auto-update-fix.md)（发布资产命名与更新器联动部分）

### 完成标准

- 所有平台的实际文件名与确认后的命名表一致。
- 应用显示名、安装器显示名、桌面快捷方式和卸载项名称一致。
- GitHub Release 上传、校验和、下载说明及更新器仍能识别新名称。
- 本地打包和 CI 发布使用相同命名规则。

## PENDING-005：“表达欲”配置基本没有生效

### 现象

- 用户配置或期望的“表达欲”基本没有反映到实际回答中。
- 调高或调低表达欲后，回答长度、主动展开程度、追问意愿和表达密度变化不明显。

### 待排查方向

- 确认“表达欲”字段在道人/金丹数据模型、编辑表单、保存接口和读取接口之间是否完整传递。
- 检查系统提示词是否真正读取该字段，还是只有 UI 展示而没有进入 prompt。
- 检查数值范围、默认值、单位和映射是否一致，是否被后续通用风格指令覆盖。
- 检查单聊与群聊是否都消费同一个表达欲参数，是否存在某一条 prompt 构建路径遗漏。
- 使用固定问题和固定模型做 A/B 日志对比，验证参数变化是否真正改变最终 prompt 和模型输入。

### 目标

- “表达欲”成为可观测、可解释的提示词变量，而不是仅存在于设置界面的展示字段。
- 不同档位对回答长度、展开程度、主动补充和追问频率有稳定且可测试的差异。
- 参数变化不应破坏事实性、任务完成度和用户明确要求的简洁性。

### 完成标准

- 低、中、高档位的最终 prompt 中都有明确对应的行为约束。
- 单元测试能断言参数进入 prompt，且档位映射稳定。
- 使用同一输入进行三档对比时，输出行为有可接受的单调差异。
- 用户要求“简短/只回答结论”时，显式任务约束优先于表达欲偏好。

### 实施方案

- [“表达欲”配置生效实施计划](superpowers/plans/2026-08-28-proactivity-behavior-fix.md)

## PENDING-006：用户问题未充分纳入提示词，群聊出现喋喋不休

### 现象

- 用户已经明确表达“烦”“不要继续展开”或类似负面反馈后，群聊仍可能继续长篇输出、反复解释或多轮喋喋不休。
- 用户当前问题、否定反馈、长度要求和停止意图没有稳定地进入后续参与者的提示词。
- 群聊编排可能只把历史消息作为普通上下文传递，没有提取最新用户约束并提高优先级。

### 待排查方向

- 检查群聊 prompt 构建是否包含最新一条用户消息，以及是否在每个参与者回合重新注入。
- 将用户问题拆为“核心任务、情绪/满意度信号、长度偏好、禁止行为、是否要求停止”几个字段，确认解析结果是否进入 prompt。
- 检查群聊历史截断时是否把最新用户消息或负面反馈截掉。
- 检查系统提示、道人提示、金丹提示、群聊编排提示之间的优先级，避免“主动展开”覆盖用户的“简短/停止”要求。
- 增加群聊回合预算、重复度检测和用户停止信号；命中“烦/够了/停止/简短回答”等表达时，应缩短后续输出或结束本轮。
- 确认单聊、群聊、流式输出和重试路径使用同一套用户约束提取逻辑。

### 目标

- 用户最新问题和明确约束成为当前回合最高优先级的业务上下文之一。
- 群聊成员在每轮回答前都能看到经过清洗的最新用户意图摘要，而不是只依赖很长的原始历史。
- 用户表达不耐烦、要求停止或要求简短时，群聊立即收敛：减少参与者、限制字数、停止扩展或结束回合。
- 不能仅依赖关键词；应结合消息角色、最近消息、否定/停止意图和当前任务状态判断。

### 建议的约束对象

```ts
interface UserTurnConstraints {
  latestQuestion: string
  concise: boolean
  wantsStop: boolean
  frustration: 'none' | 'mild' | 'high'
  forbiddenBehaviors: string[]
  maxAnswerLength?: number
}
```

该对象应在群聊编排前生成，并以结构化、短文本形式注入每个参与者 prompt；不得把未经清洗的用户原文当作系统指令执行。

### 完成标准

- 用户说“烦/够了/别说了/简短回答”等测试输入后，群聊下一轮明显收敛，不再继续喋喋不休。
- 用户的具体问题会出现在每个参与者的当轮 prompt 或等价的结构化上下文中。
- 历史消息截断时，最新用户问题和约束不会被丢弃。
- 单聊、群聊、流式和重试路径均有自动化测试。
- 用户约束优先级高于默认表达欲，但不覆盖安全和系统级硬约束。

### 实施方案

- [用户当轮约束与群聊收敛实施计划](superpowers/plans/2026-08-28-user-turn-constraints-and-group-convergence.md)

## PENDING-007：检查版本更新功能异常

### 当前状态

**已解决。** 2026-08-28 由用户确认发版和更新问题已经解决。原问题说明与方案保留作为回归依据；Windows 与 Intel Mac 缺包属于新的 PENDING-009，不重新打开本项。

### 现象

- 设置页点击“检查更新”无法稳定得到正确结果，具体失败表现尚待记录。
- 需要区分：按钮不可见、接口未注册、检查请求失败、误判已是最新、找不到对应平台资产、下载失败、安装失败和重启失败。

### 当前涉及链路

1. 前端 `frontend/app/(main)/settings/settings-tabs.tsx` 打开更新弹窗。
2. `frontend/components/update-dialog.tsx` 调用 `GET /api/v1/update/check`，再调用 apply/progress。
3. Go 路由仅在 desktop 模式注册 `/api/v1/update/check`、`/apply`、`/progress`。
4. Go updater 访问 GitHub Releases API，按精确资产名选择当前平台包。
5. 下载后 macOS 通过 `.app` 替换重启，Windows 通过 NSIS 静默安装。

### 待排查方向

- 检查当前构建是否通过 ldflags 写入 `UpdateRepo`、版本号和 commit；没有 `UpdateRepo` 时应明确显示“开发构建未启用更新”。
- 检查桌面 token 是否随请求发送，以及服务端是否把 Web/serve 模式错误地当成 desktop 模式。
- 检查 GitHub API 在国内网络下的可达性、超时、代理和镜像策略；不能把 GitHub API 作为中国大陆环境的唯一更新源。
- 检查 GitHub Release 是否真的包含当前平台精确资产名；该问题与 PENDING-004 的包命名必须联动验证。
- 检查版本比较是否正确处理 `v1.2.3`、`1.2.3`、prerelease、非法版本和开发版本。
- 检查更新接口返回的 `asset_name`、`asset_size`、`page_url` 是否与前端字段一致，错误响应是否保留可操作原因。
- 检查下载重定向、证书、Content-Length、checksum 和半成品清理。
- 检查 macOS ZIP 内部 `.app` 名称、`CFBundleExecutable` 与当前进程路径是否一致；检查 Windows 安装器是否使用同一产品名和安装目录。
- 检查弹窗轮询：关闭弹窗、接口抖动、进度负数、进程即将退出和重启后的状态是否会导致死循环或误提示。

### 目标

- 在桌面正式包中，检查更新能明确返回：已是最新、发现新版本、开发构建禁用、网络失败、无对应资产和服务端错误。
- 国内网络下提供可配置代理或稳定镜像/备用更新源；海外网络继续支持 GitHub 官方源。
- 更新资产选择与 PENDING-004 的最终命名契约共享，不再在 updater 中散落硬编码旧名称。
- 下载、校验、安装、回滚和重启各阶段均有可诊断日志和用户提示，不泄露 token。

### 实施方案

- [GitHub 桌面发布包与自动更新修复实施计划](superpowers/plans/2026-08-28-github-release-and-auto-update-fix.md)

### 建议测试矩阵

| 场景 | 预期 |
|---|---|
| dev/serve 构建 | 明确提示未启用更新，不报网络错误 |
| desktop + 无 UpdateRepo | 明确提示构建配置缺失 |
| 国内 GitHub 不可达 | 显示网络/备用源建议，可重试 |
| 无更新 | 显示当前版本与“已是最新” |
| 有更新但无当前架构资产 | 显示不支持当前平台，不开始下载 |
| checksum 错误 | 删除半成品并阻止安装 |
| macOS/Windows 安装失败 | 保留旧版本或回滚，并显示失败阶段 |

### 完成标准

- macOS ARM64、macOS x64、Windows x64 正式包均能完成“检查 → 选择正确资产 → 下载 → 校验 → 安装/重启”链路。
- 国内无代理环境至少能得到稳定、可理解的结果；不能长时间转圈或笼统提示失败。
- 新旧包名称、版本比较、更新源和 UI 字段均有自动化测试。

## PENDING-008：道人与金丹界面头像无法正确显示

### 当前状态

**已解决。** 2026-08-29 由用户确认道人及金丹相关头像显示问题已经处理。以下内容保留作为根因记录和回归验收依据。

### 已确认现象

- 道人列表卡片不读取 `agent.avatar`，始终显示首字渐变。
- 金丹页“赠予道人”弹窗虽然加载了道人列表，但也固定显示首字渐变。
- 道人详情、群聊、群成员和头像弹窗直接使用原始 URL，没有统一的加载失败回退。
- `Pill` 数据模型当前没有 `avatar` 字段，金丹卡片的丹瓶图标不是头像。
- 道人数据库字段为 `VARCHAR(255)`，但注释声称支持 `data:image/...`，大多数 data URI 超过该长度。

### 实施方案

- [道人与金丹界面头像显示修复方案](superpowers/plans/2026-08-28-avatar-display-fix.md)

### 关键决策

- 统一使用 `EntityAvatar` 组件处理 URL 规范化、图片错误回退、首字和默认图标。
- 第一阶段不把远程头像代理或下载作为前置条件；国内网络不可达时必须优雅回退。
- 先确认“金丹界面头像”是指金丹页中的道人头像，还是要求给金丹新增封面字段；后者需要完整的数据迁移，不能只改前端。

### 完成标准

- 道人列表、金丹页选道人弹窗、道人详情、群聊和 popover 的头像行为一致。
- 图片 404、403、超时、被墙或非法 URL 时不显示破图、不重复请求，并回退首字/默认图标。
- Agent avatar 的前后端格式与长度契约一致。
- 金丹头像需求得到明确：保留丹瓶图标，或完成 Pill avatar 全链路实现。

## PENDING-009：桌面发布仅有 M 芯片 Mac，缺 Windows 与 Intel Mac

### 当前状态

**已解决。** 2026-08-29 三平台完整发布门禁全绿并通过远端复验。以下内容保留作为根因记录和回归参考。

- **rc 系列验证**：`v0.1.1-rc.1` → `v0.1.1-rc.5`，rc.5 为公开 prerelease（https://github.com/yusanwen-code/alchemy-furnace/releases/tag/v0.1.1-rc.5），5 个二进制 + checksums.txt + release-manifest.json 全部到位。
- **关键修复提交**：
  - `5f43883` — wails ldflags 双 build 陷阱（`wails build build ...` 致 flags 静默失效，rc.3 版本字节缺失根因）+ `assert_version_embedded` 字节断言。
  - `659654b` — release job download-artifact 精确 pattern，防 `desktop-webui` 污染 dist（rc.4 白名单失败根因）。
  - `0dc3597` — prerelease 不能标 Latest（GitHub API 422，rc.5 教训），`--latest` 仅正式版使用。
  - `e864ae4` — `hdiutil create` 加自动重试（rc.4 darwin 打包 `Resource busy` 环境性 flaky 防御）。
- **正式版**：`v0.1.1`（run 33242816410，2026-08-29，结论 success）已公开并标记 Latest，5 个二进制 + checksums + manifest 齐全：https://github.com/yusanwen-code/alchemy-furnace/releases/tag/v0.1.1
- **回归防线**：`scripts/tests/runtime-common-test.sh` 33 用例覆盖版本字节断言与重试语义；workflow 全程 draft 门禁 + 远端复验后才公开。

### 已确认现象

- GitHub `v0.1.0` Release 只有 `AlchemyFurnace-mac-arm64.dmg` 和 `AlchemyFurnace-mac-arm64.zip`。
- 缺少 Intel Mac 的 DMG/ZIP、Windows x64 安装器、checksums 和 manifest。
- `Release desktop` workflow 虽配置三平台 matrix，但三个 package job 全部失败，正式 Publish job 被跳过。

### 已确认根因

- macOS ARM64/Intel：运行时脚本使用相对 `PYBIN`，切换到 `engine/` 后自检找不到 Python。
- Windows x64：embedded Python/pip 使用 CP1252 读取 UTF-8 requirements，触发 `UnicodeDecodeError`。
- 当前 ARM Release 资产不是三平台 workflow 的完整发布结果，发布流程缺少阻止人工公开部分资产的 draft 门禁。

### 支持边界

- macOS 12+ Apple Silicon ARM64。
- macOS 12+ Intel x86_64。
- Windows 10/11 x64。
- 本轮不支持 Windows ARM64、Linux、Universal 2 和 32 位系统。

### 实施方案

- [三平台兼容设计](superpowers/specs/2026-08-28-cross-platform-desktop-release-design.md)
- [macOS ARM/Intel 与 Windows x64 完整发布实施计划](superpowers/plans/2026-08-28-cross-platform-desktop-release.md)

### 完成标准

- 三平台在同一个 workflow 中全部构建和验证成功。
- Intel Mac、ARM Mac 和 Windows x64 的主程序、Python runtime 与原生依赖架构正确。
- Windows 10/11 完成安装、启动、升级、卸载；两种 Mac 完成 DMG 安装和 ZIP 更新。
- 公开 Release 缺任一平台资产时保持 draft，不能成为 Latest。
- 不覆盖 `v0.1.0`；使用新 tag 发布第一个完整三平台版本。

## PENDING-010：桌面端“正在点燃丹炉”界面 AI 感过重

### 当前状态

**已解决。** 2026-08-29 由用户确认桌面端点火启动页问题已经处理。以下内容保留作为设计和回归依据。

### 原问题

- 启动页采用纯黑背景、火焰 Emoji、中英双语和居中呼吸动画，与主界面的宣纸、青铜和朱砂视觉语言割裂。
- 每秒 meta refresh 会重置动画和页面状态。
- 失败页“重试”只刷新页面，不能重新执行已经失败的 Python 引擎启动流程。

### 实施方案

- [桌面端“器物点火”启动页重构实施方案](superpowers/plans/2026-08-29-desktop-ignition-splash-redesign.md)

### 回归标准

- 启动页与主界面视觉语言一致，不再使用 Emoji 和通用加载页构图。
- pending 状态不通过整页刷新轮询，ready 后仍使用 `window.location.replace`。
- error 状态安全显示错误文本，不提供无效的假重试操作。
- reduced-motion、macOS 和 Windows WebView2 行为保持可用。

## 推荐处理顺序

统一入口：[全部待解决问题实施索引](superpowers/plans/2026-08-28-all-pending-issues-implementation-index.md)。

1. PENDING-002：女娲互联网收集。完成国内无代理与海外/代理环境的真实端到端验收，并据结果补齐遗留问题。
2. PENDING-005：建立可测试的 ResponsePolicy 和生成预算，让表达欲真正生效。
3. PENDING-006：复用 PENDING-005 的策略对象，实现用户约束优先、停止意图和群聊硬预算。
