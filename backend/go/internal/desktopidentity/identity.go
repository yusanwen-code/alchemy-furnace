// Package desktopidentity 固定「显示名 炼丹炉 / 技术名 AlchemyFurnace」双层命名契约
//
// 用户可见位置(窗口标题、安装器、快捷方式、托盘菜单)必须使用 DisplayName;
// 可执行文件、数据目录、Bundle ID、更新 ZIP 根目录与 Release 资产名必须使用
// TechnicalName(ASCII)。详见 docs/superpowers/specs/2026-08-30-desktop-identity-tray-no-console-design.md
package desktopidentity

const (
	DisplayName       = "炼丹炉"
	TechnicalName     = "AlchemyFurnace"
	BundleID          = "com.alchemyfurnace.desktop"
	MacBundleName     = DisplayName + ".app"
	WindowsExecutable = TechnicalName + ".exe"
)
