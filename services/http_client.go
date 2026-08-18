package services

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"DomainManager/config"
	"DomainManager/models"
)

// systemProxyURL returns the system-wide HTTP proxy (PROXY_URL) or "".
func systemProxyURL() string {
	if config.AppConfig != nil && config.AppConfig.ProxyURL != "" {
		return config.AppConfig.ProxyURL
	}
	return strings.TrimSpace(config.GetSetting(config.SettingProxyURL))
}

// NewRegistrarClient builds an HTTP client for registrar API calls. When the
// registrar opts in via UseProxy and a system proxy (PROXY_URL) is configured,
// requests are routed through the proxy. http:// and https:// proxies are
// supported (socks5:// is not).
func NewRegistrarClient(registrar models.Registrar, timeout time.Duration) *http.Client {
	client := &http.Client{Timeout: timeout}
	if !registrar.UseProxy {
		return client
	}
	raw := systemProxyURL()
	if raw == "" {
		return client
	}
	proxyURL, err := url.Parse(raw)
	if err != nil || proxyURL.Host == "" || (proxyURL.Scheme != "http" && proxyURL.Scheme != "https") {
		return client
	}
	client.Transport = &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		DialContext:         (&net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
	}
	return client
}
