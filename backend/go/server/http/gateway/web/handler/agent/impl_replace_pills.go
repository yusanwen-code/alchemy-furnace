// 旧完整服丹编排入口（任务 5 起下线）
// 能力编排改 PUT /api/v1/agents/:id/effects（全量提交 + expected_effects_revision 乐观锁）
// （plan 任务 5 行 430：composition 写路由返回 410 pill.legacy_api_removed）。
package agent

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// ReplacePillsItem 旧完整服丹编排水（保留以文档化旧客户端载荷；接口已下线）
type ReplacePillsItem struct {
	PillID string  `json:"pill_id" binding:"required"` // 金丹 UUID
	Weight float64 `json:"weight" binding:"required"`  // 剂量/权重
}

// ReplacePillsRequest 旧完整服丹编排请求（pills 为空数组表示清空全部服用关系）
type ReplacePillsRequest struct {
	Pills []ReplacePillsItem `json:"pills"`
}

// ReplacePills 旧完整服丹编排已下线
// PUT /api/v1/agents/:uuid/pills → 410
func (cls *Agent) ReplacePills(c *gin.Context) (response.Code, any, error) {
	return 0, nil, errors.ErrorGone("pill.legacy_api_removed",
		"金丹消耗品重构：旧完整服丹编排接口已下线，能力编排请使用 /api/v1/agents/:id/effects")
}
