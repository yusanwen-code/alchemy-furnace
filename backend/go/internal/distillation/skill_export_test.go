package distillation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alchemy-furnace/server/model"
)

func validSkill() *ExportableSkill {
	return &ExportableSkill{
		Name:        "结构化金丹",
		Slug:        "",
		Description: "一份结构化的语言风格技能包",
		SkillSchema: model.JSONMap{"identity_card": "我是金丹"},
		Tags:        []string{"语言", "风格"},
		Sources: []Source{
			{Title: "公开资料", URL: "https://example.com/intro", Dimension: "decision_heuristics"},
		},
		GeneratedAt: "2026-08-27T10:00:00Z",
	}
}

func TestRejectCredentialFields(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		rejected bool
	}{
		{"api_key 顶层键", `{"pill_id":"x","format":"codex","api_key":"sk-secret"}`, true},
		{"apiKey 驼峰键", `{"format":"codex","apiKey":"sk-secret"}`, true},
		{"model_key 键", `{"model_key":"abc"}`, true},
		{"token 键", `{"token":"abc"}`, true},
		{"password 键", `{"password":"abc"}`, true},
		{"credential 键", `{"credential":"abc"}`, true},
		{"干净请求体", `{"pill_id":"x","format":"codex"}`, false},
		{"结构化 skill 负载", `{"skill":{"name":"金丹","description":"d","skillSchema":{},"generatedAt":"2026-08-27T10:00:00Z"},"format":"codex"}`, false},
		{"非法 JSON 交给绑定层", `{not json`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := RejectCredentialFields([]byte(c.body))
			if c.rejected != (err != nil) {
				t.Fatalf("RejectCredentialFields(%q) = %v, want rejected=%v", c.body, err, c.rejected)
			}
		})
	}
}

func TestValidateExportable_ValidPasses(t *testing.T) {
	if err := ValidateExportable(validSkill()); err != nil {
		t.Fatalf("有效内容被拒绝: %v", err)
	}
}

func TestValidateExportable_RejectsInvalid(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*ExportableSkill)
		wantField string
	}{
		{"空名称", func(s *ExportableSkill) { s.Name = "" }, "name"},
		{"超长名称", func(s *ExportableSkill) { s.Name = strings.Repeat("名", MaxExportNameLength+1) }, "name"},
		{"名称含控制字符", func(s *ExportableSkill) { s.Name = "a\x00b" }, "name"},
		{"名称疑似密钥", func(s *ExportableSkill) { s.Name = "sk-0123456789abcdef" }, "name"},
		{"名称疑似数据库 ID", func(s *ExportableSkill) { s.Name = "550e8400-e29b-41d4-a716-446655440000" }, "name"},
		{"空描述", func(s *ExportableSkill) { s.Description = "" }, "description"},
		{"超长描述", func(s *ExportableSkill) { s.Description = strings.Repeat("d", MaxExportDescriptionLength+1) }, "description"},
		{"描述疑似密钥", func(s *ExportableSkill) { s.Description = "api_key: abcdefgh" }, "description"},
		{"非法 slug", func(s *ExportableSkill) { s.Slug = "Bad Slug!" }, "slug"},
		{"超长 slug", func(s *ExportableSkill) { s.Slug = strings.Repeat("a", 50) }, "slug"},
		{"标签过多", func(s *ExportableSkill) { s.Tags = make([]string, MaxExportTags+1) }, "tags"},
		{"标签超长", func(s *ExportableSkill) { s.Tags = []string{strings.Repeat("t", MaxExportTagLength+1)} }, "tags"},
		{"来源过多", func(s *ExportableSkill) { s.Sources = make([]Source, MaxExportSources+1) }, "sources"},
		{"来源标题超长", func(s *ExportableSkill) {
			s.Sources = []Source{{Title: strings.Repeat("t", MaxExportSourceTitleLength+1), URL: "https://example.com", Dimension: "d"}}
		}, "sources.title"},
		{"来源维度超长", func(s *ExportableSkill) {
			s.Sources = []Source{{Title: "t", URL: "https://example.com", Dimension: strings.Repeat("d", MaxExportSourceDimensionLength+1)}}
		}, "sources.dimension"},
		{"来源非 http 协议", func(s *ExportableSkill) {
			s.Sources = []Source{{Title: "t", URL: "javascript:alert(1)", Dimension: "d"}}
		}, "sources.url"},
		{"来源 URL 带凭据", func(s *ExportableSkill) {
			s.Sources = []Source{{Title: "t", URL: "https://user:pass@example.com/", Dimension: "d"}}
		}, "sources.url"},
		{"来源 URL 含空白", func(s *ExportableSkill) {
			s.Sources = []Source{{Title: "t", URL: "https://example.com/a b", Dimension: "d"}}
		}, "sources.url"},
		{"来源 URL 缺主机名", func(s *ExportableSkill) {
			s.Sources = []Source{{Title: "t", URL: "https://", Dimension: "d"}}
		}, "sources.url"},
		{"来源 URL 超长", func(s *ExportableSkill) {
			s.Sources = []Source{{Title: "t", URL: "https://example.com/" + strings.Repeat("a", MaxExportURLLength+1), Dimension: "d"}}
		}, "sources.url"},
		{"generated_at 非法", func(s *ExportableSkill) { s.GeneratedAt = "不是时间" }, "generated_at"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			skill := validSkill()
			c.mutate(skill)
			err := ValidateExportable(skill)
			if err == nil {
				t.Fatal("期望拒绝,实际通过")
			}
			var v *ExportValidationError
			if !errors.As(err, &v) {
				t.Fatalf("err = %v, want *ExportValidationError", err)
			}
			if v.Field != c.wantField {
				t.Fatalf("field = %q, want %q (reason=%s)", v.Field, c.wantField, v.Reason)
			}
		})
	}
}

func TestClientSkillExport_PostsSnakeCasePayloadAndReadsZip(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Header().Set("Content-Disposition", `attachment; filename="alchemy-skill-structure-codex.zip"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("PK\x03\x04zipbytes"))
	}))
	defer server.Close()
	client := NewDynamicClient(func() string { return server.URL })

	result, err := client.SkillExport(context.Background(), validSkill(), "codex")
	if err != nil {
		t.Fatalf("SkillExport failed: %v", err)
	}
	if gotPath != "/api/v1/distillation/skill-export" || gotMethod != http.MethodPost {
		t.Fatalf("请求 %s %s, want POST /api/v1/distillation/skill-export", gotMethod, gotPath)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatalf("请求体非法 JSON: %v", err)
	}
	if _, ok := payload["skill_schema"]; !ok {
		t.Fatalf("请求体缺 skill_schema(snake_case): %s", gotBody)
	}
	if payload["generated_at"] != "2026-08-27T10:00:00Z" {
		t.Fatalf("generated_at = %v, want 2026-08-27T10:00:00Z", payload["generated_at"])
	}
	if payload["format"] != "codex" {
		t.Fatalf("format = %v, want codex", payload["format"])
	}
	if payload["name"] != "结构化金丹" {
		t.Fatalf("name = %v, want 结构化金丹", payload["name"])
	}
	if result.Filename != "alchemy-skill-structure-codex.zip" {
		t.Fatalf("filename = %q, want alchemy-skill-structure-codex.zip", result.Filename)
	}
	if string(result.Content) != "PK\x03\x04zipbytes" {
		t.Fatalf("content = %q", result.Content)
	}
}

func TestClientSkillExport_FallsBackToGenericFilename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("PK\x03\x04"))
	}))
	defer server.Close()
	client := NewDynamicClient(func() string { return server.URL })

	result, err := client.SkillExport(context.Background(), validSkill(), "claude")
	if err != nil {
		t.Fatalf("SkillExport failed: %v", err)
	}
	if result.Filename != "alchemy-skill-export.zip" {
		t.Fatalf("fallback filename = %q, want alchemy-skill-export.zip", result.Filename)
	}
}

func TestClientSkillExport_DecodesStructuredRemoteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"detail":{"code":"skill_export_invalid","stage":"export","message":"Skill 导出内容无效: name 长度不足","retryable":false,"details":{"field":"name","reason":"长度不足"}}}`))
	}))
	defer server.Close()
	client := NewDynamicClient(func() string { return server.URL })

	_, err := client.SkillExport(context.Background(), validSkill(), "codex")
	var remote *RemoteError
	if !errors.As(err, &remote) {
		t.Fatalf("err = %v, want *RemoteError", err)
	}
	if remote.Status != http.StatusUnprocessableEntity || remote.Code != "skill_export_invalid" {
		t.Fatalf("remote = %+v", remote)
	}
	if remote.Stage != "export" || remote.Retryable {
		t.Fatalf("stage/retryable = %s/%v", remote.Stage, remote.Retryable)
	}
	if remote.Details["field"] != "name" {
		t.Fatalf("details = %v", remote.Details)
	}
}
