package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"DomainManager/config"
	"DomainManager/models"
)

func TestNewRegistrarClientNoProxy(t *testing.T) {
	config.AppConfig = &config.Config{ProxyURL: "http://127.0.0.1:7890", DBSettings: map[string]string{}}

	// Without UseProxy the client is plain (no proxy transport).
	c := NewRegistrarClient(models.Registrar{UseProxy: false}, 15)
	if c.Transport != nil {
		t.Fatalf("unexpected transport: %+v", c.Transport)
	}

	// UseProxy with an empty system proxy also stays direct.
	config.AppConfig.ProxyURL = ""
	c = NewRegistrarClient(models.Registrar{UseProxy: true}, 15)
	if c.Transport != nil {
		t.Fatal("expected direct client when no proxy is configured")
	}
}

func TestNewRegistrarClientProxy(t *testing.T) {
	config.AppConfig = &config.Config{ProxyURL: "http://127.0.0.1:7890", DBSettings: map[string]string{}}

	c := NewRegistrarClient(models.Registrar{UseProxy: true}, 15)
	if c.Transport == nil {
		t.Fatal("expected a proxy transport")
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", c.Transport)
	}
	if tr.Proxy == nil {
		t.Fatal("expected a Proxy function on the transport")
	}
	req, _ := http.NewRequest("GET", "https://api.example.com", nil)
	proxyURL, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("proxy lookup error: %v", err)
	}
	if proxyURL == nil || proxyURL.String() != "http://127.0.0.1:7890" {
		t.Fatalf("proxy = %v, want http://127.0.0.1:7890", proxyURL)
	}
}

func TestSystemProxyURLPrecedence(t *testing.T) {
	// The runtime value (kept in sync from the DB by ApplyDBSettings) wins.
	config.AppConfig = &config.Config{
		ProxyURL:   "http://runtime:1",
		DBSettings: map[string]string{config.SettingProxyURL: "http://db:2"},
	}
	if got := systemProxyURL(); got != "http://runtime:1" {
		t.Fatalf("systemProxyURL() = %q, want runtime value", got)
	}

	// Falls back to the DB setting when the runtime value is empty.
	config.AppConfig = &config.Config{
		ProxyURL:   "",
		DBSettings: map[string]string{config.SettingProxyURL: "http://db:2"},
	}
	if got := systemProxyURL(); got != "http://db:2" {
		t.Fatalf("systemProxyURL() = %q, want db value", got)
	}
}

func TestNewRegistrarClientRoutesThroughProxy(t *testing.T) {
	proxied := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = true
		if r.URL.Host != "target.invalid" {
			t.Errorf("proxy received host %q, want target.invalid", r.URL.Host)
		}
		w.Write([]byte("ok"))
	}))
	defer proxy.Close()

	config.AppConfig = &config.Config{ProxyURL: proxy.URL, DBSettings: map[string]string{}}
	client := NewRegistrarClient(models.Registrar{UseProxy: true}, 15*time.Second)
	resp, err := client.Get("http://target.invalid/x")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	resp.Body.Close()
	if !proxied {
		t.Fatal("request was not routed through the proxy")
	}
}
