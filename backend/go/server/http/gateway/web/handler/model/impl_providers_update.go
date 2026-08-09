package model

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/request"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// UpdateProviderRequest 更新供应商请求(指针字段区分「未传」与「置空/置零」)
// api_key: 不传(nil)=不修改,传空字符串=清除密钥,传值=重新加密存储
type UpdateProviderRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=50"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=100"`
	Protocol    *string `json:"protocol" binding:"omitempty,max=50"`
	BaseURL     *string `json:"base_url" binding:"omitempty,max=255"`
	APIKey      *string `json:"api_key"`
	IsEnabled   *bool   `json:"is_enabled"`
	SortOrder   *int    `json:"sort_order"`
	Remark      *string `json:"remark" binding:"omitempty,max=255"`
}

// UpdateProvider 更新供应商
// PUT /api/v1/providers/:uuid
func (cls *Model) UpdateProvider(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	var body UpdateProviderRequest
	if err := request.ShouldBindJSON(c, &body); err != nil {
		return response.InvalidParams, nil, err
	}

	v, serr := cls.provider.UpdateProvider(contextutil.NewContextWithGin(c), uid,
		body.Name, body.DisplayName, body.Protocol, body.BaseURL, body.APIKey,
		body.IsEnabled, body.SortOrder, body.Remark)
	if serr != nil {
		return 0, nil, serr
	}
	return response.Ok, toProviderResponse(v), nil
}
