package model

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// GetProvider 供应商详情(掩码形式)
// GET /api/v1/providers/:uuid
func (cls *Model) GetProvider(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	v, serr := cls.provider.GetProviderByUUID(contextutil.NewContextWithGin(c), uid)
	if serr != nil {
		return 0, nil, serr
	}
	return response.Ok, toProviderResponse(v), nil
}
