package turnpolicy

// MemorySnippet 注入提示词的单条本地记忆(spec §10.4;P3 填充)
type MemorySnippet struct {
	Kind    string
	Content string
}

// TurnPlan 当轮策略合并结果(spec §8.2)
type TurnPlan struct {
	Stop           bool
	MustAnswer     bool
	LatestQuestion string // 用户最新问题原文(供动态分区渲染,§8.1)
	Concise        bool
	Detailed       bool
	Frustration    FrustrationLevel
	MaxSentences   int
	MaxTokens      int
	MaxTurnTokens  int
	MaxSpeakers    int
	MaxRounds      int
	OneEach        bool
	ActivatedRules []ActivatedPillRule
	Memories       []MemorySnippet
}

// 预算常量(spec §8.2 行为表逐字)
const (
	budgetConciseTokens  = 256
	budgetOneEachTokens  = 160
	budgetTurnNormal     = 1280
	budgetTurnDetailed   = 3072
	budgetSingleDetailed = 2048
	budgetGroupDetailed  = 1536
	hardTokenCeiling     = 8192 // Python 契约上限(任何路径不得突破)
)

// BuildTurnPlan 合并用户约束与表达欲策略(§8.2 行为表)。
// memberCount:群聊成员数;单聊传 1。优先级:停止 > 简短/烦躁 > 每人一句 > 详细 > 普通。
func BuildTurnPlan(constraints UserTurnConstraints, policy ResponsePolicy, memberCount int, activated []ActivatedPillRule) *TurnPlan {
	if memberCount < 1 {
		memberCount = 1
	}
	plan := &TurnPlan{
		MustAnswer:     memberCount == 1, // 单聊始终必答(§7.1)
		LatestQuestion: constraints.LatestQuestion,
		Concise:        constraints.Concise,
		Detailed:       constraints.Detailed,
		Frustration:    constraints.Frustration,
		MaxSentences:   policy.MaxSentences,
		MaxTokens:      policy.MaxTokens,
		MaxSpeakers:    2,
		MaxRounds:      2,
		MaxTurnTokens:  budgetTurnNormal,
		ActivatedRules: activated,
	}
	if constraints.WantsStop {
		plan.Stop = true
		plan.MaxSentences, plan.MaxTokens, plan.MaxTurnTokens = 0, 0, 0
		plan.MaxSpeakers, plan.MaxRounds = 0, 0
		return plan
	}
	switch {
	case constraints.Concise || constraints.Frustration == FrustrationAnnoyed:
		plan.MaxSpeakers, plan.MaxRounds = 1, 1
		plan.MaxSentences = minInt(plan.MaxSentences, 2)
		plan.MaxTokens = minInt(plan.MaxTokens, budgetConciseTokens)
		plan.MaxTurnTokens = plan.MaxTokens
	case constraints.OneEach:
		plan.MaxSpeakers, plan.MaxRounds = memberCount, 1
		plan.OneEach = true
		plan.MaxSentences, plan.MaxTokens = 1, budgetOneEachTokens
		plan.MaxTurnTokens = minInt(memberCount*budgetOneEachTokens, 4096)
	case constraints.Detailed:
		plan.MaxSpeakers, plan.MaxRounds = 3, 2
		plan.MaxTurnTokens = budgetTurnDetailed
		if memberCount == 1 {
			plan.MaxTokens = minInt(maxInt(policy.MaxTokens, budgetSingleDetailed), hardTokenCeiling)
		} else {
			plan.MaxTokens = minInt(maxInt(policy.MaxTokens, budgetGroupDetailed), hardTokenCeiling)
		}
	}
	return plan
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
