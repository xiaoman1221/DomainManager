package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/models"

	"github.com/gin-gonic/gin"
)

// isKnownSetting reports whether the key is a DB-managed runtime setting.
func isKnownSetting(key string) bool {
	for _, k := range config.DBSettingKeys {
		if k == key {
			return true
		}
	}
	return false
}

// upsertSetting creates or updates a single system setting row.
func upsertSetting(key, value string) error {
	var setting models.SystemSetting
	result := database.DB.Where("`key` = ?", key).First(&setting)
	if result.Error == nil {
		setting.Value = value
		return database.DB.Save(&setting).Error
	}
	setting = models.SystemSetting{Key: key, Value: value}
	return database.DB.Create(&setting).Error
}

func GetSystemSettings(c *gin.Context) {
	var settings []models.SystemSetting
	if err := database.DB.Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	result := map[string]string{}
	for _, s := range settings {
		result[s.Key] = s.Value
	}

	c.JSON(http.StatusOK, result)
}

// UpdateSystemSettings bulk-updates DB-managed runtime settings and reloads
// them into the running config immediately.
// PUT /api/settings  {"WHOIS_API_URL": "...", "ICP_API_URL": "..."}
func UpdateSystemSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no settings provided"})
		return
	}

	for key := range req {
		if !isKnownSetting(key) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown setting key: " + key})
			return
		}
	}

	for key, value := range req {
		if err := upsertSetting(key, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setting " + key})
			return
		}
	}

	database.LoadSettingsIntoConfig()
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

func GetSystemInfo(c *gin.Context) {
	var domainCount, certCount, registrarCount, userCount int64
	database.DB.Model(&models.Domain{}).Count(&domainCount)
	database.DB.Model(&models.Certificate{}).Count(&certCount)
	database.DB.Model(&models.Registrar{}).Count(&registrarCount)
	database.DB.Model(&models.User{}).Count(&userCount)

	c.JSON(http.StatusOK, gin.H{
		"domains":      domainCount,
		"certificates": certCount,
		"registrars":   registrarCount,
		"users":        userCount,
		"version":      "1.0.7",
	})
}

func UpdateUserProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req struct {
		Nickname string `json:"nickname"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Email != "" {
		// Check if email is already taken by another user
		var existing models.User
		if err := database.DB.Where("email = ? AND id != ?", req.Email, userID).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
		user.Email = req.Email
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func ChangePassword(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if !user.CheckPassword(req.OldPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "incorrect old password"})
		return
	}

	if err := user.SetPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

func ListAllUsers(c *gin.Context) {
	var users []models.User
	if err := database.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// UpdateUserRoleGroup changes a user's role group (admin / user).
// PUT /api/settings/users/:id/role-group {"role_group": "admin"}
func UpdateUserRoleGroup(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		RoleGroup string `json:"role_group" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.RoleGroup != "admin" && req.RoleGroup != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_group must be admin or user"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.RoleGroup = req.RoleGroup
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role group"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUserGroup changes a user's user-group marker.
// PUT /api/settings/users/:id/user-group {"user_group": "运维组"}
func UpdateUserGroup(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		UserGroup string `json:"user_group"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.UserGroup) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_group too long (max 64)"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.UserGroup = req.UserGroup
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user group"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func AdminUpdateUserPassword(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := user.SetPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}

// UpdateUser edits another user's profile fields (admin only).
// PUT /api/settings/users/:id {"nickname":"...","email":"...","role_group":"admin","user_group":"运维组"}
func UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Nickname  string `json:"nickname"`
		Email     string `json:"email"`
		RoleGroup string `json:"role_group"`
		UserGroup string `json:"user_group"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.UserGroup) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_group too long (max 64)"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if req.Email != "" && req.Email != user.Email {
		if len(req.Email) > 128 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email too long"})
			return
		}
		var existing models.User
		if err := database.DB.Where("email = ? AND id != ?", req.Email, user.ID).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
		user.Email = req.Email
	}
	if req.Nickname != "" && req.Nickname != user.Nickname {
		user.Nickname = req.Nickname
	}
	if req.RoleGroup != "" && req.RoleGroup != user.RoleGroup {
		if req.RoleGroup != "admin" && req.RoleGroup != "user" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role_group must be admin or user"})
			return
		}
		// Prevent demoting the last remaining admin.
		if user.RoleGroup == "admin" && req.RoleGroup == "user" {
			var adminCount int64
			database.DB.Model(&models.User{}).Where("role_group = ?", "admin").Count(&adminCount)
			if adminCount <= 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "不能移除最后一个管理员"})
				return
			}
		}
		user.RoleGroup = req.RoleGroup
	}
	if req.UserGroup != user.UserGroup {
		user.UserGroup = req.UserGroup
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser soft-deletes a user (admin only).
// DELETE /api/settings/users/:id
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	currentUserID := c.MustGet("user_id").(uint)

	if strconv.FormatUint(uint64(currentUserID), 10) == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除当前登录的账号"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.RoleGroup == "admin" {
		var adminCount int64
		database.DB.Model(&models.User{}).Where("role_group = ?", "admin").Count(&adminCount)
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除最后一个管理员"})
			return
		}
	}

	if err := database.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GetSiteConfig returns public site-wide configuration (footer, SNS links)
// used by the login page. No authentication required.
// GET /api/site/config
func GetSiteConfig(c *gin.Context) {
	var settings []models.SystemSetting
	if err := database.DB.Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}
	values := map[string]string{}
	for _, s := range settings {
		values[s.Key] = s.Value
	}

	c.JSON(http.StatusOK, gin.H{
		"site_name":            "Domain Manager",
		"registration_enabled": config.IsRegistrationEnabled(),
		"footer": gin.H{
			"description": values[config.SettingFooterDescription],
			"copyright":   values[config.SettingFooterCopyright],
			"icp":         values[config.SettingFooterICP],
			"police":      values[config.SettingFooterPolice],
			"links":       parseJSONArray(values[config.SettingFooterLinks]),
		},
		"sns": parseJSONArray(values[config.SettingSNSConfig]),
	})
}

// parseJSONArray parses a JSON array string into []map[string]string, returning
// an empty slice on invalid/empty input (never errors).
func parseJSONArray(raw string) []map[string]string {
	var out []map[string]string
	_ = json.Unmarshal([]byte(raw), &out)
	if out == nil {
		out = []map[string]string{}
	}
	return out
}
