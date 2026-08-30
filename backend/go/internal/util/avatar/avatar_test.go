package avatar

import (
	"strings"
	"testing"
)

// 共享头像契约:
//   - 空值合法(清空头像)
//   - 完整 http/https URL:长度 ≤2048,不允许内嵌凭据(user:pass@)
//   - data:image/(png|jpeg|webp|gif);base64,:URI 总长 ≤1_500_000 字符,payload 仅 base64 字符
//   - 其余(相对路径 / javascript: / vbscript: / blob: / 其他 MIME / 超长)→ 错误

func TestValidate(t *testing.T) {
	valid := []string{
		"",
		"https://cdn.example.com/a.png",
		"http://cdn.example.com/avatar.png?size=64",
		"data:image/png;base64,AAAA",
		"data:image/jpeg;base64,AAAA",
		"data:image/webp;base64,AAAA",
		"data:image/gif;base64,AAAA",
	}
	for _, value := range valid {
		if err := Validate(value); err != nil {
			t.Fatalf("Validate(%q) = %v, want nil", value, err)
		}
	}

	invalid := []string{
		"/a.png",                       // 相对路径
		"javascript:alert(1)",          // 可执行协议
		"vbscript:alert(1)",            // 可执行协议
		"blob:https://example.com/abc", // blob 协议
		"http://user:pass@cdn.example.com/a.png", // 内嵌凭据
		"https://user@cdn.example.com/a.png",     // 内嵌用户名
		"data:image/svg+xml;base64,AAAA",         // SVG 不在白名单
		"data:image/png,AAAA",                    // 非 base64 编码
		"data:image/png;base64,@@@",              // 非法字符
		"data:image/png;base64,A A",              // 空格非法
	}
	for _, value := range invalid {
		if err := Validate(value); err == nil {
			t.Fatalf("Validate(%q) = nil, want error", value)
		}
	}
}

func TestValidateURLLengthBoundary(t *testing.T) {
	prefix := "https://cdn.example.com/"
	ok := prefix + strings.Repeat("a", MaxURLLen-len(prefix))
	if len(ok) != MaxURLLen {
		t.Fatalf("测试用例构造错误: len=%d, want %d", len(ok), MaxURLLen)
	}
	if err := Validate(ok); err != nil {
		t.Fatalf("Validate(恰好 MaxURLLen) = %v, want nil", err)
	}
	over := prefix + strings.Repeat("a", MaxURLLen-len(prefix)+1)
	if len(over) != MaxURLLen+1 {
		t.Fatalf("测试用例构造错误: len=%d, want %d", len(over), MaxURLLen+1)
	}
	if err := Validate(over); err == nil {
		t.Fatal("Validate(超 MaxURLLen) = nil, want error")
	}
}

func TestValidateDataURILengthBoundary(t *testing.T) {
	prefix := "data:image/png;base64,"
	ok := prefix + strings.Repeat("A", MaxDataURILen-len(prefix))
	if len(ok) != MaxDataURILen {
		t.Fatalf("测试用例构造错误: len=%d, want %d", len(ok), MaxDataURILen)
	}
	if err := Validate(ok); err != nil {
		t.Fatalf("Validate(恰好 MaxDataURILen) = %v, want nil", err)
	}
	over := prefix + strings.Repeat("A", MaxDataURILen-len(prefix)+1)
	if len(over) != MaxDataURILen+1 {
		t.Fatalf("测试用例构造错误: len=%d, want %d", len(over), MaxDataURILen+1)
	}
	if err := Validate(over); err == nil {
		t.Fatal("Validate(超 MaxDataURILen) = nil, want error")
	}
}
