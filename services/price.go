package services

import (
	"fmt"
	"strings"

	"DomainManager/models"
)

// DefaultTLDs returns the list of TLDs with built-in reference prices.
func DefaultTLDs() []string {
	return []string{
		"com", "net", "org", "io", "co", "info", "biz", "xyz",
		"site", "online", "store", "tech", "app", "dev", "cloud",
	}
}

// QueryRegistrarPrices returns reference prices for the domain's TLD.
//
// NOTE: live scraping of registrar websites is intentionally NOT performed.
// The values come from the built-in baseline table in GetFallbackPrices and
// are estimates only; they are flagged with Reference=true so the UI can label
// them as 参考价.
func QueryRegistrarPrices(domain string) ([]models.PriceResult, error) {
	parts := strings.Split(strings.TrimSpace(strings.ToLower(domain)), ".")
	if len(parts) < 2 || parts[len(parts)-1] == "" {
		return nil, fmt.Errorf("invalid domain")
	}
	tld := parts[len(parts)-1]
	return GetFallbackPrices(tld), nil
}

// GetFallbackPrices returns hard-coded baseline (reference) prices in CNY.
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
			Reference:     true,
		},
		{
			Registrar:     "GoDaddy",
			TLD:           tld,
			RegisterPrice: base * 1.2 * usdToCNY,
			RenewPrice:    base * 1.4 * usdToCNY,
			TransferPrice: base * 1.1 * usdToCNY,
			Currency:      "CNY",
			URL:           fmt.Sprintf("https://www.godaddy.com/domainsearch/find?domainToCheck=example.%s", tld),
			Reference:     true,
		},
		{
			Registrar:     "Cloudflare",
			TLD:           tld,
			RegisterPrice: base * 0.95 * usdToCNY,
			RenewPrice:    base * 0.95 * usdToCNY,
			TransferPrice: base * 0.95 * usdToCNY,
			Currency:      "CNY",
			URL:           "https://www.cloudflare.com/products/registrar/",
			Reference:     true,
		},
		{
			Registrar:     "Porkbun",
			TLD:           tld,
			RegisterPrice: base * 0.9 * usdToCNY,
			RenewPrice:    base * 0.9 * usdToCNY,
			TransferPrice: base * 0.85 * usdToCNY,
			Currency:      "CNY",
			URL:           fmt.Sprintf("https://porkbun.com/checkout/search?q=example.%s", tld),
			Reference:     true,
		},
		{
			Registrar:     "Dynadot",
			TLD:           tld,
			RegisterPrice: base * 1.05 * usdToCNY,
			RenewPrice:    base * 1.05 * usdToCNY,
			TransferPrice: base * 1.0 * usdToCNY,
			Currency:      "CNY",
			URL:           fmt.Sprintf("https://www.dynadot.com/domain/search?domain=example.%s", tld),
			Reference:     true,
		},
	}

	return results
}
