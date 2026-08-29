// splash.go — L3 启动屏三态 handler
// pending: 火焰动画 + 1s 自刷新(等 engineproc + DB 就绪)
// ready:   JS window.location.replace 跳到真 origin(沿用 newRedirectHandler 的坑注释: WKWebView 跨 scheme 302 不跟随)
// err:     错误页含 reload 重试按钮
// 自包含内联 CSS,无外部资源(防 wails asset server 还没起就 404)
package desktop

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// splashState — 启动屏三态
type splashState string

const (
	splashPending splashState = "pending"
	splashReady   splashState = "ready"
	splashError   splashState = "error"
)

// splashStatus — 状态探针 JSON 契约(Task 2 起由 /__alchemy_boot_status 提供)
type splashStatus struct {
	State   splashState `json:"state"`
	Target  string      `json:"target,omitempty"`
	Message string      `json:"message,omitempty"`
}

// resolveSplashStatus — 启动状态模型
// - readiness nil 保持原语义:直接视为 ready
// - readiness 返回 err:error 态,Message 透传(供 JSON 编码,不进 HTML)
// - ready 但 target nil:返回可读错误,避免 nil panic
func resolveSplashStatus(target func() string, readiness func() (bool, error)) splashStatus {
	ready := readiness == nil
	if readiness != nil {
		var err error
		ready, err = readiness()
		if err != nil {
			return splashStatus{State: splashError, Message: err.Error()}
		}
	}
	if !ready {
		return splashStatus{State: splashPending}
	}
	if target == nil {
		return splashStatus{State: splashError, Message: "启动地址不可用"}
	}
	return splashStatus{State: splashReady, Target: target()}
}

const splashCSS = `body{margin:0;height:100vh;display:flex;flex-direction:column;align-items:center;justify-content:center;background:#09090b;color:#d4d4d8;font-family:ui-sans-serif,system-ui}
.flame{font-size:56px;animation:f 1.2s ease-in-out infinite}
@keyframes f{0%,100%{transform:scale(1);opacity:.85}50%{transform:scale(1.12);opacity:1}}
.msg{margin-top:18px;font-size:14px;opacity:.75;letter-spacing:.08em}
.err{color:#f87171;font-size:13px;max-width:70%;text-align:center}
.retry{margin-top:16px;padding:8px 22px;border:1px solid #52525b;border-radius:8px;color:#d4d4d8;cursor:pointer;background:transparent}
.retry:hover{background:#27272a}`

// splashStatusPath — 同源状态探针(页面 JS 轮询,代替整页 meta refresh)
const splashStatusPath = "/__alchemy_boot_status"

// writeSplashStatus — JSON 探针响应
func writeSplashStatus(w http.ResponseWriter, status splashStatus) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_ = json.NewEncoder(w).Encode(status)
}

// writeSplashDocument — 启动文档(Task 2:三态渲染,去掉整页 refresh;Task 3 起为固定骨架)
func writeSplashDocument(w http.ResponseWriter, target func() string, readiness func() (bool, error)) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	status := resolveSplashStatus(target, readiness)
	switch status.State {
	case splashError:
		fmt.Fprintf(w, `<!doctype html><meta charset=utf-8><title>炼丹炉</title><style>%s</style>
<body><div class=flame>🔥</div><p class=msg>丹炉点火失败 · Failed to ignite</p>
<p class=err>%s</p><button class=retry onclick="location.reload()">重试 · Retry</button>`, splashCSS, status.Message)
	case splashReady:
		// JS replace 而非 302: WKWebView 跨 scheme 302 (wails:// → http://) 经常不跟随 → 白屏
		fmt.Fprintf(w, `<!doctype html><meta charset=utf-8><title>炼丹炉</title><body><script>window.location.replace(%q);</script></body>`, status.Target)
	default:
		fmt.Fprintf(w, `<!doctype html><meta charset=utf-8><title>炼丹炉</title><style>%s</style>
<body><div class=flame>🔥</div><p class=msg>正在点燃丹炉 · Lighting the furnace…</p>`, splashCSS)
	}
}

// newSplashHandler 返回三态 http.Handler
// - readiness 由调用方提供(查询 engineproc.Start + DB 是否就绪,可能为 nil 表示直接走 ready 路径)
// - target 提供 ready 时的跳转 URL
// 路由:GET /__alchemy_boot_status -> JSON 探针;其他路径 -> 启动文档(只加载一次,页面 JS 轮询)
func newSplashHandler(target func() string, readiness func() (bool, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == splashStatusPath {
			writeSplashStatus(w, resolveSplashStatus(target, readiness))
			return
		}
		writeSplashDocument(w, target, readiness)
	})
}
