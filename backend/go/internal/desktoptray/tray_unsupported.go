//go:build !windows && !darwin

package desktoptray

import "errors"

// unsupportedBackend 非 Windows/macOS 平台明确返回不支持(Linux CI 可编译)
type unsupportedBackend struct{}

func (unsupportedBackend) Start(Callbacks) error {
	return errors.New("desktoptray: unsupported platform")
}

func (unsupportedBackend) Stop() error { return nil }

// New 平台工厂: 非 Windows/macOS 返回明确报错的 Controller
func New() *Controller {
	if disabledByEnvironment() {
		return newController(disabledBackend{})
	}
	return newController(unsupportedBackend{})
}
