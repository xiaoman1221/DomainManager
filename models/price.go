package models

import "time"

type DomainPrice struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Registrar     string    `json:"registrar" gorm:"size:128;not null"`
	TLD           string    `json:"tld" gorm:"index;size:16;not null"`
	RegisterPrice float64   `json:"register_price"`
	RenewPrice    float64   `json:"renew_price"`
	TransferPrice float64   `json:"transfer_price"`
	Currency      string    `json:"currency" gorm:"size:8;default:'CNY'"`
	URL           string    `json:"url" gorm:"size:256"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PriceCompareRequest struct {
	Domain string `json:"domain" binding:"required"`
}

type PriceResult struct {
	Registrar     string  `json:"registrar"`
	TLD           string  `json:"tld"`
	RegisterPrice float64 `json:"register_price"`
	RenewPrice    float64 `json:"renew_price"`
	TransferPrice float64 `json:"transfer_price"`
	Currency      string  `json:"currency"`
	URL           string  `json:"url"`
}
