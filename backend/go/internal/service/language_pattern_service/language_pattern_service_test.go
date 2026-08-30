package language_pattern_service

import (
	"context"
	std "errors"
	"strings"
	"testing"

	appErrors "github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/behavior"
	"github.com/alchemy-furnace/server/internal/interface/dao"
	"github.com/alchemy-furnace/server/internal/service/credential"
	"github.com/alchemy-furnace/server/internal/synthesis"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

// ---------- 测试替身 ----------

type fakeAgentDAO struct {
	dao.Agent
	agent *model.DaoAgent
	saved []*model.LanguagePattern
}

func (f *fakeAgentDAO) TakeAgentDetailByID(ctx context.Context, agentID uint) (*model.DaoAgent, appErrors.Error) {
	if f.agent == nil {
		return nil, appErrors.ErrorRecordNotFound("fake.agent.missing")
	}
	return f.agent, nil
}

func (f *fakeAgentDAO) SaveLanguagePattern(ctx context.Context, pattern *model.LanguagePattern) appErrors.Error {
	f.saved = append(f.saved, pattern)
	return nil
}

type fakeSynthesis struct {
	synthesis.Client
	resp      *synthesis.CombineResponse
	err       error
	calls     int
	lastPills []synthesis.PillInput
	lastCreds *credential.ModelCredentials
}

func (f *fakeSynthesis) Combine(ctx context.Context, personality string, pills []synthesis.PillInput, creds *credential.ModelCredentials) (*synthesis.CombineResponse, error) {
	f.calls++
	f.lastPills = pills
	f.lastCreds = creds
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

type fakeCreds struct {
	credential.Resolver
	err error
}

func (f *fakeCreds) ResolveSynthesisCredentials(ctx context.Context) (*credential.ModelCredentials, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &credential.ModelCredentials{Model: "fake-synthesis-model"}, nil
}

// ---------- 夹具 ----------

func markerSkillSchema() model.JSONMap {
	return model.JSONMap{
		"identity_card":       "IDENTITY_MARKER",
		"description":         "DESCRIPTION_MARKER",
		"expression_dna":      model.JSONMap{"vocabulary": "DNA_MARKER"},
		"mental_models":       []any{map[string]any{"name": "MENTAL_MODEL_MARKER"}},
		"decision_heuristics": []any{map[string]any{"condition": "HEURISTIC_MARKER"}},
		"values":              []any{"VALUE_MARKER"},
		"anti_patterns":       []any{"ANTI_PATTERN_MARKER"},
		"honest_limits":       []any{"HONEST_LIMIT_MARKER"},
		"example_dialogues":   []any{map[string]any{"user": "EXAMPLE_MARKER"}},
		"future_key_2026":     "UNKNOWN_FIELD_MARKER",
	}
}

func newMarkerAgent() *model.DaoAgent {
	return &model.DaoAgent{
		ID:          1,
		Name:        "测试道人",
		Personality: "沉稳内敛",
		AgentPills: []model.AgentPill{
			{
				Weight:    1.0,
				SortOrder: 0,
				Pill: model.ElixirPill{
					UUID:        uuid.New(),
					Name:        "标记金丹",
					SkillSchema: markerSkillSchema(),
				},
			},
		},
	}
}

var allMarkers = []string{
	"IDENTITY_MARKER", "DNA_MARKER", "MENTAL_MODEL_MARKER", "HEURISTIC_MARKER",
	"VALUE_MARKER", "ANTI_PATTERN_MARKER", "HONEST_LIMIT_MARKER",
	"EXAMPLE_MARKER", "UNKNOWN_FIELD_MARKER",
}

func assertPromptHasAllMarkers(t *testing.T, prompt string) {
	t.Helper()
	for _, marker := range allMarkers {
		if !strings.Contains(prompt, marker) {
			t.Errorf("渲染提示词缺少标记 %s;提示词:\n%s", marker, prompt)
		}
	}
}

// ---------- 用例 ----------

// TestGetOrBuildPatternCacheHitNewFormat 新格式缓存(behavior_profile 非空 + 版本匹配 + 指纹一致)直接命中,不调合成
func TestGetOrBuildPatternCacheHitNewFormat(t *testing.T) {
	agent := newMarkerAgent()
	fp, err := computeFingerprint(agent.Personality, agent.AgentPills)
	if err != nil {
		t.Fatalf("computeFingerprint: %v", err)
	}
	agent.LanguagePattern = &model.LanguagePattern{
		AgentID:           agent.ID,
		SystemPrompt:      "缓存的提示词",
		BehaviorProfile:   model.JSONMap{"version": 1},
		ProfileVersion:    behavior.ProfileVersion,
		SourceFingerprint: fp,
		IsValid:           true,
	}
	fakeAgent := &fakeAgentDAO{agent: agent}
	fakeSynth := &fakeSynthesis{}

	svc := New(fakeAgent, fakeSynth, &fakeCreds{})
	got, err := svc.GetOrBuildPattern(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("GetOrBuildPattern: %v", err)
	}
	if got != agent.LanguagePattern {
		t.Error("缓存命中应返回原对象")
	}
	if fakeSynth.calls != 0 {
		t.Errorf("缓存命中不应调用合成,实际 %d 次", fakeSynth.calls)
	}
}

// TestGetOrBuildPatternOldCacheRebuilds 旧缓存(无 behavior_profile)视为失效,自动重建
func TestGetOrBuildPatternOldCacheRebuilds(t *testing.T) {
	agent := newMarkerAgent()
	fp, _ := computeFingerprint(agent.Personality, agent.AgentPills)
	agent.LanguagePattern = &model.LanguagePattern{
		AgentID:           agent.ID,
		SystemPrompt:      "旧格式提示词",
		SourceFingerprint: fp,
		IsValid:           true,
	}
	fakeAgent := &fakeAgentDAO{agent: agent}
	fakeSynth := &fakeSynthesis{resp: &synthesis.CombineResponse{
		EmergenceRules: model.JSONList{"涌现规则甲"},
		InnerTensions:  []synthesis.InnerTension{{Dimension: "formality", Description: "正式程度相冲", Severity: "high"}},
		Fingerprint:    fp,
		Model:          "fake-synthesis-model",
	}}

	svc := New(fakeAgent, fakeSynth, &fakeCreds{})
	got, err := svc.GetOrBuildPattern(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("GetOrBuildPattern: %v", err)
	}
	if fakeSynth.calls != 1 {
		t.Fatalf("旧缓存应触发重建,实际 %d 次", fakeSynth.calls)
	}
	if len(fakeAgent.saved) != 1 {
		t.Fatal("重建结果应落库")
	}
	if got.BehaviorProfile == nil || got.ProfileVersion != behavior.ProfileVersion {
		t.Error("落库缓存缺少新档案字段")
	}
	assertPromptHasAllMarkers(t, got.SystemPrompt)
	if !strings.Contains(got.SystemPrompt, "涌现规则甲") {
		t.Error("涌现规则必须渲染进提示词")
	}
}

// TestGetOrBuildPatternFingerprintMismatchRebuilds 指纹不一致触发重建
func TestGetOrBuildPatternFingerprintMismatchRebuilds(t *testing.T) {
	agent := newMarkerAgent()
	agent.LanguagePattern = &model.LanguagePattern{
		AgentID:           agent.ID,
		SystemPrompt:      "过期缓存",
		BehaviorProfile:   model.JSONMap{"version": 1},
		ProfileVersion:    behavior.ProfileVersion,
		SourceFingerprint: "sha256:stale",
		IsValid:           true,
	}
	fakeAgent := &fakeAgentDAO{agent: agent}
	fakeSynth := &fakeSynthesis{resp: &synthesis.CombineResponse{
		EmergenceRules: model.JSONList{},
		Fingerprint:    "sha256:stale",
		Model:          "fake-synthesis-model",
	}}

	svc := New(fakeAgent, fakeSynth, &fakeCreds{})
	_, err := svc.GetOrBuildPattern(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("GetOrBuildPattern: %v", err)
	}
	if fakeSynth.calls != 1 {
		t.Errorf("指纹不一致应重建,实际 %d 次", fakeSynth.calls)
	}
}

// TestGetOrBuildPatternProfileVersionMismatchRebuilds 档案版本不一致触发重建
func TestGetOrBuildPatternProfileVersionMismatchRebuilds(t *testing.T) {
	agent := newMarkerAgent()
	fp, _ := computeFingerprint(agent.Personality, agent.AgentPills)
	agent.LanguagePattern = &model.LanguagePattern{
		AgentID:           agent.ID,
		SystemPrompt:      "旧版本档案",
		BehaviorProfile:   model.JSONMap{"version": 0},
		ProfileVersion:    0,
		SourceFingerprint: fp,
		IsValid:           true,
	}
	fakeAgent := &fakeAgentDAO{agent: agent}
	fakeSynth := &fakeSynthesis{resp: &synthesis.CombineResponse{
		EmergenceRules: model.JSONList{},
		Fingerprint:    fp,
		Model:          "fake-synthesis-model",
	}}

	svc := New(fakeAgent, fakeSynth, &fakeCreds{})
	if _, err := svc.GetOrBuildPattern(context.Background(), agent.ID); err != nil {
		t.Fatalf("GetOrBuildPattern: %v", err)
	}
	if fakeSynth.calls != 1 {
		t.Errorf("版本不一致应重建,实际 %d 次", fakeSynth.calls)
	}
}

// TestGetOrBuildPatternSuccessPersistsLossless 合成成功:无损渲染落库,档案与版本齐备
func TestGetOrBuildPatternSuccessPersistsLossless(t *testing.T) {
	agent := newMarkerAgent()
	fakeAgent := &fakeAgentDAO{agent: agent}
	fakeSynth := &fakeSynthesis{resp: &synthesis.CombineResponse{
		EmergenceRules: model.JSONList{"涌现规则甲", "涌现规则乙"},
		InnerTensions:  []synthesis.InnerTension{{Dimension: "sentence_length", Description: "句式相冲", Severity: "medium"}},
		Fingerprint:    "sha256:fp",
		Model:          "fake-synthesis-model",
	}}

	svc := New(fakeAgent, fakeSynth, &fakeCreds{})
	got, err := svc.GetOrBuildPattern(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("GetOrBuildPattern: %v", err)
	}

	if len(fakeAgent.saved) != 1 {
		t.Fatal("成功路径应落库")
	}
	saved := fakeAgent.saved[0]
	if !saved.IsValid {
		t.Error("成功路径 IsValid 应为 true")
	}
	if saved.ProfileVersion != behavior.ProfileVersion {
		t.Errorf("ProfileVersion = %d, want %d", saved.ProfileVersion, behavior.ProfileVersion)
	}
	if saved.BehaviorProfile == nil {
		t.Error("BehaviorProfile 必须写入")
	}
	if len(saved.EmergenceRules) != 2 {
		t.Errorf("EmergenceRules = %+v", saved.EmergenceRules)
	}
	if !strings.Contains(saved.SystemPrompt, "〔冲突调和〕") || !strings.Contains(saved.SystemPrompt, "句式相冲") {
		t.Error("冲突调和建议必须渲染")
	}
	assertPromptHasAllMarkers(t, saved.SystemPrompt)
	if got != saved {
		t.Error("返回值应为落库对象")
	}
}

// TestGetOrBuildPatternDegradedNotPersisted 降级(无凭证/涌现不可用):无损渲染返回,不落库
func TestGetOrBuildPatternDegradedNotPersisted(t *testing.T) {
	agent := newMarkerAgent()
	fakeAgent := &fakeAgentDAO{agent: agent}
	fakeSynth := &fakeSynthesis{resp: &synthesis.CombineResponse{
		EmergenceRules: model.JSONList{},
		Degraded:       true,
		DegradedReason: "no_credentials",
	}}

	svc := New(fakeAgent, fakeSynth, &fakeCreds{})
	got, err := svc.GetOrBuildPattern(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("降级路径不应返回错误: %v", err)
	}
	if got.IsValid {
		t.Error("降级结果 IsValid 应为 false(不落库)")
	}
	if len(fakeAgent.saved) != 0 {
		t.Error("降级结果不得落库")
	}
	assertPromptHasAllMarkers(t, got.SystemPrompt)
	if strings.Contains(got.SystemPrompt, "〔涌现规则〕") {
		t.Error("降级路径不应有涌现规则子节")
	}
}

// TestGetOrBuildPatternCombineErrorLosslessTemp 合成调用失败:无损渲染返回,不落库,不返回错误
func TestGetOrBuildPatternCombineErrorLosslessTemp(t *testing.T) {
	agent := newMarkerAgent()
	fakeAgent := &fakeAgentDAO{agent: agent}
	fakeSynth := &fakeSynthesis{err: std.New("python engine down")}

	svc := New(fakeAgent, fakeSynth, &fakeCreds{})
	got, err := svc.GetOrBuildPattern(context.Background(), agent.ID)
	if err != nil {
		t.Fatalf("合成失败不应阻断聊天(返回无损渲染): %v", err)
	}
	if got.IsValid {
		t.Error("合成失败结果 IsValid 应为 false")
	}
	if len(fakeAgent.saved) != 0 {
		t.Error("合成失败不得落库")
	}
	assertPromptHasAllMarkers(t, got.SystemPrompt)
}

// TestGetOrBuildPatternNoCredentialsStillCallsCombine 凭证解析失败不阻断:以 nil 凭证调合成
func TestGetOrBuildPatternNoCredentialsStillCallsCombine(t *testing.T) {
	agent := newMarkerAgent()
	fakeAgent := &fakeAgentDAO{agent: agent}
	fakeSynth := &fakeSynthesis{resp: &synthesis.CombineResponse{
		Degraded:       true,
		DegradedReason: "no_credentials",
	}}
	creds := &fakeCreds{err: std.New("no synthesis model configured")}

	svc := New(fakeAgent, fakeSynth, creds)
	if _, err := svc.GetOrBuildPattern(context.Background(), agent.ID); err != nil {
		t.Fatalf("GetOrBuildPattern: %v", err)
	}
	if fakeSynth.calls != 1 {
		t.Fatalf("应调用合成,实际 %d 次", fakeSynth.calls)
	}
	if fakeSynth.lastCreds != nil {
		t.Error("凭证解析失败时应以 nil 凭证调用合成")
	}
}
