package chat

import (
	"strconv"

	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListSessions 会话列表(分页;可选按 agent_id 过滤)
// GET /api/v1/chat/sessions?page=1&page_size=10&agent_id=<uuid>
func (cls *Chat) ListSessions(c *gin.Context) (int64, int, int, any, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// agent_id 为可选过滤项;空串视为不按道人过滤(uuid.Nil)
	agentUID := uuid.Nil
	if raw := c.Query("agent_id"); raw != "" {
		uid, err := uuid.Parse(raw)
		if err != nil {
			return 0, 0, 0, nil, errors.New(errors.ErrorTypeInvalidRequest, "handler.chat.list_agent_uuid", "道人ID格式不正确")
		}
		agentUID = uid
	}

	total, sessions, err := cls.chat.ListSessions(contextutil.NewContextWithGin(c), agentUID, page, pageSize)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	return total, page, pageSize, toSessionResponseList(sessions), nil
}
