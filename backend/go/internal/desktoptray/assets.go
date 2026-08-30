package desktoptray

import _ "embed"

// macOS Template 图标(1x/2x): 纯黑轮廓 + alpha, 系统按菜单栏深浅自动着色
//go:embed assets/furnace-tray-macos.png
var macTemplateIcon []byte

//go:embed assets/furnace-tray-macos@2x.png
var macTemplateIcon2x []byte

// Windows 通知区域图标(16/20/24/32/48 多帧)
//go:embed assets/furnace-tray-windows.ico
var windowsIcon []byte
