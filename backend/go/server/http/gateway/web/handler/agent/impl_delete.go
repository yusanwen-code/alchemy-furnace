package agent

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// Delete 删除道人
// DELETE /api/v1/agents/:uuid
func (cls *Agent) Delete(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	if serr := cls.agent.DeleteAgent(contextutil.NewContextWithGin(c), uid); serr != nil {
		return 0, nil, serr
	}
	return response.Ok, gin.H{"deleted": true}, nil
}
