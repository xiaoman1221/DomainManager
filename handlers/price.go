package handlers

import (
	"net/http"
	"strings"

	"DomainManager/database"
	"DomainManager/models"
	"DomainManager/services"

	"github.com/gin-gonic/gin"
)

func ComparePrices(c *gin.Context) {
	var req models.PriceCompareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	domain = strings.TrimSuffix(domain, ".")
	parts := strings.Split(domain, ".")
	if len(parts) < 2 || parts[len(parts)-1] == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain format"})
		return
	}
	tld := parts[len(parts)-1]

	var dbPrices []models.DomainPrice
	database.DB.Where("tld = ?", tld).Find(&dbPrices)

	apiPrices, err := services.QueryRegistrarPrices(domain)
	if err != nil {
		apiPrices = []models.PriceResult{}
	}

	allPrices := make([]models.PriceResult, 0)
	// DB rows are legacy cached data; label them as reference prices.
	for _, p := range dbPrices {
		allPrices = append(allPrices, models.PriceResult{
			Registrar:     p.Registrar,
			TLD:           p.TLD,
			RegisterPrice: p.RegisterPrice,
			RenewPrice:    p.RenewPrice,
			TransferPrice: p.TransferPrice,
			Currency:      p.Currency,
			URL:           p.URL,
			Reference:     true,
		})
	}
	allPrices = append(allPrices, apiPrices...)

	c.JSON(http.StatusOK, gin.H{
		"domain": domain,
		"tld":    tld,
		"prices": allPrices,
	})
}

func GetSupportedTLDs(c *gin.Context) {
	var prices []models.DomainPrice
	database.DB.Distinct("tld").Find(&prices)

	tlds := make([]string, 0, len(prices))
	for _, p := range prices {
		tlds = append(tlds, p.TLD)
	}

	if len(tlds) == 0 {
		tlds = services.DefaultTLDs()
	}

	c.JSON(http.StatusOK, gin.H{"tlds": tlds})
}

func RefreshPrices(c *gin.Context) {
	domain := strings.ToLower(strings.TrimSpace(c.Query("domain")))
	if domain == "" {
		domain = "example.com"
	}

	// Live registrar prices are not available; return the built-in reference
	// prices without writing them to the database as if they were real quotes.
	prices, err := services.QueryRegistrarPrices(domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "当前仅提供估算参考价，未写入数据库",
		"prices":  prices,
		"count":   len(prices),
	})
}
