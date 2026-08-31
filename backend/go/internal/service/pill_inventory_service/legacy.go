// 旧实体映射解析（任务 5 旧入口封堵）
// 旧 /pills 详情跳转与旧 pill ID 导出共用：kind=pill 旧定义→丹方 UUID，
// kind=bind 旧绑定→能力 UUID。无映射 404，不假装旧 ID 是可用金丹（plan 任务 5）。
package pill_inventory_service

import (
	"context"
	stderrors "errors"

	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResolveLegacy 按 (kind, legacyID) 唯一键查旧实体映射。
// legacyID 按不透明字符串匹配（迁移写入的是旧表 UUID 字符串）；
// 未知 kind → 400；无映射 → 404 pill.legacy_not_found。
func (s *Inventory) ResolveLegacy(ctx context.Context, kind, legacyID string) (uuid.UUID, errors.Error) {
	if kind != "pill" && kind != "bind" {
		return uuid.Nil, errors.New(errors.ErrorTypeInvalidRequest,
			"pill.invalid_legacy_kind", "未知的旧实体类型: %s", kind)
	}
	m, err := dao.PillLegacyMapByKindID(s.db.WithContext(ctx), kind, legacyID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, errors.ErrorRecordNotFound("pill.legacy_not_found")
		}
		return uuid.Nil, errors.ErrorServerInternalError("pill.legacy_query_failed")
	}
	return m.TargetUUID, nil
}
