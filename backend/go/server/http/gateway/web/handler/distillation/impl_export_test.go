// Skill 导出接口回归测试: POST /api/v1/distillation/skill-export
// 真实 sqlite 内存库 + 真实 service + 真实 handler,fake client 打桩 Python 引擎
package distillation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alchemy-furnace/server/internal/dao"
	nudist "github.com/alchemy-furnace/server/internal/distillation"
	"github.com/alchemy-furnace/server/internal/service/credential"
	"github.com/alchemy-furnace/server/internal/service/distillation_service"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/server/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// setupTestDB 初始化 sqlite 内存库并注入全局 dao.DB
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:distilltest%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("打开 sqlite 内存库失败: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("获取底层连接失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.ElixirPill{}); err != nil {
		t.Fatalf("迁移测试表失败: %v", err)
	}
	dao.DB = db
	t.Cleanup(func() { dao.DB = nil })
	return db
}

type fakeExportClient struct {
	skill  *nudist.ExportableSkill
	format string
	result *nudist.ExportResult
	err    error
}

func (f *fakeExportClient) Distill(_ context.Context, _, _, _ string, _ *credential.ModelCredentials) (*nudist.Response, error) {
	return nil, nil
}

func (f *fakeExportClient) SkillExport(_ context.Context, skill *nudist.ExportableSkill, format string) (*nudist.ExportResult, error) {
	f.skill = skill
	f.format = format
	return f.result, f.err
}

type fakeExportResolver struct{}

func (fakeExportResolver) ResolveCredentials(context.Context, string) (*credential.ModelCredentials, error) {
	return nil, nil
}

func (fakeExportResolver) ResolveSynthesisCredentials(context.Context) (*credential.ModelCredentials, error) {
	return nil, nil
}

func (fakeExportResolver) ResolveFusionCredentials(context.Context) (*credential.ModelCredentials, error) {
	return nil, nil
}

func setupSkillExportRouter(db *gorm.DB, client nudist.Client) (*gin.Engine, *fakeExportClient) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	fake, _ := client.(*fakeExportClient)
	h := New(distillation_service.New(client, fakeExportResolver{}, dao.NewPillDao()))
	r.POST("/api/v1/distillation/skill-export", h.SkillExport)
	return r, fake
}

// postSkillExport 发起导出请求,返回 (状态码, 原始响应体)
func postSkillExport(t *testing.T, r *gin.Engine, body string) (int, []byte, http.Header) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/distillation/skill-export", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, w.Body.Bytes(), w.Header()
}

func decodeEnvelope(t *testing.T, raw []byte) map[string]interface{} {
	t.Helper()
	var envelope map[string]interface{}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("响应不是 JSON 信封: %v, body=%s", err, raw)
	}
	return envelope
}

const validStructuredBody = `{
	"skill": {
		"name": "结构化金丹",
		"description": "一份结构化的语言风格技能包",
		"skillSchema": {"identity_card": "我是金丹"},
		"tags": ["语言"],
		"sources": [{"title": "公开资料", "url": "https://example.com/intro", "dimension": "decision_heuristics"}],
		"generatedAt": "2026-08-27T10:00:00Z"
	},
	"format": "codex"
}`

func TestSkillExport_StructuredModeReturnsZip(t *testing.T) {
	db := setupTestDB(t)
	client := &fakeExportClient{result: &nudist.ExportResult{Filename: "alchemy-skill-structure-codex.zip", Content: []byte("PK\x03\x04zip")}}
	r, fake := setupSkillExportRouter(db, client)

	status, raw, headers := postSkillExport(t, r, validStructuredBody)

	if status != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d, body=%s", status, raw)
	}
	if headers.Get("Content-Type") != "application/zip" {
		t.Fatalf("Content-Type = %q, want application/zip", headers.Get("Content-Type"))
	}
	if headers.Get("Content-Disposition") != `attachment; filename="alchemy-skill-structure-codex.zip"` {
		t.Fatalf("Content-Disposition = %q", headers.Get("Content-Disposition"))
	}
	if string(raw) != "PK\x03\x04zip" {
		t.Fatalf("ZIP 字节 = %q", raw)
	}
	if fake.skill == nil || fake.skill.Name != "结构化金丹" || fake.format != "codex" {
		t.Fatalf("service 未透传 skill/format: skill=%+v format=%q", fake.skill, fake.format)
	}
}

func TestSkillExport_PillIDModeLoadsSavedPill(t *testing.T) {
	db := setupTestDB(t)
	pill := model.ElixirPill{Name: "已保存金丹", Description: "已保存的金丹简介", SkillSchema: model.JSONMap{"identity_card": "x"}, Tags: model.JSONList{"语言"}}
	if err := db.Create(&pill).Error; err != nil {
		t.Fatalf("创建金丹失败: %v", err)
	}
	client := &fakeExportClient{result: &nudist.ExportResult{Filename: "alchemy-skill-x-claude.zip", Content: []byte("PK")}}
	r, fake := setupSkillExportRouter(db, client)

	body := fmt.Sprintf(`{"pill_id": %q, "format": "claude"}`, pill.UUID.String())
	status, raw, _ := postSkillExport(t, r, body)

	if status != http.StatusOK {
		t.Fatalf("期望 200, 实际 %d, body=%s", status, raw)
	}
	if fake.skill == nil || fake.skill.Name != "已保存金丹" || fake.skill.Description != "已保存的金丹简介" {
		t.Fatalf("服务端未按 pill_id 重装载金丹: %+v", fake.skill)
	}
	if fake.skill.EvidenceLevel != "limited" || fake.skill.GeneratedAt == "" {
		t.Fatalf("pill 模式投影字段异常: %+v", fake.skill)
	}
	// 接口只读: 金丹不得被删除或修改
	var after model.ElixirPill
	if err := db.First(&after, "uuid = ?", pill.UUID.String()).Error; err != nil {
		t.Fatalf("导出后金丹丢失: %v", err)
	}
	if after.Name != "已保存金丹" {
		t.Fatalf("导出修改了金丹: %+v", after)
	}
}

func TestSkillExport_InvalidFormat(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupSkillExportRouter(db, &fakeExportClient{})

	status, raw, _ := postSkillExport(t, r, `{"skill": {"name": "金丹", "description": "d", "skillSchema": {"identity_card": "x"}, "generatedAt": "2026-08-27T10:00:00Z"}, "format": "yaml"}`)

	if status != http.StatusBadRequest {
		t.Fatalf("期望 400, 实际 %d, body=%s", status, raw)
	}
}

func TestSkillExport_MissingTarget(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupSkillExportRouter(db, &fakeExportClient{})

	status, raw, _ := postSkillExport(t, r, `{"format": "codex"}`)

	if status != http.StatusBadRequest {
		t.Fatalf("期望 400, 实际 %d, body=%s", status, raw)
	}
}

func TestSkillExport_InvalidStructuredFields(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupSkillExportRouter(db, &fakeExportClient{})

	status, raw, _ := postSkillExport(t, r, `{
		"skill": {"name": "金丹", "description": "", "skillSchema": {"identity_card": "x"}, "generatedAt": "2026-08-27T10:00:00Z"},
		"format": "codex"
	}`)

	if status != http.StatusBadRequest {
		t.Fatalf("期望 400, 实际 %d, body=%s", status, raw)
	}
	envelope := decodeEnvelope(t, raw)
	if envelope["error_code"] != "service.skill_export.invalid" {
		t.Fatalf("error_code = %v, want service.skill_export.invalid", envelope["error_code"])
	}
}

func TestSkillExport_CredentialFieldsForbidden(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupSkillExportRouter(db, &fakeExportClient{})

	for _, body := range []string{
		`{"pill_id": "550e8400-e29b-41d4-a716-446655440000", "format": "codex", "api_key": "sk-secret"}`,
		`{"format": "codex", "token": "abc"}`,
	} {
		status, raw, _ := postSkillExport(t, r, body)
		if status != http.StatusForbidden {
			t.Fatalf("body=%s 期望 403, 实际 %d, resp=%s", body, status, raw)
		}
		envelope := decodeEnvelope(t, raw)
		if envelope["error_code"] != "skill_export_forbidden" {
			t.Fatalf("error_code = %v, want skill_export_forbidden", envelope["error_code"])
		}
	}
}

func TestSkillExport_PillNotFound(t *testing.T) {
	db := setupTestDB(t)
	r, _ := setupSkillExportRouter(db, &fakeExportClient{})

	body := fmt.Sprintf(`{"pill_id": %q, "format": "codex"}`, uuid.NewString())
	status, raw, _ := postSkillExport(t, r, body)

	if status != http.StatusNotFound {
		t.Fatalf("期望 404, 实际 %d, body=%s", status, raw)
	}
}

func TestSkillExport_RemoteUnavailableRetryable(t *testing.T) {
	db := setupTestDB(t)
	client := &fakeExportClient{err: &nudist.RemoteError{
		Status:    http.StatusServiceUnavailable,
		Code:      "skill_export_unavailable",
		Stage:     "export",
		Message:   "导出服务暂不可用",
		Retryable: true,
		Details:   map[string]any{"reason": "engine down"},
	}}
	r, _ := setupSkillExportRouter(db, client)

	status, raw, _ := postSkillExport(t, r, validStructuredBody)

	if status != http.StatusServiceUnavailable {
		t.Fatalf("期望 503, 实际 %d, body=%s", status, raw)
	}
	envelope := decodeEnvelope(t, raw)
	if envelope["error_code"] != "skill_export_unavailable" {
		t.Fatalf("error_code = %v, want skill_export_unavailable", envelope["error_code"])
	}
	if envelope["message"] != "导出服务暂不可用" {
		t.Fatalf("message = %v, want 公开 message 保留", envelope["message"])
	}
	data, ok := envelope["data"].(map[string]interface{})
	if !ok || data["retryable"] != true || data["stage"] != "export" {
		t.Fatalf("data = %v, want {retryable:true stage:export}", envelope["data"])
	}
}
