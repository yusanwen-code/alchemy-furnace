// Package service 流式对话取消语义单元测试（T021）
// 使用 httptest 伪造慢速 Python SSE 引擎，验证 context 取消贯穿取消链
package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alchemy-furnace/server/pkg/config"
)

// newSlowSSEEngine 返回一个慢速 SSE 引擎：先发一个内容块，然后一直挂起
// 直到客户端（ctx 取消）断开连接
func newSlowSSEEngine(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"content\": \"你\"}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		// 模拟长时间生成：直到客户端断开才返回
		select {
		case <-time.After(30 * time.Second):
			fmt.Fprint(w, "data: [DONE]\n\n")
		case <-r.Context().Done():
		}
	}))
}

// pointEngineAt 将全局配置的 Python 引擎地址指向 mock 服务器
func pointEngineAt(t *testing.T, engineURL string) {
	t.Helper()
	t.Setenv("AF_PYTHON_ENGINE_BASE_URL", engineURL)
	if _, err := config.Load(); err != nil {
		t.Fatalf("加载测试配置失败: %v", err)
	}
	if got := config.Get().PythonEngine.BaseURL; got != engineURL {
		t.Fatalf("配置未指向 mock 引擎: got %s, want %s", got, engineURL)
	}
}

// TestStreamChatCancellation 取消后应在 1s 内返回已累积的部分内容与 canceled 标记
func TestStreamChatCancellation(t *testing.T) {
	engine := newSlowSSEEngine(t)
	defer engine.Close()
	pointEngineAt(t, engine.URL)

	svc := NewChatService()
	ctx, cancel := context.WithCancel(context.Background())

	var chunks []string
	start := time.Now()
	// 收到首个内容块后触发取消（模拟前端 stop）
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	full, canceled, err := svc.StreamChat(ctx, []map[string]string{
		{"role": "user", "content": "道友请讲"},
	}, &ModelCredentials{Model: "gpt-4o"}, func(chunk string) {
		chunks = append(chunks, chunk)
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("取消场景不应返回错误: %v", err)
	}
	if !canceled {
		t.Error("应返回 canceled=true")
	}
	if elapsed > 1*time.Second {
		t.Errorf("取消后应在 1s 内返回，实际耗时 %v", elapsed)
	}
	if full != "你" {
		t.Errorf("部分内容不符: got %q, want %q", full, "你")
	}
	if len(chunks) != 1 || chunks[0] != "你" {
		t.Errorf("回调内容块不符: %v", chunks)
	}
}

// TestStreamChatNormalCompletion 正常流式完成：聚合全部内容块并以 done 结束
func TestStreamChatNormalCompletion(t *testing.T) {
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, chunk := range []string{"道", "可", "道"} {
			fmt.Fprintf(w, "data: {\"content\": %q}\n\n", chunk)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer engine.Close()
	pointEngineAt(t, engine.URL)

	svc := NewChatService()
	var chunks []string
	full, canceled, err := svc.StreamChat(context.Background(), []map[string]string{
		{"role": "user", "content": "hi"},
	}, &ModelCredentials{Model: "gpt-4o"}, func(chunk string) {
		chunks = append(chunks, chunk)
	})

	if err != nil {
		t.Fatalf("正常流程不应返回错误: %v", err)
	}
	if canceled {
		t.Error("正常完成不应标记 canceled")
	}
	if full != "道可道" {
		t.Errorf("完整内容不符: got %q", full)
	}
	if len(chunks) != 3 {
		t.Errorf("内容块数量不符: %v", chunks)
	}
}

// TestStreamChatEngineAuthError 引擎返回 401 时应映射为凭证无效的中文错误
func TestStreamChatEngineAuthError(t *testing.T) {
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"invalid api key"}`, http.StatusUnauthorized)
	}))
	defer engine.Close()
	pointEngineAt(t, engine.URL)

	svc := NewChatService()
	_, _, err := svc.StreamChat(context.Background(), []map[string]string{
		{"role": "user", "content": "hi"},
	}, &ModelCredentials{Model: "gpt-4o"}, nil)
	if err == nil {
		t.Fatal("引擎 401 应返回错误")
	}
	if got := err.Error(); got != "模型凭证无效，请检查模型管理中的 API Key" {
		t.Errorf("错误映射不符: %q", got)
	}
}

// TestStreamChatSSEErrorEvent SSE error 事件应透传其消息
func TestStreamChatSSEErrorEvent(t *testing.T) {
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"content\": \"你\"}\n\n")
		fmt.Fprint(w, "data: {\"error\": \"上游模型限流，请稍后重试\"}\n\n")
	}))
	defer engine.Close()
	pointEngineAt(t, engine.URL)

	svc := NewChatService()
	full, _, err := svc.StreamChat(context.Background(), []map[string]string{
		{"role": "user", "content": "hi"},
	}, &ModelCredentials{Model: "gpt-4o"}, nil)
	if err == nil || err.Error() != "上游模型限流，请稍后重试" {
		t.Fatalf("SSE error 事件应透传: %v", err)
	}
	if full != "你" {
		t.Errorf("已生成内容应保留: %q", full)
	}
}

// TestStreamChatLongLine 超过 64KB 的单行 SSE 数据应正常解析（ReadBytes 无 Scanner 行限制）
func TestStreamChatLongLine(t *testing.T) {
	long := make([]byte, 0, 70*1024)
	for i := 0; i < 70*1024; i++ {
		long = append(long, 'a')
	}
	engine := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"content\": \"%s\"}\n\n", string(long))
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer engine.Close()
	pointEngineAt(t, engine.URL)

	svc := NewChatService()
	full, _, err := svc.StreamChat(context.Background(), []map[string]string{
		{"role": "user", "content": "hi"},
	}, &ModelCredentials{Model: "gpt-4o"}, nil)
	if err != nil {
		t.Fatalf("长行解析不应报错: %v", err)
	}
	if full != string(long) {
		t.Errorf("长行内容不符: got %d bytes, want %d bytes", len(full), len(long))
	}
}
