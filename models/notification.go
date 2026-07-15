package models

import (
	"time"

	"gorm.io/gorm"
)

type NotificationChannel struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"index;not null"`
	Name      string         `json:"name" gorm:"size:128;not null"`
	Type      string         `json:"type" gorm:"size:32;not null"`
	Enabled   bool           `json:"enabled" gorm:"default:true"`
	Config    string         `json:"config" gorm:"type:text"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type NotificationLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ChannelID uint      `json:"channel_id" gorm:"index;not null"`
	Title     string    `json:"title" gorm:"size:255"`
	Content   string    `json:"content" gorm:"type:text"`
	Status    string    `json:"status" gorm:"size:32;default:'success'"`
	Error     string    `json:"error" gorm:"size:512"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationChannelCreateRequest struct {
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Config  string `json:"config"`
	Enabled *bool  `json:"enabled"`
}

type NotificationChannelUpdateRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Config  string `json:"config"`
	Enabled *bool  `json:"enabled"`
}

type NotificationTestRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}
