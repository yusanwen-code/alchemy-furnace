package turnpolicy

import (
	"testing"
)

func TestBuildTurnPlanStopZeroCalls(t *testing.T) {
	c := ExtractUserTurnConstraints("够了，别说了")
	p := BuildTurnPlan(c, PolicyForProactivity(50), 3, nil)
	if !p.Stop {
		t.Fatal("明确停止应标记 Stop")
	}
	if p.MaxSpeakers != 0 || p.MaxRounds != 0 || p.MaxTokens != 0 || p.MaxTurnTokens != 0 {
		t.Fatalf("停止时预算应为 0: %+v", p)
	}
}

func TestBuildTurnPlanConciseOrAnnoyed(t *testing.T) {
	for _, msg := range []string{"烦，直接说结论", "说重点"} {
		c := ExtractUserTurnConstraints(msg)
		p := BuildTurnPlan(c, PolicyForProactivity(90), 3, nil)
		if p.MaxSpeakers != 1 || p.MaxRounds != 1 {
			t.Fatalf("%q → speakers=%d rounds=%d, want 1/1", msg, p.MaxSpeakers, p.MaxRounds)
		}
		if p.MaxSentences > 2 || p.MaxTokens > 256 {
			t.Fatalf("%q → 预算超限: %+v", msg, p)
		}
	}
}

func TestBuildTurnPlanOneEach(t *testing.T) {
	c := ExtractUserTurnConstraints("大家每人一句")
	p := BuildTurnPlan(c, PolicyForProactivity(50), 4, nil)
	if !p.OneEach || p.MaxSpeakers != 4 || p.MaxRounds != 1 {
		t.Fatalf("每人一句 → %+v, want speakers=4 rounds=1", p)
	}
	if p.MaxSentences > 1 || p.MaxTokens > 160 {
		t.Fatalf("每人一句预算超限: %+v", p)
	}
	if p.MaxTurnTokens > 4096 {
		t.Fatalf("MaxTurnTokens = %d, want ≤4096", p.MaxTurnTokens)
	}
}

func TestBuildTurnPlanNormalDiscussion(t *testing.T) {
	c := ExtractUserTurnConstraints("你怎么看这件事")
	p := BuildTurnPlan(c, PolicyForProactivity(50), 3, nil)
	if p.MaxSpeakers != 2 || p.MaxRounds != 2 {
		t.Fatalf("普通讨论 → %+v, want ≤2/≤2", p)
	}
	if p.MaxSentences != 3 || p.MaxTokens != 384 {
		t.Fatalf("基础策略 → %+v, want 3句/384tok", p)
	}
	if p.MaxTurnTokens != 1280 {
		t.Fatalf("MaxTurnTokens = %d, want 1280", p.MaxTurnTokens)
	}
}

func TestBuildTurnPlanDetailed(t *testing.T) {
	// 单聊(memberCount=1):MaxTokens ≤2048(§8.2)
	c := ExtractUserTurnConstraints("详细讲讲")
	p := BuildTurnPlan(c, PolicyForProactivity(50), 1, nil)
	if p.MaxTokens > 2048 {
		t.Fatalf("单聊详细 MaxTokens = %d, want ≤2048", p.MaxTokens)
	}
	if p.MaxSpeakers != 3 || p.MaxRounds != 2 {
		t.Fatalf("详细讨论 → %+v, want 3/2", p)
	}
	// 群聊(memberCount=4):单人 ≤1536
	p = BuildTurnPlan(c, PolicyForProactivity(50), 4, nil)
	if p.MaxTokens > 1536 {
		t.Fatalf("群聊详细单人 MaxTokens = %d, want ≤1536", p.MaxTokens)
	}
	if p.MaxTurnTokens != 3072 {
		t.Fatalf("详细 MaxTurnTokens = %d, want 3072", p.MaxTurnTokens)
	}
}

func TestBuildTurnPlanSingleChatMustAnswer(t *testing.T) {
	// 单聊始终 must_answer=true(§7.1);群聊由编排器对被@者设 true
	p := BuildTurnPlan(ExtractUserTurnConstraints("你好"), PolicyForProactivity(10), 1, nil)
	if !p.MustAnswer {
		t.Fatal("单聊应始终 MustAnswer")
	}
	if p.MaxSentences != 1 || p.MaxTokens != 160 {
		t.Fatalf("低表达欲单聊预算 → %+v", p)
	}
}

func TestBuildTurnPlanPassesActivatedRules(t *testing.T) {
	rule := ActivatedPillRule{PillID: "p1", PillName: "古琴丹"}
	c := ExtractUserTurnConstraints("聊音乐")
	p := BuildTurnPlan(c, PolicyForProactivity(50), 2, []ActivatedPillRule{rule})
	if len(p.ActivatedRules) != 1 || p.ActivatedRules[0].PillID != "p1" {
		t.Fatalf("ActivatedRules 未透传: %+v", p.ActivatedRules)
	}
}
