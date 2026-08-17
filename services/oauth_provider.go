package services

import (
	"strings"

	"DomainManager/config"
)

// OauthProviderKindOauthGo and OauthProviderKindRainbow are the supported
// third-party login services selectable via the OAUTH_PROVIDER setting.
const (
	OauthProviderKindOauthGo = "oauthgo"
	OauthProviderKindRainbow = "rainbow"
)

// ActiveOauthKind returns the login service selected in system settings:
// "oauthgo" (default) or "rainbow" (彩虹聚合登录).
func ActiveOauthKind() string {
	kind := strings.TrimSpace(config.ActiveOauthProvider())
	if kind == "" {
		return OauthProviderKindOauthGo
	}
	return kind
}

// ActiveOauthEnabled reports whether the selected login service is configured.
func ActiveOauthEnabled() bool {
	if ActiveOauthKind() == OauthProviderKindRainbow {
		return RainbowEnabled()
	}
	return OauthGoEnabled()
}

// ActiveOauthProviders returns the enabled login channels of the active service.
func ActiveOauthProviders() ([]OauthGoProvider, error) {
	if ActiveOauthKind() == OauthProviderKindRainbow {
		return RainbowProviders()
	}
	return OauthGoProviders()
}

// ActiveOauthLoginURL requests a third-party authorization URL.
func ActiveOauthLoginURL(loginType, redirectURI string) (string, error) {
	if ActiveOauthKind() == OauthProviderKindRainbow {
		return RainbowLoginURL(loginType, redirectURI)
	}
	return OauthGoLoginURL(loginType, redirectURI)
}

// ActiveOauthUserInfo exchanges a one-time code for the user profile.
func ActiveOauthUserInfo(loginType, code string) (*OauthGoProfile, error) {
	if ActiveOauthKind() == OauthProviderKindRainbow {
		return RainbowUserInfo(loginType, code)
	}
	return OauthGoUserInfo(loginType, code)
}
