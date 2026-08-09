package pill

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// Delete 删除金丹(级联删除服用记录并失效相关语言模式缓存)
// DELETE /api/v1/pills/:uuid
func (cls *Pill) Delete(c *gin.Context) (response.Code, any, error) {
	uid, err := parseUUID(c)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	if serr := cls.pill.DeletePill(contextutil.NewContextWithGin(c), uid); serr != nil {
		return 0, nil, serr
	}
	return response.Ok, gin.H{"deleted": true}, nil
}
