// splash.go — L3 启动屏三态 handler
// pending: 火焰动画 + 1s 自刷新(等 engineproc + DB 就绪)
// ready:   JS window.location.replace 跳到真 origin(沿用 newRedirectHandler 的坑注释: WKWebView 跨 scheme 302 不跟随)
// err:     错误页含 reload 重试按钮
// 自包含内联 CSS,无外部资源(防 wails asset server 还没起就 404)
package desktop

import (
	"encoding/json"
	"io"
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

// splashHTML — 固定启动文档"器物点火":宣纸 × 青铜鼎 × 三炉膛。
// 完全自包含(内联 CSS/SVG/JS),不引用 /ding.png、CDN 或网络字体;
// 状态经 JS 轮询 /__alchemy_boot_status 驱动,ready 用 window.location.replace 跳转。
// 注意:Go 原始字符串内不能出现反引号,JS 字符串拼接不用模板字面量。
const splashHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>炼丹炉</title>
<style>
:root {
  --boot-paper: #f7f3ed;
  --boot-paper-deep: #efe8dc;
  --boot-ink: #1c1c1c;
  --boot-ink-muted: #6b655d;
  --boot-bronze: #716044;
  --boot-bronze-dark: #3d352b;
  --boot-gold: #c9a96e;
  --boot-cinnabar: #b54a3f;
  --boot-coal: #17120f;
  --boot-ember: #d85f2e;
  --boot-flame-core: #f3b55b;
  --boot-serif: "Songti SC", "STSong", "SimSun", ui-serif, serif;
  --boot-sans: -apple-system, BlinkMacSystemFont, "Segoe UI", "Microsoft YaHei", sans-serif;
}
* { box-sizing: border-box; }
html, body { margin: 0; min-height: 100%; }
body {
  min-height: 100vh;
  overflow: hidden;
  color: var(--boot-ink);
  background:
    radial-gradient(circle at 78% 54%, rgba(201,169,110,.10), transparent 34%),
    var(--boot-paper);
  font-family: var(--boot-sans);
}
.boot-shell {
  width: min(1180px, 100%);
  min-height: calc(100vh - 58px);
  margin: 0 auto;
  padding: clamp(32px, 7vw, 96px);
  display: grid;
  grid-template-columns: 42% 58%;
  align-items: center;
}
.boot-copy { position: relative; z-index: 2; }
.brand {
  margin: 0 0 72px;
  color: var(--boot-cinnabar);
  font: 700 13px/1 var(--boot-serif);
  letter-spacing: .42em;
}
h1 {
  margin: 0;
  font: 800 clamp(64px, 8vw, 118px)/.9 var(--boot-serif);
  letter-spacing: .14em;
}
.status { margin-top: 40px; width: min(340px, 100%); }
.status-primary { margin: 0; font-size: 16px; letter-spacing: .14em; }
.status-secondary { margin: 14px 0 0; color: var(--boot-ink-muted); font-size: 13px; }
.heat-line { position: relative; width: 220px; height: 1px; margin-top: 22px; background: rgba(61,53,43,.22); }
.heat-line span {
  position: absolute;
  top: -3px;
  left: 0;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--boot-cinnabar);
  animation: heat-line 2.4s ease-in-out infinite;
}
.furnace-stage { margin: 0; display: grid; place-items: center; transform: translate(7%, 4%); }
.furnace { display: block; width: min(52vw, 640px); height: auto; }
.furnace-aura { opacity: .20; animation: aura-breathe 3.8s ease-in-out infinite; }
.hearth { opacity: 0; animation: hearth-ignite 720ms ease-out forwards; }
.hearth-left { animation-delay: 0ms; }
.hearth-center { animation-delay: 120ms; }
.hearth-right { animation-delay: 240ms; }
.flame-outer, .flame-main, .flame-core { transform-box: fill-box; transform-origin: 50% 100%; }
.hearth-left .flame-outer, .hearth-left .flame-main, .hearth-left .flame-core { animation: flame-sway 1.8s ease-in-out infinite; }
.hearth-center .flame-outer, .hearth-center .flame-main, .hearth-center .flame-core { animation: flame-sway 1.55s ease-in-out -.4s infinite reverse; }
.hearth-right .flame-outer, .hearth-right .flame-main, .hearth-right .flame-core { animation: flame-sway 2.05s ease-in-out -.8s infinite; }
footer {
  position: fixed;
  left: clamp(32px, 7vw, 96px);
  bottom: 26px;
  color: var(--boot-ink-muted);
  font-size: 12px;
  letter-spacing: .12em;
}
.error-actions { margin-top: 22px; max-width: 420px; }
.copy-error {
  padding: 9px 15px;
  color: var(--boot-ink);
  background: transparent;
  border: 1px solid rgba(61,53,43,.35);
  border-radius: 2px;
  font: 13px/1 var(--boot-sans);
  cursor: pointer;
}
details { margin-top: 16px; color: var(--boot-ink-muted); font-size: 12px; }
.error-detail, .error-fallback {
  width: 100%;
  max-height: 120px;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  color: var(--boot-ink-muted);
  background: var(--boot-paper-deep);
  border: 0;
  padding: 12px;
}
body[data-state="error"] .flame-main,
body[data-state="error"] .flame-core { opacity: 0; animation: none; }
body[data-state="error"] .flame-outer { opacity: .3; animation: none; transform: scaleY(.34); }
body[data-state="error"] .furnace-aura,
body[data-state="error"] .heat-line span { opacity: .12; animation: none; }
@keyframes hearth-ignite { from { opacity: .16; } to { opacity: 1; } }
@keyframes flame-sway { 0%, 100% { transform: rotate(-1.6deg) scaleY(.97); } 50% { transform: rotate(1.8deg) scaleY(1.03); } }
@keyframes aura-breathe { 0%, 100% { opacity: .16; } 50% { opacity: .24; } }
@keyframes heat-line { 0% { transform: translateX(0); opacity: .35; } 50% { opacity: 1; } 100% { transform: translateX(213px); opacity: .35; } }
@media (max-width: 760px) {
  .boot-shell { grid-template-columns: 1fr; padding-top: 56px; }
  .brand { margin-bottom: 38px; }
  h1 { font-size: clamp(54px, 15vw, 72px); }
  .furnace-stage { transform: translate(8%, -3%); }
  .furnace { width: min(88vw, 420px); }
}
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation: none !important; }
  .hearth { opacity: 1; }
  .heat-line span { transform: translateX(106px); }
}
</style>
</head>
<body data-state="pending">
  <main class="boot-shell">
    <section class="boot-copy" aria-labelledby="boot-title">
      <p class="brand">炼丹炉</p>
      <h1 id="boot-title">启 炉</h1>
      <div class="status" aria-live="polite" aria-atomic="true">
        <p class="status-primary">正在温炉，请稍候</p>
        <div class="heat-line" aria-hidden="true"><span></span></div>
        <p class="status-secondary">本地引擎启动后将自动进入</p>
      </div>
      <div class="error-actions" hidden>
        <button type="button" class="copy-error">复制故障信息</button>
        <details><summary>故障详情</summary><pre class="error-detail"></pre></details>
        <textarea class="error-fallback" readonly hidden aria-label="可复制的故障信息"></textarea>
      </div>
    </section>
    <figure class="furnace-stage">
      <svg class="furnace" viewBox="0 0 640 560" role="img" aria-label="正在点燃的青铜丹炉">
        <defs>
          <radialGradient id="hearth-glow" cx="50%" cy="70%" r="55%">
            <stop offset="0" stop-color="#f3b55b" stop-opacity=".72" />
            <stop offset=".52" stop-color="#d85f2e" stop-opacity=".34" />
            <stop offset="1" stop-color="#17120f" stop-opacity="0" />
          </radialGradient>
        </defs>
        <g class="furnace-aura" aria-hidden="true"><ellipse cx="332" cy="324" rx="246" ry="166" fill="url(#hearth-glow)" /></g>
        <g class="furnace-body" fill="#716044" stroke="#3d352b" stroke-width="8">
          <path d="M176 204C190 124 246 88 320 88s130 36 144 116Z" />
          <path d="M142 206h356c-8 34-22 51-42 58 5 150-52 221-136 221s-141-71-136-221c-20-7-34-24-42-58Z" />
          <path d="M149 228C72 203 55 269 95 303c25 21 58 4 79-21" fill="none" stroke-width="20" />
          <path d="M491 228c77-25 94 41 54 75-25 21-58 4-79-21" fill="none" stroke-width="20" />
          <path d="M232 464l-28 72h58l17-58M408 464l28 72h-58l-17-58M320 480v68" stroke-linecap="round" />
        </g>
        <g class="furnace-windows">
          <g class="hearth hearth-left" transform="translate(206 300)">
            <path class="hearth-cavity" d="M0 70V32a32 32 0 0 1 64 0v38Z" fill="#17120f" />
            <path class="flame-outer" d="M9 67c-3-23 16-27 14-51 19 13 34 30 30 51Z" fill="#d85f2e" />
            <path class="flame-main" d="M19 67c-1-16 12-22 14-38 11 12 17 23 13 38Z" fill="#ed873c" />
            <path class="flame-core" d="M28 67c0-10 7-14 8-23 7 8 9 15 7 23Z" fill="#f3b55b" />
          </g>
          <g class="hearth hearth-center" transform="translate(288 292)">
            <path class="hearth-cavity" d="M0 78V36a36 36 0 0 1 72 0v42Z" fill="#17120f" />
            <path class="flame-outer" d="M10 75c-4-26 18-31 16-57 21 15 38 34 34 57Z" fill="#d85f2e" />
            <path class="flame-main" d="M22 75c-2-18 13-25 15-43 12 13 19 27 14 43Z" fill="#ed873c" />
            <path class="flame-core" d="M31 75c0-11 8-16 10-26 7 9 10 17 7 26Z" fill="#f3b55b" />
          </g>
          <g class="hearth hearth-right" transform="translate(378 300)">
            <path class="hearth-cavity" d="M0 70V32a32 32 0 0 1 64 0v38Z" fill="#17120f" />
            <path class="flame-outer" d="M9 67c-3-23 16-27 14-51 19 13 34 30 30 51Z" fill="#d85f2e" />
            <path class="flame-main" d="M19 67c-1-16 12-22 14-38 11 12 17 23 13 38Z" fill="#ed873c" />
            <path class="flame-core" d="M28 67c0-10 7-14 8-23 7 8 9 15 7 23Z" fill="#f3b55b" />
          </g>
        </g>
        <g class="furnace-details" fill="none" stroke="#c9a96e" stroke-width="5" opacity=".56" aria-hidden="true">
          <path d="M210 195c24-45 61-68 110-68s86 23 110 68" />
          <path d="M201 399c74 43 164 43 238 0" />
          <path d="M264 246h112" />
        </g>
      </svg>
    </figure>
  </main>
  <footer>本地运行 · 数据留在此设备</footer>
<script>
// 单请求轮询:页面只加载一次,状态走同源 JSON 探针;
// 请求失败只退避(最大 1800ms),不把瞬时故障误判为永久错误。
const SPLASH_STATUS_PATH = "./__alchemy_boot_status";
const body = document.body;
const title = document.querySelector("#boot-title");
const statusRegion = document.querySelector(".status");
const primary = document.querySelector(".status-primary");
const secondary = document.querySelector(".status-secondary");
const detail = document.querySelector(".error-detail");
const actions = document.querySelector(".error-actions");
const copyButton = document.querySelector(".copy-error");
const fallback = document.querySelector(".error-fallback");
let pollDelay = 450;
let lastError = "";

function showError(message) {
  lastError = message || "未知启动错误";
  body.dataset.state = "error";
  statusRegion.setAttribute("role", "alert");
  statusRegion.setAttribute("aria-live", "assertive");
  title.textContent = "炉火未成";
  primary.textContent = "丹炉未能正常启动";
  secondary.textContent = "请关闭应用后重新打开；如仍失败，请复制故障信息";
  detail.textContent = lastError;
  actions.hidden = false;
}

async function pollStatus() {
  try {
    const response = await fetch(SPLASH_STATUS_PATH, { cache: "no-store" });
    if (!response.ok) throw new Error("启动状态读取失败");
    const status = await response.json();
    if (status.state === "ready" && status.target) {
      window.location.replace(status.target);
      return;
    }
    if (status.state === "error") {
      showError(status.message);
      return;
    }
    pollDelay = 650;
  } catch (_error) {
    pollDelay = Math.min(Math.round(pollDelay * 1.5), 1800);
  }
  window.setTimeout(pollStatus, pollDelay);
}

copyButton.addEventListener("click", async () => {
  const text = "炼丹炉启动失败\n" + lastError;
  try {
    await navigator.clipboard.writeText(text);
    copyButton.textContent = "已复制";
  } catch (_error) {
    fallback.hidden = false;
    fallback.value = text;
    fallback.focus();
    fallback.select();
    copyButton.textContent = "请按 Ctrl/Cmd+C 复制";
  }
});

pollStatus();
</script>
</body>
</html>`

// splashStatusPath — 同源状态探针(页面 JS 轮询,代替整页 meta refresh)
const splashStatusPath = "/__alchemy_boot_status"

// writeSplashStatus — JSON 探针响应
func writeSplashStatus(w http.ResponseWriter, status splashStatus) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_ = json.NewEncoder(w).Encode(status)
}

// writeSplashDocument — 固定启动文档输出
func writeSplashDocument(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.WriteString(w, splashHTML)
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
		writeSplashDocument(w)
	})
}
