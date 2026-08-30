//go:build windows

package engineproc

import (
	"fmt"
	"os/exec"
	"syscall"

	winapi "golang.org/x/sys/windows"
)

// noWindowProcAttr 统一隐藏控制台创建参数:
//   - CREATE_NO_WINDOW 在创建进程时阻止任何控制台窗口出现(禁止事后 ShowWindow 补救)
//   - HideWindow 供 Go 内部 STARTF_USESHOWWINDOW 兜底
func noWindowProcAttr(extra uint32) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: extra | uint32(winapi.CREATE_NO_WINDOW),
		HideWindow:    true,
	}
}

// setProcGroup Python 子进程: 新进程组(退出时杀整树) + 无窗口(防黑框)
func setProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = noWindowProcAttr(uint32(syscall.CREATE_NEW_PROCESS_GROUP))
}

// newHiddenCommand 退出阶段任务等一次性命令: 无窗口创建
func newHiddenCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = noWindowProcAttr(0)
	return cmd
}

func killProcGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = newHiddenCommand("taskkill", "/pid", fmt.Sprint(cmd.Process.Pid), "/T", "/F").Run()
	}
}
