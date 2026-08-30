// Package turnpolicy 用户当轮约束与表达欲策略引擎(spec §7/§8)
// 本包是唯一策略源(§16.9):纯函数、无模型、无 DB;群聊编排器与单聊 handler 消费
package turnpolicy

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// ResponsePolicy 表达欲档位(spec §7.1)
type ResponsePolicy struct {
	Band             string
	VolunteerPercent int
	MaxSentences     int
	MaxTokens        int
}

// policyBands 固定映射表(spec §7.1 逐字,禁止配置化)
var policyBands = []struct {
	min          int
	max          int
	band         string
	maxSentences int
	maxTokens    int
}{
	{0, 20, "quiet", 1, 160},
	{21, 40, "reserved", 2, 256},
	{41, 60, "balanced", 3, 384},
	{61, 80, "talkative", 5, 640},
	{81, 100, "expansive", 8, 896},
}

// PolicyForProactivity 表达欲 0-100 → 固定策略档位(§7.1 表)
func PolicyForProactivity(proactivity int) ResponsePolicy {
	if proactivity < 0 {
		proactivity = 0
	}
	if proactivity > 100 {
		proactivity = 100
	}
	for _, b := range policyBands {
		if proactivity >= b.min && proactivity <= b.max {
			return ResponsePolicy{Band: b.band, VolunteerPercent: proactivity, MaxSentences: b.maxSentences, MaxTokens: b.maxTokens}
		}
	}
	// 不可达;防御性回退 quiet
	return ResponsePolicy{Band: "quiet", VolunteerPercent: proactivity, MaxSentences: 1, MaxTokens: 160}
}

// WantsToVolunteer SHA256(sessionUUID|agentID|userMessageUUID|round) 稳定桶(§7.1)。
// 哈希首 8 字节按大端取 mod 100,小于 VolunteerPercent 即自愿发言;禁止全局随机数。
func WantsToVolunteer(policy ResponsePolicy, sessionUUID string, agentID uint, userMessageUUID string, round int) bool {
	if policy.VolunteerPercent <= 0 {
		return false
	}
	if policy.VolunteerPercent >= 100 {
		return true
	}
	key := fmt.Sprintf("%s|%d|%s|%d", sessionUUID, agentID, userMessageUUID, round)
	sum := sha256.Sum256([]byte(key))
	v := binary.BigEndian.Uint64(sum[:8]) % 100
	return int(v) < policy.VolunteerPercent
}
