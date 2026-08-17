package handlers

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/models"
	"DomainManager/services"
	"DomainManager/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// errOauthNotBound is returned when a third-party identity has no local account
// and automatic registration is disabled (OAUTHGO_AUTO_REGISTER=false). The
// user must log in normally first and bind the account from the profile page.
var errOauthNotBound = errors.New("该第三方账号未绑定任何本地账号，请先登录后到个人设置中绑定")

// ---------- short-lived one-time login tickets ----------
// The OauthGo callback exchanges the authorization code on the server side and
// hands the freshly minted JWT to the SPA through a one-time ticket instead of
// putting the token in the URL (which would leak into access logs).

type oauthTicket struct {
	token   string
	expires time.Time
}

var oauthTicketStore = struct {
	sync.Mutex
	m map[string]oauthTicket
}{m: make(map[string]oauthTicket)}

const oauthTicketTTL = 2 * time.Minute

func issueOauthTicket(token string) string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	ticket := hex.EncodeToString(buf)

	oauthTicketStore.Lock()
	oauthTicketStore.m[ticket] = oauthTicket{token: token, expires: time.Now().Add(oauthTicketTTL)}
	oauthTicketStore.Unlock()
	return ticket
}

func redeemOauthTicket(ticket string) (string, bool) {
	oauthTicketStore.Lock()
	defer oauthTicketStore.Unlock()

	t, ok := oauthTicketStore.m[ticket]
	if !ok || time.Now().After(t.expires) {
		delete(oauthTicketStore.m, ticket)
		return "", false
	}
	delete(oauthTicketStore.m, ticket)
	return t.token, true
}

func cleanupOauthTickets() {
	oauthTicketStore.Lock()
	defer oauthTicketStore.Unlock()

	now := time.Now()
	for k, t := range oauthTicketStore.m {
		if now.After(t.expires) {
			delete(oauthTicketStore.m, k)
		}
	}
}

// ---------- handlers ----------

// OauthGoProviders returns the third-party login channels enabled both on the
// active login service (OauthGo or 彩虹聚合登录) and selected in system settings.
// GET /api/auth/oauth/providers (public)
func OauthGoProviders(c *gin.Context) {
	cleanupOauthTickets()

	if !services.ActiveOauthEnabled() {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "providers": []services.OauthGoProvider{}})
		return
	}

	providers, err := services.ActiveOauthProviders()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"enabled":   true,
			"providers": []services.OauthGoProvider{},
			"error":     err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": true, "providers": filterOauthProviders(providers, activeOauthTypesSettingKey())})
}

// activeOauthTypesSettingKey returns the enabled-types setting key of the
// active login service (OAUTHGO_ENABLED_TYPES or RAINBOW_ENABLED_TYPES).
func activeOauthTypesSettingKey() string {
	if services.ActiveOauthKind() == services.OauthProviderKindRainbow {
		return config.SettingRainbowEnabledTypes
	}
	return config.SettingOauthGoEnabledTypes
}

// filterOauthProviders keeps only the login methods selected in system
// settings. An empty selection means all enabled channels are shown.
func filterOauthProviders(providers []services.OauthGoProvider, settingKey string) []services.OauthGoProvider {
	raw := strings.TrimSpace(config.GetSetting(settingKey))
	if raw == "" {
		return providers
	}
	var selected []string
	if err := json.Unmarshal([]byte(raw), &selected); err != nil {
		return providers
	}
	if len(selected) == 0 {
		return providers
	}
	set := make(map[string]bool, len(selected))
	for _, s := range selected {
		set[strings.TrimSpace(s)] = true
	}
	out := make([]services.OauthGoProvider, 0, len(providers))
	for _, p := range providers {
		if set[p.Name] {
			out = append(out, p)
		}
	}
	return out
}

// filterOauthGoProviders keeps only the OauthGo login methods selected in
// system settings (OAUTHGO_ENABLED_TYPES).
func filterOauthGoProviders(providers []services.OauthGoProvider) []services.OauthGoProvider {
	return filterOauthProviders(providers, config.SettingOauthGoEnabledTypes)
}

// oauthCallbackBase returns the public scheme://host used to build third-party
// login redirect URIs. It prefers the scheme+host of the configured redirect
// URI (which is whitelisted in the login service admin panel) and falls back
// to the incoming request, honoring X-Forwarded-Proto / X-Forwarded-Host for
// reverse proxies.
func oauthCallbackBase(c *gin.Context, configured string) string {
	if configured != "" {
		if u, err := url.Parse(configured); err == nil && u.Scheme != "" && u.Host != "" {
			return u.Scheme + "://" + u.Host
		}
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if fwd := c.GetHeader("X-Forwarded-Proto"); fwd != "" {
		scheme = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	host := c.Request.Host
	if fwdHost := c.GetHeader("X-Forwarded-Host"); fwdHost != "" {
		host = strings.TrimSpace(strings.Split(fwdHost, ",")[0])
	}
	return scheme + "://" + host
}

// OauthGoLogin starts a third-party login and returns the authorization URL.
// POST /api/auth/oauth/login {"type": "qq"}
func OauthGoLogin(c *gin.Context) {
	var req struct {
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}
	if !services.ActiveOauthEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "第三方登录未配置，请在系统设置中配置 OauthGo 或彩虹登录"})
		return
	}

	redirectURI := config.AppConfig.OauthGoRedirectURI
	if redirectURI == "" {
		redirectURI = oauthCallbackBase(c, "") + "/api/auth/oauth/callback"
	}

	loginURL, err := services.ActiveOauthLoginURL(req.Type, redirectURI)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": loginURL, "type": req.Type})
}

// OauthGoCallback is the redirect_uri target. OauthGo 302s back here with
// ?type=<provider>&code=<one-time code>. The handler exchanges the code for the
// user profile, finds/creates the local user, issues a JWT, and redirects the
// browser to /login?oauth_ticket=<ticket>.
// GET /api/auth/oauth/callback?type=qq&code=xxxx
func OauthGoCallback(c *gin.Context) {
	loginType := c.Query("type")
	code := c.Query("code")

	fail := func(code string) {
		c.Redirect(http.StatusFound, "/login?oauth_error="+code)
	}

	if loginType == "" || code == "" {
		fail("1")
		return
	}

	info, err := services.ActiveOauthUserInfo(loginType, code)
	if err != nil {
		fail("1")
		return
	}

	user, err := findOrCreateOauthUser(info)
	if err != nil {
		if errors.Is(err, errOauthNotBound) {
			fail("not_bound")
			return
		}
		fail("1")
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		fail("1")
		return
	}

	ticket := issueOauthTicket(token)
	if ticket == "" {
		fail("1")
		return
	}

	c.Redirect(http.StatusFound, "/login?oauth_ticket="+ticket)
}

// OauthGoRedeem exchanges a one-time ticket for a JWT + user profile.
// POST /api/auth/oauth/ticket {"ticket": "..."}
func OauthGoRedeem(c *gin.Context) {
	var req struct {
		Ticket string `json:"ticket" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket is required"})
		return
	}

	token, ok := redeemOauthTicket(req.Ticket)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ticket 无效或已过期，请重新登录"})
		return
	}

	claims, err := utils.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: user})
}

// ---------- user mapping ----------

// findOrCreateOauthUser resolves an OauthGo profile to a local user. It first
// consults the user_oauth_bindings table, then legacy user columns, and finally
// creates a new local user (with an unguessable password) plus a binding so the
// account can be accessed through the third-party provider.
func findOrCreateOauthUser(info *services.OauthGoProfile) (*models.User, error) {
	var binding models.UserOAuthBinding
	err := database.DB.Where("provider = ? AND openid = ?", info.Type, info.OpenID).First(&binding).Error
	if err == nil {
		var user models.User
		if err := database.DB.First(&user, binding.UserID).Error; err != nil {
			return nil, err
		}
		if err := upsertOauthBinding(user.ID, info); err != nil {
			return nil, err
		}
		refreshUserProfile(&user, info)
		return &user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Legacy users created before the binding table keep oauth_provider/openid.
	var user models.User
	err = database.DB.Where("oauth_provider = ? AND oauth_openid = ?", info.Type, info.OpenID).First(&user).Error
	if err == nil {
		if err := upsertOauthBinding(user.ID, info); err != nil {
			return nil, err
		}
		refreshUserProfile(&user, info)
		return &user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Automatic account creation is disabled: the third-party identity must be
	// bound to an existing local account first.
	if !config.IsOauthGoAutoRegister() {
		return nil, errOauthNotBound
	}

	user = models.User{
		Username:      buildOauthUsername(info.Type, info.OpenID),
		Email:         buildOauthEmail(info.Type, info.OpenID),
		Nickname:      firstNonEmpty(info.Nickname, buildOauthUsername(info.Type, info.OpenID)),
		RoleGroup:     "user",
		OauthProvider: info.Type,
		OauthOpenID:   info.OpenID,
		OauthAvatar:   info.Avatar,
	}
	if err := user.SetPassword(randomPassword(24)); err != nil {
		return nil, err
	}

	if err := database.DB.Create(&user).Error; err == nil {
		if err := upsertOauthBinding(user.ID, info); err != nil {
			return nil, err
		}
		return &user, nil
	} else if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		return nil, err
	}

	// Race or username collision: retry the lookup, then retry with a suffix.
	if err2 := database.DB.Where("oauth_provider = ? AND oauth_openid = ?", info.Type, info.OpenID).First(&user).Error; err2 == nil {
		return &user, nil
	}

	user.Username = buildOauthUsername(info.Type, info.OpenID) + randomSuffix(4)
	if err := database.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	if err := upsertOauthBinding(user.ID, info); err != nil {
		return nil, err
	}
	return &user, nil
}

// upsertOauthBinding creates or refreshes a third-party binding for a user.
// It refuses to attach an identity that is already bound to another user.
func upsertOauthBinding(userID uint, info *services.OauthGoProfile) error {
	var binding models.UserOAuthBinding
	err := database.DB.Where("provider = ? AND openid = ?", info.Type, info.OpenID).First(&binding).Error
	if err == nil {
		if binding.UserID != userID {
			return errors.New("该第三方账号已绑定其他用户")
		}
		changed := false
		if info.Nickname != "" && info.Nickname != binding.Nickname {
			binding.Nickname = info.Nickname
			changed = true
		}
		if info.Avatar != "" && info.Avatar != binding.Avatar {
			binding.Avatar = info.Avatar
			changed = true
		}
		if changed {
			database.DB.Save(&binding)
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	binding = models.UserOAuthBinding{
		UserID:   userID,
		Provider: info.Type,
		OpenID:   info.OpenID,
		Nickname: info.Nickname,
		Avatar:   info.Avatar,
	}
	return database.DB.Create(&binding).Error
}

// refreshUserProfile syncs display fields from a third-party profile.
func refreshUserProfile(user *models.User, info *services.OauthGoProfile) {
	changed := false
	if info.Nickname != "" && info.Nickname != user.Nickname {
		user.Nickname = info.Nickname
		changed = true
	}
	if info.Avatar != "" && info.Avatar != user.OauthAvatar {
		user.OauthAvatar = info.Avatar
		changed = true
	}
	if info.Type != "" && info.Type != user.OauthProvider {
		user.OauthProvider = info.Type
		user.OauthOpenID = info.OpenID
		changed = true
	}
	if changed {
		database.DB.Save(user)
	}
}

// buildOauthUsername produces a stable, unique username from a provider account.
func buildOauthUsername(loginType, openID string) string {
	clean := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				b.WriteRune(r)
			}
		}
		return b.String()
	}

	name := fmt.Sprintf("oauth_%s_%s", clean(strings.ToLower(loginType)), clean(openID))
	if name == "oauth__" {
		name = "oauth_user"
	}
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}

// buildOauthEmail produces a stable synthetic email so the unique/not-null
// constraints are satisfied. It is never used for delivery.
func buildOauthEmail(loginType, openID string) string {
	sum := md5.Sum([]byte(openID))
	return fmt.Sprintf("oauth-%s-%s@oauth.local", strings.ToLower(loginType), hex.EncodeToString(sum[:])[:16])
}

func randomPassword(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("pw-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func randomSuffix(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)[:n]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
