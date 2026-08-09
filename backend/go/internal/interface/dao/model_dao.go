package dao

import (
	"context"

	"github.com/alchemy-furnace/server/internal/errors"
)

// Model 模型配置查询接口(US1 仅道人模型名校验所需的最小面;US2 迁移完整模型域)
type Model interface {
	// CountEnabledModelByName 统计「已启用供应商下的已启用模型」中指定模型名的数量
	CountEnabledModelByName(ctx context.Context, name string) (int64, errors.Error)
}
