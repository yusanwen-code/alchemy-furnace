// 幂等操作查询：断线恢复（客户端重试前先查已提交结果）
package pill_inventory

import (
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// GetOperation 读取已提交操作结果（幂等键=操作 ID；断线恢复用）
// GET /api/v1/pill-operations/:id
func (h *Handler) GetOperation(c *gin.Context) (response.Code, any, error) {
	opID, err := pathUUID(c, "id")
	if err != nil {
		return response.InvalidParams, nil, err
	}
	result, err := h.inventory.GetOperation(ctx(c), opID)
	if err != nil {
		return 0, nil, err
	}
	return response.Ok, operationResultOut(result), nil
}
