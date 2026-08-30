// cmd/desktop/open_external_test.go - 桌面端外部链接校验回归
// WKWebView 不实现 createWebViewWith,target="_blank" 静默失效,
// 链接经 HTTP 桥接交给系统浏览器;校验仅放行 http/https。
package desktop

import "testing"

func TestValidateExternalURL(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "github https", raw: "https://github.com/yusanwen-code/alchemy-furnace"},
		{name: "普通 http", raw: "http://example.com/page?q=1"},
		{name: "端口与路径", raw: "https://example.com:8443/path"},
		{name: "空链接", raw: "", wantErr: true},
		{name: "javascript 协议", raw: "javascript:alert(1)", wantErr: true},
		{name: "file 协议", raw: "file:///etc/passwd", wantErr: true},
		{name: "data 协议", raw: "data:text/html,<script>alert(1)</script>", wantErr: true},
		{name: "无协议文本", raw: "not a url", wantErr: true},
		{name: "缺主机名", raw: "https://", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateExternalURL(c.raw)
			if c.wantErr && err == nil {
				t.Fatalf("validateExternalURL(%q) = nil, 期望错误", c.raw)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("validateExternalURL(%q) = %v, 期望 nil", c.raw, err)
			}
		})
	}
}
