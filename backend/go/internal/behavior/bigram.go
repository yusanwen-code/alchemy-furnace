package behavior

import (
	"strings"
	"unicode"
)

// bigramSet 规范化字符串的字符 bigram 集合
// 规范化:unicode 转小写、剔除空白;长度 <2 返回空集合
func bigramSet(s string) map[string]struct{} {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	runes := []rune(b.String())
	if len(runes) < 2 {
		return nil
	}
	set := make(map[string]struct{}, len(runes)-1)
	for i := 0; i < len(runes)-1; i++ {
		set[string(runes[i:i+2])] = struct{}{}
	}
	return set
}

// BigramJaccard 规范化字符 bigram 的 Jaccard 相似度(0-1;任一输入为空/过短返回 0)
func BigramJaccard(a, b string) float64 {
	sa, sb := bigramSet(a), bigramSet(b)
	if len(sa) == 0 || len(sb) == 0 {
		return 0
	}
	inter, union := 0, len(sa)
	for k := range sb {
		if _, ok := sa[k]; ok {
			inter++
		} else {
			union++
		}
	}
	return float64(inter) / float64(union)
}
