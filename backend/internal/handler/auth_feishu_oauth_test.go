// [bmai-fork] feishu
package handler

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestFeishuParseUserInfo_PrefersEnterpriseEmailWhenRequired(t *testing.T) {
	body := `{
		"sub":"on_abc",
		"name":"张三",
		"picture":"https://x.example/a.jpg",
		"email":"zhangsan@personal.example",
		"enterprise_email":"zhangsan@company.example",
		"tenant_key":"t_xyz",
		"open_id":"on_abc",
		"union_id":"un_abc"
	}`
	cfg := config.FeishuConnectConfig{RequireEnterpriseEmail: true}
	res, err := feishuParseUserInfo(body, cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Email != "zhangsan@company.example" {
		t.Fatalf("expected enterprise email, got %s", res.Email)
	}
	if res.Subject != "on_abc" || res.TenantKey != "t_xyz" {
		t.Fatalf("missing subject/tenant: %+v", res)
	}
}

func TestFeishuParseUserInfo_FallsBackToSyntheticEmailWhenMissing(t *testing.T) {
	body := `{"sub":"on_abc","tenant_key":"t","name":""}`
	cfg := config.FeishuConnectConfig{}
	res, err := feishuParseUserInfo(body, cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.HasSuffix(res.Email, "@feishu-connect.invalid") {
		t.Fatalf("expected synthetic email, got %s", res.Email)
	}
	if res.Username == "" {
		t.Fatalf("username should fall back to feishu_<subject>")
	}
}

func TestFeishuParseUserInfo_RejectsMissingSubject(t *testing.T) {
	body := `{"name":"x"}`
	if _, err := feishuParseUserInfo(body, config.FeishuConnectConfig{}); err == nil {
		t.Fatal("expected error for missing subject")
	}
}

func TestFeishuParseUserInfo_FallsBackToNormalEmailWhenNoEnterprise(t *testing.T) {
	body := `{"sub":"on_xyz","email":"user@example.com","tenant_key":"t1"}`
	cfg := config.FeishuConnectConfig{}
	res, err := feishuParseUserInfo(body, cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Email != "user@example.com" {
		t.Fatalf("expected normal email, got %s", res.Email)
	}
}

func TestFeishuParseUserInfo_RequireEnterpriseEmail_FallsBackToSyntheticWhenMissing(t *testing.T) {
	body := `{"sub":"on_xyz","email":"user@example.com","tenant_key":"t1"}`
	cfg := config.FeishuConnectConfig{RequireEnterpriseEmail: true}
	res, err := feishuParseUserInfo(body, cfg)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.HasSuffix(res.Email, "@feishu-connect.invalid") {
		t.Fatalf("expected synthetic email when enterprise_email missing, got %s", res.Email)
	}
}

func TestIsSafeFeishuSubject(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"on_abc123", true},
		{"ou-ABC.def_123", true},
		{"", false},
		{strings.Repeat("a", 129), false},
		{"has space", false},
		{"has@at", false},
		{"has/slash", false},
	}
	for _, tc := range cases {
		got := isSafeFeishuSubject(tc.input)
		if got != tc.want {
			t.Errorf("isSafeFeishuSubject(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
