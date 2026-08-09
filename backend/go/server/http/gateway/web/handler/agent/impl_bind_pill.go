package agent

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/server/http/request"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BindPillRequest 服用金丹请求:pill_id 为金丹 UUID 字符串
type BindPillRequest struct {
	PillID    string  `json:"pill_id" binding:"required"`
	Weight    float64 `json:"weight" binding:"gte=0,lte=10"`
	SortOrder int     `json:"sort_order" binding:"gte=0"`
}

// BindPill 道人服用金丹
// POST /api/v1/agents/:uuid/pills
func (cls *Agent) BindPill(c *gin.Context) (response.Code, any, error) {
	agentUID, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	var body BindPillRequest
	if berr := request.ShouldBindJSON(c, &body); berr != nil {
		return response.InvalidParams, nil, berr
	}

	pillUID, perr := uuid.Parse(body.PillID)
	if perr != nil {
		return response.InvalidParams, nil, errors.New(errors.ErrorTypeInvalidRequest, "handler.agent.bind.pill_uuid", "金丹ID格式不正确")
	}

	if serr := cls.agent.BindPill(contextutil.NewContextWithGin(c), agentUID, pillUID, body.Weight, body.SortOrder); serr != nil {
		return response.CodeBindPillFailed, nil, serr
	}
	return response.Ok, gin.H{"bound": true}, nil
}
