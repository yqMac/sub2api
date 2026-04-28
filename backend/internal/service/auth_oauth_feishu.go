// [bmai-fork] feishu
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// FeishuSyntheticEmail 基于飞书 subject (open_id) 生成稳定的合成邮箱。
// 用于在 userinfo 不返回真实邮箱、或 require_enterprise_email=true 但 enterprise_email 缺失时兜底。
func FeishuSyntheticEmail(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("feishu\x1f" + subject))
	return "feishu-" + hex.EncodeToString(sum[:16]) + FeishuConnectSyntheticEmailDomain
}

// IsFeishuTenantAllowed 判定 tenant_key 是否在白名单内。
// allowlist 为空表示不限制（仅适合内网灰度，生产应配置）。
func IsFeishuTenantAllowed(allowlist []string, tenantKey string) bool {
	tenantKey = strings.TrimSpace(strings.ToLower(tenantKey))
	if len(allowlist) == 0 {
		return true
	}
	for _, t := range allowlist {
		if strings.EqualFold(strings.TrimSpace(t), tenantKey) {
			return true
		}
	}
	return false
}
