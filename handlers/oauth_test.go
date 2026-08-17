package handlers

import (
	"errors"
	"path/filepath"
	"testing"

	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/services"
)

func TestFindOrCreateOauthUser(t *testing.T) {
	config.AppConfig = &config.Config{
		DBPath: filepath.Join(t.TempDir(), "test.db"),
		JWTKey: "test-jwt-key",
	}
	database.Init()
	t.Cleanup(func() {
		if sqlDB, err := database.DB.DB(); err == nil {
			sqlDB.Close()
		}
	})

	info := &services.OauthGoProfile{
		Type:     "qq",
		OpenID:   "10001",
		Nickname: "张三",
		Avatar:   "https://example.com/avatar.png",
		Email:    "zhangsan@example.com",
	}

	// First login creates a bound local user.
	u1, err := findOrCreateOauthUser(info)
	if err != nil {
		t.Fatalf("findOrCreateOauthUser() error = %v", err)
	}
	if u1.Username != "oauth_qq_10001" {
		t.Fatalf("username = %q, want oauth_qq_10001", u1.Username)
	}
	if u1.Nickname != "张三" {
		t.Fatalf("nickname = %q, want 张三", u1.Nickname)
	}
	if u1.OauthProvider != "qq" || u1.OauthOpenID != "10001" {
		t.Fatalf("binding = %s/%s, want qq/10001", u1.OauthProvider, u1.OauthOpenID)
	}
	if u1.RoleGroup != "user" {
		t.Fatalf("role_group = %q, want user", u1.RoleGroup)
	}
	if u1.Password == "" {
		t.Fatal("expected an unguessable password hash for OAuth users")
	}

	// Second login with the same provider account maps to the same user.
	u2, err := findOrCreateOauthUser(info)
	if err != nil {
		t.Fatalf("findOrCreateOauthUser() second call error = %v", err)
	}
	if u2.ID != u1.ID {
		t.Fatalf("second call returned user %d, want same user %d", u2.ID, u1.ID)
	}

	// A different provider openid gets a distinct user.
	other := &services.OauthGoProfile{Type: "gitee", OpenID: "10001", Nickname: "张三"}
	u3, err := findOrCreateOauthUser(other)
	if err != nil {
		t.Fatalf("findOrCreateOauthUser() other error = %v", err)
	}
	if u3.ID == u1.ID {
		t.Fatal("different provider should map to a different user")
	}
}

func TestFindOrCreateOauthUserAutoRegisterDisabled(t *testing.T) {
	config.AppConfig = &config.Config{
		DBPath:     filepath.Join(t.TempDir(), "test.db"),
		JWTKey:     "test-jwt-key",
		DBSettings: map[string]string{},
	}
	database.Init()
	t.Cleanup(func() {
		if sqlDB, err := database.DB.DB(); err == nil {
			sqlDB.Close()
		}
	})

	// While auto-registration is enabled the first login binds a local user.
	bound := &services.OauthGoProfile{Type: "wechat", OpenID: "30003", Nickname: "王五"}
	u1, err := findOrCreateOauthUser(bound)
	if err != nil {
		t.Fatalf("findOrCreateOauthUser() error = %v", err)
	}

	// Disable automatic registration; the bound identity must still log in.
	config.AppConfig.DBSettings[config.SettingOauthGoAutoRegister] = "false"
	u2, err := findOrCreateOauthUser(bound)
	if err != nil {
		t.Fatalf("bound identity rejected after disabling auto-register: %v", err)
	}
	if u2.ID != u1.ID {
		t.Fatalf("bound identity returned user %d, want same user %d", u2.ID, u1.ID)
	}

	// An unbound third-party identity must be rejected instead of creating a
	// local account.
	unbound := &services.OauthGoProfile{Type: "qq", OpenID: "20002", Nickname: "李四"}
	if _, err := findOrCreateOauthUser(unbound); !errors.Is(err, errOauthNotBound) {
		t.Fatalf("findOrCreateOauthUser() error = %v, want errOauthNotBound", err)
	}
}

func TestOauthTicket(t *testing.T) {
	ticket := issueOauthTicket("jwt-token-abc")
	if ticket == "" {
		t.Fatal("issueOauthTicket() returned empty ticket")
	}

	token, ok := redeemOauthTicket(ticket)
	if !ok || token != "jwt-token-abc" {
		t.Fatalf("redeemOauthTicket() = %q, %v; want jwt-token-abc, true", token, ok)
	}

	// Tickets are one-time.
	if _, ok := redeemOauthTicket(ticket); ok {
		t.Fatal("ticket should be single-use")
	}

	// Unknown tickets are rejected.
	if _, ok := redeemOauthTicket("nope"); ok {
		t.Fatal("unknown ticket should be rejected")
	}
}
