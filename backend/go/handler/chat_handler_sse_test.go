// SSE 流式对话 handler 测试
// 覆盖：事件序列格式（event:/data: 行、双换行分隔）、chunk→done 完整流、
// 中文错误事件、客户端中断取消链（部分内容保存）
package handler

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alchemy-furnace/server/model"
	"github.com/alchemy-furnace/server/service"
	"github.com/gin-gonic/gin"
)

// ---------- mocks ----------

type savedMessage struct {
	role    string
	content string
}

type mockChatFlow struct {
	agentID   uint
	modelName string
	agentErr  error
	messages  []model.ChatMessage

	mu    sync.Mutex
	saved []savedMessage

	// streamFn 模拟 StreamChat；started 非空时在调用后关闭（取消测试用）
	streamFn func(ctx context.Context, onChunk func(string)) (string, bool, error)
	started  chan struct{}
}

func (m *mockChatFlow) GetSessionAgentInfo(sessionID uint) (uint, string, error) {
	return m.agentID, m.modelName, m.agentErr
}

func (m *mockChatFlow) GetMessages(sessionID uint, page, pageSize int) ([]model.ChatMessage, int64, error) {
	return m.messages, int64(len(m.messages)), nil
}

func (m *mockChatFlow) SaveMessage(sessionID uint, role, content string, sources model.JSONMap) (*model.ChatMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saved = append(m.saved, savedMessage{role: role, content: content})
	return &model.ChatMessage{Role: role, Content: content}, nil
}

func (m *mockChatFlow) StreamChat(ctx context.Context, messages []map[string]string, creds *service.ModelCredentials, onChunk func(string)) (string, bool, error) {
	if m.started != nil {
		close(m.started)
	}
	return m.streamFn(ctx, onChunk)
}

func (m *mockChatFlow) savedMessages() []savedMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]savedMessage(nil), m.saved...)
}

type mockPattern struct{}

func (mockPattern) GetOrBuildPattern(agentID uint) (*model.LanguagePattern, error) {
	return &model.LanguagePattern{SystemPrompt: "你是测试道人"}, nil
}

type mockCreds struct{ err error }

func (m mockCreds) ResolveCredentials(name string) (*service.ModelCredentials, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &service.ModelCredentials{Model: name, BaseURL: "http://engine", APIKey: "sk-test"}, nil
}

// ---------- 辅助 ----------

func setupSSEContext(flow *mockChatFlow, creds credentialResolver, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("POST", "/api/v1/chat/sse/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "session_id", Value: "1"}}
	return c, recorder
}

func newTestHandler(flow *mockChatFlow, creds credentialResolver) *ChatHandler {
	return &ChatHandler{flow: flow, pattern: mockPattern{}, creds: creds}
}

// ---------- 测试 ----------

// TestSSEChatEventFormat 事件序列格式：chunk→done 完整流，SSE 帧格式与响应头正确
func TestSSEChatEventFormat(t *testing.T) {
	flow := &mockChatFlow{
		agentID:   1,
		modelName: "test-model",
		streamFn: func(ctx context.Context, onChunk func(string)) (string, bool, error) {
			onChunk("道可道")
			onChunk("，非常道")
			return "道可道，非常道", false, nil
		},
	}
	h := newTestHandler(flow, mockCreds{})
	c, recorder := setupSSEContext(flow, mockCreds{}, `{"content":"何为道"}`)

	h.SSEChat(c)

	if ct := recorder.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	if h := recorder.Header().Get("X-Accel-Buffering"); h != "no" {
		t.Fatalf("X-Accel-Buffering = %q, want no", h)
	}

	want := "event: chunk\ndata: {\"content\":\"道可道\"}\n\n" +
		"event: chunk\ndata: {\"content\":\"，非常道\"}\n\n" +
		"event: done\ndata: {}\n\n"
	if got := recorder.Body.String(); got != want {
		t.Fatalf("SSE body mismatch:\ngot:\n%q\nwant:\n%q", got, want)
	}

	// 用户消息 + 完整 assistant 回复入库
	saved := flow.savedMessages()
	if len(saved) != 2 || saved[0].role != "user" || saved[0].content != "何为道" {
		t.Fatalf("saved[0] = %+v, want user 何为道", saved)
	}
	if saved[1].role != "assistant" || saved[1].content != "道可道，非常道" {
		t.Fatalf("saved[1] = %+v, want assistant 完整回复", saved[1])
	}
}

// TestSSEChatErrorEvent 凭证解析失败输出中文 error 事件
func TestSSEChatErrorEvent(t *testing.T) {
	flow := &mockChatFlow{agentID: 1, modelName: "disabled-model"}
	h := newTestHandler(flow, mockCreds{err: errors.New("该道人使用的模型已停用，请更换模型")})
	c, recorder := setupSSEContext(flow, mockCreds{}, `{"content":"在吗"}`)

	h.SSEChat(c)

	body := recorder.Body.String()
	if !strings.Contains(body, "event: error\n") {
		t.Fatalf("expected error event, got: %q", body)
	}
	if !strings.Contains(body, "该道人使用的模型已停用") {
		t.Fatalf("expected Chinese error message, got: %q", body)
	}
}

// TestSSEChatStreamError 流中途错误：已发 chunk 后输出 error 事件，不落库
func TestSSEChatStreamError(t *testing.T) {
	flow := &mockChatFlow{
		agentID:   1,
		modelName: "test-model",
		streamFn: func(ctx context.Context, onChunk func(string)) (string, bool, error) {
			onChunk("半句")
			return "半句", false, errors.New("语言引擎响应超时，请稍后重试")
		},
	}
	h := newTestHandler(flow, mockCreds{})
	c, recorder := setupSSEContext(flow, mockCreds{}, `{"content":"继续"}`)

	h.SSEChat(c)

	body := recorder.Body.String()
	if !strings.Contains(body, "event: chunk\n") || !strings.Contains(body, "event: error\n") {
		t.Fatalf("expected chunk + error events, got: %q", body)
	}
	if strings.Contains(body, "event: done") {
		t.Fatalf("unexpected done event on error path: %q", body)
	}
	// 仅用户消息入库（错误路径不保存 assistant）
	for _, s := range flow.savedMessages() {
		if s.role == "assistant" {
			t.Fatalf("error path should not save assistant message, got: %+v", s)
		}
	}
}

// TestSSEChatCancel 客户端中断：取消链贯穿，部分内容保存为 assistant 消息，不再写事件
func TestSSEChatCancel(t *testing.T) {
	started := make(chan struct{})
	flow := &mockChatFlow{
		agentID:   1,
		modelName: "test-model",
		started:   started,
		streamFn: func(ctx context.Context, onChunk func(string)) (string, bool, error) {
			onChunk("已生成的部分")
			<-ctx.Done() // 阻塞直至客户端中断
			return "已生成的部分", true, nil
		},
	}
	h := newTestHandler(flow, mockCreds{})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("POST", "/api/v1/chat/sse/1", strings.NewReader(`{"content":"长文生成"}`)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = gin.Params{{Key: "session_id", Value: "1"}}

	done := make(chan struct{})
	go func() {
		h.SSEChat(c)
		close(done)
	}()

	<-started      // 等待流式调用开始（首个 chunk 已转发）
	cancel()       // 模拟客户端 abort

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel path did not return within 2s")
	}

	// 部分内容已入库
	var assistant *savedMessage
	for _, s := range flow.savedMessages() {
		if s.role == "assistant" {
			msg := s
			assistant = &msg
		}
	}
	if assistant == nil || assistant.content != "已生成的部分" {
		t.Fatalf("partial assistant message not saved, saved: %+v", flow.savedMessages())
	}

	// 连接已断：不应出现 done 事件
	if strings.Contains(recorder.Body.String(), "event: done") {
		t.Fatalf("unexpected done event after cancel: %q", recorder.Body.String())
	}
}
