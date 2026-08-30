package turnpolicy

import (
	"strings"
)

// FrustrationLevel 用户烦躁级别(§8.1)
type FrustrationLevel string

const (
	FrustrationCalm    FrustrationLevel = "calm"
	FrustrationAnnoyed FrustrationLevel = "annoyed"
)

// UserTurnConstraints 用户当轮要求(§8.1;只保留可信布尔值与最新问题原文)
type UserTurnConstraints struct {
	LatestQuestion string
	Concise        bool
	Detailed       bool
	WantsStop      bool
	Frustration    FrustrationLevel
	OneEach        bool
}

// 停止标记:明确结束意图(§8.2「烦」本身不等于停止)
var stopMarkers = []string{"别说了", "不用说了", "不用再说", "够了", "到此为止", "别再说了", "不用回了", "不聊了"}

// 简短标记
var conciseMarkers = []string{"直接说结论", "说重点", "简短", "别啰嗦", "少说", "简洁", "一句话说清", "简而言之"}

// 详细标记
var detailedMarkers = []string{"详细", "展开", "多说", "说清楚", "完整分析", "具体讲讲", "深入"}

// 烦躁标记
var annoyedMarkers = []string{"烦", "够了", "别废话", "不耐烦", "气死", "无语"}

// 否定前缀(「不/别/莫/无须」+ 立即跟随的停止/烦躁词 → 否定其意义)
var negationPrefixes = []string{"不要", "别", "不", "莫", "无须", "无需"}

// ExtractUserTurnConstraints 无模型纯函数提取用户当轮要求(§8.1)。
// 顺序:1) 否定与继续表达优先(「不要停,继续说」→ 不停止);2) 再识别停止;3) 简短/详细/烦躁/每人一句。
func ExtractUserTurnConstraints(userMessage string) UserTurnConstraints {
	c := UserTurnConstraints{
		LatestQuestion: strings.TrimSpace(userMessage),
		Frustration:    FrustrationCalm,
	}
	msg := strings.TrimSpace(userMessage)
	if msg == "" {
		return c
	}
	lower := strings.ToLower(msg)

	// 每人一句
	for _, m := range []string{"每人一句", "每人都说一句", "每人说一句", "一人一句", "一个一个来"} {
		if strings.Contains(lower, m) {
			c.OneEach = true
			break
		}
	}

	// 继续表达(「继续说/不要停/别停/继续」)优先否定停止
	continueSignal := strings.Contains(lower, "继续说") ||
		strings.Contains(lower, "接着说") ||
		strings.Contains(lower, "不要停") ||
		strings.Contains(lower, "别停") ||
		strings.Contains(lower, "别断")

	// 停止:命中的停止标记未被否定前缀修饰
	if !continueSignal {
		for _, marker := range stopMarkers {
			idx := strings.Index(lower, marker)
			if idx < 0 {
				continue
			}
			negated := false
			for _, neg := range negationPrefixes {
				start := idx - len(neg)
				if start >= 0 && lower[start:idx] == neg {
					negated = true
					break
				}
			}
			if !negated {
				c.WantsStop = true
				break
			}
		}
		// 独立「停」字(前后非汉字)
		if !c.WantsStop && isBareStop(lower) {
			c.WantsStop = true
		}
	}

	// 简短
	for _, marker := range conciseMarkers {
		if strings.Contains(lower, marker) {
			c.Concise = true
			break
		}
	}
	// 详细
	for _, marker := range detailedMarkers {
		if strings.Contains(lower, marker) {
			c.Detailed = true
			break
		}
	}
	// 烦躁(排除「不烦」)
	for _, marker := range annoyedMarkers {
		if strings.Contains(lower, marker) {
			negated := false
			for _, neg := range negationPrefixes {
				if strings.Contains(lower, neg+marker) {
					negated = true
					break
				}
			}
			if !negated {
				c.Frustration = FrustrationAnnoyed
				break
			}
		}
	}
	return c
}

// isBareStop 独立「停」字(前后无汉字);按 rune 遍历避免字节下标错位
func isBareStop(lower string) bool {
	rs := []rune(lower)
	for i, r := range rs {
		if r != '停' {
			continue
		}
		beforeOK := i == 0 || !isHan(rs[i-1])
		afterOK := i+1 >= len(rs) || !isHan(rs[i+1])
		if beforeOK && afterOK {
			return true
		}
	}
	return false
}

func isHan(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF)
}
