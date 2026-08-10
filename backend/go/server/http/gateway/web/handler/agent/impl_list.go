package agent

import (
	"strconv"

	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/gin-gonic/gin"
)

// List 道人列表
// GET /api/v1/agents?page=1&page_size=10&status=active
func (cls *Agent) List(c *gin.Context) (int64, int, int, any, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")

	ctx := contextutil.NewContextWithGin(c)
	total, agents, err := cls.agent.ListAgents(ctx, page, pageSize, status)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	return total, page, pageSize, toResponseList(agents), nil
}
