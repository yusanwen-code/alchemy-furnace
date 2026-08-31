// 迁移摘要只读端点（任务 8：升级用户展示）
// GET /api/v1/migration-summary
// 读迁移完成标记（pill_migration_states）返回迁移时计数；
// 无标记 → migrated=false。纯读端点，禁止触发迁移。
package pill_inventory

import (
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// MigrationSummary 迁移摘要：已保存丹方/保留能力/可用金丹计数 + 备份路径 + 完成时间
func (h *Handler) MigrationSummary(c *gin.Context) (response.Code, any, error) {
	s, err := h.inventory.MigrationSummary(ctx(c))
	if err != nil {
		return 0, nil, err
	}
	return response.Ok, s, nil
}
