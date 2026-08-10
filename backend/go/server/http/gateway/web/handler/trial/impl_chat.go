package trial

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/alchemy-furnace/server/server/http/request"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// ChatRequest 试丹-临时对话请求
type ChatRequest struct {
	Personality string              `json:"personality"`                 // 基础性格描述
	Pills       []PillInput         `json:"pills"`                       // 临时组合的金丹列表
	Messages    []map[string]string `json:"messages" binding:"required"` // 对话消息(不含 system)
	Model       string              `json:"model"`                       // 对话模型(可选)
	Temperature float64             `json:"temperature"`                 // 温度(可选)
	MaxTokens   int                 `json:"max_tokens"`                  // 最大 token 数(可选)
}

// Chat 试丹-临时对话(非流式)
// POST /api/v1/trial/chat
func (cls *Trial) Chat(c *gin.Context) (response.Code, any, error) {
	var body ChatRequest
	if err := request.ShouldBindJSON(c, &body); err != nil {
		return response.InvalidParams, nil, err
	}

	pills, err := parsePillInputs(body.Pills)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	req := &service.TrialChatRequest{
		Personality: body.Personality,
		Pills:       pills,
		Messages:    body.Messages,
		Model:       body.Model,
		Temperature: body.Temperature,
		MaxTokens:   body.MaxTokens,
	}
	result, err := cls.trial.Chat(contextutil.NewContextWithGin(c), req)
	if err != nil {
		return 0, nil, err
	}
	return response.Ok, result, nil
}
