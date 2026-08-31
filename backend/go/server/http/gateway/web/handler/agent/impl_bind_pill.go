// 旧服用入口（任务 5 起下线）
// 服用金丹改 POST /api/v1/agents/:id/consume（Idempotency-Key 幂等契约）
// （plan 任务 5 行 430：绑定写路由返回 410 pill.legacy_api_removed）。
package agent

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// BindPillRequest 旧服用金丹请求契约（保留以文档化旧客户端载荷；接口已下线）
type BindPillRequest struct {
	PillID    string  `json:"pill_id" binding:"required"`
	Weight    float64 `json:"weight" binding:"gte=0,lte=10"`
	SortOrder int     `json:"sort_order" binding:"gte=0"`
}

// BindPill 旧服用入口已下线
// POST /api/v1/agents/:uuid/pills → 410
func (cls *Agent) BindPill(c *gin.Context) (response.Code, any, error) {
	return 0, nil, errors.ErrorGone("pill.legacy_api_removed",
		"金丹消耗品重构：旧绑定接口已下线，服用请使用 /api/v1/agents/:id/consume")
}
