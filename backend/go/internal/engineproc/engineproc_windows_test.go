//go:build windows

package engineproc

import (
	"os/exec"
	"syscall"
	"testing"

	winapi "golang.org/x/sys/windows"
)

// Python 子进程必须同时带新进程组(退出时杀整树)与无窗口标志(防黑框)
func TestSetProcGroupPreventsConsoleWindow(t *testing.T) {
	cmd := exec.Command("python.exe", "-V")
	setProcGroup(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	want := uint32(syscall.CREATE_NEW_PROCESS_GROUP) | uint32(winapi.CREATE_NO_WINDOW)
	if cmd.SysProcAttr.CreationFlags&want != want {
		t.Fatalf("CreationFlags=%#x want bits %#x", cmd.SysProcAttr.CreationFlags, want)
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("HideWindow=false")
	}
}

// 退出阶段的 taskkill 必须隐藏, 否则关窗时闪黑框
func TestNewHiddenCommandPreventsTaskkillConsole(t *testing.T) {
	cmd := newHiddenCommand("taskkill", "/pid", "123")
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("taskkill must be hidden")
	}
	if cmd.SysProcAttr.CreationFlags&uint32(winapi.CREATE_NO_WINDOW) == 0 {
		t.Fatal("taskkill missing CREATE_NO_WINDOW")
	}
}
