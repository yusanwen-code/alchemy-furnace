// Package dockreopen 修复 wails v2 macOS 缺失的 Dock 重开处理。
//
// 背景: wails v2 的 AppDelegate 未实现 applicationShouldHandleReopen:hasVisibleWindows:,
// 点击 Dock 图标时 macOS 只恢复"最小化"窗口, 对 orderOut 隐藏的窗口不做任何事,
// 导致"关闭到托盘后点 Dock 无法恢复主界面"。本包向该 delegate 类动态注册此方法,
// 转发到业务回调(恢复主窗口)。非 macOS 平台为 no-op。
package dockreopen

// Install 注册"点击 Dock 图标"回调(替换旧回调); show 为 nil 时仅注册空回调。
// 必须在 wails OnStartup(NSApp delegate 就绪)后调用; 非 macOS 平台为 no-op。
func Install(show func()) {
	install(show)
}
