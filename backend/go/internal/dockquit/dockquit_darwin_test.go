//go:build darwin

package dockquit

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// Dock 上的“退出”必须走完整应用退出回调，不能被 Wails 的窗口关闭处理吞掉。
func TestInstallRoutesDockQuitToCallback(t *testing.T) {
	require.Equal(t, 0, ensureAppDelegate(), "AppDelegate 测试夹具应就绪")
	var calls int32

	Install(func() { atomic.AddInt32(&calls, 1) })

	require.Equal(t, 1, fireTerminate(), "Dock 的退出请求应触达原生终止回调")
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "Dock 退出应恰好请求一次完整退出")
}

// 重复安装不应叠加退出动作；生命周期重入时只能使用最新回调。
func TestInstallReplacesDockQuitCallback(t *testing.T) {
	require.Equal(t, 0, ensureAppDelegate(), "AppDelegate 测试夹具应就绪")
	var first, second int32
	Install(func() { atomic.AddInt32(&first, 1) })
	Install(func() { atomic.AddInt32(&second, 1) })

	require.Equal(t, 1, fireTerminate())
	require.Equal(t, int32(0), atomic.LoadInt32(&first))
	require.Equal(t, int32(1), atomic.LoadInt32(&second))
}
