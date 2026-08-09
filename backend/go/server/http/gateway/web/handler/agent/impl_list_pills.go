package agent

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/gateway/web/handler/pill"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// ListPills 道人已服用的金丹列表(金丹对象数组,按服用顺序)
// GET /api/v1/agents/:uuid/pills
func (cls *Agent) ListPills(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	pills, serr := cls.agent.ListAgentPills(contextutil.NewContextWithGin(c), uid)
	if serr != nil {
		return 0, nil, serr
	}

	list := make([]*pill.Response, 0, len(pills))
	for _, p := range pills {
		list = append(list, pill.ToResponse(p))
	}
	return response.Ok, list, nil
}
