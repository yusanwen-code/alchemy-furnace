// Package trial 试丹 HTTP 处理器(新网关;对齐 Luna-CY 模板 handler 分包风格)
// 提供无需创建道人即可临时组合「基础性格 + 金丹」快速预览效果的接口
// 路由: /api/v1/trial/synthesis, /api/v1/trial/chat
package trial

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/alchemy-furnace/server/internal/synthesis"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
)

// Trial 试丹处理器
type Trial struct {
	trial service.Trial
}

// New 构造试丹处理器
func New(trial service.Trial) *Trial {
	return &Trial{trial: trial}
}

// ---------- 请求/响应 DTO ----------

// PillInput 试丹请求中的单颗金丹引用(pill_id 为对外 UUID 字符串)
type PillInput struct {
	PillID    string  `json:"pill_id" binding:"required"`
	Weight    float64 `json:"weight"`
	SortOrder int     `json:"sort_order"`
}

// SynthesizeResponse 试丹合成预览响应
type SynthesizeResponse struct {
	SystemPrompt   string                   `json:"system_prompt"`
	EmergenceRules model.JSONList           `json:"emergence_rules"`
	InnerTensions  []synthesis.InnerTension `json:"inner_tensions"`
	Fingerprint    string                   `json:"fingerprint"`
	Model          string                   `json:"model"`
}

// ---------- 解析工具 ----------

// parsePillInputs 将请求金丹引用解析为业务层输入(UUID 边界在此解析,非法返回 400)
func parsePillInputs(inputs []PillInput) ([]service.TrialPillInput, errors.Error) {
	result := make([]service.TrialPillInput, 0, len(inputs))
	for i, in := range inputs {
		uid, err := uuid.Parse(in.PillID)
		if err != nil {
			return nil, errors.New(errors.ErrorTypeInvalidRequest, "handler.trial.pill_id_parse", "第%d颗金丹ID格式不正确", i+1)
		}
		result = append(result, service.TrialPillInput{
			PillID:    uid,
			Weight:    in.Weight,
			SortOrder: in.SortOrder,
		})
	}
	return result, nil
}
