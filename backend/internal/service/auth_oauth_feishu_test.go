package service

import (
	"strings"
	"testing"
)

func TestFeishuSyntheticEmail_ProducesStableLowercaseSuffix(t *testing.T) {
	got := FeishuSyntheticEmail("on_abc123XYZ")
	if !strings.HasSuffix(got, FeishuConnectSyntheticEmailDomain) {
		t.Fatalf("expected suffix %s, got %s", FeishuConnectSyntheticEmailDomain, got)
	}
	if !strings.HasPrefix(got, "feishu-") {
		t.Fatalf("expected prefix feishu-, got %s", got)
	}
	if got != FeishuSyntheticEmail("on_abc123XYZ") {
		t.Fatalf("synthetic email is not stable for the same subject")
	}
	if got == FeishuSyntheticEmail("on_other") {
		t.Fatalf("different subjects must produce different emails")
	}
}

func TestFeishuSyntheticEmail_EmptySubjectReturnsEmpty(t *testing.T) {
	if got := FeishuSyntheticEmail(""); got != "" {
		t.Fatalf("expected empty string for empty subject, got %s", got)
	}
	if got := FeishuSyntheticEmail("   "); got != "" {
		t.Fatalf("expected empty string for whitespace subject, got %s", got)
	}
}

func TestIsFeishuTenantAllowed_NilOrEmptyAllowAll(t *testing.T) {
	if !IsFeishuTenantAllowed(nil, "tenant_a") {
		t.Fatal("nil allowlist should permit any tenant")
	}
	if !IsFeishuTenantAllowed([]string{}, "tenant_a") {
		t.Fatal("empty allowlist should permit any tenant")
	}
}

func TestIsFeishuTenantAllowed_MatchExact(t *testing.T) {
	allow := []string{"tenant_a", "tenant_b"}
	if !IsFeishuTenantAllowed(allow, "tenant_b") {
		t.Fatal("expected tenant_b to be allowed")
	}
	if IsFeishuTenantAllowed(allow, "tenant_c") {
		t.Fatal("expected tenant_c to be rejected")
	}
}

func TestIsFeishuTenantAllowed_TrimsAndCaseInsensitive(t *testing.T) {
	allow := []string{"  Tenant_A  "}
	if !IsFeishuTenantAllowed(allow, "tenant_a") {
		t.Fatal("expected case-insensitive trim match")
	}
}

func TestIsFeishuTenantAllowed_EmptyTenantWithNonEmptyAllowlistRejects(t *testing.T) {
	allow := []string{"tenant_a"}
	if IsFeishuTenantAllowed(allow, "") {
		t.Fatal("empty tenant with non-empty allowlist should be rejected")
	}
}
