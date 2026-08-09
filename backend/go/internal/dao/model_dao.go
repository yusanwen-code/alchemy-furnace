// Package dao 模型配置只读查询(新架构;US1 道人创建/更新校验依赖,US2 迁移完整模型域)
package dao

import (
	"context"

	"github.com/alchemy-furnace/server/internal/errors"
)

// ModelDao 模型配置查询实现
type ModelDao struct{}

// NewModelDao 构造模型 DAO
func NewModelDao() *ModelDao {
	return &ModelDao{}
}

// CountEnabledModelByName 统计「已启用供应商下的已启用模型」中指定模型名的数量
func (d *ModelDao) CountEnabledModelByName(ctx context.Context, name string) (int64, errors.Error) {
	var count int64
	if err := GetDB().WithContext(ctx).Table("llm_models").
		Joins("JOIN llm_providers ON llm_providers.id = llm_models.provider_id").
		Where("llm_models.name = ? AND llm_models.is_enabled = ? AND llm_providers.is_enabled = ?", name, true, true).
		Count(&count).Error; err != nil {
		return 0, errors.ErrorServerInternalError("dao.model.count_enabled_by_name")
	}
	return count, nil
}
