package models

import (
	"time"

	"gorm.io/gorm"
)

type Certificate struct {
	ID                 uint           `json:"id" gorm:"primaryKey"`
	UserID             uint           `json:"user_id" gorm:"index;not null"`
	CertimateID        string         `json:"certimate_id" gorm:"size:64"`
	Domain             string         `json:"domain" gorm:"index;size:255;not null"`
	Issuer             string         `json:"issuer" gorm:"size:255"`
	SerialNumber       string         `json:"serial_number" gorm:"size:255"`
	NotBefore          *time.Time     `json:"not_before"`
	NotAfter           *time.Time     `json:"not_after"`
	SubjectAltNames    string         `json:"subject_alt_names" gorm:"size:1024"`
	KeyAlgorithm       string         `json:"key_algorithm" gorm:"size:64"`
	SignatureAlgorithm string         `json:"signature_algorithm" gorm:"size:64"`
	IsExpired          bool           `json:"is_expired" gorm:"default:false"`
	Source             string         `json:"source" gorm:"size:64;default:'certimate'"`
	Certificate        string         `json:"certificate" gorm:"type:text"`
	PrivateKey         string         `json:"private_key" gorm:"type:text"`
	Note               string         `json:"note" gorm:"size:512"`
	Status             string         `json:"status" gorm:"size:32;default:'active'"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"-" gorm:"index"`
}

type CertificateCreateRequest struct {
	Domain          string `json:"domain" binding:"required"`
	CertimateID     string `json:"certimate_id"`
	Issuer          string `json:"issuer"`
	NotBefore       string `json:"not_before"`
	NotAfter        string `json:"not_after"`
	SubjectAltNames string `json:"subject_alt_names"`
	KeyAlgorithm    string `json:"key_algorithm"`
	Source          string `json:"source"`
	Certificate     string `json:"certificate"`
	PrivateKey      string `json:"private_key"`
	Note            string `json:"note"`
}

type CertificateUpdateRequest struct {
	Domain          string `json:"domain"`
	Issuer          string `json:"issuer"`
	NotAfter        string `json:"not_after"`
	SubjectAltNames string `json:"subject_alt_names"`
	KeyAlgorithm    string `json:"key_algorithm"`
	Source          string `json:"source"`
	Certificate     string `json:"certificate"`
	PrivateKey      string `json:"private_key"`
	Note            string `json:"note"`
	Status          string `json:"status"`
}

type CertimateConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	// Token is a runtime-only PocketBase superuser token. It is never persisted
	// through the API; it is kept for reading legacy configs saved with a token.
	Token string `json:"token,omitempty"`
}

type CertimateConfigRequest struct {
	URL      string `json:"url" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
