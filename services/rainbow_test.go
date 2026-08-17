package services

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"DomainManager/config"
)

func rainbowTestConfig(srv *httptest.Server) {
	config.AppConfig = &config.Config{
		RainbowBaseURL: srv.URL,
		RainbowAppID:   "10001",
		RainbowAppKey:  "secret",
		RainbowEnabled: true,
		DBSettings:     map[string]string{},
	}
}

func TestRainbowLoginURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("act"); got != "login" {
			t.Errorf("act = %q, want login", got)
		}
		if got := r.URL.Query().Get("appid"); got != "10001" {
			t.Errorf("appid = %q, want 10001", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"succ","url":"https://example.com/authorize"}`))
	}))
	defer srv.Close()
	rainbowTestConfig(srv)

	u, err := RainbowLoginURL("qq", "https://example.com/callback")
	if err != nil {
		t.Fatalf("RainbowLoginURL() error = %v", err)
	}
	if u != "https://example.com/authorize" {
		t.Fatalf("RainbowLoginURL() url = %q", u)
	}
}

func TestRainbowLoginURLFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":1,"msg":"redirect_uri 不在白名单"}`))
	}))
	defer srv.Close()
	rainbowTestConfig(srv)

	if _, err := RainbowLoginURL("qq", "https://example.com/callback"); err == nil {
		t.Fatal("expected login URL request to fail")
	}
}

func TestRainbowUserInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.FormValue("act"); got != "callback" {
			t.Errorf("act = %q, want callback", got)
		}
		if got := r.FormValue("code"); got != "code-123" {
			t.Errorf("code = %q, want code-123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"msg":"succ","access_token":"tok","social_uid":"u-1","faceimg":"https://example.com/a.png","nickname":"张三","location":"广东深圳","gender":"男"}`))
	}))
	defer srv.Close()
	rainbowTestConfig(srv)

	info, err := RainbowUserInfo("qq", "code-123")
	if err != nil {
		t.Fatalf("RainbowUserInfo() error = %v", err)
	}
	if info.Type != "qq" || info.OpenID != "u-1" || info.Nickname != "张三" || info.Avatar != "https://example.com/a.png" {
		t.Fatalf("RainbowUserInfo() = %+v", info)
	}
}

func TestRainbowUserInfoPending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":2,"msg":"未完成登录"}`))
	}))
	defer srv.Close()
	rainbowTestConfig(srv)

	if _, err := RainbowUserInfo("qq", "stale"); !errors.Is(err, ErrOauthGoPending) {
		t.Fatalf("RainbowUserInfo() error = %v, want ErrOauthGoPending", err)
	}
}

func TestActiveOauthKind(t *testing.T) {
	config.AppConfig = &config.Config{DBSettings: map[string]string{}}

	if got := ActiveOauthKind(); got != OauthProviderKindOauthGo {
		t.Fatalf("ActiveOauthKind() default = %q, want oauthgo", got)
	}

	config.AppConfig.DBSettings[config.SettingOauthProvider] = OauthProviderKindRainbow
	if got := ActiveOauthKind(); got != OauthProviderKindRainbow {
		t.Fatalf("ActiveOauthKind() = %q, want rainbow", got)
	}
}
