package models

import (
	"time"

	"gorm.io/gorm"
)

type Registrar struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"uniqueIndex;size:128;not null"`
	Type        string         `json:"type" gorm:"size:32;not null"` // aliyun, tencent, cloudflare, godaddy, namecheap, digitalplat, etc.
	APIEndpoint string         `json:"api_endpoint" gorm:"size:256"`
	APIKey      string         `json:"api_key" gorm:"size:256"`
	APISecret   string         `json:"api_secret" gorm:"size:256"`
	APIExtra    string         `json:"api_extra" gorm:"size:512"` // JSON extra params
	Enabled     bool           `json:"enabled" gorm:"default:true"`
	SyncEnabled bool           `json:"sync_enabled" gorm:"default:false"`
	LastSyncAt  *time.Time     `json:"last_sync_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type RegistrarCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Type        string `json:"type" binding:"required"`
	APIEndpoint string `json:"api_endpoint"`
	APIKey      string `json:"api_key"`
	APISecret   string `json:"api_secret"`
	APIExtra    string `json:"api_extra"`
	Enabled     bool   `json:"enabled"`
	SyncEnabled bool   `json:"sync_enabled"`
}

type RegistrarUpdateRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	APIEndpoint string `json:"api_endpoint"`
	APIKey      string `json:"api_key"`
	APISecret   string `json:"api_secret"`
	APIExtra    string `json:"api_extra"`
	Enabled     *bool  `json:"enabled"`
	SyncEnabled *bool  `json:"sync_enabled"`
}

type ImportDomainsRequest struct {
	RegistrarID uint   `json:"registrar_id" binding:"required"`
	Domains     string `json:"domains"` // newline-separated domain list for manual import
}
