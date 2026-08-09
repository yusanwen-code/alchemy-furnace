package model

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// Options 已启用供应商下的已启用模型精简列表(含供应商显示名),供道人表单下拉使用
// GET /api/v1/models/options
func (cls *Model) Options(c *gin.Context) (response.Code, any, error) {
	options, err := cls.model.Options(contextutil.NewContextWithGin(c))
	if err != nil {
		return 0, nil, err
	}
	return response.Ok, options, nil
}
