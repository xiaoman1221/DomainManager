package handlers

import (
	"DomainManager/database"
	"DomainManager/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

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

func UpdateSystemSetting(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var setting models.SystemSetting
	result := database.DB.Where("`key` = ?", req.Key).First(&setting)
	if result.Error == nil {
		setting.Value = req.Value
		database.DB.Save(&setting)
	} else {
		setting = models.SystemSetting{Key: req.Key, Value: req.Value}
		database.DB.Create(&setting)
	}

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
		"version":      "1.0.0",
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

func UpdateUserRole(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Role != "admin" && req.Role != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or user"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	user.Role = req.Role
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
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
