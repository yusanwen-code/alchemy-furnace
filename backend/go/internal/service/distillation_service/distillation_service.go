package distillation_service

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"
	"time"

	"github.com/alchemy-furnace/server/internal/distillation"
	appErrors "github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/interface/dao"
	"github.com/alchemy-furnace/server/internal/service/credential"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

type Service struct {
	client     distillation.Client
	credential credential.Resolver
	pills      dao.Pill
}

func New(client distillation.Client, resolver credential.Resolver, pills dao.Pill) *Service {
	return &Service{client: client, credential: resolver, pills: pills}
}

func (s *Service) Distill(ctx context.Context, subject, brief, locale string) (*distillation.Response, appErrors.Error) {
	subject, brief = strings.TrimSpace(subject), strings.TrimSpace(brief)
	if len([]rune(subject)) < 2 || len([]rune(brief)) < 4 {
		return nil, appErrors.New(appErrors.ErrorTypeInvalidRequest, "service.distillation.input", "请填写人物或主题，以及至少一句炼制目标")
	}
	if locale != "en" {
		locale = "zh-CN"
	}
	creds, err := s.credential.ResolveSynthesisCredentials(ctx)
	if err != nil {
		return nil, appErrors.New(appErrors.ErrorTypeInvalidRequest, "service.distillation.credentials", err.Error())
	}
	result, err := s.client.Distill(ctx, subject, brief, locale, creds)
	if err != nil {
		var remote *distillation.RemoteError
		if stderrors.As(err, &remote) {
			// 透传远端语义: error_code=远端 code,data={stage,retryable,details}。
			// 可重试的远端 503 映射为 ErrorTypeServiceUnavailable(HTTP 503),
			// 不被通用 Wrapper 隐藏成"服务器内部错误"。
			errorType := appErrors.ErrorTypeInvalidRequest
			switch {
			case remote.Status == http.StatusServiceUnavailable:
				errorType = appErrors.ErrorTypeServiceUnavailable
			case remote.Status >= http.StatusInternalServerError:
				errorType = appErrors.ErrorTypeServerInternalError
			}
			return nil, appErrors.NewWithData(errorType, remote.Code, map[string]any{
				"stage":     remote.Stage,
				"retryable": remote.Retryable,
				"details":   remote.Details,
			}, remote.Message)
		}
		return nil, appErrors.New(appErrors.ErrorTypeServerInternalError, "service.distillation.call", err.Error())
	}
	return result, nil
}

// SkillExport 只读导出: 服务端重校验(格式/目标唯一性/字段长度/slug/来源协议/敏感内容),
// 绝不接收或透传 API Key;pill_id 模式从数据库重新装载金丹,结构化模式重校验后透传。
// 接口不删除、不修改金丹;远端可重试错误映射为 503,内容错误映射为 400。
func (s *Service) SkillExport(ctx context.Context, input *distillation.SkillExportInput) (*distillation.ExportResult, appErrors.Error) {
	if input == nil {
		return nil, appErrors.New(appErrors.ErrorTypeInvalidRequest, "service.skill_export.input", "缺少导出请求")
	}
	format := strings.TrimSpace(input.Format)
	if format != "codex" && format != "claude" {
		return nil, appErrors.New(appErrors.ErrorTypeInvalidRequest, "service.skill_export.format", "format 必须是 codex 或 claude")
	}
	hasPill, hasSkill := strings.TrimSpace(input.PillID) != "", input.Skill != nil
	if hasPill == hasSkill {
		return nil, appErrors.New(appErrors.ErrorTypeInvalidRequest, "service.skill_export.target", "必须且只能提供 pill_id 或 skill 之一")
	}

	skill := input.Skill
	if hasPill {
		uid, err := uuid.Parse(strings.TrimSpace(input.PillID))
		if err != nil {
			return nil, appErrors.New(appErrors.ErrorTypeInvalidRequest, "service.skill_export.pill_id", "非法金丹 ID")
		}
		pill, derr := s.pills.TakePillByUUID(ctx, uid)
		if derr != nil {
			return nil, derr
		}
		// 只投影导出所需的规范化字段,绝不序列化整行数据库记录
		skill = &distillation.ExportableSkill{
			Name:          pill.Name,
			Description:   pill.Description,
			SkillSchema:   pill.SkillSchema,
			Tags:          jsonListToStrings(pill.Tags),
			Sources:       make([]distillation.Source, 0), // 空来源必须发 [] 而非 null(Pydantic sources: List 拒绝 null)
			GeneratedAt:   pill.UpdatedAt.UTC().Format(time.RFC3339),
			EvidenceLevel: "limited",
		}
	}

	if err := distillation.ValidateExportable(skill); err != nil {
		var v *distillation.ExportValidationError
		if stderrors.As(err, &v) {
			return nil, appErrors.NewWithData(appErrors.ErrorTypeInvalidRequest, "service.skill_export.invalid",
				map[string]any{"field": v.Field, "reason": v.Reason}, err.Error())
		}
		return nil, appErrors.New(appErrors.ErrorTypeInvalidRequest, "service.skill_export.invalid", err.Error())
	}

	result, err := s.client.SkillExport(ctx, skill, format)
	if err != nil {
		var remote *distillation.RemoteError
		if stderrors.As(err, &remote) {
			// 透传远端语义: error_code=远端 code,data={stage,retryable,details};
			// 可重试的远端 503 映射为 ErrorTypeServiceUnavailable(HTTP 503)
			errorType := appErrors.ErrorTypeInvalidRequest
			switch {
			case remote.Status == http.StatusServiceUnavailable:
				errorType = appErrors.ErrorTypeServiceUnavailable
			case remote.Status >= http.StatusInternalServerError:
				errorType = appErrors.ErrorTypeServerInternalError
			}
			return nil, appErrors.NewWithData(errorType, remote.Code, map[string]any{
				"stage":     remote.Stage,
				"retryable": remote.Retryable,
				"details":   remote.Details,
			}, remote.Message)
		}
		return nil, appErrors.New(appErrors.ErrorTypeServerInternalError, "service.skill_export.call", err.Error())
	}
	return result, nil
}

// jsonListToStrings model.JSONList(interface{} 列表)投影为导出模型使用的字符串列表
func jsonListToStrings(list model.JSONList) []string {
	result := make([]string, 0, len(list))
	for _, item := range list {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}
