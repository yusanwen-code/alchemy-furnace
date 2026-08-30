// cmd/desktop/open_external.go - 桌面端外部链接交给系统浏览器
// WKWebView 未实现 createWebViewWith,target="_blank" 点击被静默吞掉
// (设置关于的 GitHub 仓库链接打不开即此因);链接经 HTTP 桥接交给
// 系统默认浏览器打开(open / rundll32)。仅 desktop 模式注册(见 main.go),
// 走同一 DesktopGuard。校验仅放行 http/https,防 javascript:/file: 注入。
package desktop

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/gin-gonic/gin"
)

// validateExternalURL 校验可交系统浏览器打开的链接:仅 http/https 且带主机名
func validateExternalURL(raw string) error {
	if raw == "" {
		return errors.New("链接为空")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("非法链接: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https 链接: %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("链接缺少主机名")
	}
	return nil
}

// OpenExternalURL 用系统默认浏览器打开链接(darwin: open;windows: rundll32)
func OpenExternalURL(raw string) error {
	if err := validateExternalURL(raw); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", raw).Run()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", raw).Run()
	default:
		return fmt.Errorf("当前平台不支持打开外部链接: %s", runtime.GOOS)
	}
}

// RegisterOpenExternalEndpoints 注册桌面端外部链接端点(仅 desktop 模式调用)
func RegisterOpenExternalEndpoints(engine *gin.Engine) {
	engine.POST("/api/v1/desktop/open-url", func(c *gin.Context) {
		var req struct {
			URL string `json:"url"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "desktop.open_url.invalid", "message": "请求体非法"})
			return
		}
		if err := OpenExternalURL(req.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "desktop.open_url.rejected", "message": "无法打开链接: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": nil})
	})
}
