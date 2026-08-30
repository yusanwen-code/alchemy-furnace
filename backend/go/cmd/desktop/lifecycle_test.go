package desktop

import (
	"context"
	"errors"
	"testing"

	"github.com/alchemy-furnace/server/internal/desktoptray"
	"github.com/stretchr/testify/require"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// fakeWindowRuntime 记录窄接口调用
type fakeWindowRuntime struct {
	hides       int
	shows       int
	unminimises int
	abouts      int
	quits       int
	saved       int // 窗口几何落盘次数
}

func (f *fakeWindowRuntime) Hide(context.Context)       { f.hides++ }
func (f *fakeWindowRuntime) Show(context.Context)       { f.shows++ }
func (f *fakeWindowRuntime) Unminimise(context.Context) { f.unminimises++ }
func (f *fakeWindowRuntime) About(context.Context)      { f.abouts++ }
func (f *fakeWindowRuntime) Quit(context.Context)       { f.quits++ }

type fakeTrayBackend struct {
	err   error
	stops int
}

func (f *fakeTrayBackend) Start(desktoptray.Callbacks) error { return f.err }
func (f *fakeTrayBackend) Stop() error                       { f.stops++; return nil }

// fixture: trayErr=nil → 托盘就绪; 非 nil → Start 失败(降级路径)
// saveState 注入 fake: 真实 saveWindowState 需要 Wails frontend ctx,测试环境无则 log.Fatalf
func newLifecycleFixture(trayErr error) (*desktopLifecycle, *fakeWindowRuntime, *fakeTrayBackend) {
	win := &fakeWindowRuntime{}
	tb := &fakeTrayBackend{err: trayErr}
	l := newDesktopLifecycle(win, desktoptray.NewController(tb))
	l.saveState = func(context.Context) error { win.saved++; return nil }
	return l, win, tb
}

// 托盘就绪: 关闭 → 保存窗口状态 + 隐藏窗口, 返回 true(吞掉关闭)
func TestLifecycleBeforeCloseHidesWhenTrayReady(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	require.True(t, l.BeforeClose(context.Background()))
	require.Equal(t, 1, win.saved)
	require.Equal(t, 1, win.hides)
}

// Start 必须注册 Dock 重开回调(wails 缺失的 applicationShouldHandleReopen 修复),
// 且触发该回调应恢复主窗口
func TestLifecycleStartInstallsDockReopen(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	var installed func()
	l.dockReopen = func(fn func()) { installed = fn }

	require.NoError(t, l.Start(context.Background()))
	require.NotNil(t, installed, "Start 应注册 Dock 重开回调")

	installed() // 模拟点击 Dock 图标
	require.Equal(t, 1, win.shows, "触发 Dock 重开应恢复主窗口")
}

// 托盘失败: 关闭 → 不隐藏, 返回 false(真正退出)
func TestLifecycleBeforeCloseQuitsWhenTrayNotReady(t *testing.T) {
	l, win, _ := newLifecycleFixture(errors.New("tray boom"))
	require.Error(t, l.Start(context.Background()))
	require.False(t, l.BeforeClose(context.Background()))
	require.Equal(t, 0, win.hides)
}

// context 未注入时 HideMainWindow 不执行(不 panic, 不调 Wails runtime)
func TestLifecycleHideWithoutContextIsNoop(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	l.HideMainWindow() // 未 Start: ctx 为 nil
	require.Equal(t, 0, win.hides)
	require.NoError(t, l.Start(context.Background()))
	l.HideMainWindow()
	require.Equal(t, 1, win.hides)
}

// 恢复窗口: 先 Unminimise 再 Show
func TestLifecycleShowMainWindowUnminimiseThenShow(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	l.ShowMainWindow()
	require.Equal(t, 1, win.unminimises)
	require.Equal(t, 1, win.shows)
}

// 关于对话框: context 可用时打开
func TestLifecycleShowAboutOpensDialog(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	l.ShowAbout()
	require.Equal(t, 1, win.abouts)
}

// RequestQuit 只触发一次 Quit; 二次调用无动作
func TestLifecycleRequestQuitOnce(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	l.RequestQuit()
	l.RequestQuit()
	require.Equal(t, 1, win.quits)
}

// quitting 后 BeforeClose 直接返回 false(允许退出流程继续)
func TestLifecycleBeforeCloseAfterQuitReturnsFalse(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	require.NoError(t, l.Start(context.Background()))
	l.RequestQuit()
	require.False(t, l.BeforeClose(context.Background()))
	require.Equal(t, 0, win.hides)
}

// Shutdown 只调一次托盘 Stop
func TestLifecycleShutdownStopsTrayOnce(t *testing.T) {
	_, _, tb := newLifecycleFixture(nil)
	tray := desktoptray.NewController(tb)
	l := newDesktopLifecycle(&fakeWindowRuntime{}, tray)
	l.saveState = func(context.Context) error { return nil }
	require.NoError(t, l.Start(context.Background()))
	l.Shutdown()
	l.Shutdown()
	require.Equal(t, 1, tb.stops)
}

// 装配契约: HideWindowOnClose=false(否则绕过托盘失败降级), 生命周期字段全部接线
func TestConfigureDesktopLifecycleFields(t *testing.T) {
	l, _, _ := newLifecycleFixture(nil)
	opts := &options.App{}
	configureDesktopLifecycle(opts, l)
	require.False(t, opts.HideWindowOnClose)
	require.NotNil(t, opts.OnBeforeClose)
	require.NotNil(t, opts.OnStartup)
	require.NotNil(t, opts.OnShutdown)
	require.NotNil(t, opts.SingleInstanceLock)
	require.NotNil(t, opts.SingleInstanceLock.OnSecondInstanceLaunch)
}

// 单实例唤回: 第二实例通知触发 ShowMainWindow(Unminimise + Show)
func TestConfigureSecondInstanceShowsMainWindow(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	opts := &options.App{}
	configureDesktopLifecycle(opts, l)
	require.NoError(t, l.Start(context.Background()))
	opts.SingleInstanceLock.OnSecondInstanceLaunch(options.SecondInstanceData{})
	require.Equal(t, 1, win.unminimises)
	require.Equal(t, 1, win.shows)
}

// 唤回早于 startup(context 未注入): pendingShow 记录, Start 注入 ctx 后消费一次
func TestLifecyclePendingShowConsumedOnStart(t *testing.T) {
	l, win, _ := newLifecycleFixture(nil)
	l.ShowMainWindow() // 未 Start: ctx 为 nil, 只记 pendingShow
	require.Equal(t, 0, win.shows)
	require.NoError(t, l.Start(context.Background()))
	require.Equal(t, 1, win.shows)
}
