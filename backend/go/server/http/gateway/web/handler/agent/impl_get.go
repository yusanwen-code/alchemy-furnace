package agent

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// Get 道人详情(含已服用金丹与语言模式缓存)
// GET /api/v1/agents/:uuid
func (cls *Agent) Get(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	agent, serr := cls.agent.GetAgentDetailByUUID(contextutil.NewContextWithGin(c), uid)
	if serr != nil {
		return response.NotFound, nil, serr
	}
	return response.Ok, toDetailResponse(agent), nil
}
