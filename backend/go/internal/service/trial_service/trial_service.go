// Package trial_service 试丹业务逻辑实现(新架构 internal 分层)
// 提供无需创建道人即可临时组合「基础性格 + 金丹」快速预览效果的接口
// 对应 RESTful API: /api/v1/trial/synthesis, /api/v1/trial/chat
package trial_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/alchemy-furnace/server/internal/configuration"
	"github.com/alchemy-furnace/server/internal/behavior"
	"github.com/alchemy-furnace/server/internal/errors"
	idao "github.com/alchemy-furnace/server/internal/interface/dao"
	iservice "github.com/alchemy-furnace/server/internal/interface/service"
	"github.com/alchemy-furnace/server/internal/service/credential"
	"github.com/alchemy-furnace/server/internal/service/engine"
	"github.com/alchemy-furnace/server/internal/synthesis"
	"github.com/alchemy-furnace/server/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Trial service.Trial 接口实现
type Trial struct {
	pill       idao.Pill
	synthesis  synthesis.Client // 接口,便于单测 mock
	credential credential.Resolver
	httpClient httpDoer // 接口化: 生产为 *http.Client,测试可注入假实现
}

// httpDoer 可注入的 HTTP 客户端(http.Client 天然满足)
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// New 构造试丹业务实例
func New(pill idao.Pill, synthesisClient synthesis.Client, credential credential.Resolver) *Trial {
	return &Trial{
		pill:       pill,
		synthesis:  synthesisClient,
		credential: credential,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// loadTrialPills 按 UUID 批量加载金丹并组装为合成输入
// 排序键 (sort_order, uuid_str) 字典序,保证合成指纹稳定
func (s *Trial) loadTrialPills(ctx context.Context, inputs []iservice.TrialPillInput) ([]synthesis.PillInput, errors.Error) {
	if len(inputs) == 0 {
		return []synthesis.PillInput{}, nil
	}

	// 排序键 (sort_order, uuid_str) 字典序,与 Python 端指纹算法对齐
	sorted := make([]iservice.TrialPillInput, len(inputs))
	copy(sorted, inputs)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].SortOrder != sorted[j].SortOrder {
			return sorted[i].SortOrder < sorted[j].SortOrder
		}
		return sorted[i].PillID.String() < sorted[j].PillID.String()
	})

	uids := make([]uuid.UUID, 0, len(sorted))
	for _, in := range sorted {
		uids = append(uids, in.PillID)
	}

	pills, err := s.pill.FindPillsByUUIDs(ctx, uids)
	if err != nil {
		return nil, err.Relation(errors.ErrorServerInternalError("service.trial.load_pills"))
	}
	pillMap := make(map[string]*model.ElixirPill, len(pills))
	for _, p := range pills {
		pillMap[p.UUID.String()] = p
	}

	result := make([]synthesis.PillInput, 0, len(sorted))
	for _, in := range sorted {
		pill, ok := pillMap[in.PillID.String()]
		if !ok {
			return nil, errors.New(errors.ErrorTypeRecordNotFound, "service.trial.pill_missing", "金丹(id=%s)不存在", in.PillID.String())
		}
		weight := in.Weight
		if weight <= 0 {
			weight = 1.0
		}
		result = append(result, synthesis.PillInput{
			ID:          pill.UUID.String(),
			Name:        pill.Name,
			Weight:      weight,
			SortOrder:   in.SortOrder,
			SkillSchema: pill.SkillSchema,
		})
	}
	return result, nil
}

// Synthesize 试丹-合成预览:不写入缓存,返回行为引擎渲染结果
// 指定了 modelName 时按该模型解析凭证,否则使用合成专用模型凭证
func (s *Trial) Synthesize(ctx context.Context, personality string, pills []iservice.TrialPillInput, modelName string) (*iservice.TrialSynthesisResult, errors.Error) {
	loaded, err := s.loadTrialPills(ctx, pills)
	if err != nil {
		return nil, err
	}
	return s.renderTrialPrompt(ctx, personality, loaded, s.resolveTrialCredentials(ctx, modelName))
}

// renderTrialPrompt 试丹公共渲染: 确定性编译 + 涌现合并 + 渲染完整提示词。
// 合成失败/降级不返回错误: 返回无损确定性渲染(degraded=true),聊天不阻断(spec §12)
func (s *Trial) renderTrialPrompt(ctx context.Context, personality string, loaded []synthesis.PillInput, creds *credential.ModelCredentials) (*iservice.TrialSynthesisResult, errors.Error) {
	profile := behavior.CompileProfile(personality, loaded)

	combined, e := s.synthesis.Combine(ctx, personality, loaded, creds)
	if e != nil {
		profile.WithEmergence(nil, nil, true, "combine_error")
		return &iservice.TrialSynthesisResult{
			SystemPrompt:   behavior.RenderSystemPrompt(profile, ""),
			EmergenceRules: model.JSONList{},
			Degraded:       true,
			DegradedReason: "combine_error",
		}, nil
	}
	profile.WithEmergence(combined.EmergenceRules, combined.InnerTensions, combined.Degraded, combined.DegradedReason)

	emergenceRules := combined.EmergenceRules
	if emergenceRules == nil {
		emergenceRules = model.JSONList{}
	}
	return &iservice.TrialSynthesisResult{
		SystemPrompt:   behavior.RenderSystemPrompt(profile, ""),
		EmergenceRules: emergenceRules,
		InnerTensions:  combined.InnerTensions,
		Fingerprint:    combined.Fingerprint,
		Model:          combined.Model,
		Degraded:       combined.Degraded,
		DegradedReason: combined.DegradedReason,
	}, nil
}

// Chat 试丹-临时对话:先合成系统提示词,再调用语言引擎非流式对话
func (s *Trial) Chat(ctx context.Context, req *iservice.TrialChatRequest) (*iservice.TrialChatResponse, errors.Error) {
	loaded, err := s.loadTrialPills(ctx, req.Pills)
	if err != nil {
		return nil, err
	}

	// 合成系统提示词(失败/降级时返回无损确定性渲染,不阻塞试丹对话)
	result, e := s.renderTrialPrompt(ctx, req.Personality, loaded, s.resolveTrialCredentials(ctx, ""))
	if e != nil {
		return nil, e
	}

	// 组装消息:行为引擎渲染的 system 提示词 + 用户提供的消息
	messages := make([]map[string]string, 0, len(req.Messages)+1)
	messages = append(messages, map[string]string{"role": "system", "content": result.SystemPrompt})
	messages = append(messages, req.Messages...)

	// 对话模型凭证:指定 model 时按名解析,否则用默认/合成凭证
	creds := s.resolveTrialCredentials(ctx, req.Model)
	modelName := req.Model
	if modelName == "" {
		if creds != nil && creds.Model != "" {
			modelName = creds.Model
		} else {
			modelName = configuration.Configuration.LLM.DefaultModel
		}
	}
	temperature := req.Temperature
	if temperature <= 0 {
		temperature = 0.7
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	reqBody := map[string]interface{}{
		"messages":    messages,
		"model":       modelName,
		"temperature": temperature,
		"max_tokens":  maxTokens,
	}
	if creds != nil {
		if creds.BaseURL != "" {
			reqBody["base_url"] = creds.BaseURL
		}
		if creds.APIKey != "" {
			reqBody["api_key"] = creds.APIKey
		}
	}
	jsonBody, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/api/v1/chat/completions", configuration.Configuration.PythonEngine.BaseURL)
	httpReq, rerr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if rerr != nil {
		return nil, errors.ErrorServerInternalError("service.trial.chat_build_request")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, herr := s.httpClient.Do(httpReq)
	if herr != nil {
		return nil, errors.New(errors.ErrorTypeServerInternalError, "service.trial.chat_request", engine.MapEngineError(herr))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		engineErr := &engine.EngineError{Op: "语言引擎对话接口", StatusCode: resp.StatusCode, Body: string(body)}
		return nil, errors.New(errors.ErrorTypeServerInternalError, "service.trial.chat_engine", engine.MapEngineError(engineErr))
	}

	// Python /chat/completions 以 BaseResponse 信封返回 {code,message,data:{content,...}},
	// 需解包 data 后再取 Content/Model/Usage(与 /synthesis/combine 直返 CombineResponse 不同)
	var envelope struct {
		Code int                         `json:"code"`
		Data iservice.TrialChatResponse  `json:"data"`
	}
	if derr := json.NewDecoder(resp.Body).Decode(&envelope); derr != nil {
		return nil, errors.ErrorServerInternalError("service.trial.chat_decode")
	}
	return &envelope.Data, nil
}

// resolveTrialCredentials 解析试丹请求指定模型的凭证;空模型名走合成专用解析
// 解析失败时降级为 nil(Python 回退环境变量配置),不阻塞试丹流程
func (s *Trial) resolveTrialCredentials(ctx context.Context, modelName string) *credential.ModelCredentials {
	var creds *credential.ModelCredentials
	var err error
	if modelName != "" {
		creds, err = s.credential.ResolveCredentials(ctx, modelName)
	} else {
		creds, err = s.credential.ResolveSynthesisCredentials(ctx)
	}
	if err != nil {
		zap.L().Warn("[炼丹炉] 试丹模型凭证解析失败，回退环境变量配置",
			zap.String("model", modelName), zap.Error(err))
		return nil
	}
	return creds
}
