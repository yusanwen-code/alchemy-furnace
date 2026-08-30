# 道人行为、金丹化性与本地记忆统一设计

**日期：** 2026-08-30

**状态：** 已经用户确认方案 1，等待实施计划

**产品边界：** 仅维护 Wails 桌面版

**取代方案：**

- `docs/superpowers/plans/2026-08-28-proactivity-behavior-fix.md`
- `docs/superpowers/plans/2026-08-28-user-turn-constraints-and-group-convergence.md`

上述两份旧计划不再单独执行。本设计将“金丹效果可感知”“表达欲生效”“群聊收敛”“本地记忆与真人感”合并为一个原子产品任务，避免多个提示词补丁互相覆盖。

## 1. 背景与问题定义

当前道人虽然能服用金丹，聊天链路也会读取合成后的 `LanguagePattern.SystemPrompt`，但实际体验仍接近直接与基础模型聊天。用户已经明确指出以下问题：

1. 金丹给道人后缺少稳定、可辨认的行为变化。
2. `proactivity`（表达欲）基本只停留在数据字段和提示词文案，没有稳定控制回答长度、主动发言概率和生成预算。
3. 围炉论道没有可靠执行“简短、停止、别展开”等用户要求，多个道人可能连续输出重复内容。
4. 道人无法跨会话记住用户偏好、关系变化和未完成约定，缺少持续相处感。
5. 现有合成降级会丢失心智模型、决策启发式、价值观、反模式和示例对话。用户已明确确认：这种降级不可接受。

本任务不以“增加更多提示词”为目标，而是建立一套可观察、可测试、可降级但不丢语义的统一行为系统。

## 2. 已确认的根因

### 2.1 金丹链路不是完全断开，而是影响被压缩和旁路

现有数据流为：

```text
AgentPill + ElixirPill.SkillSchema
  → LanguagePatternService.GetOrBuildPattern
  → Python LanguageSynthesisService.combine
  → LanguagePattern.SystemPrompt
  → 单聊或群聊模型请求
```

代码证据：

- `backend/go/internal/service/language_pattern_service/language_pattern_service.go` 会加载道人、金丹、剂量和顺序，并调用 Python 合成。
- `backend/go/server/http/gateway/web/handler/chat/impl_sse_chat.go` 的单聊请求会使用 `pattern.SystemPrompt`。
- `backend/go/internal/service/chat_service/group_orchestrator.go` 的群聊请求也会以 `pattern.SystemPrompt` 为基础。

因此问题不是“金丹完全没有进入请求”，而是进入请求后的语义保障不足。

### 2.2 `EmergenceRules` 没有进入实际聊天

Python 会返回 `emergence_rules`，Go 会将其保存到 `LanguagePattern.EmergenceRules`，前端道人详情也会展示；但单聊和群聊构造模型消息时只使用 `SystemPrompt`。独立保存的涌现规则没有在运行时形成强约束。

### 2.3 合成降级不是无损降级

`LanguageSynthesisService._fallback_prompt` 当前只保留基础性格、部分表达 DNA、词汇、禁忌词和诚实边界。下列字段会消失：

- `identity_card`
- `description`
- `mental_models`
- `decision_heuristics`
- `values`
- `anti_patterns`
- `example_dialogues`
- 多丹组合产生的确定性冲突处理结果

这意味着合成模型不可用、输出为空或 JSON 解析失败时，金丹的核心能力会被静默削弱。

### 2.4 表达欲与群聊收敛只依赖模型自觉

`BuildGroupSystemPrompt` 会告诉模型“表达欲越高越健谈”，但编排器仍可能为所有道人启动模型调用，最多运行三轮。单聊也没有将表达欲映射到句数和 token。用户停止、简短、烦躁等当轮约束尚未形成代码级策略。

### 2.5 现有测试没有证明“金丹可辨认”

现有测试能证明：

- 金丹字段进入结构化合并器；
- 指纹和缓存失效工作；
- 合成器能够返回系统提示词；
- 群聊提示词包含表达欲文案。

但没有证明：

- 同一道人服丹前后的回答可被稳定区分；
- 某颗金丹的心智模型或启发式实际在相关问题中生效；
- 合成降级保留全部字段；
- 低、中、高表达欲产生单调可测的行为差异。

## 3. 设计目标

### 3.1 必须实现

1. 每颗已服用金丹在每轮至少贡献身份、表达 DNA、价值观、反模式和诚实边界。
2. 与问题相关的心智模型、决策启发式和示例对话必须在本轮显式激活。
3. 合成成功和降级路径都不得丢失任何已知金丹字段。
4. LLM 合成只能增加涌现层，不能取代或覆盖原始金丹事实。
5. 表达欲必须由代码控制发言资格、默认句数和 `max_tokens`。
6. 用户当轮要求必须高于表达欲和默认群聊策略。
7. 群聊必须有发言人数、轮数、重复度和停止条件硬上限。
8. 每位道人拥有相互隔离、仅保存在本地数据库的记忆。
9. 记忆允许使用当前已配置模型提炼，但不接入 Mem0 云服务、外部记忆 API 或外部向量数据库。
10. 用户能在道人详情中查看、编辑、固定、删除、清空和关闭自动记忆。
11. 所有验收以 Wails 桌面版为准。

### 3.2 明确不做

- 不引入 AutoGen、Mem0 或 SillyTavern 作为运行时依赖。
- 不恢复 Qdrant，不新增独立向量数据库。
- 不构建独立 Web 产品能力。
- 不让道人虚构身体、现实经历或人类身份来伪装真人。
- 不通过随机等待、假输入状态或故意拖慢回答制造真人感。
- 不让记忆内容取得 system 指令权限。
- 不将用户聊天正文、记忆或提示词写入普通日志。

## 4. 外部参考及采用边界

### 4.1 Mem0

- 项目：<https://github.com/mem0ai/mem0>
- 调研时约 64.3k Star，Apache-2.0。
- 借鉴：候选记忆提炼、跨会话检索、时间与实体信号、记忆评测思路。
- 不采用：云服务、SDK、托管存储、外部 embedding 和向量库。

### 4.2 Microsoft AutoGen

- 项目：<https://github.com/microsoft/autogen>
- 调研时约 60.7k Star，代码 MIT；项目已进入维护模式。
- 借鉴：候选发言者选择、禁止连续重复发言、终止条件、最大轮数。
- 不采用：框架依赖、额外的 LLM 发言选择器和 AutoGen 运行时。

### 4.3 SillyTavern

- 项目：<https://github.com/SillyTavern/SillyTavern>
- 群聊说明：<https://docs.sillytavern.app/usage/core-concepts/groupchats/>
- 调研时约 32.8k Star，AGPL-3.0。
- 借鉴：点名必答、Talkativeness 控制未点名角色发言、每次只注入当前发言者角色信息。
- 不复制其 AGPL 代码，不使用其角色卡实现。

### 4.4 Generative Agents

- 项目：<https://github.com/joonspk-research/generative_agents>
- 论文：<https://arxiv.org/abs/2304.03442>
- 调研时约 22k Star，Apache-2.0。
- 借鉴：观察、记忆检索、反思共同提高行为可信度的设计结论。
- 不采用：模拟世界、日程规划和自主行动系统。

## 5. 总体架构

新增统一的“道人行为引擎”，由四个可独立测试的单元组成：

```text
┌──────────────────────────────────────────────────────────┐
│                   Daoist Behavior Engine                  │
├────────────────┬────────────────┬────────────────────────┤
│ Pill Compiler  │ Memory Service │ Turn Policy Engine     │
│ 金丹无损编译    │ 本地记忆       │ 约束/表达欲/群聊预算    │
├────────────────┴────────────────┴────────────────────────┤
│ Prompt Composer：按权限分区构造最终模型请求               │
└──────────────────────────────────────────────────────────┘
```

每轮数据流：

```text
最新用户消息
  → ExtractUserTurnConstraints
  → ActivatePillRules
  → RetrieveAgentMemories
  → PolicyForProactivity
  → BuildTurnPlan
  → ComposeMessages
  → StreamChat(max_tokens)
  → 保存回复
  → 异步提炼候选记忆
  → 校验、去重后写入本地数据库
```

统一优先级：

```text
安全硬约束
  > 用户当轮明确要求
  > 金丹禁忌与诚实边界
  > 本轮激活的金丹能力
  > 本地记忆事实
  > 道人基础性格
  > 表达欲默认行为
  > 群聊默认值
```

## 6. 金丹无损编译

### 6.1 核心原则

原始 `ElixirPill.SkillSchema` 是唯一事实来源。LLM 生成的自然语言提示词不是事实来源。

每次缓存重建分为三个阶段：

1. **Canonical Compile：** 纯函数将基础性格、金丹、权重和顺序编译为完整结构化档案。
2. **Optional Emergence：** 合成模型只产生涌现规则、冲突调和建议和可选的语言润色层。
3. **Deterministic Render：** 纯函数把完整结构化档案与可用的涌现层渲染为运行时提示词。

如果第二阶段失败，直接跳过涌现层；第一和第三阶段仍必须成功，并完整保留所有字段。

### 6.2 数据契约

```go
type CompiledPillProfile struct {
    PillID             string                 `json:"pill_id"`
    Name               string                 `json:"name"`
    Weight             float64                `json:"weight"`
    SortOrder          int                    `json:"sort_order"`
    IdentityCard       string                 `json:"identity_card"`
    Description        string                 `json:"description"`
    ExpressionDNA      map[string]any         `json:"expression_dna"`
    MentalModels       []map[string]any       `json:"mental_models"`
    DecisionHeuristics []map[string]any       `json:"decision_heuristics"`
    Values             []string               `json:"values"`
    AntiPatterns       []string               `json:"anti_patterns"`
    HonestLimits       []string               `json:"honest_limits"`
    ExampleDialogues   []map[string]any       `json:"example_dialogues"`
    UnknownFields      map[string]any         `json:"unknown_fields"`
}

type DaoistBehaviorProfile struct {
    Version           int                   `json:"version"`
    BasePersonality   string                `json:"base_personality"`
    Pills             []CompiledPillProfile `json:"pills"`
    EmergenceRules    []string              `json:"emergence_rules"`
    InnerTensions     []InnerTension        `json:"inner_tensions"`
    EmergenceDegraded bool                  `json:"emergence_degraded"`
    DegradedReason    string                `json:"degraded_reason,omitempty"`
}
```

`UnknownFields` 用于保证未来新增的 `skill_schema` 键不会在编译过程中消失。它们必须保存在档案中，并在“扩展字段”分区渲染为 JSON；低阶模型不得用白名单序列化覆盖整个原对象。

### 6.3 缓存模型

在 `language_patterns` 增加：

- `behavior_profile JSONB NOT NULL`
- `profile_version INTEGER NOT NULL DEFAULT 1`

保留现有 `system_prompt` 作为预渲染缓存，避免每轮重复序列化。`SourceFingerprint` 继续覆盖基础性格、金丹 ID、名称、权重、顺序和完整 `skill_schema`。

旧缓存缺少 `behavior_profile` 时视为失效，在下一次聊天前自动重建。

### 6.4 无损降级契约

降级渲染必须包含：

| 字段 | 每轮核心区 | 相关时激活区 | 可否丢失 |
|---|---:|---:|---:|
| identity_card | 是 | — | 否 |
| description | 摘要 | 完整 | 否 |
| expression_dna | 是 | — | 否 |
| mental_models | 名称 | 细节 | 否 |
| decision_heuristics | 条件索引 | 命中的完整规则 | 否 |
| values | 是 | — | 否 |
| anti_patterns | 是 | — | 否 |
| honest_limits | 是 | — | 否 |
| example_dialogues | 索引 | 最相关示例 | 否 |
| unknown_fields | 扩展分区 | — | 否 |

“相关时激活”只影响本轮提示词预算，不代表从档案或缓存删除。每颗金丹每轮都必须在核心区留下可见贡献。

### 6.5 金丹规则激活

候选关键词来自：

- 金丹名称和标签；
- `description`；
- `expression_dna.vocabulary`；
- `mental_models.name`、`one_liner`、`application` 和 `detection_questions`；
- `decision_heuristics.condition` 和 `case`；
- `example_dialogues.user`。

使用本地规范化、关键词命中和字符 bigram 相似度评分，不额外调用选择模型。每颗金丹最多激活：

- 2 个心智模型；
- 3 条决策启发式；
- 1 条示例对话。

如果没有任何相关规则命中，按权重和服用顺序为每颗金丹选取一个代表性心智模型或启发式，确保金丹不会完全隐身。

## 7. 表达欲与生成预算

### 7.1 基础策略

```go
type ResponsePolicy struct {
    Band             string
    VolunteerPercent int
    MaxSentences     int
    MaxTokens        int
}
```

固定映射：

| 表达欲 | Band | VolunteerPercent | MaxSentences | MaxTokens |
|---|---|---:|---:|---:|
| 0–20 | quiet | 原始值 | 1 | 160 |
| 21–40 | reserved | 原始值 | 2 | 256 |
| 41–60 | balanced | 原始值 | 3 | 384 |
| 61–80 | talkative | 原始值 | 5 | 640 |
| 81–100 | expansive | 原始值 | 8 | 896 |

规则：

- 单聊始终 `must_answer=true`，表达欲只控制默认长度和主动展开。
- 群聊明确被 `@` 的道人必须回答。
- 未被点名的道人先通过相关性门槛，再通过稳定表达欲桶。
- 稳定桶使用 `SHA256(sessionUUID|agentID|userMessageUUID|round)`，禁止全局随机数。
- 当前用户明确要求详细时可提高默认句数和 token，但不得超过模型配置和全局 8192 上限。
- 当前用户要求简短时必须收紧高表达欲道人。

### 7.2 生成接口

Go 的聊天接口增加显式生成选项：

```go
type GenerationOptions struct {
    MaxTokens int
}

StreamChat(
    ctx context.Context,
    messages []map[string]string,
    creds *credential.ModelCredentials,
    options GenerationOptions,
    onChunk func(string),
) (fullContent string, canceled bool, err error)
```

所有调用点必须显式传值；禁止用可变参数或隐藏默认值掩盖漏改。标题生成继续使用独立小预算。

## 8. 用户当轮约束

### 8.1 约束对象

```go
type FrustrationLevel string

type UserTurnConstraints struct {
    LatestQuestion string
    Concise        bool
    Detailed       bool
    WantsStop      bool
    Frustration    FrustrationLevel
    OneEach        bool
}
```

提取器为无模型纯函数，先识别否定与继续表达，再识别停止，避免把“不要停，继续说”误判为停止。

用户原文只保留在 `user` role；system patch 只写可信布尔值和整数，不把用户原文提升为系统指令。

### 8.2 合并结果

```go
type TurnPlan struct {
    Stop           bool
    MustAnswer     bool
    MaxSentences   int
    MaxTokens      int
    MaxTurnTokens  int
    MaxSpeakers    int
    MaxRounds      int
    OneEach        bool
    ActivatedRules []ActivatedPillRule
    Memories       []MemorySnippet
}
```

固定行为：

| 用户意图 | MaxSpeakers | MaxRounds | MaxSentences | 单人 MaxTokens | MaxTurnTokens |
|---|---:|---:|---:|---:|---:|
| 明确停止 | 0 | 0 | 0 | 0 | 0 |
| 简短或高烦躁 | 1 | 1 | ≤2 | ≤256 | ≤256 |
| 每人一句 | 成员数 | 1 | 1 | ≤160 | `min(成员数×160, 4096)` |
| 普通讨论 | ≤2 | ≤2 | 基础策略 | 基础策略 | ≤1280 |
| 明确详细讨论 | ≤3 | ≤2 | 可高于基础策略 | 群聊≤1536 | ≤3072 |

“烦”本身不等于停止；只有明确的结束意图才零调用。

单聊明确要求详细时单次 `MaxTokens` 上限为 2048；任何路径都不得突破模型自身配置和 Python 请求契约的 8192 上限。群聊编排器必须在启动下一位道人前检查剩余 `MaxTurnTokens`，不能只限制每个调用而忽略整轮总量。

## 9. 群聊编排

### 9.1 发言候选排序

每轮按以下顺序选择：

1. 用户明确点名或 `@`；
2. 金丹激活规则与问题的相关性；
3. 本地记忆与问题的相关性；
4. 表达欲稳定分数；
5. 最近发言冷却惩罚。

普通问题至少选出一名可用的主回答者，除非用户明确停止或全部道人不可用。即使所有道人表达欲为 0，系统也会将相关度最高者指定为本轮 `must_answer` 的主回答者；这属于回应用户，不计为自愿发言。第二名道人只有在能提供不同金丹视角且通过表达欲门槛时才补充。

### 9.2 轮次与终止

- 同一道人不得在同一用户回合连续发言。
- 默认只运行一轮；只有明确 `@` 交接时允许第二轮。
- 达到 `MaxSpeakers`、`MaxRounds` 或 token 预算立即结束。
- 用户停止时保存用户消息并直接发送 `turn_done(spoke=0)`。
- 回复与本轮已有回答的规范化 bigram Jaccard 相似度达到 `0.85` 时，保存当前已输出回复，但不再启动下一名道人。
- 短于 8 个 Unicode 字符的回复不做重复拦截。
- 单个自愿发言者失败时允许下一名合格道人继续；明确被点名者失败时必须向用户显示该道人失败，不能静默换人冒充。

### 9.3 金丹身份隔离

每次生成只注入当前发言道人的完整行为档案。其他成员只以姓名、最近发言和公开头像身份进入上下文，禁止把全部成员金丹合并到同一 system prompt，以免角色串味。

## 10. 本地记忆

### 10.1 存储边界

新增 `agent_memories` 表。所有记录存入桌面应用现有本地数据库；不发送给任何记忆服务，也不建立外部索引。

```go
type AgentMemory struct {
    ID              uint
    UUID            uuid.UUID
    AgentID         uint
    Kind            string
    Content         string
    Keywords        JSONList
    Importance      int
    Confidence      float64
    Pinned          bool
    Status          string
    SourceSessionID *uint
    SourceMessageID *uint
    ContentHash     string
    LastAccessedAt  *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

`Kind` 仅允许：

- `user_fact`
- `user_preference`
- `relationship`
- `open_loop`
- `episode`

`Status` 仅允许 `active`、`superseded`、`archived`。

在 `dao_agents` 增加 `memory_enabled BOOLEAN NOT NULL DEFAULT TRUE`。

### 10.2 提炼流程

完成一轮回复后，将本轮用户消息、道人回复和最少必要上下文提交给当前已配置模型，要求输出结构化候选：

```json
{
  "memories": [
    {
      "kind": "user_preference",
      "content": "用户偏好先看结论再看论证",
      "keywords": ["结论", "论证", "表达偏好"],
      "importance": 4,
      "confidence": 0.9
    }
  ]
}
```

约束：

- 提炼调用不得阻塞聊天正文展示；使用单 worker、容量 32 的本地后台队列串行执行。队列已满时跳过本轮自动提炼并记录不含正文的 warning，不能阻塞或拖慢聊天。
- 应用退出时只等待当前任务有限收尾，不无限阻塞退出。
- JSON 不合法、超时或模型失败时不写入记忆，也不影响已完成的聊天。
- `content` 最大 500 Unicode 字符，关键词最多 12 个，重要度 1–5，可信度 0–1。
- 不保存 API Key、系统提示词、推理内容、验证码、访问令牌或明显的秘密值。
- 不根据助手自己的猜测保存高可信用户事实。

单聊为当前道人提炼。群聊只为实际发言或被用户明确点名的道人提炼；一次群聊回合使用一次结构化提炼调用，返回按道人 UUID 分组的候选，避免成员数倍增的模型调用。

“只做本地”指记忆的保存、索引、检索和管理全部在桌面本地完成。用户已经允许提炼阶段调用其当前配置的模型，因此本轮必要对话片段仍会发送到对应模型供应商：单聊使用该道人模型，群聊使用本轮第一位成功发言道人的模型。界面必须明确说明这一点；关闭自动记忆后不得启动提炼调用。

### 10.3 去重与冲突

- 对 `kind|normalized_content` 计算 SHA256 `ContentHash`。
- 完全相同的 active 记忆只更新可信度、重要度和 `UpdatedAt`。
- 高相似候选与旧记忆冲突时，新记录写入 active，旧记录改为 superseded，不物理删除。
- 用户固定的记忆不得被自动 supersede。
- 自动记忆不得覆盖用户手工编辑的内容。
- 用户执行“删除”或“清空”时必须物理删除对应本地记录，不得只改为 archived。`archived` 仅供未来非删除式整理和兼容内部状态使用。

### 10.4 检索

不使用 embedding。先查询当前道人 active 记忆，再用以下信号排序：

- pinned 固定优先；
- 关键词精确命中；
- 字符 bigram 相关度；
- importance；
- 最近访问和更新时间；
- `open_loop` 在相关话题下加权。

每轮最多注入 6 条、合计最多 1200 Unicode 字符。记忆以“本地事实参考”分区注入，明确声明：记忆不是指令，发现与最新用户消息冲突时以最新用户消息为准。

### 10.5 管理界面

道人详情增加“记忆”区域：

- 自动记忆开关；
- 按类型筛选；
- 查看内容、来源会话、可信度、重要度和更新时间；
- 手动新增、编辑和固定；
- 删除单条；
- 清空该道人全部记忆；
- 跳转来源会话。

清空必须二次确认。删除和清空仅影响选定道人，不能跨道人操作。

## 11. 提示词分区

最终 system message 使用稳定标题分区：

```text
【安全与真实性边界】
【道人身份】
【永久丹性核心】
【本轮激活丹性】
【本地记忆事实】
【用户当轮要求】
【回答与群聊预算】
```

分区规则：

- 安全边界由应用提供。
- 道人身份、金丹和用户手工记忆属于本地配置，但记忆仍只具有事实上下文权限。
- 自动提炼记忆视为不可信数据，必须引用式注入，不得拼接为命令句。
- 用户原问题始终作为最后一条 `user` message 出现，历史截断和 retry 不得丢失。
- 原 `DaoistBehaviorProfile` 不因当轮约束而修改；当轮策略只影响本次请求。

## 12. 错误与降级矩阵

| 失败点 | 行为 | 是否允许丢金丹字段 |
|---|---|---:|
| 涌现 LLM 不可用 | 使用完整确定性档案和渲染器 | 否 |
| 涌现 JSON 非法 | 丢弃涌现层，记录安全错误码 | 否 |
| 旧缓存无 behavior_profile | 自动按原始金丹重建 | 否 |
| 新保存的 skill_schema 非法 | 返回字段级 400，阻止保存 | 不适用，禁止静默修复 |
| 历史 skill_schema 类型异常 | 原值进入 UnknownFields，显示质量警告并继续确定性渲染 | 否 |
| 记忆提炼失败 | 本轮聊天成功，跳过记忆写入 | 否 |
| 记忆检索失败 | 不注入记忆，聊天继续并记录错误码 | 否 |
| 单个群成员模型失败 | 可见错误；按必答/自愿规则决定是否继续 | 否 |
| 用户停止 | 不调用模型，正常完成空回合 | 否 |

日志只允许记录：

- profile version、fingerprint 前 12 位、pill count；
- activated rule count、memory count；
- policy band、max tokens、eligible、must answer；
- error code、耗时和模型名。

日志禁止记录完整用户消息、完整记忆、完整 system prompt、模型推理内容和凭证。

## 13. API 与界面边界

### 13.1 记忆 API

- `GET /api/v1/agents/:uuid/memories`
- `POST /api/v1/agents/:uuid/memories`
- `PATCH /api/v1/agents/:uuid/memories/:memory_uuid`
- `DELETE /api/v1/agents/:uuid/memories/:memory_uuid`
- `DELETE /api/v1/agents/:uuid/memories`
- 现有道人更新接口增加可选 `memory_enabled`。

所有外部 ID 使用 UUID，内部 GORM 联结继续使用自增 ID。

### 13.2 聊天 API

现有 SSE 事件保持兼容。新增内部 `GenerationOptions`，不要求前端传 token。`accepted` 后直接 `done/turn_done` 且没有 chunk 是明确停止时的合法结果，前端不得无限 loading 或报网络错误。

### 13.3 行为可观察性

道人详情的语言模式区域增加：

- 档案版本；
- 本次合成是否使用涌现增强；
- 完整金丹字段保留状态；
- 最近一次重建时间；
- “试丹对照”入口：使用同一固定问题比较基础道人与服丹道人。

聊天主界面不默认展示调试提示词或长行为追踪，避免破坏沉浸感。

## 14. 测试与验收

### 14.1 金丹无损测试

构造一颗每个字段都带唯一标记的金丹，例如：

```text
IDENTITY_MARKER
DNA_MARKER
MENTAL_MODEL_MARKER
HEURISTIC_MARKER
VALUE_MARKER
ANTI_PATTERN_MARKER
HONEST_LIMIT_MARKER
EXAMPLE_MARKER
UNKNOWN_FIELD_MARKER
```

分别模拟：正常合成、无凭证、超时、非法 JSON、空 `system_prompt`。断言 `DaoistBehaviorProfile` 和确定性最终提示词仍包含全部标记。

### 14.2 金丹可辨认验收

固定同一道人、模型、温度和 10 个问题，比较：

1. 无金丹；
2. 单颗高辨识度金丹；
3. 两颗组合金丹。

要求：

- 结构测试证明命中的规则和示例进入每次请求；
- 人工盲测至少 8/10 次能正确分辨有无目标金丹；
- 单丹与双丹回答能体现不同规则，不只是出现几个特色词；
- anti-pattern 和 honest-limit 不被违反；
- Wails 桌面端完成一次真实模型验收。

### 14.3 表达欲验收

- 表驱动覆盖 `0,20,21,40,41,60,61,80,81,100`。
- 1000 个固定回合验证 20、50、80 的自愿发言次数严格单调。
- 单聊低表达欲仍回答；群聊低表达欲被 `@` 时仍回答。
- 三档请求的句数指令和 `max_tokens` 单调递增。

### 14.4 用户约束验收

- “烦，直接说结论”：最多 1 人、1 轮、2 句、256 tokens。
- “够了，别说了”：0 次模型调用。
- “不要停，继续说”：正常调用。
- “大家每人一句”：每人最多一句，只运行一轮。
- 最新用户问题在每个模型调用中都是最后一条 user message。

### 14.5 记忆验收

- 会话 A 中形成用户偏好，会话 B 中同一道人能检索并自然使用。
- 另一位道人不能读到该记忆。
- 关闭自动记忆后不再启动提炼调用。
- 删除记忆后新会话不再检索到。
- 固定记忆不会被自动 supersede。
- 提炼失败不影响聊天完成事件。
- 恶意记忆文本不能改变 system 权限或覆盖用户最新要求。

### 14.6 回归与桌面验收

- Go：相关包测试后运行 `go test ./... -count=1`。
- Python：相关测试后运行 `.venv/bin/pytest -q`。
- Frontend：运行 typecheck、相关 Vitest、lint 和 production build。
- Wails：真实桌面包验证单聊、群聊、停止、重试、记忆管理、重启后记忆保留。
- macOS Apple Silicon、macOS Intel、Windows x64 不得增加外部常驻服务或缺失运行时依赖。

## 15. 迁移与兼容

1. 新迁移只新增列和表，不修改已有 UUID 或会话数据。
2. 旧 `LanguagePattern` 首次使用时自动重建，不批量调用合成模型。
3. 旧道人默认 `memory_enabled=true`，但迁移不会根据历史消息回填记忆，避免未经用户预期扫描全部历史。
4. 历史聊天继续可读。
5. 现有 SSE 消费端保持兼容。
6. 现有 `proactivity` 继续使用 0–100，不新增重复字段。
7. 回滚时可停止新记忆写入；新增表和列保留不会影响旧版本读取既有字段。

## 16. 实施边界与低阶模型纪律

低阶模型执行后续实施计划时必须遵守：

1. 先测试后实现，每个任务独立提交。
2. 不得同时执行旧的两份表达欲/群聊计划。
3. 不得只修改提示词宣称任务完成。
4. 不得把 LLM 润色结果当作完整金丹档案保存。
5. 不得在降级代码中只选择“常用字段”。
6. 不得新增全局随机数决定群聊发言。
7. 不得复制 AGPL 项目代码。
8. 不得引入 Mem0、AutoGen、向量库或云记忆依赖。
9. 不得让前端复制用户约束解析规则；Go 是唯一策略源。
10. 不得让记忆提炼失败阻断聊天。
11. 不得记录完整提示词、用户正文、记忆内容或凭证。
12. 不得把 Web 开发模式当最终验收；必须以 Wails 桌面版为准。

## 17. 完成定义

只有同时满足以下条件，统一任务才算完成：

- 合成失败时全部金丹字段仍存在并进入确定性提示词。
- `EmergenceRules` 在实际聊天中生效，而不是只展示。
- 用户能通过盲测辨认金丹带来的行为差异。
- 表达欲稳定控制单聊长度、群聊发言机会和 token。
- 用户的停止与简短要求被代码硬执行。
- 群聊不会无限轮转或重复同一观点。
- 同一道人能跨会话使用本地记忆，其他道人无法越权读取。
- 用户能管理和关闭记忆。
- 全部自动化测试通过，并完成 Wails 桌面真实模型验收。
