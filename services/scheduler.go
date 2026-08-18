package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"DomainManager/config"
	"DomainManager/database"
	"DomainManager/models"
)

const schedulerTick = 1 * time.Minute

// StartScheduler launches the background job runner. Due tasks are executed
// once shortly after startup and then on a one-minute tick. The first run is
// delayed a few seconds so the HTTP server is ready before any push fires.
func StartScheduler() {
	go func() {
		time.Sleep(10 * time.Second)
		RunDueScheduledTasks()
		ticker := time.NewTicker(schedulerTick)
		defer ticker.Stop()
		for range ticker.C {
			RunDueScheduledTasks()
		}
	}()
	log.Println("scheduler started")
}

// RunDueScheduledTasks executes every enabled task whose interval has elapsed.
func RunDueScheduledTasks() {
	var tasks []models.ScheduledTask
	if err := database.DB.Where("enabled = ?", true).Find(&tasks).Error; err != nil {
		log.Printf("[scheduler] failed to load tasks: %v", err)
		return
	}
	now := time.Now()
	for _, t := range tasks {
		if !IsScheduledTaskDue(t, now) {
			continue
		}
		if err := RunScheduledTaskNow(t); err != nil {
			log.Printf("[scheduler] task %d (%s) failed: %v", t.ID, t.Name, err)
		}
	}
}

// IsScheduledTaskDue reports whether a task should run at the given time.
// Tasks that never ran are always due.
func IsScheduledTaskDue(task models.ScheduledTask, now time.Time) bool {
	if task.LastRunAt == nil {
		return true
	}
	interval := time.Duration(task.IntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = time.Duration(models.ScheduledTaskIntervalDefault) * time.Minute
	}
	return now.Sub(*task.LastRunAt) >= interval
}

// RunScheduledTaskNow executes a task immediately and records the run time.
func RunScheduledTaskNow(task models.ScheduledTask) error {
	var err error
	switch task.Type {
	case models.ScheduledTaskTypeSystemInfo:
		err = pushSystemInfoToUser(task.UserID)
	default:
		err = fmt.Errorf("unsupported scheduled task type %q", task.Type)
	}
	now := time.Now()
	if dbErr := database.DB.Model(&task).Update("last_run_at", now).Error; dbErr != nil {
		log.Printf("[scheduler] failed to update last_run_at for task %d: %v", task.ID, dbErr)
	}
	return err
}

// pushSystemInfoToUser builds the system-information report and sends it
// through every enabled notification channel owned by the user.
func pushSystemInfoToUser(userID uint) error {
	var channels []models.NotificationChannel
	if err := database.DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&channels).Error; err != nil {
		return err
	}
	if len(channels) == 0 {
		return nil // nothing to send
	}

	title := "系统信息定时推送"
	content := BuildSystemInfoReport()
	svc := NewNotificationService()
	for _, ch := range channels {
		cfg := ApplyGlobalSMTPDefaults(ch.Type, ch.Config)
		sendErr := svc.SendByType(ch.Type, cfg, title, content)
		logEntry := models.NotificationLog{
			ChannelID: ch.ID,
			Title:     title,
			Content:   content,
			Status:    "success",
		}
		if sendErr != nil {
			logEntry.Status = "failed"
			logEntry.Error = sendErr.Error()
		}
		if dbErr := database.DB.Create(&logEntry).Error; dbErr != nil {
			log.Printf("[scheduler] failed to save notification log: %v", dbErr)
		}
	}
	return nil
}

// BuildSystemInfoReport returns a plain-text snapshot of the system state.
func BuildSystemInfoReport() string {
	now := time.Now()
	month := now.AddDate(0, 0, 30)

	var domains, certs, registrars, users int64
	database.DB.Model(&models.Domain{}).Count(&domains)
	database.DB.Model(&models.Certificate{}).Count(&certs)
	database.DB.Model(&models.Registrar{}).Count(&registrars)
	database.DB.Model(&models.User{}).Count(&users)

	var domainsExpiring, certsExpiring int64
	database.DB.Model(&models.Domain{}).
		Where("expiry_date IS NOT NULL AND expiry_date > ? AND expiry_date <= ?", now, month).
		Count(&domainsExpiring)
	database.DB.Model(&models.Certificate{}).
		Where("not_after IS NOT NULL AND not_after > ? AND not_after <= ?", now, month).
		Count(&certsExpiring)

	var b strings.Builder
	b.WriteString("【系统信息】\n")
	b.WriteString(fmt.Sprintf("生成时间：%s\n", now.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("域名总数：%d（30天内到期 %d）\n", domains, domainsExpiring))
	b.WriteString(fmt.Sprintf("证书总数：%d（30天内到期 %d）\n", certs, certsExpiring))
	b.WriteString(fmt.Sprintf("注册商数量：%d\n", registrars))
	b.WriteString(fmt.Sprintf("用户数量：%d\n", users))
	b.WriteString(fmt.Sprintf("系统版本：%s\n", config.Version))
	return b.String()
}
