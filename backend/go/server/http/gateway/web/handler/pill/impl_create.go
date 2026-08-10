package pill

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/server/http/request"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// CreateRequest 创建金丹请求
type CreateRequest struct {
	Name        string         `json:"name" binding:"required,max=100"`
	Description string         `json:"description"`
	SkillSchema model.JSONMap  `json:"skill_schema" binding:"required"`
	Tags        model.JSONList `json:"tags"`
	Author      string         `json:"author" binding:"max=100"`
	Version     string         `json:"version" binding:"max=20"`
}

// Create 创建金丹
// POST /api/v1/pills
func (cls *Pill) Create(c *gin.Context) (response.Code, any, error) {
	var body CreateRequest
	if err := request.ShouldBindJSON(c, &body); err != nil {
		return response.InvalidParams, nil, err
	}

	// 错误路径返回码传 0:Wrapper 按错误类型映射(400 校验失败/500 内部错误)
	pill, err := cls.pill.CreatePill(contextutil.NewContextWithGin(c),
		body.Name, body.Description, body.SkillSchema, body.Tags, body.Author, body.Version)
	if err != nil {
		return 0, nil, err
	}
	return response.CodeCreated, ToResponse(pill), nil
}
