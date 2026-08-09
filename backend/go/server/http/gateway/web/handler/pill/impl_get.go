package pill

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// parseUUID 解析路径中的 :uuid 参数;非法形态返回 400
func parseUUID(c *gin.Context) (uuid.UUID, errors.Error) {
	uid, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		return uuid.Nil, errors.New(errors.ErrorTypeInvalidRequest, "handler.pill.uuid_parse", "金丹ID格式不正确")
	}
	return uid, nil
}

// Get 金丹详情
// GET /api/v1/pills/:uuid
func (cls *Pill) Get(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	pill, serr := cls.pill.GetPillByUUID(contextutil.NewContextWithGin(c), uid)
	if serr != nil {
		return response.NotFound, nil, serr
	}
	return response.Ok, ToResponse(pill), nil
}
