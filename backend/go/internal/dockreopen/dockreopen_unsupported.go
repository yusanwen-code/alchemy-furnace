//go:build !darwin

package dockreopen

// install 非 macOS 平台无 Dock 概念, no-op
func install(show func()) {}
