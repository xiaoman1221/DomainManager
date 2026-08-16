package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"DomainManager/database"
	"DomainManager/models"

	"github.com/gin-gonic/gin"
)

func CreateDomain(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req models.DomainCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	domain := models.Domain{
		UserID:      userID,
		Name:        req.Name,
		Registrar:   req.Registrar,
		Nameservers: req.Nameservers,
		AutoRenew:   req.AutoRenew,
		Note:        req.Note,
		Group:       req.Group,
		Tags:        req.Tags,
	}

	if req.ExpiryDate != "" {
		t, err := time.Parse("2006-01-02", req.ExpiryDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expiry_date format, use YYYY-MM-DD"})
			return
		}
		domain.ExpiryDate = &t
	}

	if err := database.DB.Create(&domain).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create domain"})
		return
	}

	c.JSON(http.StatusCreated, domain)
}

func ListDomains(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	query := database.DB.Where("user_id = ?", userID)

	if status := c.Query("status"); status != "" {
		switch status {
		case "active":
			query = query.Where("status = ?", "active")
		case "expired":
			now := time.Now()
			query = query.Where("(status = ? OR (expiry_date IS NOT NULL AND expiry_date < ?))", "expired", now)
		case "expiring_30":
			now := time.Now()
			limit := now.AddDate(0, 0, 30)
			query = query.Where("expiry_date IS NOT NULL AND expiry_date > ? AND expiry_date <= ?", now, limit)
		case "icp_registered":
			query = query.Where("icp_status = ?", "registered")
		case "icp_not_registered":
			query = query.Where("icp_status = ? OR icp_status = ? OR icp_status IS NULL", "not_found", "failed")
		}
	}
	if keyword := c.Query("keyword"); keyword != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(keyword)
		query = query.Where("name LIKE ? ESCAPE '\\'", "%"+escaped+"%")
	}

	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "DESC")
	allowedSorts := map[string]bool{
		"expiry_date": true, "renewal_price": true, "created_at": true, "name": true,
	}
	if !allowedSorts[sortBy] {
		sortBy = "created_at"
	}
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}
	// Sort with NULL values always last, regardless of direction.
	query = query.Order(fmt.Sprintf("CASE WHEN %s IS NULL THEN 1 ELSE 0 END, %s %s", sortBy, sortBy, sortOrder))

	var total int64
	if err := query.Model(&models.Domain{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list domains"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var domains []models.Domain
	if err := query.Order(fmt.Sprintf("CASE WHEN %s IS NULL THEN 1 ELSE 0 END, %s %s", sortBy, sortBy, sortOrder)).
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list domains"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": domains, "total": total})
}

func GetDomain(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var domain models.Domain
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&domain).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	c.JSON(http.StatusOK, domain)
}

func UpdateDomain(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var domain models.Domain
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&domain).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	var req models.DomainUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Registrar != "" {
		updates["registrar"] = req.Registrar
	}
	if req.ExpiryDate != "" {
		t, err := time.Parse("2006-01-02", req.ExpiryDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expiry_date format"})
			return
		}
		updates["expiry_date"] = t
	}
	if req.Nameservers != "" {
		updates["nameservers"] = req.Nameservers
	}
	if req.AutoRenew != nil {
		updates["auto_renew"] = *req.AutoRenew
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Note != "" {
		updates["note"] = req.Note
	}
	if req.Group != nil {
		updates["group"] = *req.Group
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.CertCount != nil {
		updates["cert_count"] = *req.CertCount
	}
	if req.AutoUpdate != nil {
		updates["auto_update"] = *req.AutoUpdate
	}
	if req.UpdateICP != nil {
		updates["update_icp"] = *req.UpdateICP
	}
	if req.ExpiryReminder != nil {
		updates["expiry_reminder"] = *req.ExpiryReminder
	}
	if req.RenewalPrice != nil {
		updates["renewal_price"] = *req.RenewalPrice
	}

	if err := database.DB.Model(&domain).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update domain"})
		return
	}

	database.DB.Where("id = ?", id).First(&domain)
	c.JSON(http.StatusOK, domain)
}

func DeleteDomain(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&models.Domain{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "domain deleted"})
}

func BatchDeleteDomains(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := database.DB.Where("id IN ? AND user_id = ?", req.IDs, userID).Delete(&models.Domain{})
	c.JSON(http.StatusOK, gin.H{"deleted": result.RowsAffected})
}

func BatchUpdateDomains(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req struct {
		IDs    []uint                 `json:"ids" binding:"required"`
		Fields map[string]interface{} `json:"fields" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only allow updating a fixed set of safe fields.
	allowedFields := map[string]bool{
		"name": true, "registrar": true, "status": true, "note": true,
		"group": true, "tags": true, "cert_count": true,
		"auto_renew": true, "auto_update": true, "update_icp": true,
		"expiry_reminder": true, "renewal_price": true, "nameservers": true,
	}
	filtered := map[string]interface{}{}
	for k, v := range req.Fields {
		if allowedFields[k] {
			filtered[k] = v
		}
	}
	if len(filtered) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no allowed fields to update"})
		return
	}

	result := database.DB.Where("id IN ? AND user_id = ?", req.IDs, userID).Model(&models.Domain{}).Updates(filtered)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to batch update domains"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": result.RowsAffected})
}

func GetDomainStats(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var total int64
	var active int64
	var expiringSoon int64

	database.DB.Model(&models.Domain{}).Where("user_id = ?", userID).Count(&total)
	now := time.Now()
	database.DB.Model(&models.Domain{}).
		Where("user_id = ? AND status = ? AND (expiry_date IS NULL OR expiry_date >= ?)", userID, "active", now).
		Count(&active)

	oneMonthLater := now.AddDate(0, 1, 0)
	database.DB.Model(&models.Domain{}).
		Where("user_id = ? AND expiry_date IS NOT NULL AND expiry_date > ? AND expiry_date <= ?", userID, now, oneMonthLater).
		Count(&expiringSoon)

	c.JSON(http.StatusOK, gin.H{
		"total":         total,
		"active":        active,
		"expiring_soon": expiringSoon,
	})
}

func ExportDomains(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var domains []models.Domain
	if err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export domains"})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=domains_export.csv")
	// UTF-8 BOM so Excel opens Chinese headers correctly
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"域名", "注册商", "到期时间", "分组", "标签", "备注", "自动续费", "NS服务器", "证书数量"})

	for _, d := range domains {
		expiry := ""
		if d.ExpiryDate != nil {
			expiry = d.ExpiryDate.Format("2006-01-02")
		}
		w.Write([]string{
			d.Name,
			d.Registrar,
			expiry,
			d.Group,
			d.Tags,
			d.Note,
			strconv.FormatBool(d.AutoRenew),
			d.Nameservers,
			strconv.Itoa(d.CertCount),
		})
	}
	w.Flush()
}

func ImportDomainsCSV(c *gin.Context) {
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

	var imported, skipped, updated int
	for _, row := range records[1:] {
		domain := getCol(row, "域名")
		if domain == "" || !strings.Contains(domain, ".") {
			skipped++
			continue
		}

		var existing models.Domain
		result := database.DB.Where("name = ? AND user_id = ?", domain, userID).First(&existing)
		if result.Error == nil {
			updates := map[string]interface{}{}
			if reg := getCol(row, "注册商"); reg != "" {
				updates["registrar"] = reg
			}
			if expiry := getCol(row, "到期时间"); expiry != "" {
				if t, err := time.Parse("2006-01-02", expiry); err == nil {
					updates["expiry_date"] = t
				}
			}
			if g := getCol(row, "分组"); g != "" {
				updates["group"] = g
			}
			if t := getCol(row, "标签"); t != "" {
				updates["tags"] = t
			}
			if n := getCol(row, "备注"); n != "" {
				updates["note"] = n
			}
			if ar := getCol(row, "自动续费"); ar != "" {
				updates["auto_renew"] = ar == "true" || ar == "TRUE" || ar == "1"
			}
			if ns := getCol(row, "NS服务器"); ns != "" {
				updates["nameservers"] = ns
			}
			if cc := getCol(row, "证书数量"); cc != "" {
				if v, err := strconv.Atoi(cc); err == nil {
					updates["cert_count"] = v
				}
			}
			database.DB.Model(&existing).Updates(updates)
			updated++
		} else {
			newDomain := models.Domain{
				UserID:      userID,
				Name:        domain,
				Registrar:   getCol(row, "注册商"),
				Status:      "active",
				Note:        getCol(row, "备注"),
				Group:       getCol(row, "分组"),
				Tags:        getCol(row, "标签"),
				Nameservers: getCol(row, "NS服务器"),
			}
			if expiry := getCol(row, "到期时间"); expiry != "" {
				if t, err := time.Parse("2006-01-02", expiry); err == nil {
					newDomain.ExpiryDate = &t
				}
			}
			if ar := getCol(row, "自动续费"); ar == "true" || ar == "TRUE" || ar == "1" {
				newDomain.AutoRenew = true
			}
			if cc := getCol(row, "证书数量"); cc != "" {
				if v, err := strconv.Atoi(cc); err == nil {
					newDomain.CertCount = v
				}
			}
			database.DB.Create(&newDomain)
			imported++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"imported": imported,
		"updated":  updated,
		"skipped":  skipped,
		"message":  fmt.Sprintf("导入完成: 新增 %d, 更新 %d, 跳过 %d", imported, updated, skipped),
	})
}
