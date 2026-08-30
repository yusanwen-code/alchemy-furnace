// Package avatar 共享头像字段契约校验(道人/用户头像共用)
//
// 契约(与前端编辑表单保持一致):
//   - 空值合法(清空头像)
//   - 完整 http/https URL:长度 ≤MaxURLLen,不允许内嵌凭据(user:pass@)
//   - data:image/(png|jpeg|webp|gif);base64,:URI 总长 ≤MaxDataURILen,payload 仅 base64 字符
//   - 其余(相对路径 / javascript: / vbscript: / blob: / 其他 MIME / 超长)→ error
//
// 错误消息只描述规则,绝不携带原始头像值(完整 data URI 不进响应/日志)。
package avatar

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// MaxURLLen 完整 http/https URL 最大长度(字符)
	MaxURLLen = 2048
	// MaxDataURILen data:image URI 最大总长(字符)
	MaxDataURILen = 1_500_000
)

// avatarAllowedImageMIMEs data URI 允许的图片 MIME 子类型
var avatarAllowedImageMIMEs = map[string]bool{
	"png":  true,
	"jpeg": true,
	"webp": true,
	"gif":  true,
}

// Validate 校验头像字段(创建与更新共用);空值合法,返回 nil
func Validate(value string) error {
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "data:image/") {
		return validateDataURI(value)
	}
	return validateURL(value)
}

// validateURL 完整 URL 校验:http/https、长度、主机名、无内嵌凭据
func validateURL(raw string) error {
	if len(raw) > MaxURLLen {
		return fieldError("头像 URL 过长(上限 %d 字符)", MaxURLLen)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fieldError("头像 URL 格式不正确")
	}
	if scheme := strings.ToLower(u.Scheme); scheme != "http" && scheme != "https" {
		return fieldError("头像仅支持完整 http/https URL 或 data:image 数据 URI")
	}
	if u.Host == "" {
		return fieldError("头像 URL 缺少主机名")
	}
	if u.User != nil {
		return fieldError("头像 URL 不允许包含用户名或密码")
	}
	return nil
}

// validateDataURI data URI 校验:MIME 白名单 + base64 编码 + 长度 + 字符集
func validateDataURI(uri string) error {
	if len(uri) > MaxDataURILen {
		return fieldError("头像 data URI 过长(上限 %d 字符)", MaxDataURILen)
	}
	rest, ok := strings.CutPrefix(uri, "data:image/")
	if !ok {
		return fieldError("头像仅支持完整 http/https URL 或 data:image 数据 URI")
	}
	semi := strings.IndexByte(rest, ';')
	if semi < 0 {
		return fieldError("头像 data URI 缺少 MIME 参数")
	}
	if !avatarAllowedImageMIMEs[rest[:semi]] {
		return fieldError("头像 data URI 仅支持 png/jpeg/webp/gif 图片")
	}
	tail := rest[semi+1:]
	if !strings.HasPrefix(tail, "base64,") {
		return fieldError("头像 data URI 仅支持 base64 编码")
	}
	payload := tail[len("base64,"):]
	if payload == "" {
		return fieldError("头像 data URI 缺少图片数据")
	}
	for i := 0; i < len(payload); i++ {
		if !isBase64Char(payload[i]) {
			return fieldError("头像 data URI 包含非法字符")
		}
	}
	return nil
}

// isBase64Char base64 字母表(A-Za-z0-9+/=)
func isBase64Char(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '/' || c == '='
}

// fieldError 头像字段错误(仅描述规则,不含头像值)
func fieldError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
