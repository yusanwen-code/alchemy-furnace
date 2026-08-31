// 任务 8：迁移摘要只读查询
// 读迁移完成标记（pill_migration_states）的 ReportJSON，供升级用户界面展示
// 「已保存 X 份丹方、保留 Y 项已吸收能力、可用金丹 Z 枚」。
// 纯读：不在此触发迁移（迁移只在启动链 MigratePillInventory 执行）。
package pill_inventory_service

import (
	"context"

	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/alchemy-furnace/server/model"
	"gorm.io/gorm"
)

// MigrationSummary 读迁移完成标记；无标记返回 Migrated=false（前端不展示）。
// ReportJSON 的数值是迁移时刻快照，非实时计数——升级摘要语义就是迁移时状态。
func (s *Inventory) MigrationSummary(ctx context.Context) (*service.MigrationSummary, errors.Error) {
	var state model.PillMigrationState
	err := s.db.WithContext(ctx).Where("key = ?", dao.PillInventoryMigrationKey).First(&state).Error
	if err == gorm.ErrRecordNotFound {
		return &service.MigrationSummary{}, nil
	}
	if err != nil {
		return nil, errors.New(errors.ErrorTypeServerInternalError,
			"pill.migration_summary_read", "读取迁移完成标记失败: %v", err)
	}

	// GORM serializer:json 已解码为 JSONMap（迁移报告由迁移代码构造，无 schema 全文/密钥）
	report := state.ReportJSON

	out := &service.MigrationSummary{
		Migrated:    true,
		CompletedAt: state.CompletedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	assignInt(&out.LegacyPills, report["legacy_pills"])
	assignInt(&out.LegacyBinds, report["legacy_binds"])
	assignInt(&out.Recipes, report["recipes"])
	assignInt(&out.AvailableItems, report["available_items"])
	assignInt(&out.HistoryItems, report["history_items"])
	assignInt(&out.Effects, report["effects"])
	if v, ok := report["is_fresh_install"].(bool); ok {
		out.IsFreshInstall = v
	}
	if v, ok := report["backup_path"].(string); ok {
		out.BackupPath = v
	}
	return out, nil
}

// assignInt 报告 JSON 数值可能是 float64（JSON 解码）或 int64（Go 直接构造），
// 两种来源都吸收；缺失/类型异常保持 0 并随报告展示（报告由迁移代码构造，不应出现）。
func assignInt(dst *int64, v any) {
	switch n := v.(type) {
	case float64:
		*dst = int64(n)
	case int64:
		*dst = n
	case int:
		*dst = int64(n)
	default:
		// 未知类型：不崩，保留 0（迁移报告字段缺失时语义即 0）
	}
}
