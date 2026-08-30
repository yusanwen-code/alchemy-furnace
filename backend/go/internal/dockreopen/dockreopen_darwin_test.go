//go:build darwin

package dockreopen

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// 安装后 wails AppDelegate 类必须获得 reopen 方法(Dock 点击的入口)
func TestInstallAddsReopenMethodToAppDelegate(t *testing.T) {
	require.Equal(t, 0, ensureAppDelegate(), "AppDelegate 测试夹具应就绪")
	require.Equal(t, 0, hasReopenMethod(), "安装前不应已有 reopen 方法")

	Install(nil)
	require.Equal(t, 1, hasReopenMethod(), "Install 后 AppDelegate 必须响应 applicationShouldHandleReopen:")
}

// 触发 reopen 消息必须调用注册的回调(即恢复主界面)
func TestDockReopenTriggersCallback(t *testing.T) {
	require.Equal(t, 0, ensureAppDelegate(), "AppDelegate 测试夹具应就绪")
	var calls int32
	Install(func() { atomic.AddInt32(&calls, 1) })

	require.Equal(t, 1, fireReopen(), "reopen 消息应能触达 IMP")
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "点击 Dock 应恰好触发一次回调")
}

// 重复 Install 不叠加方法, 且回调更新为最新注册的
func TestInstallReplacesCallbackIdempotently(t *testing.T) {
	require.Equal(t, 0, ensureAppDelegate(), "AppDelegate 测试夹具应就绪")
	var first, second int32
	Install(func() { atomic.AddInt32(&first, 1) })
	Install(func() { atomic.AddInt32(&second, 1) })

	require.Equal(t, 1, fireReopen())
	require.Equal(t, int32(0), atomic.LoadInt32(&first), "旧回调不应再被触发")
	require.Equal(t, int32(1), atomic.LoadInt32(&second), "最新回调应被触发")
}
