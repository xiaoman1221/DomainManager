package handlers

import (
	"net/http"

	"DomainManager/database"
	"DomainManager/models"
	"DomainManager/services"

	"github.com/gin-gonic/gin"
)

// ListScheduledTasks returns the current user's scheduled tasks.
// GET /api/notifications/schedules
func ListScheduledTasks(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var tasks []models.ScheduledTask
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list scheduled tasks"})
		return
	}
	if tasks == nil {
		tasks = []models.ScheduledTask{}
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks})
}

// CreateScheduledTask creates a scheduled task for the current user.
// POST /api/notifications/schedules
func CreateScheduledTask(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req models.ScheduledTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	interval := req.IntervalMinutes
	if interval <= 0 {
		interval = models.ScheduledTaskIntervalDefault
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	name := req.Name
	if name == "" {
		name = "系统信息定时推送"
	}

	task := models.ScheduledTask{
		UserID:          userID,
		Type:            req.Type,
		Name:            name,
		Enabled:         enabled,
		IntervalMinutes: interval,
	}
	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create scheduled task"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// UpdateScheduledTask updates an existing scheduled task.
// PUT /api/notifications/schedules/:id
func UpdateScheduledTask(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var task models.ScheduledTask
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scheduled task not found"})
		return
	}

	var req models.ScheduledTaskUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		task.Name = req.Name
	}
	if req.Enabled != nil {
		task.Enabled = *req.Enabled
	}
	if req.IntervalMinutes > 0 {
		task.IntervalMinutes = req.IntervalMinutes
	}

	if err := database.DB.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update scheduled task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// DeleteScheduledTask removes a scheduled task.
// DELETE /api/notifications/schedules/:id
func DeleteScheduledTask(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.ScheduledTask{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete scheduled task"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// RunScheduledTaskNow executes a scheduled task immediately and records the
// run time so the next automatic run is counted from now.
// POST /api/notifications/schedules/:id/run
func RunScheduledTaskNow(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var task models.ScheduledTask
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scheduled task not found"})
		return
	}

	if err := services.RunScheduledTaskNow(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "run failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "task executed"})
}
