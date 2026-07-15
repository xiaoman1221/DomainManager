package handlers

import (
	"DomainManager/database"
	"DomainManager/models"
	"DomainManager/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func ListNotificationChannels(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var channels []models.NotificationChannel
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&channels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list channels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": channels})
}

func CreateNotificationChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req models.NotificationChannelCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	channel := models.NotificationChannel{
		UserID:  userID,
		Name:    req.Name,
		Type:    req.Type,
		Enabled: enabled,
		Config:  req.Config,
	}

	if err := database.DB.Create(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create channel"})
		return
	}

	c.JSON(http.StatusCreated, channel)
}

func UpdateNotificationChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var channel models.NotificationChannel
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&channel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	var req models.NotificationChannelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		channel.Name = req.Name
	}
	if req.Type != "" {
		channel.Type = req.Type
	}
	if req.Config != "" {
		channel.Config = req.Config
	}
	if req.Enabled != nil {
		channel.Enabled = *req.Enabled
	}

	if err := database.DB.Save(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update channel"})
		return
	}

	c.JSON(http.StatusOK, channel)
}

func DeleteNotificationChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.NotificationChannel{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete channel"})
		return
	}

	// Also delete logs
	database.DB.Where("channel_id = ?", id).Delete(&models.NotificationLog{})

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func TestNotificationChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var channel models.NotificationChannel
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&channel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	var req models.NotificationTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := services.NewNotificationService()
	err := svc.SendByType(channel.Type, channel.Config, req.Title, req.Content)

	log := models.NotificationLog{
		ChannelID: channel.ID,
		Title:     req.Title,
		Content:   req.Content,
		Status:    "success",
	}
	if err != nil {
		log.Status = "failed"
		log.Error = err.Error()
		database.DB.Create(&log)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "test failed: " + err.Error()})
		return
	}

	database.DB.Create(&log)
	c.JSON(http.StatusOK, gin.H{"message": "test sent successfully"})
}

func ListNotificationLogs(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	channelID := c.Query("channel_id")

	var logs []models.NotificationLog
	query := database.DB.Joins("JOIN notification_channels ON notification_logs.channel_id = notification_channels.id").
		Where("notification_channels.user_id = ?", userID)

	if channelID != "" {
		query = query.Where("notification_logs.channel_id = ?", channelID)
	}

	if err := query.Order("notification_logs.created_at DESC").Limit(100).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}

func ToggleNotificationChannel(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var channel models.NotificationChannel
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&channel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "channel not found"})
		return
	}

	channel.Enabled = !channel.Enabled
	if err := database.DB.Save(&channel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle channel"})
		return
	}

	c.JSON(http.StatusOK, channel)
}

func GetNotificationTypes(c *gin.Context) {
	types := []map[string]string{
		{"value": "bark", "label": "Bark", "description": "iOS 推送通知"},
		{"value": "telegram", "label": "Telegram Bot", "description": "Telegram 机器人推送"},
		{"value": "email", "label": "邮件", "description": "SMTP 邮件通知"},
		{"value": "webhook", "label": "Webhook", "description": "自定义 Webhook 推送"},
	}
	c.JSON(http.StatusOK, gin.H{"data": types})
}

func SendDomainExpiryNotifications(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	// Get all enabled notification channels
	var channels []models.NotificationChannel
	if err := database.DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&channels).Error; err != nil || len(channels) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "no enabled channels"})
		return
	}

	// Get domains expiring in 7 days
	now := time.Now()
	limit := now.AddDate(0, 0, 7)
	var domains []models.Domain
	database.DB.Where("user_id = ? AND expiry_date IS NOT NULL AND expiry_date > ? AND expiry_date <= ? AND expiry_reminder = ?",
		userID, now, limit, true).Find(&domains)

	if len(domains) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "no expiring domains"})
		return
	}

	// Build notification content
	title := "域名到期提醒"
	content := "以下域名将在7天内到期：\n"
	for _, d := range domains {
		expiryStr := ""
		if d.ExpiryDate != nil {
			expiryStr = d.ExpiryDate.Format("2006-01-02")
		}
		content += "- " + d.Name + " (到期: " + expiryStr + ")\n"
	}

	svc := services.NewNotificationService()
	sent := 0
	for _, ch := range channels {
		if err := svc.SendByType(ch.Type, ch.Config, title, content); err != nil {
			log := models.NotificationLog{
				ChannelID: ch.ID,
				Title:     title,
				Content:   content,
				Status:    "failed",
				Error:     err.Error(),
			}
			database.DB.Create(&log)
		} else {
			log := models.NotificationLog{
				ChannelID: ch.ID,
				Title:     title,
				Content:   content,
				Status:    "success",
			}
			database.DB.Create(&log)
			sent++
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "notifications sent", "sent": sent, "total": len(channels)})
}
