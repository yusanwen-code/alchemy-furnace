package behavior

import (
	"strings"

	"github.com/alchemy-furnace/server/internal/service/turnpolicy"
	"github.com/alchemy-furnace/server/model"
)

// ActivatePillRules 从用户消息激活相关金丹规则(§6.5)。
// 候选关键词:丹名/描述/表达DNA词汇/心智模型/启发式/示例;评分=关键词命中(乘 2)+字符 bigram 相似度。
// 每颗金丹最多 2 心智模型 / 3 决策启发式 / 1 示例对话;全部无命中时按权重+服用顺序回退代表性规则。
func ActivatePillRules(userMessage string, profile *DaoistBehaviorProfile) []turnpolicy.ActivatedPillRule {
	if profile == nil {
		return nil
	}
	out := make([]turnpolicy.ActivatedPillRule, 0, len(profile.Pills))
	for _, pill := range profile.Pills {
		rule := turnpolicy.ActivatedPillRule{PillID: pill.PillID, PillName: pill.Name}
		keywords := pillKeywords(pill)
		rule.MentalModels = topCandidates(userMessage, pill.MentalModels, keywords, 2)
		rule.DecisionHeuristics = topCandidates(userMessage, pill.DecisionHeuristics, keywords, 3)
		rule.ExampleDialogues = topCandidates(userMessage, pill.ExampleDialogues, keywords, 1)
		if len(rule.MentalModels) == 0 && len(rule.DecisionHeuristics) == 0 && len(rule.ExampleDialogues) == 0 {
			// 回退:取该丹第一条可用的心智模型或启发式(按服用顺序,权重已在排序中体现)
			if len(pill.MentalModels) > 0 {
				rule.MentalModels = []string{formatMentalModel(pill.MentalModels[0])}
			} else if len(pill.DecisionHeuristics) > 0 {
				rule.DecisionHeuristics = []string{formatHeuristic(pill.DecisionHeuristics[0])}
			}
		}
		if len(rule.MentalModels) > 0 || len(rule.DecisionHeuristics) > 0 || len(rule.ExampleDialogues) > 0 {
			out = append(out, rule)
		}
	}
	return out
}

// ScoreUserMessageRelevance 用户消息与该道人档案的相关度(0-100)。
// 聚合全部金丹的关键词命中与 bigram 相似度;关键词命中权重高。
func ScoreUserMessageRelevance(userMessage string, profile *DaoistBehaviorProfile) int {
	if profile == nil || strings.TrimSpace(userMessage) == "" {
		return 0
	}
	total := 0.0
	for _, pill := range profile.Pills {
		keywords := pillKeywords(pill)
		hit := 0
		for _, kw := range keywords {
			if kw != "" && strings.Contains(userMessage, kw) {
				hit++
			}
		}
		best := 0.0
		for _, kw := range keywords {
			if s := BigramJaccard(userMessage, kw); s > best {
				best = s
			}
		}
		total += float64(hit)*20 + best*30
	}
	if total > 100 {
		total = 100
	}
	return int(total)
}

// pillKeywords 收集单颗金丹的候选关键词(§6.5)
func pillKeywords(pill CompiledPillProfile) []string {
	var kws []string
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" {
			kws = append(kws, s)
		}
	}
	add(pill.Name)
	add(pill.Description)
	if dna, ok := pill.ExpressionDNA["vocabulary"].([]any); ok {
		for _, v := range dna {
			if s, ok := v.(string); ok {
				add(s)
			}
		}
	}
	for _, mm := range pill.MentalModels {
		add(strField(mm, "name"))
		add(strField(mm, "one_liner"))
	}
	for _, h := range pill.DecisionHeuristics {
		add(strField(h, "condition"))
		add(strField(h, "case"))
	}
	for _, ex := range pill.ExampleDialogues {
		add(strField(ex, "user"))
	}
	return kws
}

func strField(m model.JSONMap, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

func formatMentalModel(m model.JSONMap) string {
	name, one := strField(m, "name"), strField(m, "one_liner")
	if one == "" {
		return name
	}
	return name + "：" + one
}

func formatHeuristic(h model.JSONMap) string {
	c, a, cs := strField(h, "condition"), strField(h, "action"), strField(h, "case")
	if c == "" && a == "" {
		return ""
	}
	out := c + " 则 " + a
	if cs != "" {
		out += "(例:" + cs + ")"
	}
	return out
}

func formatExample(e model.JSONMap) string {
	u, a := strField(e, "user"), strField(e, "assistant")
	if u == "" {
		return ""
	}
	return "问:" + u + " 答:" + a
}

// topCandidates 按「关键词命中 + bigram 相似度」取前 n 条(同分保序,不依赖 sort 稳定性)
func topCandidates(userMessage string, items []model.JSONMap, keywords []string, n int) []string {
	if n <= 0 || len(items) == 0 {
		return nil
	}
	type scored struct {
		text  string
		score float64
	}
	scoredList := make([]scored, 0, len(items))
	for _, item := range items {
		text := describeItem(item)
		if text == "" {
			continue
		}
		s := 0.0
		for _, kw := range keywords {
			if kw != "" && strings.Contains(userMessage, kw) {
				s += 2
			}
			if b := BigramJaccard(userMessage, kw); b > s {
				s = b
			}
		}
		scoredList = append(scoredList, scored{text: text, score: s})
	}
	for i := 1; i < len(scoredList); i++ {
		for j := i; j > 0 && scoredList[j].score > scoredList[j-1].score; j-- {
			scoredList[j], scoredList[j-1] = scoredList[j-1], scoredList[j]
		}
	}
	out := make([]string, 0, n)
	for i := 0; i < len(scoredList) && i < n; i++ {
		out = append(out, scoredList[i].text)
	}
	return out
}

// describeItem 将心智模型/启发式/示例渲染为文本(name 键=心智模型,condition 键=启发式,user 键=示例)
func describeItem(m model.JSONMap) string {
	if _, ok := m["name"]; ok {
		return formatMentalModel(m)
	}
	if _, ok := m["condition"]; ok {
		return formatHeuristic(m)
	}
	if _, ok := m["user"]; ok {
		return formatExample(m)
	}
	return ""
}
