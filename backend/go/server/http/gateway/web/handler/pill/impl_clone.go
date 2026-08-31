// 旧金丹克隆入口（任务 5 起下线）
// 炼制新金丹改 POST /api/v1/recipes/:id/craft（按不可变版本炼制一枚）
// （plan 任务 5 行 430：clone 写路由返回 410 pill.legacy_api_removed）。
package pill

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// Clone 旧金丹克隆已下线
// POST /api/v1/pills/:uuid/clone → 410
func (cls *Pill) Clone(c *gin.Context) (response.Code, any, error) {
	return 0, nil, errors.ErrorGone("pill.legacy_api_removed",
		"金丹消耗品重构：旧金丹管理接口已下线，炼制请使用 /api/v1/recipes/:id/craft")
}
