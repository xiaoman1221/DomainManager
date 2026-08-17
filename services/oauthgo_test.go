package services

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"DomainManager/config"
)

func TestOauthGoSign(t *testing.T) {
	// Docs example: sign = md5("appid=10001&code=xxxx&type=gitee&key=your_appkey")
	config.AppConfig = &config.Config{OauthGoAppKey: "your_appkey"}

	got := oauthGoSign(map[string]string{
		"appid": "10001",
		"code":  "xxxx",
		"type":  "gitee",
	})
	want := "bca4e73b2fe524faf2faadd6f1770d54"
	if got != want {
		t.Fatalf("oauthGoSign() = %q, want %q", got, want)
	}
}

func TestOauthGoProvidersBareArray(t *testing.T) {
	// The OauthGo docs return a bare JSON array from /api/oauth/providers.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name":"gitee","display_name":"Gitee","category":"social"},{"name":"qq","display_name":"QQ","category":"social"}]`))
	}))
	defer srv.Close()

	config.AppConfig = &config.Config{
		OauthGoBaseURL: srv.URL,
		OauthGoAppID:   "10001",
		OauthGoAppKey:  "secret",
		OauthGoEnabled: true,
	}

	providers, err := OauthGoProviders()
	if err != nil {
		t.Fatalf("OauthGoProviders() error = %v", err)
	}
	if len(providers) != 2 || providers[0].Name != "gitee" || providers[1].Name != "qq" {
		t.Fatalf("OauthGoProviders() = %+v", providers)
	}
}

func TestOauthGoProvidersEnvelope(t *testing.T) {
	// Self-hosted variants may wrap the list in the unified envelope.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":0,"message":"success","data":[{"name":"qq","display_name":"QQ","category":"social"}]}`))
	}))
	defer srv.Close()

	config.AppConfig = &config.Config{
		OauthGoBaseURL: srv.URL,
		OauthGoAppID:   "10001",
		OauthGoAppKey:  "secret",
		OauthGoEnabled: true,
	}

	providers, err := OauthGoProviders()
	if err != nil {
		t.Fatalf("OauthGoProviders() envelope error = %v", err)
	}
	if len(providers) != 1 || providers[0].Name != "qq" {
		t.Fatalf("OauthGoProviders() envelope = %+v", providers)
	}
}
