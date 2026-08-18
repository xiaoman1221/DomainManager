package models

import (
	"time"

	"gorm.io/gorm"
)

// ScheduledTask is a recurring background job owned by a user. Currently the
// only implemented type is "system_info" (periodic system-information push
// through the user's enabled notification channels).
type ScheduledTask struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	UserID          uint           `json:"user_id" gorm:"index;not null"`
	Type            string         `json:"type" gorm:"size:32;not null"` // system_info
	Name            string         `json:"name" gorm:"size:128;not null"`
	Enabled         bool           `json:"enabled" gorm:"default:true"`
	IntervalMinutes int            `json:"interval_minutes" gorm:"not null;default:1440"` // 1440 = once a day
	LastRunAt       *time.Time     `json:"last_run_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// ScheduledTaskTypeSystemInfo is the only supported task type for now.
const ScheduledTaskTypeSystemInfo = "system_info"

// ScheduledTaskIntervalDefault is the default interval in minutes (1 day).
const ScheduledTaskIntervalDefault = 1440

type ScheduledTaskCreateRequest struct {
	Type            string `json:"type" binding:"required"`
	Name            string `json:"name"`
	Enabled         *bool  `json:"enabled"`
	IntervalMinutes int    `json:"interval_minutes"`
}

type ScheduledTaskUpdateRequest struct {
	Name            string `json:"name"`
	Enabled         *bool  `json:"enabled"`
	IntervalMinutes int    `json:"interval_minutes"`
}
