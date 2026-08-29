// splash_test.go - L3 启动屏三态单测: pending/ready/err
// 覆盖三态返回 HTML 结构(自刷新/JS 跳转/错误页含重试)
package desktop

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSplashStates(t *testing.T) {
	// pending → 启动页(火焰动画,不再整页刷新)
	h := newSplashHandler(
		func() string { return "http://x/?token=t&platform=darwin" },
		func() (bool, error) { return false, nil },
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "点燃") {
		t.Fatalf("pending 态应含'点燃'文案:\n%s", body)
	}

	// ready → 跳转页(JS location.replace,避免 WKWebView 跨 scheme 302)
	h = newSplashHandler(
		func() string { return "http://x/?token=t&platform=darwin" },
		func() (bool, error) { return true, nil },
	)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), "window.location.replace") {
		t.Fatalf("ready 态应返回跳转页:\n%s", rec.Body.String())
	}

	// err → 错误页(包含错误原因 + reload 重试按钮)
	h = newSplashHandler(
		nil,
		func() (bool, error) { return false, errors.New("引擎挂了") },
	)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	body = rec.Body.String()
	if !strings.Contains(body, "引擎挂了") {
		t.Fatalf("err 态应含错误原因:\n%s", body)
	}
	if !strings.Contains(body, "location.reload") {
		t.Fatalf("err 态应含重试按钮:\n%s", body)
	}
}

// TestResolveSplashStatus — 启动状态模型表格测试// pending: readiness false,不调用 target
// ready:   readiness true,返回 target() 结果
// error:   readiness 返回 err,Message 透传 err.Error()
// nil readiness: 直接视为 ready(调用方不提供探针)
// ready 但 target nil: 拒绝 nil panic,返回可读错误
func TestResolveSplashStatus(t *testing.T) {
	cases := []struct {
		name      string
		target    func() string
		readiness func() (bool, error)
		want      splashStatus
	}{
		{
			name:      "pending",
			target:    func() string { return "http://127.0.0.1:1234/?token=x" },
			readiness: func() (bool, error) { return false, nil },
			want:      splashStatus{State: splashPending},
		},
		{
			name:      "ready",
			target:    func() string { return "http://127.0.0.1:1234/?token=x" },
			readiness: func() (bool, error) { return true, nil },
			want:      splashStatus{State: splashReady, Target: "http://127.0.0.1:1234/?token=x"},
		},
		{
			name:      "readiness error",
			readiness: func() (bool, error) { return false, errors.New("引擎启动失败") },
			want:      splashStatus{State: splashError, Message: "引擎启动失败"},
		},
		{
			name:      "nil readiness",
			target:    func() string { return "http://127.0.0.1:1234/" },
			readiness: nil,
			want:      splashStatus{State: splashReady, Target: "http://127.0.0.1:1234/"},
		},
		{
			name:      "ready but nil target",
			readiness: func() (bool, error) { return true, nil },
			want:      splashStatus{State: splashError, Message: "启动地址不可用"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveSplashStatus(tc.target, tc.readiness)
			if got != tc.want {
				t.Fatalf("resolveSplashStatus() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestSplashStatusEndpoint — 状态探针 HTTP 契约
// GET /__alchemy_boot_status → application/json + no-store;ready 返回非空 Target
func TestSplashStatusEndpoint(t *testing.T) {
	h := newSplashHandler(
		func() string { return "http://127.0.0.1:1234/?token=x" },
		func() (bool, error) { return true, nil },
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", splashStatusPath, nil))
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	var status splashStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("JSON decode: %v", err)
	}
	if status.State != splashReady || status.Target == "" {
		t.Fatalf("ready 态应返回非空 Target, got %+v", status)
	}
}

// TestSplashStatusEscaping — 错误文本必须 JSON 编码,往返不变
// 曾直接把 err.Error() 拼进 HTML——注入与路径泄露风险;JSON 解码后必须与输入相同
func TestSplashStatusEscaping(t *testing.T) {
	msg := `</script><script>alert(1)</script>`
	h := newSplashHandler(nil, func() (bool, error) { return false, errors.New(msg) })
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", splashStatusPath, nil))
	var status splashStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("JSON decode: %v", err)
	}
	if status.Message != msg {
		t.Fatalf("Message 往返 = %q, want %q", status.Message, msg)
	}
}
