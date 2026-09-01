package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestInitWithOptionsWritesJSONLogToAppFile(t *testing.T) {
	dir := t.TempDir()
	if err := InitWithOptions(Options{Mode: "release", DataDir: dir, BootID: "boot-test", Console: false}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Sync)
	L.Info("测试事件", zap.String("api_key", "sk-file-secret"))
	Sync()

	data, err := os.ReadFile(filepath.Join(dir, "logs", "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "boot-test") || !strings.Contains(text, "测试事件") {
		t.Fatalf("日志缺少 boot_id 或消息: %s", text)
	}
	if strings.Contains(text, "sk-file-secret") {
		t.Fatalf("文件日志泄露密钥: %s", text)
	}
}

func TestRedactTextRemovesSecretsButKeepsDiagnosticCode(t *testing.T) {
	input := "engine.health_timeout Authorization: Bearer secret X-Alchemy-Token: desktop-secret api_key=sk-example-secret"
	got := RedactText(input)
	for _, secret := range []string{"secret", "desktop-secret", "sk-example-secret"} {
		if strings.Contains(got, secret) {
			t.Fatalf("脱敏结果仍包含 %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, "engine.health_timeout") {
		t.Fatalf("脱敏丢失错误码: %s", got)
	}
}
