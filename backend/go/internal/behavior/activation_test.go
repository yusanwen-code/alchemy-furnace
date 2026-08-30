package behavior

import (
	"testing"

	"github.com/alchemy-furnace/server/model"
)

func sampleProfile() *DaoistBehaviorProfile {
	return &DaoistBehaviorProfile{
		Version:         ProfileVersion,
		BasePersonality: "沉稳",
		Pills: []CompiledPillProfile{
			{
				PillID: "p1", Name: "古琴丹", Weight: 2.0, SortOrder: 0,
				Description: "以古琴之道应答,论音乐与静心",
				ExpressionDNA: map[string]any{"vocabulary": []any{"古琴", "琴韵", "高山流水"}},
				MentalModels: []model.JSONMap{
					{"name": "知音", "one_liner": "先问对方所好再谈琴"},
					{"name": "松沉", "one_liner": "遇事先沉一口气"},
				},
				DecisionHeuristics: []model.JSONMap{
					{"condition": "被问及音乐", "action": "引用琴典作答", "case": "论乐"},
				},
				ExampleDialogues: []model.JSONMap{
					{"user": "你会弹琴吗", "assistant": "略通一二,愿闻其详。"},
				},
			},
			{
				PillID: "p2", Name: "棋弈丹", Weight: 1.0, SortOrder: 1,
				Description: "围棋布局之道",
				ExpressionDNA: map[string]any{"vocabulary": []any{"围棋", "布局"}},
				MentalModels: []model.JSONMap{{"name": "全局观", "one_liner": "先看大局再看局部"}},
			},
		},
	}
}

func TestActivatePillRulesMatchesRelevantPill(t *testing.T) {
	profile := sampleProfile()
	rules := ActivatePillRules("今天心情烦躁,聊点围棋布局", profile)
	if len(rules) == 0 {
		t.Fatal("应激活至少一颗金丹的规则")
	}
	found := false
	for _, r := range rules {
		if r.PillID == "p2" {
			found = true
			if len(r.MentalModels) == 0 {
				t.Fatal("棋弈丹应激活心智模型")
			}
		}
	}
	if !found {
		t.Fatalf("rules = %+v, 应包含棋弈丹(与话题相关)", rules)
	}
}

func TestActivatePillRulesRespectsCaps(t *testing.T) {
	// 2 颗丹全部命中音乐主题:每颗仍 ≤2 心智/≤3 启发/≤1 示例
	rules := ActivatePillRules("古琴与高山流水,谈琴论乐", sampleProfile())
	for _, r := range rules {
		if len(r.MentalModels) > 2 || len(r.DecisionHeuristics) > 3 || len(r.ExampleDialogues) > 1 {
			t.Fatalf("caps violated: %+v", r)
		}
	}
}

func TestActivatePillRulesFallbackWhenNoMatch(t *testing.T) {
	profile := sampleProfile()
	rules := ActivatePillRules("今天天气如何", profile)
	if len(rules) != len(profile.Pills) {
		t.Fatalf("无命中时每颗金丹应有代表性规则: %+v", rules)
	}
	for _, r := range rules {
		if len(r.MentalModels)+len(r.DecisionHeuristics) == 0 {
			t.Fatalf("代表规则不应为空: %+v", r)
		}
	}
}

func TestScoreUserMessageRelevanceHigherForMatchingPill(t *testing.T) {
	profile := sampleProfile()
	high := ScoreUserMessageRelevance("围棋布局怎么看", profile)
	low := ScoreUserMessageRelevance("今天的天气不错", profile)
	if high <= low {
		t.Fatalf("相关度 high=%d low=%d, 话题命中应更高", high, low)
	}
}

func TestScoreUserMessageRelevanceNilProfile(t *testing.T) {
	if got := ScoreUserMessageRelevance("随便聊聊", nil); got != 0 {
		t.Fatalf("nil profile 应返回 0, got %d", got)
	}
}
