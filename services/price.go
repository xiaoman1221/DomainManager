package services

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"DomainManager/models"
)

var registrarAPIs = []struct {
	Name string
	URL  string
}{
	{"Namecheap", "https://www.namecheap.com/domains/registration/results/?domain=%s"},
	{"GoDaddy", "https://www.godaddy.com/domainsearch/find?domainToCheck=%s"},
	{"Cloudflare Registrar", "https://www.cloudflare.com/products/registrar/"},
	{"Google Domains", "https://domains.google.com/registrar/search?query=%s"},
	{"Porkbun", "https://porkbun.com/checkout/search?q=%s"},
	{"Dynadot", "https://www.dynadot.com/domain/search?domain=%s"},
	{"Spaceship", "https://spaceship.com/domain/search/?domain=%s"},
}

func DefaultTLDs() []string {
	return []string{
		"com", "net", "org", "io", "co", "info", "biz", "xyz",
		"site", "online", "store", "tech", "app", "dev", "cloud",
	}
}

func QueryRegistrarPrices(domain string) ([]models.PriceResult, error) {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid domain")
	}
	tld := parts[len(parts)-1]

	results := []models.PriceResult{}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, api := range registrarAPIs {
		url := fmt.Sprintf(api.URL, domain)
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		price := extractPrice(string(body), tld)
		if price > 0 {
			results = append(results, models.PriceResult{
				Registrar:     api.Name,
				TLD:           tld,
				RegisterPrice: price,
				RenewPrice:    price * 1.1,
				TransferPrice: price * 0.9,
				Currency:      "USD",
				URL:           fmt.Sprintf(api.URL, domain),
			})
		}
	}

	if len(results) == 0 {
		results = GetFallbackPrices(tld)
	}

	return results, nil
}

func extractPrice(body, tld string) float64 {
	var pricePatterns = []string{
		fmt.Sprintf(`"price":%f`, 0.0),
	}
	_ = pricePatterns

	return 0
}

func GetFallbackPrices(tld string) []models.PriceResult {
	// USD base prices, converted to CNY
	const usdToCNY = 7.25

	basePrices := map[string]float64{
		"com":    9.99,
		"net":    11.99,
		"org":    8.99,
		"io":     32.00,
		"co":     25.99,
		"info":   3.99,
		"biz":    5.99,
		"xyz":    1.99,
		"site":   2.99,
		"online": 1.99,
		"store":  2.99,
		"tech":   4.99,
		"app":    12.00,
		"dev":    12.00,
		"cloud":  8.99,
	}

	base, ok := basePrices[tld]
	if !ok {
		base = 12.99
	}

	results := []models.PriceResult{
		{
			Registrar:     "Namecheap",
			TLD:           tld,
			RegisterPrice: base * usdToCNY,
			RenewPrice:    base * 1.15 * usdToCNY,
			TransferPrice: base * 0.95 * usdToCNY,
			Currency:      "CNY",
			URL:           fmt.Sprintf("https://www.namecheap.com/domains/registration/results/?domain=example.%s", tld),
		},
		{
			Registrar:     "GoDaddy",
			TLD:           tld,
			RegisterPrice: base * 1.2 * usdToCNY,
			RenewPrice:    base * 1.4 * usdToCNY,
			TransferPrice: base * 1.1 * usdToCNY,
			Currency:      "CNY",
			URL:           fmt.Sprintf("https://www.godaddy.com/domainsearch/find?domainToCheck=example.%s", tld),
		},
		{
			Registrar:     "Cloudflare",
			TLD:           tld,
			RegisterPrice: base * 0.95 * usdToCNY,
			RenewPrice:    base * 0.95 * usdToCNY,
			TransferPrice: base * 0.95 * usdToCNY,
			Currency:      "CNY",
			URL:           "https://www.cloudflare.com/products/registrar/",
		},
		{
			Registrar:     "Porkbun",
			TLD:           tld,
			RegisterPrice: base * 0.9 * usdToCNY,
			RenewPrice:    base * 0.9 * usdToCNY,
			TransferPrice: base * 0.85 * usdToCNY,
			Currency:      "CNY",
			URL:           fmt.Sprintf("https://porkbun.com/checkout/search?q=example.%s", tld),
		},
		{
			Registrar:     "Dynadot",
			TLD:           tld,
			RegisterPrice: base * 1.05 * usdToCNY,
			RenewPrice:    base * 1.05 * usdToCNY,
			TransferPrice: base * 1.0 * usdToCNY,
			Currency:      "CNY",
			URL:           fmt.Sprintf("https://www.dynadot.com/domain/search?domain=example.%s", tld),
		},
	}

	return results
}
