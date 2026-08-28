// export_save_test.go - 桌面 Skill 导出落盘单测
// 注入临时数据目录(paths.SetDataDirOverrideForTest),不触碰真实用户目录。
package desktop

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/alchemy-furnace/server/internal/paths"
)

func setupExportTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	paths.SetDataDirOverrideForTest(dir)
	t.Cleanup(func() { paths.SetDataDirOverrideForTest("") })
	return dir
}

func TestValidateExportFilename(t *testing.T) {
	valid := []string{
		"alchemy-skill-my-skill-codex.zip",
		"alchemy-skill-x-claude.zip",
		"alchemy-skill-a-1-2-3-codex.zip",
		// slug 上限 49 字符(与 Python MAX_SLUG_LENGTH 对齐)
		"alchemy-skill-" + "a" + strings.Repeat("b", 48) + "-codex.zip",
	}
	for _, name := range valid {
		if err := ValidateExportFilename(name); err != nil {
			t.Errorf("ValidateExportFilename(%q) = %v, want nil", name, err)
		}
	}
	invalid := []string{
		"", "x.zip", "alchemy-skill-x.zip",
		"../evil.zip", "a/../../evil.zip", "alchemy-skill-..-codex.zip",
		"/etc/passwd", "alchemy-skill-x-codex.zip/..",
		"alchemy-skill-中文-codex.zip",
		"alchemy-skill-x-yaml.zip", "alchemy-skill-x-codex.ZIP", "alchemy-skill-x-codex.tar.gz",
		"alchemy-skill-" + strings.Repeat("a", 60) + "-codex.zip",
		"alchemy-skill--x-codex.zip", // 首字符必须是字母数字
	}
	for _, name := range invalid {
		if err := ValidateExportFilename(name); err == nil {
			t.Errorf("ValidateExportFilename(%q) = nil, want error", name)
		}
	}
}

func TestSaveExportBytes_RoundTrip(t *testing.T) {
	setupExportTest(t)
	content := []byte("PK\x03\x04 fake zip content")
	saved, err := SaveExportBytes("alchemy-skill-my-skill-codex.zip", content)
	if err != nil {
		t.Fatalf("SaveExportBytes: %v", err)
	}
	base, _ := paths.DataDir()
	if !strings.HasPrefix(saved, filepath.Join(base, "exports", "")) {
		t.Fatalf("saved path %q 不在 exports 目录下", saved)
	}
	got, err := os.ReadFile(saved)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("read back = %q, want %q", got, content)
	}
}

func TestSaveExportBytes_RejectsInvalidName(t *testing.T) {
	base := setupExportTest(t)
	for _, name := range []string{"../evil.zip", "x.zip", "alchemy-skill-x-codex.zip/.."} {
		if _, err := SaveExportBytes(name, []byte("PK")); err == nil {
			t.Errorf("SaveExportBytes(%q) = nil, want error", name)
		}
	}
	// 非法名不得落盘(exports 目录可能因从未成功保存而不存在,视为空)
	entries, err := os.ReadDir(filepath.Join(base, "exports"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read exports dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("exports 目录应为空,实际 %d 个文件", len(entries))
	}
}

func TestRevealExportPath_RejectsOutsideExports(t *testing.T) {
	base := setupExportTest(t)
	// 写一个 exports 外的文件
	outside := filepath.Join(filepath.Dir(base), "evil.zip")
	if err := os.WriteFile(outside, []byte("PK"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		outside,
		filepath.Join(base, "exports", "..", "evil.zip"),
		filepath.Join(base, "exports", "missing.zip"),
	} {
		if _, err := revealCommand(p); err == nil {
			t.Errorf("revealCommand(%q) = nil, want error", p)
		}
	}
}

func TestRevealExportPath_CommandForExistingFile(t *testing.T) {
	if runtime.GOOS == "linux" {
		// desktop 仅支持 darwin/windows,linux 上 revealCommand 明确返回不支持错误
		t.Skip("desktop 不支持 linux 定位")
	}
	setupExportTest(t)
	saved, err := SaveExportBytes("alchemy-skill-my-skill-codex.zip", []byte("PK"))
	if err != nil {
		t.Fatalf("SaveExportBytes: %v", err)
	}
	cmd, err := revealCommand(saved)
	if err != nil {
		t.Fatalf("revealCommand: %v", err)
	}
	if runtime.GOOS == "darwin" {
		// exec.Command 会解析 PATH,cmd.Path 是绝对路径,只断言 Args
		if len(cmd.Args) != 3 || cmd.Args[0] != "open" || cmd.Args[1] != "-R" || cmd.Args[2] != saved {
			t.Fatalf("cmd = %v, want open -R %s", cmd.Args, saved)
		}
	}
	if runtime.GOOS == "windows" {
		if len(cmd.Args) != 3 || cmd.Args[0] != "explorer" || cmd.Args[1] != "/select," || cmd.Args[2] != saved {
			t.Fatalf("cmd = %v, want explorer /select, %s", cmd.Args, saved)
		}
	}
}

func TestExportsDir_CreatesDirectory(t *testing.T) {
	setupExportTest(t)
	dir, err := ExportsDir()
	if err != nil {
		t.Fatalf("ExportsDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("exports dir 未创建: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("%s 不是目录", dir)
	}
}
