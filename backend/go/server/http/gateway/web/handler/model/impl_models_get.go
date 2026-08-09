package model

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// GetModel 模型详情(含 referenced_by 引用数)
// GET /api/v1/models/:uuid
func (cls *Model) GetModel(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	v, serr := cls.model.GetModelByUUID(contextutil.NewContextWithGin(c), uid)
	if serr != nil {
		return 0, nil, serr
	}
	return response.Ok, toModelResponse(v), nil
}
