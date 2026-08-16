package handlers

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"DomainManager/database"
	"DomainManager/models"
	"DomainManager/services"

	"github.com/gin-gonic/gin"
)

func ListRegistrars(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var registrars []models.Registrar
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&registrars).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list registrars"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": registrars})
}

func GetRegistrar(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")
	var registrar models.Registrar
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&registrar).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "registrar not found"})
		return
	}
	c.JSON(http.StatusOK, registrar)
}

func CreateRegistrar(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req models.RegistrarCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	registrar := models.Registrar{
		UserID:      userID,
		Name:        req.Name,
		Type:        req.Type,
		APIEndpoint: req.APIEndpoint,
		APIKey:      req.APIKey,
		APISecret:   req.APISecret,
		APIExtra:    req.APIExtra,
		Enabled:     req.Enabled,
		SyncEnabled: req.SyncEnabled,
	}

	if err := database.DB.Create(&registrar).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "registrar name already exists"})
		return
	}

	c.JSON(http.StatusCreated, registrar)
}

func UpdateRegistrar(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")
	var registrar models.Registrar
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&registrar).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "registrar not found"})
		return
	}

	var req models.RegistrarUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.APIEndpoint != "" {
		updates["api_endpoint"] = req.APIEndpoint
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}
	if req.APISecret != "" {
		updates["api_secret"] = req.APISecret
	}
	if req.APIExtra != "" {
		updates["api_extra"] = req.APIExtra
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.SyncEnabled != nil {
		updates["sync_enabled"] = *req.SyncEnabled
	}

	if err := database.DB.Model(&registrar).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update registrar"})
		return
	}
	database.DB.Where("id = ? AND user_id = ?", id, userID).First(&registrar)
	c.JSON(http.StatusOK, registrar)
}

func DeleteRegistrar(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")
	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Registrar{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "registrar not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "registrar deleted"})
}

func ImportDomainsFromRegistrar(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req models.ImportDomainsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var registrar models.Registrar
	if err := database.DB.Where("id = ? AND user_id = ?", req.RegistrarID, userID).First(&registrar).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "registrar not found"})
		return
	}

	var imported int
	var skipped int
	var refreshed int

	if req.Domains != "" {
		lines := strings.Split(strings.TrimSpace(req.Domains), "\n")
		for _, line := range lines {
			domain := strings.TrimSpace(line)
			if domain == "" {
				continue
			}
			if !strings.Contains(domain, ".") {
				skipped++
				continue
			}

			var existing models.Domain
			result := database.DB.Where("name = ? AND user_id = ?", domain, userID).First(&existing)
			if result.Error == nil {
				// Domain exists: skip import, refresh WHOIS/ICP
				log.Printf("domain %s already exists, refreshing WHOIS/ICP", domain)
				refreshWhoisForDomain(&existing)
				refreshICPForDomain(&existing)
				now := time.Now()
				existing.WhoisUpdatedAt = &now
				if err := database.DB.Save(&existing).Error; err != nil {
					log.Printf("failed to save refreshed domain %s: %v", domain, err)
				}
				refreshed++
			} else {
				newDomain := models.Domain{
					UserID:    userID,
					Name:      domain,
					Registrar: registrar.Name,
					Status:    "active",
				}
				if err := database.DB.Create(&newDomain).Error; err != nil {
					log.Printf("failed to import domain %s: %v", domain, err)
					skipped++
					continue
				}
				imported++
			}
		}
	} else {
		log.Printf("auto-fetching domains from registrar %q (type=%q)", registrar.Name, registrar.Type)
		fetchedDomains, err := services.FetchDomainsFromRegistrar(registrar)
		if err != nil {
			log.Printf("fetch domains failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch domains: " + err.Error()})
			return
		}
		log.Printf("fetched %d domains from registrar %q", len(fetchedDomains), registrar.Name)

		for _, domain := range fetchedDomains {
			var existing models.Domain
			result := database.DB.Where("name = ? AND user_id = ?", domain.Name, userID).First(&existing)
			if result.Error == nil {
				// Domain exists: skip import, refresh WHOIS/ICP
				log.Printf("domain %s already exists, refreshing WHOIS/ICP", domain.Name)
				refreshWhoisForDomain(&existing)
				refreshICPForDomain(&existing)
				now := time.Now()
				existing.WhoisUpdatedAt = &now
				if err := database.DB.Save(&existing).Error; err != nil {
					log.Printf("failed to save refreshed domain %s: %v", domain.Name, err)
				}
				refreshed++
			} else {
				domain.UserID = userID
				domain.Registrar = registrar.Name
				if err := database.DB.Create(&domain).Error; err != nil {
					log.Printf("failed to import domain %s: %v", domain.Name, err)
					skipped++
					continue
				}
				imported++
			}
		}
	}

	now := time.Now()
	database.DB.Model(&registrar).Update("last_sync_at", now)

	c.JSON(http.StatusOK, gin.H{
		"imported":  imported,
		"refreshed": refreshed,
		"skipped":   skipped,
		"message":   fmt.Sprintf("导入完成: 新增 %d 个, 刷新 %d 个, 跳过 %d 个", imported, refreshed, skipped),
	})
}

func GetRegistrarTypes(c *gin.Context) {
	types := []map[string]string{
		{"value": "aliyun", "label": "阿里云（万网）", "region": "cn"},
		{"value": "aliyun_intl", "label": "阿里云（国际）", "region": "global"},
		{"value": "tencent", "label": "腾讯云（DNSPod）", "region": "cn"},
		{"value": "tencent_intl", "label": "腾讯云（国际）", "region": "global"},
		{"value": "huawei", "label": "华为云", "region": "cn"},
		{"value": "cloudflare", "label": "Cloudflare", "region": "global"},
		{"value": "godaddy", "label": "GoDaddy", "region": "global"},
		{"value": "namecheap", "label": "Namecheap", "region": "global"},
		{"value": "namesilo", "label": "NameSilo", "region": "global"},
		{"value": "porkbun", "label": "Porkbun", "region": "global"},
		{"value": "dynadot", "label": "Dynadot", "region": "global"},
		{"value": "digitalplat", "label": "DigitalPlat（免费域名）", "region": "global"},
		{"value": "amazon", "label": "AWS Route53", "region": "global"},
		{"value": "other", "label": "其他（手动导入）", "region": "other"},
	}
	c.JSON(http.StatusOK, gin.H{"data": types})
}

func ExportRegistrars(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var registrars []models.Registrar
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&registrars).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export registrars"})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=registrars_export.csv")
	// UTF-8 BOM so Excel opens Chinese headers correctly
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"名称", "类型", "API端点", "APIKey", "APISecret", "额外参数", "启用", "自动同步"})

	for _, r := range registrars {
		w.Write([]string{
			r.Name,
			r.Type,
			r.APIEndpoint,
			r.APIKey,
			r.APISecret,
			r.APIExtra,
			boolStr(r.Enabled),
			boolStr(r.SyncEnabled),
		})
	}
	w.Flush()
}

func ImportRegistrarsCSV(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse CSV: " + err.Error()})
		return
	}

	if len(records) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is empty or has no data rows"})
		return
	}

	header := records[0]
	colMap := map[string]int{}
	for i, h := range header {
		colMap[strings.TrimSpace(h)] = i
	}

	getCol := func(row []string, name string) string {
		if idx, ok := colMap[name]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	var created, updated, skipped int
	for _, row := range records[1:] {
		name := getCol(row, "名称")
		regType := getCol(row, "类型")
		if name == "" || regType == "" {
			skipped++
			continue
		}

		var existing models.Registrar
		if result := database.DB.Where("name = ? AND user_id = ?", name, userID).First(&existing); result.Error == nil {
			existing.Type = regType
			existing.APIEndpoint = getCol(row, "API端点")
			if v := getCol(row, "APIKey"); v != "" {
				existing.APIKey = v
			}
			if v := getCol(row, "APISecret"); v != "" {
				existing.APISecret = v
			}
			existing.APIExtra = getCol(row, "额外参数")
			if v := getCol(row, "启用"); v != "" {
				existing.Enabled = v == "true" || v == "TRUE" || v == "1"
			}
			if v := getCol(row, "自动同步"); v != "" {
				existing.SyncEnabled = v == "true" || v == "TRUE" || v == "1"
			}
			if err := database.DB.Save(&existing).Error; err != nil {
				log.Printf("failed to update registrar %s: %v", name, err)
				skipped++
				continue
			}
			updated++
		} else {
			newReg := models.Registrar{
				UserID:      userID,
				Name:        name,
				Type:        regType,
				APIEndpoint: getCol(row, "API端点"),
				APIKey:      getCol(row, "APIKey"),
				APISecret:   getCol(row, "APISecret"),
				APIExtra:    getCol(row, "额外参数"),
				Enabled:     getCol(row, "启用") == "true" || getCol(row, "启用") == "TRUE" || getCol(row, "启用") == "1",
				SyncEnabled: getCol(row, "自动同步") == "true" || getCol(row, "自动同步") == "TRUE" || getCol(row, "自动同步") == "1",
			}
			if err := database.DB.Create(&newReg).Error; err != nil {
				log.Printf("failed to create registrar %s: %v", name, err)
				skipped++
				continue
			}
			created++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"created": created,
		"updated": updated,
		"skipped": skipped,
		"message": fmt.Sprintf("导入完成: 新建 %d, 更新 %d, 跳过 %d", created, updated, skipped),
	})
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
