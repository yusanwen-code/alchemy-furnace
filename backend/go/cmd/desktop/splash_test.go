// splash_test.go - L3 启动屏三态单测: pending/ready/err
// 覆盖三态返回 HTML 结构(自刷新/JS 跳转/错误页含重试)
package desktop

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSplashStates(t *testing.T) {
	// pending → 启动页(自刷新,火焰动画)
	h := newSplashHandler(
		func() string { return "http://x/?token=t&platform=darwin" },
		func() (bool, error) { return false, nil },
	)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "refresh") {
		t.Fatalf("pending 态应返回自刷新启动页(缺 refresh meta):\n%s", body)
	}
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

// TestResolveSplashStatus — 启动状态模型表格测试
// pending: readiness false,不调用 target
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
