package pill

import (
	"strconv"

	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/internal/errors"
	"github.com/gin-gonic/gin"
)

// List 金丹列表
// GET /api/v1/pills?page=1&page_size=10&keyword=xxx&is_builtin=true
func (cls *Pill) List(c *gin.Context) (int64, int, int, any, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	var isBuiltin *bool
	if raw := c.Query("is_builtin"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return 0, 0, 0, nil, errors.New(errors.ErrorTypeInvalidRequest, "handler.pill.list.is_builtin", "is_builtin 参数格式不正确，应为 true 或 false")
		}
		isBuiltin = &v
	}

	ctx := contextutil.NewContextWithGin(c)
	total, pills, err := cls.pill.ListPills(ctx, page, pageSize, keyword, isBuiltin)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	return total, page, pageSize, toResponseList(pills), nil
}
