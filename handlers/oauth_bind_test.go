package handlers

import (
	"path/filepath"
	"testing"

	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/models"
	"DomainManager/services"
)

func TestFilterOauthGoProviders(t *testing.T) {
	config.AppConfig = &config.Config{DBSettings: map[string]string{}}
	providers := []services.OauthGoProvider{
		{Name: "qq", DisplayName: "QQ"},
		{Name: "github", DisplayName: "GitHub"},
		{Name: "wechat", DisplayName: "微信"},
	}

	// Empty selection -> all providers.
	if got := filterOauthGoProviders(providers); len(got) != 3 {
		t.Fatalf("empty selection returned %d providers, want 3", len(got))
	}

	// Selected types -> only those.
	config.AppConfig.DBSettings[config.SettingOauthGoEnabledTypes] = `["qq","github"]`
	got := filterOauthGoProviders(providers)
	if len(got) != 2 || got[0].Name != "qq" || got[1].Name != "github" {
		t.Fatalf("filtered providers = %+v, want [qq github]", got)
	}

	// Invalid JSON -> all providers.
	config.AppConfig.DBSettings[config.SettingOauthGoEnabledTypes] = "not-json"
	if got := filterOauthGoProviders(providers); len(got) != 3 {
		t.Fatalf("invalid selection returned %d providers, want 3", len(got))
	}
}

func TestUpsertOauthBindingDuplicate(t *testing.T) {
	config.AppConfig = &config.Config{DBPath: filepath.Join(t.TempDir(), "test.db"), JWTKey: "test"}
	database.Init()
	t.Cleanup(func() {
		if sqlDB, err := database.DB.DB(); err == nil {
			sqlDB.Close()
		}
	})

	u1 := models.User{Username: "u_one", Email: "u1@example.com", Password: "x"}
	u2 := models.User{Username: "u_two", Email: "u2@example.com", Password: "x"}
	if err := database.DB.Create(&u1).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.DB.Create(&u2).Error; err != nil {
		t.Fatal(err)
	}

	info := &services.OauthGoProfile{Type: "qq", OpenID: "10001", Nickname: "张三"}

	if err := upsertOauthBinding(u1.ID, info); err != nil {
		t.Fatalf("first bind error: %v", err)
	}
	// Re-binding the same identity to the same user is fine.
	if err := upsertOauthBinding(u1.ID, info); err != nil {
		t.Fatalf("re-bind same user error: %v", err)
	}
	// Binding the identity to another user must fail.
	if err := upsertOauthBinding(u2.ID, info); err == nil {
		t.Fatal("expected duplicate binding to be rejected")
	}

	// A different provider with the same openid is independent.
	other := &services.OauthGoProfile{Type: "gitee", OpenID: "10001", Nickname: "张三"}
	if err := upsertOauthBinding(u2.ID, other); err != nil {
		t.Fatalf("bind other provider error: %v", err)
	}

	var count int64
	database.DB.Model(&models.UserOAuthBinding{}).Count(&count)
	if count != 2 {
		t.Fatalf("binding count = %d, want 2", count)
	}
}
