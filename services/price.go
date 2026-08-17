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
// The table covers international registrars (USD base x exchange rate),
// major Chinese registrars (CNY base) and aggregator/price-comparison sites.
func GetFallbackPrices(tld string) []models.PriceResult {
	// USD base prices (per year), converted to CNY for display.
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

	// CNY renewal baselines for Chinese registrars (first-year promo is
	// usually cheaper; renewal price is the stable anchor).
	cnyBasePrices := map[string]float64{
		"com":    85,
		"net":    92,
		"org":    85,
		"io":     268,
		"co":     168,
		"info":   62,
		"biz":    62,
		"xyz":    48,
		"site":   45,
		"online": 45,
		"store":  55,
		"tech":   55,
		"app":    99,
		"dev":    99,
		"cloud":  85,
	}

	base, ok := basePrices[tld]
	if !ok {
		base = 12.99
	}
	cnyBase, ok := cnyBasePrices[tld]
	if !ok {
		cnyBase = 69
	}
	domain := "example." + tld

	type platform struct {
		registrar string
		register  float64 // price factor relative to the base
		renew     float64
		transfer  float64
		cny       bool // use the CNY baseline instead of USD baseline
		url       string
	}

	platforms := []platform{
		// ---- International registrars ----
		{"Namecheap", 1.00, 1.15, 0.95, false, fmt.Sprintf("https://www.namecheap.com/domains/registration/results/?domain=%s", domain)},
		{"GoDaddy", 1.20, 1.40, 1.10, false, fmt.Sprintf("https://www.godaddy.com/domainsearch/find?domainToCheck=%s", domain)},
		{"Cloudflare", 0.95, 0.95, 0.95, false, "https://www.cloudflare.com/products/registrar/"},
		{"Porkbun", 0.90, 0.90, 0.85, false, fmt.Sprintf("https://porkbun.com/checkout/search?q=%s", domain)},
		{"Dynadot", 1.05, 1.05, 1.00, false, fmt.Sprintf("https://www.dynadot.com/domain/search?domain=%s", domain)},
		{"Spaceship", 0.92, 0.92, 0.88, false, fmt.Sprintf("https://www.spaceship.com/domains/?search=%s", domain)},
		{"Sav.com", 0.85, 0.98, 0.88, false, fmt.Sprintf("https://www.sav.com/domains?q=%s", domain)},

		// ---- Major Chinese registrars (CNY) ----
		{"阿里云", 0.62, 1.00, 0.70, true, fmt.Sprintf("https://wanwang.aliyun.com/domain/searchresult/#/keyword/%s", domain)},
		{"腾讯云", 0.62, 1.00, 0.70, true, "https://buy.cloud.tencent.com/domain"},
		{"华为云", 0.70, 1.02, 0.75, true, "https://www.huaweicloud.com/product/domain.html"},
		{"西部数码", 0.58, 0.98, 0.68, true, "https://www.west.cn/domains/"},
		{"新网", 0.65, 1.05, 0.72, true, "https://www.xinnet.com/domain/"},

		// ---- Price-comparison / aggregator platforms ----
		{"nazhumi 比价", 1.00, 1.08, 0.98, false, "https://www.nazhumi.com/"},
		{"NameBeta 比价", 0.98, 1.05, 0.95, false, "https://namebeta.com/"},
		{"TLD-List 比价", 1.00, 1.10, 1.00, false, "https://zh-hans.tld-list.com/"},
		{"Domcomp 比价", 0.95, 1.02, 0.92, false, "https://www.domcomp.com/"},
	}

	results := make([]models.PriceResult, 0, len(platforms))
	for _, p := range platforms {
		ref, mult := base, usdToCNY
		if p.cny {
			ref, mult = cnyBase, 1
		}
		results = append(results, models.PriceResult{
			Registrar:     p.registrar,
			TLD:           tld,
			RegisterPrice: roundPrice(p.register * ref * mult),
			RenewPrice:    roundPrice(p.renew * ref * mult),
			TransferPrice: roundPrice(p.transfer * ref * mult),
			Currency:      "CNY",
			URL:           p.url,
			Reference:     true,
		})
	}
	return results
}

// roundPrice keeps reference prices readable (two decimals at most).
func roundPrice(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
