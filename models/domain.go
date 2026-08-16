package models

import (
	"time"

	"gorm.io/gorm"
)

type Domain struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	UserID      uint       `json:"user_id" gorm:"index;not null"`
	Name        string     `json:"name" gorm:"index;size:255;not null"`
	Status      string     `json:"status" gorm:"size:32;default:'active'"`
	Registrar   string     `json:"registrar" gorm:"size:128"`
	ExpiryDate  *time.Time `json:"expiry_date"`
	Nameservers string     `json:"nameservers" gorm:"size:512"`
	AutoRenew   bool       `json:"auto_renew" gorm:"default:false"`
	Note        string     `json:"note" gorm:"size:512"`

	Group          string `json:"group" gorm:"size:64;default:''"`
	Tags           string `json:"tags" gorm:"size:256"`
	CertCount      int    `json:"cert_count" gorm:"default:0"`
	AutoUpdate     bool   `json:"auto_update" gorm:"default:false"`
	UpdateICP      bool   `json:"update_icp" gorm:"default:false"`
	ExpiryReminder bool   `json:"expiry_reminder" gorm:"default:true"`

	RegistrationDate  *time.Time `json:"registration_date"`
	CreationDate      *time.Time `json:"creation_date"`
	UpdatedDate       *time.Time `json:"updated_date"`
	RegistrantName    string     `json:"registrant_name" gorm:"size:255"`
	RegistrantOrg     string     `json:"registrant_org" gorm:"size:255"`
	RegistrantEmail   string     `json:"registrant_email" gorm:"size:255"`
	RegistrantPhone   string     `json:"registrant_phone" gorm:"size:64"`
	RegistrantCountry string     `json:"registrant_country" gorm:"size:8"`
	DNSSEC            string     `json:"dnssec" gorm:"size:64"`
	RegistrarWhois    string     `json:"registrar_whois" gorm:"size:255"`
	RegistrarURL      string     `json:"registrar_url" gorm:"size:512"`
	WhoisServer       string     `json:"whois_server" gorm:"size:255"`
	WhoisStatus       string     `json:"whois_status" gorm:"size:255"`
	WhoisRaw          string     `json:"whois_raw" gorm:"type:text"`
	WhoisUpdatedAt    *time.Time `json:"whois_updated_at"`

	ICPNumber       string     `json:"icp_number" gorm:"size:64"`
	ICPOwnerName    string     `json:"icp_owner_name" gorm:"size:255"`
	ICPOwnerType    string     `json:"icp_owner_type" gorm:"size:32"`
	ICPVerifyStatus string     `json:"icp_verify_status" gorm:"size:32"`
	ICPFilingDate   *time.Time `json:"icp_filing_date"`
	ICPServiceName  string     `json:"icp_service_name" gorm:"size:255"`
	ICPServiceURL   string     `json:"icp_service_url" gorm:"size:512"`
	ICPStatus       string     `json:"icp_status" gorm:"size:32;default:'unknown'"`
	RenewalPrice    float64    `json:"renewal_price" gorm:"default:0"`
	PriceSource     string     `json:"price_source" gorm:"size:32;default:''"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type DomainCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Registrar   string `json:"registrar"`
	ExpiryDate  string `json:"expiry_date"`
	Nameservers string `json:"nameservers"`
	AutoRenew   bool   `json:"auto_renew"`
	Note        string `json:"note"`
	Group       string `json:"group"`
	Tags        string `json:"tags"`
}

type DomainUpdateRequest struct {
	Name           string   `json:"name"`
	Registrar      string   `json:"registrar"`
	ExpiryDate     string   `json:"expiry_date"`
	Nameservers    string   `json:"nameservers"`
	AutoRenew      *bool    `json:"auto_renew"`
	Status         string   `json:"status"`
	Note           string   `json:"note"`
	Group          *string  `json:"group"`
	Tags           *string  `json:"tags"`
	CertCount      *int     `json:"cert_count"`
	AutoUpdate     *bool    `json:"auto_update"`
	UpdateICP      *bool    `json:"update_icp"`
	ExpiryReminder *bool    `json:"expiry_reminder"`
	RenewalPrice   *float64 `json:"renewal_price"`
}
