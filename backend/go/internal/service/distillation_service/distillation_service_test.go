package distillation_service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/internal/distillation"
	appErrors "github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/service/credential"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

type fakeClient struct {
	called bool
	result *distillation.Response
	err    error

	exportCalled bool
	exportResult *distillation.ExportResult
	exportErr    error
	exportSkill  *distillation.ExportableSkill
	exportFormat string
}

func (f *fakeClient) Distill(_ context.Context, _, _, _ string, _ *credential.ModelCredentials) (*distillation.Response, error) {
	f.called = true
	return f.result, f.err
}

func (f *fakeClient) SkillExport(_ context.Context, skill *distillation.ExportableSkill, format string) (*distillation.ExportResult, error) {
	f.exportCalled = true
	f.exportSkill = skill
	f.exportFormat = format
	return f.exportResult, f.exportErr
}

type fakeResolver struct {
	credentials *credential.ModelCredentials
	err         error
}

func (f fakeResolver) ResolveCredentials(context.Context, string) (*credential.ModelCredentials, error) {
	return f.credentials, f.err
}

func (f fakeResolver) ResolveSynthesisCredentials(context.Context) (*credential.ModelCredentials, error) {
	return f.credentials, f.err
}

func (f fakeResolver) ResolveFusionCredentials(context.Context) (*credential.ModelCredentials, error) {
	return f.credentials, f.err
}

func TestDistillValidatesBeforeCallingDependencies(t *testing.T) {
	client := &fakeClient{}
	service := New(client, fakeResolver{}, dao.NewPillDao())

	result, appErr := service.Distill(context.Background(), "A", "短", "zh-CN")

	if result != nil || appErr == nil {
		t.Fatalf("result = %#v, error = %#v; want validation error", result, appErr)
	}
	if client.called {
		t.Fatal("client was called for invalid input")
	}
}

func TestDistillPassesConfiguredCredentials(t *testing.T) {
	want := &distillation.Response{Name: "结构化金丹"}
	client := &fakeClient{result: want}
	resolver := fakeResolver{credentials: &credential.ModelCredentials{Model: "configured-model", APIKey: "secret"}}
	service := New(client, resolver, dao.NewPillDao())

	got, appErr := service.Distill(context.Background(), "人物主题", "提炼公开资料中的决策方式", "en")

	if appErr != nil || got != want {
		t.Fatalf("got = %#v, error = %#v", got, appErr)
	}
	if !client.called {
		t.Fatal("client was not called")
	}
}

func TestDistillReturnsCredentialFailureWithoutCallingClient(t *testing.T) {
	client := &fakeClient{}
	service := New(client, fakeResolver{err: errors.New("未配置模型")}, dao.NewPillDao())

	result, appErr := service.Distill(context.Background(), "人物主题", "提炼公开资料中的决策方式", "zh-CN")

	if result != nil || appErr == nil || client.called {
		t.Fatalf("result = %#v, error = %#v, called = %v", result, appErr, client.called)
	}
}

func TestDistillMapsRemoteServiceUnavailableWithCodeAndData(t *testing.T) {
	remote := &distillation.RemoteError{
		Status:    http.StatusServiceUnavailable,
		Code:      "research_search_blocked",
		Stage:     "research",
		Message:   "搜索受限",
		Retryable: true,
		Details:   map[string]any{"documents": 0},
	}
	service := New(&fakeClient{err: remote}, fakeResolver{credentials: &credential.ModelCredentials{}}, dao.NewPillDao())

	_, appErr := service.Distill(context.Background(), "人物主题", "提炼公开资料中的决策方式", "zh-CN")

	if appErr == nil || !appErr.IsType(appErrors.ErrorTypeServiceUnavailable) {
		t.Fatalf("error = %#v, want ErrorTypeServiceUnavailable", appErr)
	}
	if appErr.GetCode() != "research_search_blocked" {
		t.Fatalf("code = %q, want research_search_blocked", appErr.GetCode())
	}
	if appErr.Error() != "搜索受限" {
		t.Fatalf("message = %q, want 搜索受限", appErr.Error())
	}
	withData, ok := appErr.(appErrors.ErrorWithData)
	if !ok {
		t.Fatal("expected ErrorWithData to carry stage/retryable/details")
	}
	payload, ok := withData.GetData().(map[string]any)
	if !ok || payload["stage"] != "research" || payload["retryable"] != true {
		t.Fatalf("data = %#v, want {stage:research retryable:true}", withData.GetData())
	}
	details, ok := payload["details"].(map[string]any)
	if !ok || details["documents"] != 0 {
		t.Fatalf("details = %#v, want {documents:0} 原样透传", payload["details"])
	}
}

func TestDistillMapsRemoteInvalidRequestTo400WithPublicMessage(t *testing.T) {
	remote := &distillation.RemoteError{
		Status:  http.StatusBadRequest,
		Message: "未配置可用于智能炼制的模型，请先到设置中配置模型供应商",
	}
	service := New(&fakeClient{err: remote}, fakeResolver{credentials: &credential.ModelCredentials{}}, dao.NewPillDao())

	_, appErr := service.Distill(context.Background(), "人物主题", "提炼公开资料中的决策方式", "zh-CN")

	if appErr == nil || !appErr.IsType(appErrors.ErrorTypeInvalidRequest) {
		t.Fatalf("error = %#v, want ErrorTypeInvalidRequest", appErr)
	}
	if appErr.Error() != "未配置可用于智能炼制的模型，请先到设置中配置模型供应商" {
		t.Fatalf("message = %q, want 公开 message 保留", appErr.Error())
	}
}

func validExportSkill() *distillation.ExportableSkill {
	return &distillation.ExportableSkill{
		Name:        "结构化金丹",
		Description: "一份结构化的语言风格技能包",
		SkillSchema: map[string]interface{}{"identity_card": "我是金丹"},
		Tags:        []string{"语言"},
		GeneratedAt: "2026-08-27T10:00:00Z",
	}
}

func TestSkillExport_RejectsInvalidFormatBeforeCallingClient(t *testing.T) {
	client := &fakeClient{}
	service := New(client, fakeResolver{}, dao.NewPillDao())

	_, appErr := service.SkillExport(context.Background(), &distillation.SkillExportInput{
		Skill:  validExportSkill(),
		Format: "yaml",
	})

	if appErr == nil || !appErr.IsType(appErrors.ErrorTypeInvalidRequest) {
		t.Fatalf("error = %#v, want ErrorTypeInvalidRequest", appErr)
	}
	if client.exportCalled {
		t.Fatal("client was called for invalid format")
	}
}

func TestSkillExport_RequiresExactlyOneTarget(t *testing.T) {
	client := &fakeClient{}
	service := New(client, fakeResolver{}, dao.NewPillDao())

	for _, input := range []*distillation.SkillExportInput{
		{Format: "codex"},
		{PillID: "550e8400-e29b-41d4-a716-446655440000", Skill: validExportSkill(), Format: "codex"},
	} {
		_, appErr := service.SkillExport(context.Background(), input)
		if appErr == nil || !appErr.IsType(appErrors.ErrorTypeInvalidRequest) {
			t.Fatalf("input = %+v, error = %#v, want ErrorTypeInvalidRequest", input, appErr)
		}
	}
	if client.exportCalled {
		t.Fatal("client was called without a valid single target")
	}
}

func TestSkillExport_RejectsInvalidStructuredContent(t *testing.T) {
	client := &fakeClient{}
	service := New(client, fakeResolver{}, dao.NewPillDao())
	skill := validExportSkill()
	skill.Description = "" // 空描述: 服务端重校验拦截

	_, appErr := service.SkillExport(context.Background(), &distillation.SkillExportInput{
		Skill:  skill,
		Format: "codex",
	})

	if appErr == nil || !appErr.IsType(appErrors.ErrorTypeInvalidRequest) {
		t.Fatalf("error = %#v, want ErrorTypeInvalidRequest", appErr)
	}
	if appErr.GetCode() != "service.skill_export.invalid" {
		t.Fatalf("code = %q, want service.skill_export.invalid", appErr.GetCode())
	}
	if client.exportCalled {
		t.Fatal("client was called for invalid content")
	}
}

func TestSkillExport_StructuredSuccessPassesSkillAndFormat(t *testing.T) {
	client := &fakeClient{exportResult: &distillation.ExportResult{Filename: "alchemy-skill-x-codex.zip", Content: []byte("PK")}}
	service := New(client, fakeResolver{}, dao.NewPillDao())

	result, appErr := service.SkillExport(context.Background(), &distillation.SkillExportInput{
		Skill:  validExportSkill(),
		Format: "codex",
	})

	if appErr != nil || result == nil || string(result.Content) != "PK" {
		t.Fatalf("result = %#v, error = %#v", result, appErr)
	}
	if !client.exportCalled || client.exportFormat != "codex" {
		t.Fatalf("client 未携带 format=codex: called=%v format=%q", client.exportCalled, client.exportFormat)
	}
	if client.exportSkill.Name != "结构化金丹" || client.exportSkill.GeneratedAt != "2026-08-27T10:00:00Z" {
		t.Fatalf("client skill = %+v", client.exportSkill)
	}
}

func TestSkillExport_MapsRemoteUnavailableToRetryableServiceUnavailable(t *testing.T) {
	remote := &distillation.RemoteError{
		Status:    http.StatusServiceUnavailable,
		Code:      "skill_export_unavailable",
		Stage:     "export",
		Message:   "导出服务暂不可用",
		Retryable: true,
		Details:   map[string]any{"reason": "engine down"},
	}
	service := New(&fakeClient{exportErr: remote}, fakeResolver{}, dao.NewPillDao())

	_, appErr := service.SkillExport(context.Background(), &distillation.SkillExportInput{
		Skill:  validExportSkill(),
		Format: "claude",
	})

	if appErr == nil || !appErr.IsType(appErrors.ErrorTypeServiceUnavailable) {
		t.Fatalf("error = %#v, want ErrorTypeServiceUnavailable", appErr)
	}
	if appErr.GetCode() != "skill_export_unavailable" || appErr.Error() != "导出服务暂不可用" {
		t.Fatalf("code/message = %q/%q", appErr.GetCode(), appErr.Error())
	}
	withData, ok := appErr.(appErrors.ErrorWithData)
	if !ok {
		t.Fatal("expected ErrorWithData")
	}
	payload := withData.GetData().(map[string]any)
	if payload["stage"] != "export" || payload["retryable"] != true {
		t.Fatalf("data = %#v", withData.GetData())
	}
}

func TestSkillExport_MapsRemoteInvalidTo400WithPublicCode(t *testing.T) {
	remote := &distillation.RemoteError{
		Status:  http.StatusUnprocessableEntity,
		Code:    "skill_export_invalid",
		Stage:   "export",
		Message: "Skill 导出内容无效: name 长度不足",
	}
	service := New(&fakeClient{exportErr: remote}, fakeResolver{}, dao.NewPillDao())

	_, appErr := service.SkillExport(context.Background(), &distillation.SkillExportInput{
		Skill:  validExportSkill(),
		Format: "codex",
	})

	if appErr == nil || !appErr.IsType(appErrors.ErrorTypeInvalidRequest) {
		t.Fatalf("error = %#v, want ErrorTypeInvalidRequest", appErr)
	}
	if appErr.GetCode() != "skill_export_invalid" {
		t.Fatalf("code = %q, want skill_export_invalid", appErr.GetCode())
	}
}

// fakePills 仅实现 TakePillByUUID 的最小 stub,其余 DAO 方法在本测试不触达
type fakePills struct{ pill *model.ElixirPill }

func (f *fakePills) TakePillByUUID(_ context.Context, _ uuid.UUID) (*model.ElixirPill, appErrors.Error) {
	if f.pill == nil {
		return nil, appErrors.ErrorRecordNotFound("fake.pill")
	}
	return f.pill, nil
}
func (f *fakePills) FindPillsByUUIDs(context.Context, []uuid.UUID) ([]*model.ElixirPill, appErrors.Error) { panic("unused") }
func (f *fakePills) FindPills(context.Context, int, int, string, *bool) (int64, []*model.ElixirPill, appErrors.Error) { panic("unused") }
func (f *fakePills) SavePill(context.Context, *model.ElixirPill) appErrors.Error { panic("unused") }
func (f *fakePills) UpdatePill(context.Context, *model.ElixirPill, map[string]any) appErrors.Error { panic("unused") }
func (f *fakePills) DeletePill(context.Context, *model.ElixirPill) appErrors.Error { panic("unused") }
func (f *fakePills) FindAgentIDsByPillID(context.Context, uint) ([]uint, appErrors.Error) { panic("unused") }
func (f *fakePills) InvalidateLanguagePatternsByAgentIDs(context.Context, []uint) appErrors.Error { panic("unused") }

// pill_id 模式投影必须把空来源序列化为 [] 而非 null:
// Go nil slice → JSON null → Python Pydantic sources: List 校验失败 422
// (2026-08-28 桌面验收报错 list_type@body.sources 的回归防线)
func TestSkillExport_PillIDModeNeverSendsNullSources(t *testing.T) {
	client := &fakeClient{}
	pill := &model.ElixirPill{
		UUID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:        "结构化金丹",
		Description: "一份结构化的语言风格技能包",
		SkillSchema: map[string]interface{}{"identity_card": "我是金丹"},
		Tags:        model.JSONList{"语言"},
		UpdatedAt:   time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC),
	}
	service := New(client, fakeResolver{}, &fakePills{pill: pill})

	_, appErr := service.SkillExport(context.Background(), &distillation.SkillExportInput{
		PillID: "550e8400-e29b-41d4-a716-446655440000",
		Format: "codex",
	})
	if appErr != nil {
		t.Fatalf("unexpected error = %#v", appErr)
	}
	if client.exportSkill == nil {
		t.Fatal("client was not called with a projected skill")
	}
	if client.exportSkill.Sources == nil {
		t.Fatal("Sources 为 nil: JSON 序列化为 null 会被 Pydantic 拒绝")
	}
	raw, err := json.Marshal(client.exportSkill)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes.Contains(raw, []byte(`"sources":null`)) {
		t.Fatalf("投影包含 sources:null: %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"sources":[]`)) {
		t.Fatalf("投影应含 sources:[]: %s", raw)
	}
}
