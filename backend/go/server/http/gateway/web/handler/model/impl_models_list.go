package model

import (
	"strconv"

	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// ListModels 供应商下的模型列表(含 referenced_by 引用数)
// GET /api/v1/providers/:uuid/models?page=1&page_size=100
func (cls *Model) ListModels(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	views, serr := cls.model.ListModelsByProvider(contextutil.NewContextWithGin(c), uid, page, pageSize)
	if serr != nil {
		return 0, nil, serr
	}
	return response.Ok, toModelResponseList(views), nil
}
