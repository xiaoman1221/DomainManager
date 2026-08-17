package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Username      string         `json:"username" gorm:"uniqueIndex;size:64;not null"`
	Email         string         `json:"email" gorm:"uniqueIndex;size:128;not null"`
	Password      string         `json:"-" gorm:"not null"`
	Nickname      string         `json:"nickname" gorm:"size:64"`
	RoleGroup     string         `json:"role_group" gorm:"column:role_group;size:32;default:'user'"` // 角色组: admin / user
	UserGroup     string         `json:"user_group" gorm:"column:user_group;size:64;default:''"`     // 用户组标记
	OauthProvider string         `json:"oauth_provider" gorm:"size:32;index"`
	OauthOpenID   string         `json:"oauth_openid" gorm:"column:oauth_openid;size:128;index"`
	OauthAvatar   string         `json:"oauth_avatar" gorm:"size:512"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
