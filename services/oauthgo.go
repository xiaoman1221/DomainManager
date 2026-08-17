package services

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"DomainManager/config"
)

// OauthGo is a third-party login aggregation service (https://o.1v.fit/docs).
// This client uses the self-hosted REST interface (/api/v1/oauth/*).

var (
	// ErrOauthGoNotConfigured is returned when OauthGo settings are missing.
	ErrOauthGoNotConfigured = errors.New("OauthGo is not configured")
	// ErrOauthGoPending means the one-time code is invalid/used/expired.
	ErrOauthGoPending = errors.New("OauthGo login not completed")
)

const oauthGoTimeout = 15 * time.Second

// OauthGoProvider describes an enabled third-party login channel.
type OauthGoProvider struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Category    string `json:"category"`
}

// OauthGoProfile is the user profile returned by /api/v1/oauth/userinfo.
type OauthGoProfile struct {
	Type        string `json:"type"`
	OpenID      string `json:"openid"`
	UnionID     string `json:"unionid"`
	Nickname    string `json:"nickname"`
	Avatar      string `json:"avatar"`
	Email       string `json:"email"`
	Gender      string `json:"gender"`
	Location    string `json:"location"`
	AccessToken string `json:"access_token"`
}

// oauthGoResponse is the unified {code, message, data} envelope.
type oauthGoResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func oauthGoEnabled() bool {
	return config.AppConfig.OauthGoEnabled
}

func oauthGoBaseURL() string {
	return config.AppConfig.OauthGoBaseURL
}

// oauthGoSign computes the MD5 signature required by the OauthGo REST API:
// params sorted by key ascending joined as k1=v1&k2=v2, then &key=<appkey>,
// MD5 hex (lowercase).
func oauthGoSign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	b.WriteString("&key=")
	b.WriteString(config.AppConfig.OauthGoAppKey)

	sum := md5.Sum([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func oauthGoPost(path string, payload map[string]string) (*oauthGoResponse, error) {
	if !oauthGoEnabled() {
		return nil, ErrOauthGoNotConfigured
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	client := &http.Client{Timeout: oauthGoTimeout}
	resp, err := client.Post(oauthGoBaseURL()+path, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call OauthGo: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read OauthGo response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OauthGo returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var res oauthGoResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("failed to parse OauthGo response: %w", err)
	}
	return &res, nil
}

// OauthGoEnabled reports whether OauthGo is configured.
func OauthGoEnabled() bool {
	return oauthGoEnabled()
}

// OauthGoProviders returns the list of enabled login channels.
func OauthGoProviders() ([]OauthGoProvider, error) {
	if !oauthGoEnabled() {
		return nil, ErrOauthGoNotConfigured
	}

	client := &http.Client{Timeout: oauthGoTimeout}
	resp, err := client.Get(oauthGoBaseURL() + "/api/oauth/providers")
	if err != nil {
		return nil, fmt.Errorf("failed to call OauthGo: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read OauthGo response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OauthGo returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var providers []OauthGoProvider
	if err := json.Unmarshal(data, &providers); err != nil {
		// Self-hosted variants may still wrap the list in the unified
		// {code, message, data} envelope; accept both shapes.
		var res oauthGoResponse
		if uerr := json.Unmarshal(data, &res); uerr != nil {
			return nil, fmt.Errorf("failed to parse OauthGo providers: %w", err)
		}
		if res.Code != 0 {
			return nil, fmt.Errorf("OauthGo providers failed: %s", res.Message)
		}
		if err := json.Unmarshal(res.Data, &providers); err != nil {
			return nil, fmt.Errorf("failed to parse OauthGo providers data: %w", err)
		}
	}
	return providers, nil
}

// OauthGoLoginURL requests a third-party authorization URL for the given type.
func OauthGoLoginURL(loginType, redirectURI string) (string, error) {
	res, err := oauthGoPost("/api/v1/oauth/login", map[string]string{
		"appid":        config.AppConfig.OauthGoAppID,
		"appkey":       config.AppConfig.OauthGoAppKey,
		"type":         loginType,
		"redirect_uri": redirectURI,
	})
	if err != nil {
		return "", err
	}
	if res.Code != 0 {
		return "", fmt.Errorf("OauthGo login failed: %s", res.Message)
	}

	var data struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(res.Data, &data); err != nil {
		return "", fmt.Errorf("failed to parse OauthGo login data: %w", err)
	}
	if data.URL == "" {
		return "", errors.New("OauthGo returned an empty authorization url")
	}
	return data.URL, nil
}

// OauthGoUserInfo exchanges a one-time code for the third-party user profile.
func OauthGoUserInfo(loginType, code string) (*OauthGoProfile, error) {
	params := map[string]string{
		"appid": config.AppConfig.OauthGoAppID,
		"type":  loginType,
		"code":  code,
	}

	res, err := oauthGoPost("/api/v1/oauth/userinfo", map[string]string{
		"appid": params["appid"],
		"type":  params["type"],
		"code":  params["code"],
		"sign":  oauthGoSign(params),
	})
	if err != nil {
		return nil, err
	}
	if res.Code == 2 {
		return nil, ErrOauthGoPending
	}
	if res.Code != 0 {
		return nil, fmt.Errorf("OauthGo userinfo failed: %s", res.Message)
	}

	var info OauthGoProfile
	if err := json.Unmarshal(res.Data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse OauthGo userinfo: %w", err)
	}
	if info.OpenID == "" {
		return nil, errors.New("OauthGo returned empty openid")
	}
	return &info, nil
}
