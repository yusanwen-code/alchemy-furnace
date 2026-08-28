package agent

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/alchemy-furnace/server/internal/errors"
)

// 头像字段契约(与前端编辑表单保持一致):
//   - 空值合法(清空头像)
//   - 完整 http/https URL:长度 ≤2048,不允许内嵌凭据(user:pass@)
//   - data:image/(png|jpeg|webp|gif);base64,:URI 总长 ≤1_500_000 字符,payload 仅 base64 字符
//   - 其余(相对路径 / javascript: / vbscript: / blob: / 其他 MIME / 超长)→ 400 字段错误
//
// 错误消息只描述规则,绝不携带头像值(完整 data URI 不进响应/日志)
const (
	avatarMaxURLLen     = 2048
	avatarMaxDataURILen = 1_500_000
)

// avatarAllowedImageMIMEs data URI 允许的图片 MIME 子类型
var avatarAllowedImageMIMEs = map[string]bool{
	"png":  true,
	"jpeg": true,
	"webp": true,
	"gif":  true,
}

// validateAvatar 校验头像字段(创建与更新共用)
// 返回 ErrorTypeInvalidRequest → Wrapper 映射 400 字段错误
func validateAvatar(v string) errors.Error {
	if v == "" {
		return nil // 空值合法:清空头像
	}
	if strings.HasPrefix(v, "data:image/") {
		return validateAvatarDataURI(v)
	}
	return validateAvatarURL(v)
}

// validateAvatarURL 完整 URL 校验:http/https、长度、主机名、无内嵌凭据
func validateAvatarURL(raw string) errors.Error {
	if len(raw) > avatarMaxURLLen {
		return avatarFieldError("头像 URL 过长(上限 %d 字符)", avatarMaxURLLen)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return avatarFieldError("头像 URL 格式不正确")
	}
	if scheme := strings.ToLower(u.Scheme); scheme != "http" && scheme != "https" {
		return avatarFieldError("头像仅支持完整 http/https URL 或 data:image 数据 URI")
	}
	if u.Host == "" {
		return avatarFieldError("头像 URL 缺少主机名")
	}
	if u.User != nil {
		return avatarFieldError("头像 URL 不允许包含用户名或密码")
	}
	return nil
}

// validateAvatarDataURI data URI 校验:MIME 白名单 + base64 编码 + 长度 + 字符集
func validateAvatarDataURI(uri string) errors.Error {
	if len(uri) > avatarMaxDataURILen {
		return avatarFieldError("头像 data URI 过长(上限 %d 字符)", avatarMaxDataURILen)
	}
	rest, ok := strings.CutPrefix(uri, "data:image/")
	if !ok {
		return avatarFieldError("头像仅支持完整 http/https URL 或 data:image 数据 URI")
	}
	semi := strings.IndexByte(rest, ';')
	if semi < 0 {
		return avatarFieldError("头像 data URI 缺少 MIME 参数")
	}
	if !avatarAllowedImageMIMEs[rest[:semi]] {
		return avatarFieldError("头像 data URI 仅支持 png/jpeg/webp/gif 图片")
	}
	tail := rest[semi+1:]
	if !strings.HasPrefix(tail, "base64,") {
		return avatarFieldError("头像 data URI 仅支持 base64 编码")
	}
	payload := tail[len("base64,"):]
	if payload == "" {
		return avatarFieldError("头像 data URI 缺少图片数据")
	}
	for i := 0; i < len(payload); i++ {
		if !isBase64Char(payload[i]) {
			return avatarFieldError("头像 data URI 包含非法字符")
		}
	}
	return nil
}

// isBase64Char base64 字母表(A-Za-z0-9+/=)
func isBase64Char(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '/' || c == '='
}

// avatarFieldError 头像字段错误(ErrorTypeInvalidRequest → Wrapper 映射 400)
func avatarFieldError(format string, args ...any) errors.Error {
	return errors.New(errors.ErrorTypeInvalidRequest, "handler.agent.avatar_validate", fmt.Sprintf(format, args...))
}
