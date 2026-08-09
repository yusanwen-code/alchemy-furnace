package agent

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/request"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// UpdateAgentPillRequest 更新服用记录请求(bug 修复点:body 不再有 pill_id,金丹由路径 :pill_uuid 标识)
type UpdateAgentPillRequest struct {
	Weight    *float64 `json:"weight" binding:"omitempty,gte=0,lte=10"`
	SortOrder *int     `json:"sort_order" binding:"omitempty,gte=0"`
}

// UpdateAgentPill 更新服用记录(权重/顺序)
// PUT /api/v1/agents/:uuid/pills/:pill_uuid
func (cls *Agent) UpdateAgentPill(c *gin.Context) (response.Code, any, error) {
	agentUID, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}
	pillUID, err := parsePillUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	var body UpdateAgentPillRequest
	if berr := request.ShouldBindJSON(c, &body); berr != nil {
		return response.InvalidParams, nil, berr
	}

	if serr := cls.agent.UpdateAgentPill(contextutil.NewContextWithGin(c), agentUID, pillUID, body.Weight, body.SortOrder); serr != nil {
		return response.CodeUpdateAgentPillFailed, nil, serr
	}
	return response.Ok, gin.H{"updated": true}, nil
}
