package pill

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/server/http/request"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// UpdateRequest 更新金丹请求(指针字段区分「未传」与「置空」)
type UpdateRequest struct {
	Name        *string        `json:"name" binding:"omitempty,max=100"`
	Description *string        `json:"description"`
	SkillSchema model.JSONMap  `json:"skill_schema"`
	Tags        model.JSONList `json:"tags"`
	Author      *string        `json:"author" binding:"omitempty,max=100"`
	Version     *string        `json:"version" binding:"omitempty,max=20"`
}

// Update 更新金丹;更新后所有服用该金丹的道人语言模式缓存失效
// PUT /api/v1/pills/:uuid
func (cls *Pill) Update(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	var body UpdateRequest
	if berr := request.ShouldBindJSON(c, &body); berr != nil {
		return response.InvalidParams, nil, berr
	}

	pill, serr := cls.pill.UpdatePill(contextutil.NewContextWithGin(c), uid,
		body.Name, body.Description, body.SkillSchema, body.Tags, body.Author, body.Version)
	if serr != nil {
		return 0, nil, serr
	}
	return response.Ok, ToResponse(pill), nil
}
