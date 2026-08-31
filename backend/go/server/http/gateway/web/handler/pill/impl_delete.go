// 旧金丹删除入口（任务 5 起下线）
// 弃置金丹实例改 POST /api/v1/pill-items/:id/discard（终态保留去向，不物理删除）
// （plan 任务 5 行 430：旧 /pills 写入返回 410 pill.legacy_api_removed）。
package pill

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// Delete 旧金丹删除已下线
// DELETE /api/v1/pills/:uuid → 410
func (cls *Pill) Delete(c *gin.Context) (response.Code, any, error) {
	return 0, nil, errors.ErrorGone("pill.legacy_api_removed",
		"金丹消耗品重构：旧金丹管理接口已下线，弃置请使用 /api/v1/pill-items/:id/discard")
}
