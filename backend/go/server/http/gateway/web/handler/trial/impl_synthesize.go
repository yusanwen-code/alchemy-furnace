package trial

import (
	"github.com/alchemy-furnace/server/internal/context/contextutil"
	"github.com/alchemy-furnace/server/server/http/request"
	"github.com/alchemy-furnace/server/server/http/response"
	"github.com/gin-gonic/gin"
)

// SynthesizeRequest 试丹-合成预览请求
type SynthesizeRequest struct {
	Personality string      `json:"personality"` // 基础性格描述
	Pills       []PillInput `json:"pills"`       // 临时组合的金丹列表
	ModelName   string      `json:"model_name"`  // 合成使用的模型(可选)
}

// Synthesize 试丹-合成预览
// POST /api/v1/trial/synthesis
func (cls *Trial) Synthesize(c *gin.Context) (response.Code, any, error) {
	var body SynthesizeRequest
	if err := request.ShouldBindJSON(c, &body); err != nil {
		return response.InvalidParams, nil, err
	}

	pills, err := parsePillInputs(body.Pills)
	if err != nil {
		return response.InvalidParams, nil, err
	}

	result, err := cls.trial.Synthesize(contextutil.NewContextWithGin(c), body.Personality, pills, body.ModelName)
	if err != nil {
		return 0, nil, err
	}
	return response.Ok, SynthesizeResponse{
		SystemPrompt:   result.SystemPrompt,
		EmergenceRules: result.EmergenceRules,
		InnerTensions:  result.InnerTensions,
		Fingerprint:    result.Fingerprint,
		Model:          result.Model,
	}, nil
}
