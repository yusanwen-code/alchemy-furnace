// Package pill_inventory 金丹消耗品库存 HTTP 处理器（任务 5）
// 路由契约见 docs/superpowers/plans/2026-08-31-pill-recipes-consumables.md §2.3：
// 丹方 /recipes、库存 /pill-items、服用与能力 /agents/:id/{consume,effects}、
// 融合两阶段 /fusion/{previews,confirm}、幂等操作查询 /pill-operations/:id。
// 所有写操作（预览与查询除外）要求 Idempotency-Key（UUID 全局幂等键）：
// 缺失/非法 → 400；成功响应 data.operation_id 与请求头同值。
package pill_inventory

import (
	"context"
	"math"
	"strings"

	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler 金丹消耗品库存处理器
type Handler struct {
	inventory service.PillInventory // 丹方/库存/服用/融合确认/幂等操作
	fusion    service.Fusion        // 融合预览（两阶段第一阶段，模型调用在事务外）
	agent     service.Agent         // 能力列表/全量编排/移除
}

// New 构造库存处理器
func New(inventory service.PillInventory, fusion service.Fusion, agent service.Agent) *Handler {
	return &Handler{inventory: inventory, fusion: fusion, agent: agent}
}

// ---------- 公共校验与解析 ----------

// idempotencyKey 读 Idempotency-Key 头并解析 UUID；缺失/非法 → 400
func idempotencyKey(c *gin.Context) (uuid.UUID, error) {
	raw := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if raw == "" {
		return uuid.Nil, errors.New(errors.ErrorTypeInvalidRequest,
			"handler.pill_inventory.idempotency_missing", "缺少 Idempotency-Key 请求头")
	}
	uid, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, errors.New(errors.ErrorTypeInvalidRequest,
			"handler.pill_inventory.idempotency_parse", "Idempotency-Key 不是合法 UUID")
	}
	return uid, nil
}

// pathUUID 解析路径参数为 UUID；缺失/非法 → 400
func pathUUID(c *gin.Context, name string) (uuid.UUID, error) {
	raw := c.Param(name)
	if raw == "" {
		return uuid.Nil, errors.New(errors.ErrorTypeInvalidRequest,
			"handler.pill_inventory.path_missing", "缺少路径参数 %s", name)
	}
	uid, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, errors.New(errors.ErrorTypeInvalidRequest,
			"handler.pill_inventory.uuid_parse", "路径参数 %s 不是合法 UUID", name)
	}
	return uid, nil
}

// validateWeight weight 必须在 [0,10] 且为有限数；非法 → 400
func validateWeight(name string, w float64) error {
	if math.IsNaN(w) || math.IsInf(w, 0) || w < 0 || w > 10 {
		return errors.New(errors.ErrorTypeInvalidRequest,
			"handler.pill_inventory.weight_range", "%s 必须在 0-10 之间", name)
	}
	return nil
}

// validateSortOrder sort_order 不得为负；非法 → 400
func validateSortOrder(name string, v int) error {
	if v < 0 {
		return errors.New(errors.ErrorTypeInvalidRequest,
			"handler.pill_inventory.sort_order_negative", "%s 不得为负数", name)
	}
	return nil
}

// ctx 从 gin 上下文取请求 context
func ctx(c *gin.Context) context.Context {
	return contextutil.NewContextWithGin(c)
}
