package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"DomainManager/config"
)

// 彩虹聚合登录 (Rainbow login) client. It implements the connect.php protocol
// that is also supported by OauthGo (compatible with u.cccyun.cc):
//
//	1. GET  {base}/connect.php?act=login&appid&appkey&type&redirect_uri
//	       -> {"code":0,"msg":"succ","url":"<authorize url>"}
//	2. platform 302s back to redirect_uri with ?type=<t>&code=<code>
//	3. POST {base}/connect.php?act=callback&appid&appkey&type&code
//	       -> {"code":0,"msg":"succ","social_uid":"...","nickname":"...",
//	           "faceimg":"...","location":"...","gender":"...","ip":"..."}
//
// code=0 成功，code=1 失败，code=2 登录未完成。

const rainbowTimeout = 15 * time.Second

// rainbowResponse is the flat {code, msg, ...} object returned by connect.php.
type rainbowResponse struct {
	Code        int    `json:"code"`
	Msg         string `json:"msg"`
	URL         string `json:"url"`
	AccessToken string `json:"access_token"`
	SocialUID   string `json:"social_uid"`
	Faceimg     string `json:"faceimg"`
	Nickname    string `json:"nickname"`
	Location    string `json:"location"`
	Gender      string `json:"gender"`
	IP          string `json:"ip"`
}

// RainbowEnabled reports whether the 彩虹聚合登录 client is configured.
func RainbowEnabled() bool {
	return config.AppConfig != nil && config.AppConfig.RainbowEnabled
}

// RainbowProviders returns the channels offered by the connect.php protocol.
// There is no remote listing endpoint, so the supported set is built-in.
func RainbowProviders() ([]OauthGoProvider, error) {
	if !RainbowEnabled() {
		return nil, ErrOauthGoNotConfigured
	}
	return []OauthGoProvider{
		{Name: "qq", DisplayName: "QQ", Category: "social"},
		{Name: "wx", DisplayName: "微信", Category: "social"},
		{Name: "alipay", DisplayName: "支付宝", Category: "social"},
		{Name: "sina", DisplayName: "微博", Category: "social"},
		{Name: "baidu", DisplayName: "百度", Category: "social"},
		{Name: "douyin", DisplayName: "抖音", Category: "social"},
		{Name: "dingtalk", DisplayName: "钉钉", Category: "social"},
		{Name: "gitee", DisplayName: "Gitee", Category: "social"},
		{Name: "github", DisplayName: "GitHub", Category: "social"},
		{Name: "wework", DisplayName: "企业微信", Category: "social"},
	}, nil
}

// RainbowLoginURL requests a third-party authorization URL for the given type.
func RainbowLoginURL(loginType, redirectURI string) (string, error) {
	if !RainbowEnabled() {
		return "", ErrOauthGoNotConfigured
	}

	q := url.Values{}
	q.Set("act", "login")
	q.Set("appid", config.AppConfig.RainbowAppID)
	q.Set("appkey", config.AppConfig.RainbowAppKey)
	q.Set("type", loginType)
	q.Set("redirect_uri", redirectURI)

	apiURL := strings.TrimRight(config.AppConfig.RainbowBaseURL, "/") + "/connect.php?" + q.Encode()
	client := &http.Client{Timeout: rainbowTimeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to call 彩虹登录: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read 彩虹登录 response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("彩虹登录 returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var res rainbowResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return "", fmt.Errorf("failed to parse 彩虹登录 response: %w", err)
	}
	if res.Code != 0 {
		return "", fmt.Errorf("彩虹登录 login failed: %s", firstNonEmptyRainbow(res.Msg, "未知错误"))
	}
	if res.URL == "" {
		return "", errors.New("彩虹登录 returned an empty authorization url")
	}
	return res.URL, nil
}

// RainbowUserInfo exchanges a one-time code for the third-party user profile.
func RainbowUserInfo(loginType, code string) (*OauthGoProfile, error) {
	if !RainbowEnabled() {
		return nil, ErrOauthGoNotConfigured
	}

	form := url.Values{}
	form.Set("act", "callback")
	form.Set("appid", config.AppConfig.RainbowAppID)
	form.Set("appkey", config.AppConfig.RainbowAppKey)
	form.Set("type", loginType)
	form.Set("code", code)

	client := &http.Client{Timeout: rainbowTimeout}
	resp, err := client.PostForm(strings.TrimRight(config.AppConfig.RainbowBaseURL, "/")+"/connect.php", form)
	if err != nil {
		return nil, fmt.Errorf("failed to call 彩虹登录: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read 彩虹登录 response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("彩虹登录 returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var res rainbowResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("failed to parse 彩虹登录 response: %w", err)
	}
	if res.Code == 2 {
		return nil, ErrOauthGoPending
	}
	if res.Code != 0 {
		return nil, fmt.Errorf("彩虹登录 callback failed: %s", firstNonEmptyRainbow(res.Msg, "未知错误"))
	}
	if res.SocialUID == "" {
		return nil, errors.New("彩虹登录 returned empty social_uid")
	}

	return &OauthGoProfile{
		Type:        loginType,
		OpenID:      res.SocialUID,
		Nickname:    res.Nickname,
		Avatar:      res.Faceimg,
		Gender:      res.Gender,
		Location:    res.Location,
		AccessToken: res.AccessToken,
	}, nil
}

func firstNonEmptyRainbow(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
