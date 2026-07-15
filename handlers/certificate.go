package handlers

import (
	"DomainManager/database"
	"DomainManager/models"
	"DomainManager/services"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func ListCertificates(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var certs []models.Certificate
	query := database.DB.Where("user_id = ?", userID)

	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("domain LIKE ? OR issuer LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := c.Query("status"); status != "" {
		switch status {
		case "active":
			query = query.Where("status = ?", "active")
		case "expired":
			query = query.Where("status = ?", "expired")
		case "expiring_30":
			now := time.Now()
			limit := now.AddDate(0, 0, 30)
			query = query.Where("not_after IS NOT NULL AND not_after > ? AND not_after <= ?", now, limit)
		}
	}

	if err := query.Order("not_after ASC").Find(&certs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list certificates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": certs, "total": len(certs)})
}

func GetCertificate(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var cert models.Certificate
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&cert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}

	c.JSON(http.StatusOK, cert)
}

func CreateCertificate(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req models.CertificateCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cert := models.Certificate{
		UserID:          userID,
		Domain:          req.Domain,
		CertimateID:     req.CertimateID,
		Issuer:          req.Issuer,
		SerialNumber:    "",
		SubjectAltNames: req.SubjectAltNames,
		KeyAlgorithm:    req.KeyAlgorithm,
		Source:          req.Source,
		Certificate:     req.Certificate,
		PrivateKey:      req.PrivateKey,
		Note:            req.Note,
		Status:          "active",
	}

	if req.NotBefore != "" {
		t, err := time.Parse("2006-01-02", req.NotBefore)
		if err == nil {
			cert.NotBefore = &t
		}
	}
	if req.NotAfter != "" {
		t, err := time.Parse("2006-01-02", req.NotAfter)
		if err == nil {
			cert.NotAfter = &t
			if t.Before(time.Now()) {
				cert.IsExpired = true
				cert.Status = "expired"
			}
		}
	}

	if err := database.DB.Create(&cert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create certificate"})
		return
	}

	c.JSON(http.StatusCreated, cert)
}

func UpdateCertificate(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var cert models.Certificate
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&cert).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}

	var req models.CertificateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Domain != "" {
		cert.Domain = req.Domain
	}
	if req.Issuer != "" {
		cert.Issuer = req.Issuer
	}
	if req.SubjectAltNames != "" {
		cert.SubjectAltNames = req.SubjectAltNames
	}
	if req.KeyAlgorithm != "" {
		cert.KeyAlgorithm = req.KeyAlgorithm
	}
	if req.Source != "" {
		cert.Source = req.Source
	}
	if req.Certificate != "" {
		cert.Certificate = req.Certificate
	}
	if req.PrivateKey != "" {
		cert.PrivateKey = req.PrivateKey
	}
	if req.Note != "" {
		cert.Note = req.Note
	}
	if req.Status != "" {
		cert.Status = req.Status
	}
	if req.NotAfter != "" {
		t, err := time.Parse("2006-01-02", req.NotAfter)
		if err == nil {
			cert.NotAfter = &t
			cert.IsExpired = t.Before(time.Now())
			if cert.IsExpired {
				cert.Status = "expired"
			}
		}
	}

	if err := database.DB.Save(&cert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update certificate"})
		return
	}

	c.JSON(http.StatusOK, cert)
}

func DeleteCertificate(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Certificate{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete certificate"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func SyncCertimateCertificates(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var setting models.SystemSetting
	if err := database.DB.Where("`key` = ?", "certimate_config").First(&setting).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Certimate not configured"})
		return
	}

	var config models.CertimateConfig
	if err := json.Unmarshal([]byte(setting.Value), &config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid Certimate config"})
		return
	}

	svc := services.NewCertimateService(config)
	total, err := svc.SyncCertificates(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "sync completed", "total": total})
}

func GetCertimateConfig(c *gin.Context) {
	var setting models.SystemSetting
	if err := database.DB.Where("`key` = ?", "certimate_config").First(&setting).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "token": "", "configured": false})
		return
	}

	var config models.CertimateConfig
	json.Unmarshal([]byte(setting.Value), &config)

	c.JSON(http.StatusOK, gin.H{"url": config.URL, "token": config.Token, "configured": true})
}

func SaveCertimateConfig(c *gin.Context) {
	var req models.CertimateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := models.CertimateConfig{URL: req.URL, Token: req.Token}
	jsonData, _ := json.Marshal(config)

	var setting models.SystemSetting
	result := database.DB.Where("`key` = ?", "certimate_config").First(&setting)
	if result.Error == nil {
		setting.Value = string(jsonData)
		database.DB.Save(&setting)
	} else {
		setting = models.SystemSetting{Key: "certimate_config", Value: string(jsonData)}
		database.DB.Create(&setting)
	}

	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

func GetCertificateStats(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var total, active, expired, expiringSoon int64

	database.DB.Model(&models.Certificate{}).Where("user_id = ?", userID).Count(&total)
	database.DB.Model(&models.Certificate{}).Where("user_id = ? AND status = ?", userID, "active").Count(&active)
	database.DB.Model(&models.Certificate{}).Where("user_id = ? AND status = ?", userID, "expired").Count(&expired)

	now := time.Now()
	limit := now.AddDate(0, 0, 30)
	database.DB.Model(&models.Certificate{}).Where("user_id = ? AND not_after IS NOT NULL AND not_after > ? AND not_after <= ?", userID, now, limit).Count(&expiringSoon)

	c.JSON(http.StatusOK, gin.H{
		"total":        total,
		"active":       active,
		"expired":      expired,
		"expiring_soon": expiringSoon,
	})
}

// parseTime is a helper to parse time strings
func parseTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, err
		}
	}
	return &t, nil
}

// Helper to parse uint from string
func parseUint(s string) uint {
	v, _ := strconv.ParseUint(s, 10, 64)
	return uint(v)
}
