package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/models"
	"DomainManager/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------- third-party account binding ----------
// A logged-in user can bind OauthGo identities to their local account from the
// profile page. The binding callback is reached through a one-time token placed
// in the redirect path, because OauthGo only echoes type + code on the callback.

type oauthBindPending struct {
	userID  uint
	expires time.Time
}

var oauthBindStore = struct {
	sync.Mutex
	m map[string]oauthBindPending
}{m: make(map[string]oauthBindPending)}

const oauthBindTTL = 10 * time.Minute

func issueOauthBindToken(userID uint) string {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	token := hex.EncodeToString(buf)

	oauthBindStore.Lock()
	oauthBindStore.m[token] = oauthBindPending{userID: userID, expires: time.Now().Add(oauthBindTTL)}
	oauthBindStore.Unlock()
	return token
}

func redeemOauthBindToken(token string) (uint, bool) {
	oauthBindStore.Lock()
	defer oauthBindStore.Unlock()

	p, ok := oauthBindStore.m[token]
	if !ok || time.Now().After(p.expires) {
		delete(oauthBindStore.m, token)
		return 0, false
	}
	delete(oauthBindStore.m, token)
	return p.userID, true
}

func cleanupOauthBindings() {
	oauthBindStore.Lock()
	defer oauthBindStore.Unlock()

	now := time.Now()
	for k, p := range oauthBindStore.m {
		if now.After(p.expires) {
			delete(oauthBindStore.m, k)
		}
	}
}

// OauthGoBindings lists the current user's third-party bindings.
// GET /api/auth/oauth/bindings (auth)
func OauthGoBindings(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var bindings []models.UserOAuthBinding
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&bindings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list bindings"})
		return
	}
	if bindings == nil {
		bindings = []models.UserOAuthBinding{}
	}
	c.JSON(http.StatusOK, gin.H{"data": bindings})
}

// OauthGoBindStart starts a third-party binding and returns the authorization
// URL. The callback embeds a one-time token that maps back to the logged-in user.
// POST /api/auth/oauth/bind (auth) {"type": "qq"}
func OauthGoBindStart(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req struct {
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}
	if !services.ActiveOauthEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "第三方登录未配置，无法绑定账号"})
		return
	}

	token := issueOauthBindToken(userID)
	if token == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start binding"})
		return
	}

	redirectURI := oauthCallbackBase(c, config.AppConfig.OauthGoRedirectURI) + "/api/auth/oauth/bind/callback/" + token

	loginURL, err := services.ActiveOauthLoginURL(req.Type, redirectURI)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": loginURL, "type": req.Type})
}

// OauthGoBindCallback is the redirect target for binding. It associates the
// authorized OauthGo identity with the pending user and redirects back to the
// profile page with ?oauth_bind=success|duplicate|error.
// GET /api/auth/oauth/bind/callback/:token?type=qq&code=xxxx
func OauthGoBindCallback(c *gin.Context) {
	cleanupOauthBindings()

	token := c.Param("token")
	loginType := c.Query("type")
	code := c.Query("code")

	redirect := func(status string) {
		c.Redirect(http.StatusFound, "/profile?oauth_bind="+status)
	}

	userID, ok := redeemOauthBindToken(token)
	if !ok {
		redirect("error")
		return
	}
	if loginType == "" || code == "" {
		redirect("error")
		return
	}

	info, err := services.ActiveOauthUserInfo(loginType, code)
	if err != nil {
		redirect("error")
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		redirect("error")
		return
	}

	// The identity must not belong to another user.
	var existing models.UserOAuthBinding
	if err := database.DB.Where("provider = ? AND openid = ?", info.Type, info.OpenID).First(&existing).Error; err == nil {
		if existing.UserID != userID {
			redirect("duplicate")
			return
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		redirect("error")
		return
	}

	if err := upsertOauthBinding(userID, info); err != nil {
		if strings.Contains(err.Error(), "已绑定其他用户") {
			redirect("duplicate")
			return
		}
		redirect("error")
		return
	}
	refreshUserProfile(&user, info)

	redirect("success")
}

// OauthGoUnbind removes a third-party binding.
// DELETE /api/auth/oauth/bind/:provider (auth)
func OauthGoUnbind(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}

	var binding models.UserOAuthBinding
	if err := database.DB.Where("user_id = ? AND provider = ?", userID, provider).First(&binding).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该第三方绑定"})
		return
	}
	if err := database.DB.Delete(&binding).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unbind"})
		return
	}

	// Clear the legacy user columns if they point at the removed binding.
	var user models.User
	if err := database.DB.First(&user, userID).Error; err == nil && user.OauthProvider == provider {
		user.OauthProvider = ""
		user.OauthOpenID = ""
		user.OauthAvatar = ""
		database.DB.Save(&user)
	}

	c.JSON(http.StatusOK, gin.H{"message": "unbound"})
}
