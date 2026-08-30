package desktoptray

import (
	"errors"
	"os"
	"sync"
)

// Callbacks 托盘用户动作; Open=恢复主窗口, Quit=完整退出应用
type Callbacks struct {
	Open func()
	Quit func()
}

// Backend 平台托盘实现的窄接口(Windows Shell_NotifyIconW / macOS NSStatusItem)
type Backend interface {
	Start(Callbacks) error
	Stop() error
}

// Controller 平台无关的托盘生命周期; Start/Stop 幂等, Ready 反映真实状态
type Controller struct {
	backend Backend
	mu      sync.Mutex
	started bool
	ready   bool
}

func newController(backend Backend) *Controller {
	return &Controller{backend: backend}
}

// Start 启动托盘(幂等)。锁内只改状态, 锁外调用 backend, 避免平台回调重入死锁。
// nil 回调在传给 backend 前替换为空函数, backend 可安全调用。
func (c *Controller) Start(cb Callbacks) error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = true
	c.mu.Unlock()

	if cb.Open == nil {
		cb.Open = func() {}
	}
	if cb.Quit == nil {
		cb.Quit = func() {}
	}
	if err := c.backend.Start(cb); err != nil {
		c.mu.Lock()
		c.started = false
		c.ready = false
		c.mu.Unlock()
		return err
	}
	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()
	return nil
}

// Stop 停止托盘(幂等); 未启动时直接成功, 不触碰 backend
func (c *Controller) Stop() error {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return nil
	}
	c.started = false
	c.ready = false
	c.mu.Unlock()
	return c.backend.Stop()
}

// Ready 托盘是否可用(生命周期降级判断依据)
func (c *Controller) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ready
}

// disabledByEnvironment 失败降级测试开关: ALCHEMY_TRAY_DISABLE=1 时禁用托盘
func disabledByEnvironment() bool {
	return os.Getenv("ALCHEMY_TRAY_DISABLE") == "1"
}

// disabledBackend Start 返回明确错误, 让生命周期层走关闭即退出降级
type disabledBackend struct{}

func (disabledBackend) Start(Callbacks) error {
	return errors.New("desktoptray: disabled by ALCHEMY_TRAY_DISABLE")
}

func (disabledBackend) Stop() error { return nil }
