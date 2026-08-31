// 共享的 nuwa-skill schema 校验（金丹消耗品重构任务 2：从 pill_service.go 提取）
// 逻辑唯一事实源：旧 ElixirPill 与新的丹方版本共用同一套规则，
// 禁止在 pill_inventory_service 里复制分叉一份。
package pill_service

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/model"
)

// ValidateSkillSchema 校验 nuwa-skill 结构化内容:
// expression_dna 必须存在且为非空对象;mental_models 长度 0-20;example_dialogues 长度 0-10
func ValidateSkillSchema(schema model.JSONMap) errors.Error {
	if len(schema) == 0 {
		return errors.New(errors.ErrorTypeInvalidRequest, "service.pill.schema.empty", "skill_schema 校验失败: skill_schema 不能为空")
	}

	dna, ok := schema["expression_dna"]
	if !ok || dna == nil {
		return errors.New(errors.ErrorTypeInvalidRequest, "service.pill.schema.dna_missing", "skill_schema 校验失败: 缺少 expression_dna")
	}
	if dnaMap, ok := dna.(map[string]interface{}); !ok || len(dnaMap) == 0 {
		return errors.New(errors.ErrorTypeInvalidRequest, "service.pill.schema.dna_invalid", "skill_schema 校验失败: expression_dna 必须为非空对象")
	}

	if models, ok := schema["mental_models"]; ok && models != nil {
		list, ok := models.([]interface{})
		if !ok {
			return errors.New(errors.ErrorTypeInvalidRequest, "service.pill.schema.models_type", "skill_schema 校验失败: mental_models 必须为数组")
		}
		if len(list) > 20 {
			return errors.New(errors.ErrorTypeInvalidRequest, "service.pill.schema.models_len", "skill_schema 校验失败: mental_models 长度不能超过 20")
		}
	}

	if dialogues, ok := schema["example_dialogues"]; ok && dialogues != nil {
		list, ok := dialogues.([]interface{})
		if !ok {
			return errors.New(errors.ErrorTypeInvalidRequest, "service.pill.schema.dialogues_type", "skill_schema 校验失败: example_dialogues 必须为数组")
		}
		if len(list) > 10 {
			return errors.New(errors.ErrorTypeInvalidRequest, "service.pill.schema.dialogues_len", "skill_schema 校验失败: example_dialogues 长度不能超过 10")
		}
	}

	return nil
}
