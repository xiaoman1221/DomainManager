package models

import (
	"time"

	"gorm.io/gorm"
)

type SystemSetting struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Key       string         `json:"key" gorm:"uniqueIndex;size:128;not null"`
	Value     string         `json:"value" gorm:"type:text"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}
