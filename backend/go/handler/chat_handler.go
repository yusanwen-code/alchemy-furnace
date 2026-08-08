// Package handler 对话管理 HTTP 处理器
// 处理会话管理和 WebSocket 流式对话
// 对应 API: /api/v1/chat/sessions, /api/v1/chat/ws/:session_id
// WebSocket 聊天流程（v2 协议，读写分离）:
//  1. 读泵 goroutine 常驻接收客户端消息（流式期间也能收 stop）
//  2. 每轮对话创建 context.WithCancel，stop/连接关闭即取消上游 LLM 流
//  3. 加载/合成道人的语言模式（系统提示词），解析模型凭证按请求透传
//  4. 调用 Python 语言引擎 /chat/completions/stream (SSE) 并流式转发
//  5. 心跳：30s ping / 60s 读超时；所有写操作经写锁串行化
//  6. 保存消息到数据库（停止时保存已生成的部分内容）
package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/pkg/response"
	"github.com/alchemy-furnace/server/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocket 升级器配置
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有来源的跨域 WebSocket 连接
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ChatHandler 对话 HTTP 处理器
type ChatHandler struct {
	service  *service.ChatService
	patterns *service.LanguagePatternService
	models   *service.ModelService
}

// NewChatHandler 创建对话处理器
func NewChatHandler() *ChatHandler {
	return &ChatHandler{
		service:  service.NewChatService(),
		patterns: service.NewLanguagePatternService(),
		models:   service.NewModelService(),
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

// wsClientMessage 客户端 → 服务端消息（v2 协议）
// { "content": "用户问题" } 或 { "type": "stop" }
type wsClientMessage struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// wsConn 封装 WebSocket 连接：所有写操作经写锁串行化（gorilla 单写者约束）
type wsConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

// send 发送服务端 → 客户端消息（带写超时保护）
func (w *wsConn) send(msgType, content string) {
	msg := map[string]string{
		"type":    msgType,
		"content": content,
	}

	w.writeMu.Lock()
	defer w.writeMu.Unlock()

	w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	defer w.conn.SetWriteDeadline(time.Time{})

	if err := w.conn.WriteJSON(msg); err != nil {
		zap.L().Warn("[炼丹炉] WebSocket 消息发送失败", zap.Error(err))
	}
}

// ping 发送协议层 ping 控制帧（浏览器自动回 pong）
func (w *wsConn) ping() error {
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	defer w.conn.SetWriteDeadline(time.Time{})
	return w.conn.WriteMessage(websocket.PingMessage, nil)
}

// WebSocketChat WebSocket 流式对话（v2 协议：读写分离 + stop + 心跳）
// GET /api/v1/chat/ws/:session_id
// 协议: WebSocket
//
// 消息格式（JSON）:
//
//	客户端 -> 服务端: { "content": "用户问题" }
//	客户端 -> 服务端: { "type": "stop" }                  （仅流式生成期间有效）
//	服务端 -> 客户端: { "type": "chunk", "content": "回答片段" }
//	服务端 -> 客户端: { "type": "done", "content": "" }
//	服务端 -> 客户端: { "type": "stopped", "content": "" }
//	服务端 -> 客户端: { "type": "error", "content": "可读中文错误" }
func (h *ChatHandler) WebSocketChat(c *gin.Context) {
	sessionID, err := strconv.ParseUint(c.Param("session_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "会话ID格式不正确")
		return
	}

	// 升级 HTTP 连接为 WebSocket
	rawConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.L().Error("[炼丹炉] WebSocket 升级失败", zap.Error(err))
		return
	}
	defer rawConn.Close()
	ws := &wsConn{conn: rawConn}

	zap.L().Info("[炼丹炉] 仙缘已到，WebSocket 连接已建立",
		zap.Uint64("session_id", sessionID))

	// ---------- 心跳：60s 读超时（pong 续期），30s ping ----------
	rawConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	rawConn.SetPongHandler(func(string) error {
		rawConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	pingDone := make(chan struct{})
	defer close(pingDone)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-pingDone:
				return
			case <-ticker.C:
				if err := ws.ping(); err != nil {
					zap.L().Warn("[炼丹炉] WebSocket ping 失败", zap.Error(err))
					return
				}
			}
		}
	}()

	// 获取会话关联的道人信息
	agentID, modelName, err := h.service.GetSessionAgentInfo(uint(sessionID))
	if err != nil {
		ws.send("error", "获取会话信息失败: "+err.Error())
		return
	}

	// 加载/合成道人的语言模式（系统提示词）
	pattern, err := h.patterns.GetOrBuildPattern(agentID)
	if err != nil {
		zap.L().Error("[炼丹炉] 语言模式合成失败", zap.Error(err))
		ws.send("error", "化丹为性失败: "+err.Error())
		return
	}

	zap.L().Info("[炼丹炉] 论道准备就绪",
		zap.Uint64("session_id", sessionID),
		zap.Uint("agent_id", agentID),
		zap.String("model", modelName),
		zap.String("fingerprint", pattern.SourceFingerprint))

	// ---------- 当轮生成的取消函数（stop / 连接关闭时调用） ----------
	var mu sync.Mutex
	var currentCancel context.CancelFunc
	cancelCurrent := func() {
		mu.Lock()
		defer mu.Unlock()
		if currentCancel != nil {
			currentCancel()
		}
	}
	setCurrent := func(cancel context.CancelFunc) {
		mu.Lock()
		defer mu.Unlock()
		currentCancel = cancel
	}
	clearCurrent := func() {
		mu.Lock()
		defer mu.Unlock()
		currentCancel = nil
	}

	// ---------- 读泵：常驻读取客户端消息，流式期间也能收 stop ----------
	msgCh := make(chan string)
	readErrCh := make(chan error, 1)
	go func() {
		for {
			var m wsClientMessage
			if err := rawConn.ReadJSON(&m); err != nil {
				cancelCurrent() // 连接关闭/异常也取消当轮生成
				readErrCh <- err
				return
			}
			// 收到任何客户端帧均视为活跃，续期读超时
			rawConn.SetReadDeadline(time.Now().Add(60 * time.Second))
			if m.Type == "stop" {
				cancelCurrent() // 非生成期间 currentCancel 为 nil，忽略
				continue
			}
			msgCh <- m.Content
		}
	}()

	// ---------- 主流程：逐条处理用户消息 ----------
	for {
		select {
		case readErr := <-readErrCh:
			if websocket.IsUnexpectedCloseError(readErr, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				zap.L().Warn("[炼丹炉] WebSocket 连接异常断开", zap.Error(readErr))
			}
			zap.L().Info("[炼丹炉] WebSocket 论道结束", zap.Uint64("session_id", sessionID))
			return

		case content := <-msgCh:
			// 校验消息内容
			if strings.TrimSpace(content) == "" {
				ws.send("error", "消息内容不能为空")
				continue
			}

			// 1. 保存用户消息到数据库
			if _, err := h.service.SaveMessage(uint(sessionID), "user", content, nil); err != nil {
				zap.L().Error("[炼丹炉] 保存用户消息失败", zap.Error(err))
				ws.send("error", "保存消息失败")
				continue
			}

			zap.L().Info("[炼丹炉] 收到论道问题", zap.Uint64("session_id", sessionID), zap.String("content", content))

			// 2. 获取历史消息（最近 20 条，用于上下文）
			recentMessages, _, err := h.service.GetMessages(uint(sessionID), 1, 20)
			if err != nil {
				zap.L().Error("[炼丹炉] 获取历史消息失败", zap.Error(err))
				ws.send("error", "获取历史消息失败")
				continue
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

			// 3. 解析模型凭证（每轮解析，模型停用/换钥即时生效）
			creds, err := h.models.ResolveCredentials(modelName)
			if err != nil {
				zap.L().Warn("[炼丹炉] 模型凭证解析失败",
					zap.String("model", modelName), zap.Error(err))
				ws.send("error", err.Error()) // 已是可读中文（模型停用/解密失败）
				continue
			}

			// 4. 创建当轮可取消上下文，流式调用语言引擎并转发
			ctx, cancel := context.WithCancel(context.Background())
			setCurrent(cancel)
			fullText, canceled, streamErr := h.service.StreamChat(ctx, messages, creds, func(chunk string) {
				ws.send("chunk", chunk)
			})
			clearCurrent()
			cancel()

			switch {
			case canceled:
				// 收到停止指令：保存已生成的部分内容
				if fullText != "" {
					if _, err := h.service.SaveMessage(uint(sessionID), "assistant", fullText, nil); err != nil {
						zap.L().Error("[炼丹炉] 保存停止时的部分回复失败", zap.Error(err))
					}
				}
				ws.send("stopped", "")
				zap.L().Info("[炼丹炉] 论道已被叫停",
					zap.Uint64("session_id", sessionID), zap.Int("partial_length", len(fullText)))

			case streamErr != nil:
				// 结构化中文错误（模型凭证无效 / 引擎超时 / 引擎异常等）
				ws.send("error", streamErr.Error())

			default:
				// 正常完成
				ws.send("done", "")
				if fullText != "" {
					if _, err := h.service.SaveMessage(uint(sessionID), "assistant", fullText, nil); err != nil {
						zap.L().Error("[炼丹炉] 保存助手回复失败", zap.Error(err))
					}
				}
				zap.L().Info("[炼丹炉] 论道一轮完成",
					zap.Uint64("session_id", sessionID), zap.Int("response_length", len(fullText)))
			}
		}
	}
}

// WSMessage WebSocket 消息结构（用于文档说明）
type WSMessage struct {
	Type    string `json:"type"`    // chunk / done / stopped / error
	Content string `json:"content"` // 消息内容
}
