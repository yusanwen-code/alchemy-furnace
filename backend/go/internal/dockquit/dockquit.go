// Package dockquit routes macOS Dock “Quit” requests to the application's
// complete shutdown path. Wails otherwise sends them through OnBeforeClose,
// which is intentionally used by this application for close-to-tray.
package dockquit

// Install registers the callback for a Dock “Quit” request. Repeated calls
// replace the previous callback. It is a no-op outside macOS.
func Install(quit func()) {
	install(quit)
}
