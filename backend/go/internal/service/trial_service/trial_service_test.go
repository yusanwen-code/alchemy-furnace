package trial_service

import (
	"context"
	"encoding/json"
	std "errors"
	"io"
	"net/http"
	"strings"
	"testing"

	appErrors "github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/interface/dao"
	iservice "github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/alchemy-furnace/server/internal/service/credential"
	"github.com/alchemy-furnace/server/internal/synthesis"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)
type fakePillDAO struct {
	dao.Pill
	pills []*model.ElixirPill
}

func (f *fakePillDAO) FindPillsByUUIDs(ctx context.Context, uids []uuid.UUID) ([]*model.ElixirPill, appErrors.Error) {
	return f.pills, nil
}

type fakeSynth struct {
	synthesis.Client
	resp *synthesis.CombineResponse
	err  error
}

func (f *fakeSynth) Combine(ctx context.Context, personality string, pills []synthesis.PillInput, creds *credential.ModelCredentials) (*synthesis.CombineResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

type fakeCreds struct {
	credential.Resolver
}

func (f *fakeCreds) ResolveSynthesisCredentials(ctx context.Context) (*credential.ModelCredentials, error) {
	return &credential.ModelCredentials{Model: "fake"}, nil
}

func (f *fakeCreds) ResolveCredentials(ctx context.Context, name string) (*credential.ModelCredentials, error) {
	return &credential.ModelCredentials{Model: name}, nil
}

type fakeHTTP struct {
	lastBody string
}

func (f *fakeHTTP) Do(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	f.lastBody = string(body)
	return &http.Response{
		StatusCode: 200,
		Body: io.NopCloser(strings.NewReader(
			`{"code":0,"message":"success","data":{"content":"回复内容","model":"m","usage":{}}}`,
		)),
	}, nil
}

// ---------- 夹具 ----------

func trialMarkerPill() *model.ElixirPill {
	return &model.ElixirPill{
		UUID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name: "标记金丹",
		SkillSchema: model.JSONMap{
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
		},
	}
}

var trialMarkers = []string{
	"IDENTITY_MARKER", "DNA_MARKER", "MENTAL_MODEL_MARKER", "HEURISTIC_MARKER",
	"VALUE_MARKER", "ANTI_PATTERN_MARKER", "HONEST_LIMIT_MARKER",
	"EXAMPLE_MARKER", "UNKNOWN_FIELD_MARKER",
}

func assertTrialPromptMarkers(t *testing.T, prompt string) {
	t.Helper()
	for _, marker := range trialMarkers {
		if !strings.Contains(prompt, marker) {
			t.Errorf("提示词缺少标记 %s", marker)
		}
	}
}

// ---------- 用例 ----------

// TestSynthesizeRendersLosslessPrompt 试丹合成:渲染提示词含全部金丹字段 + 涌现规则
func TestSynthesizeRendersLosslessPrompt(t *testing.T) {
	svc := &Trial{
		pill:       &fakePillDAO{pills: []*model.ElixirPill{trialMarkerPill()}},
		synthesis:  &fakeSynth{resp: &synthesis.CombineResponse{
			EmergenceRules: model.JSONList{"涌现规则甲"},
			Fingerprint:    "sha256:fp",
			Model:          "fake",
		}},
		credential: &fakeCreds{},
		httpClient: &fakeHTTP{},
	}

	result, err := svc.Synthesize(context.Background(), "沉稳内敛", []iservice.TrialPillInput{
		{PillID: trialMarkerPill().UUID, Weight: 1.0, SortOrder: 0},
	}, "")
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	assertTrialPromptMarkers(t, result.SystemPrompt)
	if !strings.Contains(result.SystemPrompt, "〔涌现规则〕") || !strings.Contains(result.SystemPrompt, "涌现规则甲") {
		t.Error("涌现规则必须渲染进提示词")
	}
	if result.Degraded {
		t.Error("非降级路径 Degraded 应为 false")
	}
	if result.Fingerprint != "sha256:fp" || result.Model != "fake" {
		t.Errorf("Fingerprint/Model 透传错误: %+v", result)
	}
}

// TestSynthesizeDegradesOnCombineError 合成失败:返回无损渲染(degraded),不返回错误
func TestSynthesizeDegradesOnCombineError(t *testing.T) {
	svc := &Trial{
		pill:       &fakePillDAO{pills: []*model.ElixirPill{trialMarkerPill()}},
		synthesis:  &fakeSynth{err: std.New("engine down")},
		credential: &fakeCreds{},
		httpClient: &fakeHTTP{},
	}

	result, err := svc.Synthesize(context.Background(), "沉稳内敛", []iservice.TrialPillInput{
		{PillID: trialMarkerPill().UUID, Weight: 1.0, SortOrder: 0},
	}, "")
	if err != nil {
		t.Fatalf("合成失败不应返回错误: %v", err)
	}
	if !result.Degraded || result.DegradedReason != "combine_error" {
		t.Errorf("降级标记错误: %+v", result)
	}
	assertTrialPromptMarkers(t, result.SystemPrompt)
	if strings.Contains(result.SystemPrompt, "〔涌现规则〕") {
		t.Error("失败路径不应有涌现规则子节")
	}
}

// TestChatUsesRenderedSystemPrompt 试丹对话:system 消息是行为引擎渲染的提示词(含全部标记)
func TestChatUsesRenderedSystemPrompt(t *testing.T) {
	fakeHTTPClient := &fakeHTTP{}
	svc := &Trial{
		pill:       &fakePillDAO{pills: []*model.ElixirPill{trialMarkerPill()}},
		synthesis:  &fakeSynth{resp: &synthesis.CombineResponse{
			EmergenceRules: model.JSONList{"涌现规则甲"},
		}},
		credential: &fakeCreds{},
		httpClient: fakeHTTPClient,
	}

	resp, err := svc.Chat(context.Background(), &iservice.TrialChatRequest{
		Personality: "沉稳内敛",
		Pills:       []iservice.TrialPillInput{{PillID: trialMarkerPill().UUID, Weight: 1.0, SortOrder: 0}},
		Messages:    []map[string]string{{"role": "user", "content": "求道"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "回复内容" {
		t.Errorf("Content = %q", resp.Content)
	}

	var sent struct {
		Messages []map[string]string `json:"messages"`
	}
	if err := json.Unmarshal([]byte(fakeHTTPClient.lastBody), &sent); err != nil {
		t.Fatalf("解析发送体失败: %v", err)
	}
	if len(sent.Messages) != 2 || sent.Messages[0]["role"] != "system" {
		t.Fatalf("消息结构错误: %+v", sent.Messages)
	}
	assertTrialPromptMarkers(t, sent.Messages[0]["content"])
	if !strings.Contains(sent.Messages[0]["content"], "涌现规则甲") {
		t.Error("system 提示词应含涌现规则")
	}
}
