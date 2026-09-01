// lifecycle.go — 关闭到托盘的后台生命周期状态机(平台无关)
// 关闭按钮: 托盘可用 → 隐藏窗口; 托盘失败 → 关闭即真正退出(降级)
// 真正退出入口: 托盘菜单"退出炼丹炉"(经 RequestQuit → Wails runtime.Quit)
package desktop

import (
	"context"
	"log"
	"sync"
	"sync/atomic"

	"github.com/alchemy-furnace/server/internal/desktoptray"
	"github.com/alchemy-furnace/server/internal/dockquit"
	"github.com/alchemy-furnace/server/internal/dockreopen"
	"github.com/wailsapp/wails/v2/pkg/options"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// windowRuntime 窄接口: 生命周期层只依赖这几个窗口操作(测试用 fake 替换)
type windowRuntime interface {
	Hide(context.Context)
	Show(context.Context)
	Unminimise(context.Context)
	About(context.Context)
	Quit(context.Context)
}

// wailsWindowRuntime 无状态适配器: 窄接口 → Wails runtime
type wailsWindowRuntime struct{}

func (wailsWindowRuntime) Hide(ctx context.Context)       { wailsruntime.WindowHide(ctx) }
func (wailsWindowRuntime) Show(ctx context.Context)       { wailsruntime.WindowShow(ctx) }
func (wailsWindowRuntime) Unminimise(ctx context.Context) { wailsruntime.WindowUnminimise(ctx) }
func (wailsWindowRuntime) Quit(ctx context.Context)       { wailsruntime.Quit(ctx) }
func (wailsWindowRuntime) About(ctx context.Context) {
	_, _ = wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
		Type:    wailsruntime.InfoDialog,
		Title:   "关于炼丹炉",
		Message: "Alchemy Furnace",
	})
}

// desktopLifecycle 关闭/恢复/退出状态机
type desktopLifecycle struct {
	mu     sync.RWMutex
	ctx    context.Context
	window windowRuntime
	tray   *desktoptray.Controller

	// saveState 窗口几何落盘(可注入: 测试用 fake, 真实实现=windowstate.saveWindowState,
	// 它需要 Wails frontend ctx, 测试环境没有会 log.Fatalf)
	saveState func(context.Context) error

	// dockReopen 注册"点击 Dock 图标"回调(可注入: 测试用 fake 记录,
	// 真实实现=dockreopen.Install 向 wails AppDelegate 添加缺失的 reopen 方法)
	dockReopen func(func())

	// dockQuit 将 Dock 的“退出”接入完整退出路径，而非 OnBeforeClose 的隐藏窗口路径。
	dockQuit func(func())

	trayReady   atomic.Bool
	quitting    atomic.Bool
	pendingShow atomic.Bool // 第二实例唤回早于 startup(context 未注入)时记录

	shutdownOnce sync.Once
}

func newDesktopLifecycle(window windowRuntime, tray *desktoptray.Controller) *desktopLifecycle {
	return &desktopLifecycle{
		window:     window,
		tray:       tray,
		saveState:  saveWindowState,
		dockReopen: dockreopen.Install,
		dockQuit:   dockquit.Install,
	}
}

// Start 注入 Wails context 并启动托盘(幂等); 托盘失败仅记录错误, 不弹阻塞对话框。
// startup 前到达的第二实例唤回在此消费一次(此时 ctx 已可用)。
func (l *desktopLifecycle) Start(ctx context.Context) error {
	l.mu.Lock()
	l.ctx = ctx
	l.mu.Unlock()
	// wails 未实现 applicationShouldHandleReopen(Dock 点击恢复窗口),
	// 由 dockreopen 向 AppDelegate 动态注册, 转发到 ShowMainWindow
	l.dockReopen(l.ShowMainWindow)
	l.dockQuit(l.RequestQuit)
	err := l.tray.Start(desktoptray.Callbacks{
		Open: l.ShowMainWindow,
		Quit: l.RequestQuit,
	})
	l.trayReady.Store(l.tray.Ready())
	if l.pendingShow.Swap(false) {
		l.ShowMainWindow()
	}
	return err
}

// BeforeClose 窗口关闭回调: true=吞掉关闭(隐藏到托盘), false=允许退出
// 先查 quitting(真正退出中不拦截), 再查 trayReady(托盘失败时关闭即退出);
// 只有托盘可用时才保存窗口状态并隐藏。
func (l *desktopLifecycle) BeforeClose(ctx context.Context) bool {
	if l.quitting.Load() {
		return false
	}
	if !l.trayReady.Load() {
		return false
	}
	if err := l.saveState(ctx); err != nil {
		log.Printf("[炼丹炉] 关闭时保存窗口状态失败: %v", err)
	}
	l.HideMainWindow()
	return true
}

// HideMainWindow 隐藏主窗口; context 未注入(Wails 未就绪)时不执行
func (l *desktopLifecycle) HideMainWindow() {
	l.mu.RLock()
	ctx := l.ctx
	l.mu.RUnlock()
	if ctx == nil {
		return
	}
	l.window.Hide(ctx)
}

// ShowMainWindow 恢复主窗口: 先取消最小化再显示
// context 未注入(第二实例通知早于 startup)时记 pendingShow, Start 后消费一次
func (l *desktopLifecycle) ShowMainWindow() {
	l.mu.RLock()
	ctx := l.ctx
	l.mu.RUnlock()
	if ctx == nil {
		l.pendingShow.Store(true)
		return
	}
	l.window.Unminimise(ctx)
	l.window.Show(ctx)
}

// ShowAbout 打开"关于炼丹炉"信息对话框
func (l *desktopLifecycle) ShowAbout() {
	l.mu.RLock()
	ctx := l.ctx
	l.mu.RUnlock()
	if ctx == nil {
		return
	}
	l.window.About(ctx)
}

// RequestQuit 完整退出(托盘"退出炼丹炉"): 只触发一次, 走 Wails OnShutdown
func (l *desktopLifecycle) RequestQuit() {
	if !l.quitting.CompareAndSwap(false, true) {
		return
	}
	l.mu.RLock()
	ctx := l.ctx
	l.mu.RUnlock()
	if ctx == nil {
		return
	}
	l.window.Quit(ctx)
}

// Shutdown 应用退出时的托盘清理(幂等, 由 OnShutdown 统一调用)
func (l *desktopLifecycle) Shutdown() {
	l.shutdownOnce.Do(func() {
		_ = l.tray.Stop()
	})
}

// configureDesktopLifecycle 装配 Wails 生命周期字段(可测试工厂)
// HideWindowOnClose 必须为 false: true 会绕过托盘失败降级(关闭即退出)
// OnShutdown 默认=lifecycle.Shutdown(幂等停托盘); main.go 需要组合关停时在其后覆盖
func configureDesktopLifecycle(opts *options.App, lifecycle *desktopLifecycle) {
	opts.HideWindowOnClose = false
	opts.OnBeforeClose = lifecycle.BeforeClose
	// Wails 回调签名无返回值, 用闭包适配(Start 失败记录错误, 不弹阻塞对话框)
	opts.OnStartup = func(ctx context.Context) { _ = lifecycle.Start(ctx) }
	opts.OnShutdown = func(context.Context) { lifecycle.Shutdown() }
	if opts.SingleInstanceLock == nil {
		opts.SingleInstanceLock = &options.SingleInstanceLock{}
	}
	opts.SingleInstanceLock.OnSecondInstanceLaunch = func(options.SecondInstanceData) {
		lifecycle.ShowMainWindow()
	}
}
