package model

import (
	"strconv"

	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/gin-gonic/gin"
)

// ListProviders 供应商列表
// GET /api/v1/providers?enabled=true&page=1&page_size=50
func (cls *Model) ListProviders(c *gin.Context) (int64, int, int, any, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	var enabled *bool
	if raw := c.Query("enabled"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return 0, 0, 0, nil, errors.New(errors.ErrorTypeInvalidRequest, "handler.model.list.enabled", "enabled 参数仅支持 true/false")
		}
		enabled = &v
	}

	ctx := contextutil.NewContextWithGin(c)
	total, views, err := cls.provider.ListProviders(ctx, page, pageSize, enabled)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	return total, page, pageSize, toProviderResponseList(views), nil
}
