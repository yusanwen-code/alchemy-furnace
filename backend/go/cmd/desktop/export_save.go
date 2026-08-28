// cmd/desktop/export_save.go - 桌面端 Skill 导出落盘
// webview 重定向到 http origin 后 Wails Bind 不可达(WKWebView 也不执行 a[download] 的 Blob 下载),
// 故导出 ZIP 经 HTTP 桥接落盘: 前端 fetch 得二进制 → base64 → POST save-export → 写入
// 数据目录 exports/;reveal-export 在 Finder/资源管理器中定位文件。
// 两端点仅 desktop 模式注册(见 main.go),走同一 DesktopGuard。
package desktop

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/alchemy-furnace/server/internal/paths"
	"github.com/gin-gonic/gin"
)

// maxExportBytes 导出 ZIP 解码上限(10MB,安全兜底;实际包通常 < 1MB)
const maxExportBytes = 10 << 20

// exportZipRe 与 skill_export 的 ZIP 命名契约同口径:
// alchemy-skill-<slug>-codex.zip / -claude.zip(slug 仅 [a-z0-9][a-z0-9-]{0,48})
var exportZipRe = regexp.MustCompile(`^alchemy-skill-[a-z0-9][a-z0-9-]{0,48}-(codex|claude)\.zip$`)

// ValidateExportFilename 校验导出文件名(白名单正则,天然拒绝路径穿越)
func ValidateExportFilename(filename string) error {
	if !exportZipRe.MatchString(filename) {
		return fmt.Errorf("非法导出文件名: %q", filename)
	}
	return nil
}

// ExportsDir 返回导出目录(数据目录/exports),并确保存在
func ExportsDir() (string, error) {
	base, err := paths.DataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "exports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建导出目录: %w", err)
	}
	return dir, nil
}

// SaveExportBytes 将导出包写入数据目录 exports/,返回绝对路径
func SaveExportBytes(filename string, data []byte) (string, error) {
	if err := ValidateExportFilename(filename); err != nil {
		return "", err
	}
	dir, err := ExportsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("写入导出文件: %w", err)
	}
	return path, nil
}

// revealCommand 返回在系统文件管理器中定位文件的命令(darwin: open -R;windows: explorer /select,)
// 仅接受 exports 目录内已存在的文件,防任意路径命令注入。
func revealCommand(path string) (*exec.Cmd, error) {
	dir, err := ExportsDir()
	if err != nil {
		return nil, err
	}
	clean := filepath.Clean(path)
	if !strings.HasPrefix(clean, filepath.Clean(dir)+string(os.PathSeparator)) {
		return nil, errors.New("拒绝非导出目录路径")
	}
	if info, err := os.Stat(clean); err != nil || info.IsDir() {
		return nil, errors.New("导出文件不存在")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", clean), nil
	case "windows":
		return exec.Command("explorer", "/select,", clean), nil
	default:
		return nil, fmt.Errorf("当前平台不支持定位: %s", runtime.GOOS)
	}
}

// RevealExportPath 在系统文件管理器中定位导出文件
func RevealExportPath(path string) error {
	cmd, err := revealCommand(path)
	if err != nil {
		return err
	}
	return cmd.Run()
}

// RegisterExportSaveEndpoints 注册桌面端导出落盘端点(仅 desktop 模式调用)
func RegisterExportSaveEndpoints(engine *gin.Engine) {
	engine.POST("/api/v1/desktop/save-export", func(c *gin.Context) {
		var req struct {
			Filename   string `json:"filename"`
			ContentB64 string `json:"content_b64"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "desktop.save_export.invalid", "message": "请求体非法"})
			return
		}
		if err := ValidateExportFilename(req.Filename); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "desktop.save_export.invalid_name", "message": "非法导出文件名"})
			return
		}
		data, err := base64.StdEncoding.DecodeString(req.ContentB64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "desktop.save_export.invalid_b64", "message": "内容不是合法 base64"})
			return
		}
		if len(data) == 0 || len(data) > maxExportBytes {
			c.JSON(http.StatusBadRequest, gin.H{"code": "desktop.save_export.size", "message": "导出内容为空或超出 10MB 上限"})
			return
		}
		savedPath, err := SaveExportBytes(req.Filename, data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": "desktop.save_export.write_failed", "message": "保存失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": map[string]string{"saved_path": savedPath}})
	})

	engine.POST("/api/v1/desktop/reveal-export", func(c *gin.Context) {
		var req struct {
			Path string `json:"path"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "desktop.reveal.invalid", "message": "请求体非法"})
			return
		}
		if err := RevealExportPath(req.Path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "desktop.reveal.rejected", "message": "无法定位: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": nil})
	})
}
