package model

import (
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// Templates 预置供应商模板清单(静态常量,前后端单一数据源)
// GET /api/v1/providers/templates
func (cls *Model) Templates(c *gin.Context) (response.Code, any, error) {
	return response.Ok, cls.provider.Templates(), nil
}
