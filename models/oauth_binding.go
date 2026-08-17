package models

import (
	"time"
)

// UserOAuthBinding links a local user to a third-party (OauthGo) identity.
// A user may bind multiple providers.
type UserOAuthBinding struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index;not null"`
	Provider  string    `json:"provider" gorm:"size:32;not null;uniqueIndex:idx_binding_provider_openid,priority:1"`
	OpenID    string    `json:"openid" gorm:"column:openid;size:128;not null;uniqueIndex:idx_binding_provider_openid,priority:2"`
	Nickname  string    `json:"nickname" gorm:"size:64"`
	Avatar    string    `json:"avatar" gorm:"size:512"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
