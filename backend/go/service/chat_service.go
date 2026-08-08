// Package service 对话业务逻辑层
// 处理会话管理和消息存储
// 流式对话的核心逻辑在 handler/chat_handler.go 的 WebSocket 中实现
package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alchemy-furnace/server/dao"
	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/config"
	"go.uber.org/zap"
)

// ChatService 对话业务逻辑
type ChatService struct {
	engineBaseURL string
}

// NewChatService 创建对话业务实例
func NewChatService() *ChatService {
	return &ChatService{
		engineBaseURL: config.Get().PythonEngine.BaseURL,
	}
}

// CreateSession 创建新的对话会话
func (s *ChatService) CreateSession(req *model.CreateSessionRequest) (*model.ChatSession, error) {
	// 检查道人是否存在
	var agent model.DaoAgent
	if err := dao.GetDB().First(&agent, req.AgentID).Error; err != nil {
		return nil, fmt.Errorf("道人(id=%d)不存在: %w", req.AgentID, err)
	}

	title := req.Title
	if title == "" {
		title = fmt.Sprintf("与 %s 的论道", agent.Name)
	}

	session := model.ChatSession{
		AgentID: req.AgentID,
		Title:   title,
	}

	if err := dao.GetDB().Create(&session).Error; err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	// 预加载道人信息
	session.Agent = agent

	zap.L().Info("[炼丹炉] 新的论道会话开启",
		zap.Uint("session_id", session.ID),
		zap.String("title", session.Title),
		zap.String("agent", agent.Name))

	return &session, nil
}

// ListSessions 获取会话列表，支持分页
func (s *ChatService) ListSessions(page, pageSize int) ([]model.ChatSession, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var sessions []model.ChatSession
	var total int64

	if err := dao.GetDB().Model(&model.ChatSession{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询会话总数失败: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := dao.GetDB().
		Preload("Agent").
		Order("updated_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&sessions).Error; err != nil {
		return nil, 0, fmt.Errorf("查询会话列表失败: %w", err)
	}

	return sessions, total, nil
}

// GetSession 根据 ID 获取会话详情
func (s *ChatService) GetSession(id uint) (*model.ChatSession, error) {
	var session model.ChatSession
	if err := dao.GetDB().Preload("Agent").Preload("Messages").First(&session, id).Error; err != nil {
		return nil, fmt.Errorf("查询会话(id=%d)失败: %w", id, err)
	}
	return &session, nil
}

// GetMessages 获取会话的消息历史，支持分页（按时间正序排列）
func (s *ChatService) GetMessages(sessionID uint, page, pageSize int) ([]model.ChatMessage, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200 // 消息查询允许更大的分页
	}

	var messages []model.ChatMessage
	var total int64

	db := dao.GetDB().Model(&model.ChatMessage{}).Where("session_id = ?", sessionID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询消息总数失败: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := db.Order("created_at ASC").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		return nil, 0, fmt.Errorf("查询消息列表失败: %w", err)
	}

	return messages, total, nil
}

// SaveMessage 保存消息到数据库
func (s *ChatService) SaveMessage(sessionID uint, role, content string, sources model.JSONMap) (*model.ChatMessage, error) {
	msg := model.ChatMessage{
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		Sources:   sources,
	}

	if err := dao.GetDB().Create(&msg).Error; err != nil {
		return nil, fmt.Errorf("保存消息失败: %w", err)
	}

	// 更新会话的 updated_at
	dao.GetDB().Model(&model.ChatSession{}).Where("id = ?", sessionID).Update("updated_at", time.Now())

	return &msg, nil
}

// EngineError 语言引擎返回的非 200 错误，保留状态码供错误映射使用
type EngineError struct {
	Op         string // 操作描述
	StatusCode int    // HTTP 状态码
	Body       string // 引擎响应体
}

func (e *EngineError) Error() string {
	return fmt.Sprintf("%s返回错误: status=%d, body=%s", e.Op, e.StatusCode, e.Body)
}

// MapEngineError 将调用语言引擎的错误映射为可读中文描述（websocket-protocol.md 错误语义）
//   - 401/403: 模型凭证无效
//   - 网络超时/408/504: 引擎响应超时
//   - 5xx: 引擎服务异常
//   - 连接失败: 无法连接语言引擎
func MapEngineError(err error) string {
	var engineErr *EngineError
	if errors.As(err, &engineErr) {
		switch {
		case engineErr.StatusCode == http.StatusUnauthorized || engineErr.StatusCode == http.StatusForbidden:
			return "模型凭证无效，请检查模型管理中的 API Key"
		case engineErr.StatusCode == http.StatusRequestTimeout || engineErr.StatusCode == http.StatusGatewayTimeout:
			return "语言引擎响应超时，请稍后重试"
		case engineErr.StatusCode >= 500:
			return "语言引擎服务异常，请稍后重试"
		default:
			return fmt.Sprintf("语言引擎请求失败（状态码 %d）", engineErr.StatusCode)
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "语言引擎响应超时，请稍后重试"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "语言引擎响应超时，请稍后重试"
	}
	return "无法连接语言引擎，请检查服务是否启动"
}

// CallChatStream 调用 Python 语言引擎的流式对话接口（SSE），返回响应流
// messages 应已包含合成后的 system 消息（由 LanguagePatternService 提供）
// ctx 取消时上游 HTTP 请求随之中断（停止指令贯穿取消链）
// creds 为按请求传递的模型凭证；base_url/api_key 为空时 Python 回退自身环境变量（向后兼容）
// creds 为 nil 时使用环境变量中的默认模型
func (s *ChatService) CallChatStream(ctx context.Context, messages []map[string]string, creds *ModelCredentials) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/api/v1/chat/completions/stream", s.engineBaseURL)

	modelName := config.Get().LLM.DefaultModel
	if creds != nil && creds.Model != "" {
		modelName = creds.Model
	}

	reqBody := map[string]interface{}{
		"messages": messages,
		"model":    modelName,
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("构建流式对话请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用语言引擎流式对话接口失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &EngineError{Op: "语言引擎流式接口", StatusCode: resp.StatusCode, Body: string(body)}
	}

	return resp.Body, nil
}

// StreamChat 调用语言引擎流式对话并逐块回调，返回完整内容与取消标记
//   - 使用 bufio.Reader.ReadBytes('\n') 解析 SSE，无 64KB 行限制
//   - ctx 取消时返回已累积的部分内容与 canceled=true，err 为 nil
//   - 引擎错误映射为可读中文描述；SSE error 事件直接透传其消息
//
// onChunk 每个内容片段回调一次（通常为转发到 WebSocket）
func (s *ChatService) StreamChat(ctx context.Context, messages []map[string]string, creds *ModelCredentials, onChunk func(string)) (fullContent string, canceled bool, err error) {
	stream, err := s.CallChatStream(ctx, messages, creds)
	if err != nil {
		if ctx.Err() != nil {
			return "", true, nil
		}
		return "", false, errors.New(MapEngineError(err))
	}
	defer stream.Close()

	var full strings.Builder
	reader := bufio.NewReader(stream)
	for {
		// 行级读取，遇 \n 返回；流中断时返回已读片段与 io.EOF
		line, readErr := reader.ReadBytes('\n')
		if ctx.Err() != nil {
			// 收到停止指令：返回已累积内容
			return full.String(), true, nil
		}

		text := strings.TrimRight(string(line), "\r\n")
		if strings.HasPrefix(text, "data: ") {
			data := strings.TrimPrefix(text, "data: ")
			if data == "[DONE]" {
				return full.String(), false, nil
			}

			// 解析 SSE JSON（语言引擎直接输出 {"content": "..."} 格式，错误时为 {"error": "..."}）
			var chunk struct {
				Content string `json:"content"`
				Error   string `json:"error"`
			}
			if jerr := json.Unmarshal([]byte(data), &chunk); jerr == nil {
				if chunk.Error != "" {
					return full.String(), false, errors.New(chunk.Error)
				}
				if chunk.Content != "" {
					full.WriteString(chunk.Content)
					if onChunk != nil {
						onChunk(chunk.Content)
					}
				}
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				// 流正常结束（未收到 [DONE] 也按完成处理）
				return full.String(), false, nil
			}
			if ctx.Err() != nil || errors.Is(readErr, context.Canceled) {
				return full.String(), true, nil
			}
			var netErr net.Error
			if errors.As(readErr, &netErr) && netErr.Timeout() {
				return full.String(), false, errors.New("语言引擎响应超时，请稍后重试")
			}
			zap.L().Warn("[炼丹炉] SSE 流读取异常", zap.Error(readErr))
			return full.String(), false, nil
		}
	}
}

// GetSessionAgentInfo 获取会话关联的道人信息（ID 与模型名）
// 用于 WebSocket 聊天时构建对话请求
func (s *ChatService) GetSessionAgentInfo(sessionID uint) (agentID uint, modelName string, err error) {
	// 查询会话
	var session model.ChatSession
	if err := dao.GetDB().First(&session, sessionID).Error; err != nil {
		return 0, "", fmt.Errorf("会话(id=%d)不存在: %w", sessionID, err)
	}

	// 查询道人信息
	var agent model.DaoAgent
	if err := dao.GetDB().First(&agent, session.AgentID).Error; err != nil {
		return 0, "", fmt.Errorf("道人(id=%d)不存在: %w", session.AgentID, err)
	}

	modelName = agent.ModelName
	if modelName == "" {
		modelName = config.Get().LLM.DefaultModel
	}

	return session.AgentID, modelName, nil
}

// UpdateSessionTitle 更新会话标题
func (s *ChatService) UpdateSessionTitle(sessionID uint, title string) error {
	if err := dao.GetDB().Model(&model.ChatSession{}).Where("id = ?", sessionID).
		Update("title", title).Error; err != nil {
		return fmt.Errorf("更新会话标题失败: %w", err)
	}
	return nil
}

// DeleteSession 删除会话，级联删除所有消息
func (s *ChatService) DeleteSession(id uint) error {
	var session model.ChatSession
	if err := dao.GetDB().First(&session, id).Error; err != nil {
		return fmt.Errorf("会话(id=%d)不存在: %w", id, err)
	}

	if err := dao.GetDB().Delete(&session).Error; err != nil {
		return fmt.Errorf("删除会话失败: %w", err)
	}

	zap.L().Info("[炼丹炉] 论道会话已结束", zap.Uint("session_id", id))
	return nil
}
