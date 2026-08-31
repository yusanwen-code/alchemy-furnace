// 旧入口封堵（任务 5）：旧 /pills/:uuid 详情仅提供 LegacyMap 跳转。
// 响应 {entity_type:"recipe", recipe_id:"..."}，旧详情 UI 显示"已升级为丹方"，
// 不假装该 ID 是可用金丹（plan 任务 5 行 431）。
package pill_inventory

import (
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// ResolveLegacyPill 旧金丹详情跳转：旧定义 UUID → 丹方 UUID。
// GET /api/v1/pills/:uuid（旧路由；任务 5 起仅提供跳转信息，无映射 404）
func (h *Handler) ResolveLegacyPill(c *gin.Context) (response.Code, any, error) {
	legacyID, err := pathUUID(c, "uuid")
	if err != nil {
		return response.InvalidParams, nil, err
	}
	target, err := h.inventory.ResolveLegacy(ctx(c), "pill", legacyID.String())
	if err != nil {
		return 0, nil, err
	}
	return response.Ok, map[string]any{
		"entity_type": "recipe",
		"recipe_id":   target.String(),
	}, nil
}
