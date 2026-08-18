package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// DB-backed runtime settings. These are migrated out of .env into the
// system_settings table (managed in the admin settings UI). Environment
// variables are only used as initial defaults on first startup.
const (
	SettingWhoisAPIURL        = "WHOIS_API_URL"
	SettingUaWhoisServer      = "UA_WHOIS_SERVER"
	SettingICPAPIURL          = "ICP_API_URL"
	SettingDigitalPlatRDAPURL = "DIGITALPLAT_RDAP_URL"

	SettingOauthGoBaseURL      = "OAUTHGO_BASE_URL"
	SettingOauthGoAppID        = "OAUTHGO_APP_ID"
	SettingOauthGoAppKey       = "OAUTHGO_APP_KEY"
	SettingOauthGoRedirectURI  = "OAUTHGO_REDIRECT_URI"
	SettingOauthGoEnabledTypes = "OAUTHGO_ENABLED_TYPES"

	// Third-party login service selector: "oauthgo" (default) or "rainbow"
	// (彩虹聚合登录, connect.php protocol compatible with u.cccyun.cc).
	SettingOauthProvider = "OAUTH_PROVIDER"

	SettingRainbowBaseURL      = "RAINBOW_BASE_URL"
	SettingRainbowAppID        = "RAINBOW_APP_ID"
	SettingRainbowAppKey       = "RAINBOW_APP_KEY"
	SettingRainbowRedirectURI  = "RAINBOW_REDIRECT_URI"
	SettingRainbowEnabledTypes = "RAINBOW_ENABLED_TYPES"

	SettingPaymentEnabled    = "PAYMENT_ENABLED"
	SettingPaymentProvider   = "PAYMENT_PROVIDER"
	SettingPaymentMerchantID = "PAYMENT_MERCHANT_ID"
	SettingPaymentAppID      = "PAYMENT_APP_ID"
	SettingPaymentAppKey     = "PAYMENT_APP_KEY"
	SettingPaymentNotifyURL  = "PAYMENT_NOTIFY_URL"

	SettingSMTPEnabled    = "SMTP_ENABLED"
	SettingSMTPHost       = "SMTP_HOST"
	SettingSMTPPort       = "SMTP_PORT"
	SettingSMTPUsername   = "SMTP_USERNAME"
	SettingSMTPPassword   = "SMTP_PASSWORD"
	SettingSMTPFrom       = "SMTP_FROM"
	SettingSMTPFromName   = "SMTP_FROM_NAME"
	SettingSMTPEncryption = "SMTP_ENCRYPTION"

	SettingSNSConfig = "SNS_CONFIG"

	SettingRegistrationEnabled = "REGISTRATION_ENABLED"
	SettingOauthGoAutoRegister = "OAUTHGO_AUTO_REGISTER"

	SettingFooterDescription = "FOOTER_DESCRIPTION"
	SettingFooterCopyright   = "FOOTER_COPYRIGHT"
	SettingFooterICP         = "FOOTER_ICP"
	SettingFooterPolice      = "FOOTER_POLICE"
	SettingFooterLinks       = "FOOTER_LINKS"
)

// DBSettingKeys lists the settings stored in the database (managed in UI).
var DBSettingKeys = []string{
	SettingWhoisAPIURL,
	SettingUaWhoisServer,
	SettingICPAPIURL,
	SettingDigitalPlatRDAPURL,

	SettingOauthGoBaseURL,
	SettingOauthGoAppID,
	SettingOauthGoAppKey,
	SettingOauthGoRedirectURI,
	SettingOauthGoEnabledTypes,

	SettingOauthProvider,

	SettingRainbowBaseURL,
	SettingRainbowAppID,
	SettingRainbowAppKey,
	SettingRainbowRedirectURI,
	SettingRainbowEnabledTypes,

	SettingPaymentEnabled,
	SettingPaymentProvider,
	SettingPaymentMerchantID,
	SettingPaymentAppID,
	SettingPaymentAppKey,
	SettingPaymentNotifyURL,

	SettingSMTPEnabled,
	SettingSMTPHost,
	SettingSMTPPort,
	SettingSMTPUsername,
	SettingSMTPPassword,
	SettingSMTPFrom,
	SettingSMTPFromName,
	SettingSMTPEncryption,

	SettingSNSConfig,

	SettingRegistrationEnabled,
	SettingOauthGoAutoRegister,

	SettingFooterDescription,
	SettingFooterCopyright,
	SettingFooterICP,
	SettingFooterPolice,
	SettingFooterLinks,
}

type Config struct {
	Port               string
	JWTKey             string
	DBPath             string
	WhoisAPIURL        string
	UaWhoisServer      string
	ICPAPIURL          string
	DigitalPlatRDAPURL string
	OauthGoBaseURL     string
	OauthGoAppID       string
	OauthGoAppKey      string
	OauthGoRedirectURI string
	OauthGoEnabled     bool

	// Active third-party login service ("oauthgo" | "rainbow").
	OauthProvider string

	RainbowBaseURL     string
	RainbowAppID       string
	RainbowAppKey      string
	RainbowRedirectURI string
	RainbowEnabled     bool

	// DBSettings holds every DB-backed setting key -> value (loaded after DB init).
	DBSettings map[string]string
}

var AppConfig *Config

// Version is the application release version shown in the UI and in scheduled
// system-information pushes.
const Version = "1.0.8"

func Load() {
	godotenv.Load()

	AppConfig = &Config{
		Port:               getEnv("PORT", "8080"),
		JWTKey:             os.Getenv("JWT_KEY"),
		DBPath:             getEnv("DB_PATH", "domain_manager.db"),
		WhoisAPIURL:        getEnv("WHOIS_API_URL", "https://who.zmh.me"),
		UaWhoisServer:      getEnv("UA_WHOIS_SERVER", "whois.ua:43"),
		ICPAPIURL:          getEnv("ICP_API_URL", "http://127.0.0.1:16181"),
		DigitalPlatRDAPURL: getEnv("DIGITALPLAT_RDAP_URL", "https://rdap.digitalplat.org"),
		OauthGoBaseURL:     strings.TrimRight(getEnv("OAUTHGO_BASE_URL", ""), "/"),
		OauthGoAppID:       os.Getenv("OAUTHGO_APP_ID"),
		OauthGoAppKey:      os.Getenv("OAUTHGO_APP_KEY"),
		OauthGoRedirectURI: getEnv("OAUTHGO_REDIRECT_URI", ""),
		OauthProvider:      getEnv("OAUTH_PROVIDER", "oauthgo"),
		RainbowBaseURL:     strings.TrimRight(getEnv("RAINBOW_BASE_URL", ""), "/"),
		RainbowAppID:       os.Getenv("RAINBOW_APP_ID"),
		RainbowAppKey:      os.Getenv("RAINBOW_APP_KEY"),
		RainbowRedirectURI: getEnv("RAINBOW_REDIRECT_URI", ""),
		DBSettings:         map[string]string{},
	}

	RefreshOauthEnabled()

	if AppConfig.JWTKey == "" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			log.Fatalf("failed to generate JWT key: %v", err)
		}
		AppConfig.JWTKey = hex.EncodeToString(key)
		log.Println("WARNING: JWT_KEY is not set; generated a random signing key. Tokens will be invalidated on every restart. Set JWT_KEY in your environment for stable sessions.")
	}
}

// GetSetting returns a DB-backed setting value (empty when unset).
func GetSetting(key string) string {
	if AppConfig == nil || AppConfig.DBSettings == nil {
		return ""
	}
	return AppConfig.DBSettings[key]
}

// IsRegistrationEnabled reports whether new user registration is allowed.
// Unset (default) means registration is enabled.
func IsRegistrationEnabled() bool {
	return GetSetting(SettingRegistrationEnabled) != "false"
}

// IsOauthGoAutoRegister reports whether third-party login may create new local
// accounts automatically. When disabled, OauthGo identities must be bound to an
// existing account first (profile -> 第三方登录). Unset (default) means allowed.
func IsOauthGoAutoRegister() bool {
	return GetSetting(SettingOauthGoAutoRegister) != "false"
}

// RefreshOauthEnabled recomputes whether the configured third-party login
// services are enabled (both OauthGo and 彩虹聚合登录 can be configured; the
// active one is selected by OAUTH_PROVIDER).
func RefreshOauthEnabled() {
	if AppConfig == nil {
		return
	}
	AppConfig.OauthGoEnabled = AppConfig.OauthGoBaseURL != "" &&
		AppConfig.OauthGoAppID != "" &&
		AppConfig.OauthGoAppKey != ""
	AppConfig.RainbowEnabled = AppConfig.RainbowBaseURL != "" &&
		AppConfig.RainbowAppID != "" &&
		AppConfig.RainbowAppKey != ""
}

// ApplyDBSettings overrides runtime settings from a key/value map (loaded from
// the system_settings table). Every known DB setting is also kept in
// AppConfig.DBSettings for runtime access.
func ApplyDBSettings(values map[string]string) {
	if AppConfig == nil {
		return
	}
	if AppConfig.DBSettings == nil {
		AppConfig.DBSettings = map[string]string{}
	}
	for _, key := range DBSettingKeys {
		if v, ok := values[key]; ok {
			AppConfig.DBSettings[key] = v
		}
	}

	if v, ok := values[SettingWhoisAPIURL]; ok {
		AppConfig.WhoisAPIURL = v
	}
	if v, ok := values[SettingUaWhoisServer]; ok {
		AppConfig.UaWhoisServer = v
	}
	if v, ok := values[SettingICPAPIURL]; ok {
		AppConfig.ICPAPIURL = v
	}
	if v, ok := values[SettingDigitalPlatRDAPURL]; ok {
		AppConfig.DigitalPlatRDAPURL = v
	}
	if v, ok := values[SettingOauthGoBaseURL]; ok {
		AppConfig.OauthGoBaseURL = strings.TrimRight(v, "/")
	}
	if v, ok := values[SettingOauthGoAppID]; ok {
		AppConfig.OauthGoAppID = v
	}
	if v, ok := values[SettingOauthGoAppKey]; ok {
		AppConfig.OauthGoAppKey = v
	}
	if v, ok := values[SettingOauthGoRedirectURI]; ok {
		AppConfig.OauthGoRedirectURI = v
	}
	if v, ok := values[SettingOauthProvider]; ok {
		AppConfig.OauthProvider = v
	}
	if v, ok := values[SettingRainbowBaseURL]; ok {
		AppConfig.RainbowBaseURL = strings.TrimRight(v, "/")
	}
	if v, ok := values[SettingRainbowAppID]; ok {
		AppConfig.RainbowAppID = v
	}
	if v, ok := values[SettingRainbowAppKey]; ok {
		AppConfig.RainbowAppKey = v
	}
	if v, ok := values[SettingRainbowRedirectURI]; ok {
		AppConfig.RainbowRedirectURI = v
	}
	RefreshOauthEnabled()
}

// ActiveOauthProvider returns the third-party login service selected in
// settings: "oauthgo" (default) or "rainbow" (彩虹聚合登录).
func ActiveOauthProvider() string {
	if kind := strings.TrimSpace(GetSetting(SettingOauthProvider)); kind != "" {
		return kind
	}
	if AppConfig != nil && AppConfig.OauthProvider != "" {
		return AppConfig.OauthProvider
	}
	return "oauthgo"
}

// DefaultDBSettingValue returns the initial value (from env) for a DB setting.
func DefaultDBSettingValue(key string) string {
	if AppConfig == nil {
		return ""
	}
	switch key {
	case SettingWhoisAPIURL:
		return AppConfig.WhoisAPIURL
	case SettingUaWhoisServer:
		return AppConfig.UaWhoisServer
	case SettingICPAPIURL:
		return AppConfig.ICPAPIURL
	case SettingDigitalPlatRDAPURL:
		return AppConfig.DigitalPlatRDAPURL
	case SettingOauthGoBaseURL:
		return AppConfig.OauthGoBaseURL
	case SettingOauthGoAppID:
		return AppConfig.OauthGoAppID
	case SettingOauthGoAppKey:
		return AppConfig.OauthGoAppKey
	case SettingOauthGoRedirectURI:
		return AppConfig.OauthGoRedirectURI
	case SettingOauthProvider:
		return AppConfig.OauthProvider
	case SettingRainbowBaseURL:
		return AppConfig.RainbowBaseURL
	case SettingRainbowAppID:
		return AppConfig.RainbowAppID
	case SettingRainbowAppKey:
		return AppConfig.RainbowAppKey
	case SettingRainbowRedirectURI:
		return AppConfig.RainbowRedirectURI
	}
	return ""
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
