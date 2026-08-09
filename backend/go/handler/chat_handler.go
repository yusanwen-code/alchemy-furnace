// Package handler 对话管理 HTTP 处理器
// 处理会话管理和 SSE 流式对话
// 对应 API: /api/v1/chat/sessions, /api/v1/chat/sse/:session_id
// SSE 聊天流程:
//  1. 加载/合成道人的语言模式（系统提示词），解析模型凭证按请求透传
//  2. 调用 Python 语言引擎 /chat/completions/stream (SSE) 并以标准 SSE 逐段转发
//  3. 停止 = 客户端中断连接（AbortController），c.Request.Context() 取消贯穿至上游 LLM 流
//  4. 保活：等待上游超过 25s 发送 SSE 注释心跳（: ping）
//  5. 保存消息到数据库（中断时保存已生成的部分内容）
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/response"
	"github.com/alchemy-furnace/server/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// chatFlowService SSEChat 依赖的对话服务（接口抽出便于单测 mock）
type chatFlowService interface {
	GetSessionAgentInfo(sessionID uint) (agentID uint, modelName string, err error)
	GetMessages(sessionID uint, page, pageSize int) ([]model.ChatMessage, int64, error)
	SaveMessage(sessionID uint, role, content string, sources model.JSONMap) (*model.ChatMessage, error)
	StreamChat(ctx context.Context, messages []map[string]string, creds *service.ModelCredentials, onChunk func(string)) (fullContent string, canceled bool, err error)
}

// patternProvider 语言模式合成（接口抽出便于单测 mock）
type patternProvider interface {
	GetOrBuildPattern(agentID uint) (*model.LanguagePattern, error)
}

// credentialResolver 模型凭证解析（接口抽出便于单测 mock）
type credentialResolver interface {
	ResolveCredentials(name string) (*service.ModelCredentials, error)
}

// ChatHandler 对话 HTTP 处理器
type ChatHandler struct {
	service  *service.ChatService
	patterns *service.LanguagePatternService
	models   *service.ModelService

	// SSE 流式对话依赖（默认与上方同实例，测试可注入 mock）
	flow    chatFlowService
	pattern patternProvider
	creds   credentialResolver
}

// NewChatHandler 创建对话处理器
func NewChatHandler() *ChatHandler {
	chatService := service.NewChatService()
	patternService := service.NewLanguagePatternService()
	modelService := service.NewModelService()
	return &ChatHandler{
		service:  chatService,
		patterns: patternService,
		models:   modelService,
		flow:     chatService,
		pattern:  patternService,
		creds:    modelService,
	}
}

// CreateSession 创建对话会话
// POST /api/v1/chat/sessions
// Body: { "agent_id": 1, "title": "论道标题" }
func (h *ChatHandler) CreateSession(c *gin.Context) {
	var req model.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	session, err := h.service.CreateSession(&req)
	if err != nil {
		zap.L().Error("[炼丹炉] 创建会话失败", zap.Error(err))
		response.InternalError(c, "创建会话失败")
		return
	}

	response.Created(c, session)
}

// ListSessions 会话列表
// GET /api/v1/chat/sessions?page=1&page_size=10
func (h *ChatHandler) ListSessions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	sessions, total, err := h.service.ListSessions(page, pageSize)
	if err != nil {
		zap.L().Error("[炼丹炉] 查询会话列表失败", zap.Error(err))
		response.InternalError(c, "查询会话列表失败")
		return
	}

	response.SuccessWithPage(c, total, page, pageSize, sessions)
}

// GetMessages 获取会话消息历史
// GET /api/v1/chat/sessions/:id/messages?page=1&page_size=20
func (h *ChatHandler) GetMessages(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "会话ID格式不正确")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	messages, total, err := h.service.GetMessages(uint(sessionID), page, pageSize)
	if err != nil {
		zap.L().Error("[炼丹炉] 查询消息历史失败", zap.Uint64("session_id", sessionID), zap.Error(err))
		response.InternalError(c, "查询消息历史失败")
		return
	}

	response.SuccessWithPage(c, total, page, pageSize, messages)
}

// ==================================================================
// 标准 SSE 流式对话
// ==================================================================

// sseHeartbeatInterval 等待上游超过该间隔则发送注释心跳（防代理空闲超时）
const sseHeartbeatInterval = 25 * time.Second

// setSSEHeaders 设置 SSE 响应头（X-Accel-Buffering: no 指示 nginx 不缓冲）
func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
}

// sseWriteEvent 写入一条标准 SSE 事件（event + data 双行，空行分隔）并立即 Flush
func sseWriteEvent(w http.ResponseWriter, flusher http.Flusher, event string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		zap.L().Warn("[炼丹炉] SSE 事件序列化失败", zap.String("event", event), zap.Error(err))
		return
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		zap.L().Warn("[炼丹炉] SSE 事件写入失败", zap.String("event", event), zap.Error(err))
	}
	flusher.Flush()
}

// sseWriteComment 写入 SSE 注释行（心跳，客户端解析器应忽略）并立即 Flush
func sseWriteComment(w http.ResponseWriter, flusher http.Flusher, comment string) {
	if _, err := fmt.Fprintf(w, ": %s\n\n", comment); err != nil {
		zap.L().Warn("[炼丹炉] SSE 心跳写入失败", zap.Error(err))
	}
	flusher.Flush()
}

// ssePayload 事件数据载体（chunk: {"content": "..."}，done: {}，error: {"content": "中文描述"}）
type ssePayload struct {
	Content string `json:"content,omitempty"`
}

// streamResult StreamChat  goroutine 的收尾结果
type streamResult struct {
	full     string
	canceled bool
	err      error
}

// SSEChat 标准 SSE 流式对话
// POST /api/v1/chat/sse/:session_id
// Body: { "content": "用户问题" }
//
// 服务端 -> 客户端事件:
//
//	event: chunk   data: {"content": "回答片段"}
//	event: done    data: {}                     （完整回复已入库）
//	event: error   data: {"content": "可读中文错误"}
//	: ping                                    （25s 注释心跳）
//
// 停止生成：客户端中断连接（AbortController.abort()），无 stopped 确认事件；
// 已生成的部分内容（非空时）保存为 assistant 消息
func (h *ChatHandler) SSEChat(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("session_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "会话ID格式不正确")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Content) == "" {
		response.BadRequest(c, "消息内容不能为空")
		return
	}
	content := req.Content

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.InternalError(c, "当前服务不支持流式响应")
		return
	}

	setSSEHeaders(c)
	w := c.Writer

	// 获取会话关联的道人信息
	agentID, modelName, err := h.flow.GetSessionAgentInfo(uint(sessionID))
	if err != nil {
		sseWriteEvent(w, flusher, "error", ssePayload{Content: "获取会话信息失败: " + err.Error()})
		return
	}

	// 加载/合成道人的语言模式（系统提示词）
	pattern, err := h.pattern.GetOrBuildPattern(agentID)
	if err != nil {
		zap.L().Error("[炼丹炉] 语言模式合成失败", zap.Error(err))
		sseWriteEvent(w, flusher, "error", ssePayload{Content: "化丹为性失败: " + err.Error()})
		return
	}

	// 保存用户消息到数据库
	if _, err := h.flow.SaveMessage(uint(sessionID), "user", content, nil); err != nil {
		zap.L().Error("[炼丹炉] 保存用户消息失败", zap.Error(err))
		sseWriteEvent(w, flusher, "error", ssePayload{Content: "保存消息失败"})
		return
	}

	zap.L().Info("[炼丹炉] 收到论道问题",
		zap.Uint64("session_id", sessionID),
		zap.Uint("agent_id", agentID),
		zap.String("model", modelName),
		zap.String("content", content))

	// 获取历史消息（最近 20 条，用于上下文）
	recentMessages, _, err := h.flow.GetMessages(uint(sessionID), 1, 20)
	if err != nil {
		zap.L().Error("[炼丹炉] 获取历史消息失败", zap.Error(err))
		sseWriteEvent(w, flusher, "error", ssePayload{Content: "获取历史消息失败"})
		return
	}

	// 构建消息列表（OpenAI 格式，首条为合成后的系统提示词）
	messages := []map[string]string{
		{"role": "system", "content": pattern.SystemPrompt},
	}
	for _, m := range recentMessages {
		messages = append(messages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	// 解析模型凭证（每轮解析，模型停用/换钥即时生效；错误已是可读中文）
	creds, err := h.creds.ResolveCredentials(modelName)
	if err != nil {
		zap.L().Warn("[炼丹炉] 模型凭证解析失败", zap.String("model", modelName), zap.Error(err))
		sseWriteEvent(w, flusher, "error", ssePayload{Content: err.Error()})
		return
	}

	// 请求级生命周期：客户端中断（abort/关闭页面/断网）→ ctx 取消 → 上游 LLM 流中断
	ctx := c.Request.Context()

	chunkCh := make(chan string)
	resultCh := make(chan streamResult, 1)
	go func() {
		full, canceled, streamErr := h.flow.StreamChat(ctx, messages, creds, func(chunk string) {
			select {
			case chunkCh <- chunk:
			case <-ctx.Done():
			}
		})
		resultCh <- streamResult{full: full, canceled: canceled, err: streamErr}
	}()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case chunk := <-chunkCh:
			sseWriteEvent(w, flusher, "chunk", ssePayload{Content: chunk})
			heartbeat.Reset(sseHeartbeatInterval)

		case <-heartbeat.C:
			// 生成期间无 chunk 间隙保活（如思考型模型长时间静默）
			sseWriteComment(w, flusher, "ping")

		case res := <-resultCh:
			h.finishSSEStream(uint(sessionID), w, flusher, res)
			return

		case <-ctx.Done():
			// 客户端已中断：StreamChat 每轮检查 ctx，会迅速收尾返回部分内容
			res := <-resultCh
			if res.full != "" {
				if _, err := h.flow.SaveMessage(uint(sessionID), "assistant", res.full, nil); err != nil {
					zap.L().Error("[炼丹炉] 保存中断时的部分回复失败", zap.Error(err))
				}
			}
			zap.L().Info("[炼丹炉] 论道已被叫停",
				zap.Uint64("session_id", sessionID), zap.Int("partial_length", len(res.full)))
			return
		}
	}
}

// finishSSEStream 处理 StreamChat 正常收尾：done / error / 取消（恰好遇完成按 done 处理）
func (h *ChatHandler) finishSSEStream(sessionID uint, w http.ResponseWriter, flusher http.Flusher, res streamResult) {
	switch {
	case res.canceled:
		// 连接已中断（取消仅由 ctx 触发）：保存部分内容，不再写事件
		if res.full != "" {
			if _, err := h.flow.SaveMessage(sessionID, "assistant", res.full, nil); err != nil {
				zap.L().Error("[炼丹炉] 保存停止时的部分回复失败", zap.Error(err))
			}
		}
		zap.L().Info("[炼丹炉] 论道已被叫停",
			zap.Uint("session_id", sessionID), zap.Int("partial_length", len(res.full)))

	case res.err != nil:
		// 结构化中文错误（模型凭证无效 / 引擎超时 / 引擎异常等）
		sseWriteEvent(w, flusher, "error", ssePayload{Content: res.err.Error()})

	default:
		if res.full != "" {
			if _, err := h.flow.SaveMessage(sessionID, "assistant", res.full, nil); err != nil {
				zap.L().Error("[炼丹炉] 保存助手回复失败", zap.Error(err))
			}
		}
		sseWriteEvent(w, flusher, "done", ssePayload{})
		zap.L().Info("[炼丹炉] 论道一轮完成",
			zap.Uint("session_id", sessionID), zap.Int("response_length", len(res.full)))
	}
}
