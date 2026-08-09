package agent

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// UnbindPill 道人解除金丹绑定
// DELETE /api/v1/agents/:uuid/pills/:pill_uuid
func (cls *Agent) UnbindPill(c *gin.Context) (response.Code, any, error) {
	agentUID, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}
	pillUID, err := parsePillUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	if serr := cls.agent.UnbindPill(contextutil.NewContextWithGin(c), agentUID, pillUID); serr != nil {
		return response.CodeUnbindPillFailed, nil, serr
	}
	return response.Ok, gin.H{"unbound": true}, nil
}
