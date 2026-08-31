// Package fusion 金丹融合 HTTP 处理器（新网关）
// 旧路由 /api/v1/fusion/fuse 任务 5 起下线：融合改两阶段
// POST /api/v1/fusion/previews + POST /api/v1/fusion/confirm（预览不扣料、确认原子扣料）。
package fusion

import (
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// Fusion 金丹融合处理器
type Fusion struct {
	fusion service.Fusion
}

// New 构造金丹融合处理器
func New(fusion service.Fusion) *Fusion {
	return &Fusion{fusion: fusion}
}

// FuseRequest 旧金丹融合请求契约（保留以文档化旧客户端载荷；接口已下线）
type FuseRequest struct {
	PillUUIDs         []string `json:"pill_uuids" binding:"required,min=2"` // 原料金丹 UUID(至少 2 枚)
	ExcludeOperatorID string   `json:"exclude_operator_id"`                 // 重试时要排除的算子 id(可选)
}

// FuseResponse 旧金丹融合响应契约（保留以文档化旧客户端载荷；接口已下线）
type FuseResponse struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	SkillSchema map[string]interface{} `json:"skill_schema"`
	Operator    interface{}            `json:"operator"`
	Model       string                 `json:"model"`
	Degraded    bool                   `json:"degraded"`
}

// Fuse 旧融合预览入口已下线
// POST /api/v1/fusion/fuse → 410
func (cls *Fusion) Fuse(c *gin.Context) (response.Code, any, error) {
	return 0, nil, errors.ErrorGone("fusion.legacy_api_removed",
		"金丹消耗品重构：旧融合接口已下线，请使用 POST /api/v1/fusion/previews + /fusion/confirm")
}
