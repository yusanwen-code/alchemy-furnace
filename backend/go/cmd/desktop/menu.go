// menu.go — L4 macOS 原生菜单栏
// 不用 menu.AppMenu(): 它会自动生成 Quit/⌘Q 第二退出入口, 与"应用内只有托盘退出"冲突
// 自定义子菜单: 关于炼丹炉(信息对话框) + 隐藏炼丹炉 ⌘H
// EditMenu: Undo/Redo/Cut/Copy/Paste/SelectAll (⌘Z/⇧⌘Z/⌘X/⌘C/⌘V/⌘A,让 webview 输入框自带快捷键)
// 快捷键(⌘,/⌘N)由前端 desktop-guards.tsx 监听,菜单里不重复(避免两套入口分歧)
package desktop

import (
	"runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
)

// buildAppMenu 仅在 mac 装配菜单;win/linux 走 webview 默认(系统菜单栏不存在)
// 操作系统关机、注销和强制结束仍可走系统终止路径(不受菜单约束)
func buildAppMenu(lifecycle *desktopLifecycle) *menu.Menu {
	m := menu.NewMenu()
	if runtime.GOOS != "darwin" {
		return m
	}
	appMenu := m.AddSubmenu("炼丹炉")
	appMenu.AddText("关于炼丹炉", nil, func(_ *menu.CallbackData) {
		lifecycle.ShowAbout()
	})
	appMenu.AddSeparator()
	appMenu.AddText("隐藏炼丹炉", keys.CmdOrCtrl("h"), func(_ *menu.CallbackData) {
		lifecycle.HideMainWindow()
	})
	m.Append(menu.EditMenu())
	return m
}
