package handlers

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"DomainManager/database"
	"DomainManager/models"
	"DomainManager/services"

	"github.com/gin-gonic/gin"
)

func QueryWhois(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain parameter is required"})
		return
	}

	info, err := services.QueryWhois(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

func QueryICP(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain parameter is required"})
		return
	}

	info, err := services.QueryICP(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

func RefreshDomainInfo(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var domain models.Domain
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&domain).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	log.Printf("refreshing WHOIS+ICP for domain %q (id=%d)", domain.Name, domain.ID)

	whoisErr := refreshWhoisForDomain(&domain)
	icpErr := refreshICPForDomain(&domain)

	now := time.Now()
	domain.WhoisUpdatedAt = &now
	database.DB.Save(&domain)

	resp := gin.H{
		"message": "refresh completed",
		"domain":  domain,
	}
	if whoisErr != nil {
		resp["whois_error"] = whoisErr.Error()
	}
	if icpErr != nil {
		resp["icp_error"] = icpErr.Error()
	}

	c.JSON(http.StatusOK, resp)
}

func refreshWhoisForDomain(domain *models.Domain) error {
	info, err := services.QueryWhois(domain.Name)
	if err != nil {
		log.Printf("WHOIS query failed for %s: %v", domain.Name, err)
		return err
	}

	if info.Registrar != "" {
		domain.RegistrarWhois = info.Registrar
	}
	if info.RegistrarURL != "" {
		domain.RegistrarURL = info.RegistrarURL
	}
	if info.WhoisServer != "" {
		domain.WhoisServer = info.WhoisServer
	}
	if info.CreationDate != nil {
		domain.CreationDate = info.CreationDate
	}
	if info.UpdatedDate != nil {
		domain.UpdatedDate = info.UpdatedDate
	}
	if info.ExpiryDate != nil {
		domain.ExpiryDate = info.ExpiryDate
	}
	if info.RegistrantName != "" {
		domain.RegistrantName = info.RegistrantName
	}
	if info.RegistrantOrg != "" {
		domain.RegistrantOrg = info.RegistrantOrg
	}
	if info.RegistrantEmail != "" {
		domain.RegistrantEmail = info.RegistrantEmail
	}
	if info.RegistrantPhone != "" {
		domain.RegistrantPhone = info.RegistrantPhone
	}
	if info.RegistrantCountry != "" {
		domain.RegistrantCountry = info.RegistrantCountry
	}
	if info.Nameservers != "" {
		domain.Nameservers = info.Nameservers
	}
	if info.DNSSEC != "" {
		domain.DNSSEC = info.DNSSEC
	}
	if info.Status != "" {
		domain.WhoisStatus = info.Status
	}
	domain.WhoisRaw = info.RawText

	log.Printf("WHOIS updated for %s: registrar=%s, expiry=%v", domain.Name, domain.RegistrarWhois, domain.ExpiryDate)
	return nil
}

func refreshICPForDomain(domain *models.Domain) error {
	info, err := services.QueryICP(domain.Name)
	if err != nil {
		log.Printf("ICP query failed for %s: %v", domain.Name, err)
		domain.ICPStatus = "failed"
		return err
	}

	if info.VerifyStatus == "未备案" {
		domain.ICPStatus = "not_found"
		domain.ICPNumber = ""
		domain.ICPOwnerName = ""
		domain.ICPOwnerType = ""
		domain.ICPVerifyStatus = ""
		domain.ICPFilingDate = nil
		domain.ICPServiceName = ""
		domain.ICPServiceURL = ""
	} else {
		domain.ICPStatus = "registered"
		if info.ICPNumber != "" {
			domain.ICPNumber = info.ICPNumber
		}
		if info.OwnerName != "" {
			domain.ICPOwnerName = info.OwnerName
		}
		if info.OwnerType != "" {
			domain.ICPOwnerType = info.OwnerType
		}
		if info.VerifyStatus != "" {
			domain.ICPVerifyStatus = info.VerifyStatus
		}
		if info.FilingDate != nil {
			domain.ICPFilingDate = info.FilingDate
		}
		if info.ServiceName != "" {
			domain.ICPServiceName = info.ServiceName
		}
		if info.ServiceURL != "" {
			domain.ICPServiceURL = info.ServiceURL
		}
	}

	log.Printf("ICP updated for %s: status=%s, number=%s", domain.Name, domain.ICPStatus, domain.ICPNumber)
	return nil
}

func QueryRenewalPrice(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)
	id := c.Param("id")

	var domain models.Domain
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&domain).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	// Try registrar API first
	if domain.Registrar != "" {
		var registrar models.Registrar
		if err := database.DB.Where("name = ?", domain.Registrar).First(&registrar).Error; err == nil {
			result, err := services.QueryRenewalPrice(domain.Name, registrar)
			if err == nil && result.Error == "" && result.Price > 0 {
				domain.RenewalPrice = result.Price
				domain.PriceSource = result.Source
				database.DB.Save(&domain)
				c.JSON(http.StatusOK, result)
				return
			}
			log.Printf("registrar price query failed for %s: %v (error: %s)", domain.Name, err, result.Error)
		}
	}

	// Fallback: use built-in price comparison data, pick lowest renewal price
	// USD to CNY fixed rate
	const usdToCNY = 7.25

	parts := strings.Split(domain.Name, ".")
	tld := ""
	if len(parts) >= 2 {
		tld = parts[len(parts)-1]
	}
	fallbackPrices := services.GetFallbackPrices(tld)
	lowestPrice := 0.0
	for _, p := range fallbackPrices {
		if lowestPrice == 0 || p.RenewPrice < lowestPrice {
			lowestPrice = p.RenewPrice
		}
	}

	if lowestPrice > 0 {
		cnyPrice := lowestPrice * usdToCNY
		domain.RenewalPrice = cnyPrice
		domain.PriceSource = "fallback"
		database.DB.Save(&domain)
		c.JSON(http.StatusOK, &services.RenewalPriceResult{
			Domain:   domain.Name,
			Price:    cnyPrice,
			Currency: "CNY",
			Source:   "fallback",
		})
		return
	}

	c.JSON(http.StatusOK, &services.RenewalPriceResult{
		Domain: domain.Name,
		Error:  "no price data available",
	})
}

func BatchQueryRenewalPrice(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
		return
	}
	if len(req.IDs) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many domains, max 200"})
		return
	}

	var domains []models.Domain
	if err := database.DB.Where("id IN ? AND user_id = ?", req.IDs, userID).Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query domains"})
		return
	}

	// Cache registrar lookups
	registrarCache := make(map[string]models.Registrar)
	var cacheMu sync.Mutex

	type result struct {
		ID     uint   `json:"id"`
		Name   string `json:"name"`
		Price  float64 `json:"price"`
		Source string `json:"source"`
		Error  string `json:"error,omitempty"`
	}

	results := make([]result, len(domains))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8) // max 8 concurrent

	for i, d := range domains {
		wg.Add(1)
		go func(idx int, dom models.Domain) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r := result{ID: dom.ID, Name: dom.Name}

			// Try registrar API first
			if dom.Registrar != "" {
				cacheMu.Lock()
				reg, ok := registrarCache[dom.Registrar]
				if !ok {
					if err := database.DB.Where("name = ?", dom.Registrar).First(&reg).Error; err == nil {
						registrarCache[dom.Registrar] = reg
						ok = true
					}
				}
				cacheMu.Unlock()

				if ok {
					priceResult, err := services.QueryRenewalPrice(dom.Name, reg)
					if err == nil && priceResult.Error == "" && priceResult.Price > 0 {
						r.Price = priceResult.Price
						r.Source = priceResult.Source
						database.DB.Model(&dom).Updates(map[string]interface{}{
							"renewal_price": priceResult.Price,
							"price_source":  priceResult.Source,
						})
						results[idx] = r
						return
					}
					log.Printf("batch price query failed for %s: %v (error: %s)", dom.Name, err, priceResult.Error)
				}
			}

			// Fallback: built-in price comparison data
			const usdToCNY = 7.25
			parts := strings.Split(dom.Name, ".")
			tld := ""
			if len(parts) >= 2 {
				tld = parts[len(parts)-1]
			}
			fallbackPrices := services.GetFallbackPrices(tld)
			lowestPrice := 0.0
			for _, p := range fallbackPrices {
				if lowestPrice == 0 || p.RenewPrice < lowestPrice {
					lowestPrice = p.RenewPrice
				}
			}
			if lowestPrice > 0 {
				cnyPrice := lowestPrice * usdToCNY
				r.Price = cnyPrice
				r.Source = "fallback"
				database.DB.Model(&dom).Updates(map[string]interface{}{
					"renewal_price": cnyPrice,
					"price_source":  "fallback",
				})
			} else {
				r.Error = "no price data available"
			}
			results[idx] = r
		}(i, d)
	}

	wg.Wait()

	c.JSON(http.StatusOK, gin.H{"data": results})
}
