package model

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// DeleteModel 删除模型
// DELETE /api/v1/models/:uuid
// 被道人引用时返回 409(携带 referenced_by)
func (cls *Model) DeleteModel(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	if serr := cls.model.DeleteModel(contextutil.NewContextWithGin(c), uid); serr != nil {
		return 0, nil, serr
	}
	return response.Ok, gin.H{"deleted": true}, nil
}
