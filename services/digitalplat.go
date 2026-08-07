package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"DomainManager/config"
	"DomainManager/models"
)

const (
	digitalPlatAPIDefault = "https://domain-api.digitalplat.org/api/v1"
	digitalPlatMaxPages   = 10
	digitalPlatPageSize   = 100
	digitalPlatUserAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

// digitalPlatSuffixes lists the free domain extensions offered by DigitalPlat.
var digitalPlatSuffixes = []string{
	"dpdns.org",
	"us.kg",
	"qzz.io",
	"xx.kg",
	"qd.je",
}

// IsDigitalPlatDomain reports whether the domain ends with a DigitalPlat free
// domain suffix.
func IsDigitalPlatDomain(domain string) bool {
	name := strings.TrimSpace(strings.ToLower(domain))
	for _, suffix := range digitalPlatSuffixes {
		if name == suffix || strings.HasSuffix(name, "."+suffix) {
			return true
		}
	}
	return false
}

// digitalPlatErrMessage extracts a human-readable error message from an API
// error body such as {"error":{"code":"invalid_api_key","message":"..."},"success":false}.
func digitalPlatErrMessage(body []byte) string {
	var payload struct {
		Error   interface{} `json:"error"`
		Message string      `json:"message"`
		Detail  string      `json:"detail"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "check the API key"
	}
	if payload.Message != "" {
		return payload.Message
	}
	if payload.Detail != "" {
		return payload.Detail
	}
	switch t := payload.Error.(type) {
	case string:
		if t != "" {
			return t
		}
	case map[string]interface{}:
		m := lowercaseKeys(t)
		if msg := getStr(m, "message", "detail", "error", "code"); msg != "" {
			return msg
		}
	}
	return "check the API key"
}

// digitalPlatClient builds an HTTP client with browser-like headers to avoid
// Cloudflare bot challenges on DigitalPlat endpoints.
func digitalPlatClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func digitalPlatBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", digitalPlatUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="126", "Google Chrome";v="126", "Not.A/Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
}

func fetchDigitalPlatDomains(registrar models.Registrar) ([]models.Domain, error) {
	token := strings.TrimSpace(registrar.APIKey)
	if token == "" {
		return nil, fmt.Errorf("digitalplat API key not configured")
	}

	base := strings.TrimSpace(registrar.APIEndpoint)
	if base == "" {
		base = digitalPlatAPIDefault
	}
	base = strings.TrimRight(base, "/")

	var domains []models.Domain
	for page := 1; page <= digitalPlatMaxPages; page++ {
		apiURL := fmt.Sprintf("%s/domains?page=%d&per_page=%d", base, page, digitalPlatPageSize)
		items, totalPages, err := digitalPlatFetchPage(apiURL, token)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			d := mapDigitalPlatDomain(item)
			if d == nil {
				continue
			}
			d.Registrar = "DigitalPlat"
			domains = append(domains, *d)
		}

		if totalPages <= page || len(items) == 0 {
			break
		}
	}

	log.Printf("[digitalplat] parsed %d domains", len(domains))
	return domains, nil
}

func digitalPlatFetchPage(apiURL, token string) ([]interface{}, int, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create digitalplat request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	digitalPlatBrowserHeaders(req)

	client := digitalPlatClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("digitalplat API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("digitalplat failed to read response: %w", err)
	}

	log.Printf("[digitalplat] HTTP %d, body length=%d", resp.StatusCode, len(body))

	switch resp.StatusCode {
	case 401:
		return nil, 0, fmt.Errorf("digitalplat API authentication failed (HTTP 401): %s", digitalPlatErrMessage(body))
	case 403:
		if strings.Contains(resp.Header.Get("Cf-Mitigated"), "challenge") {
			return nil, 0, fmt.Errorf("digitalplat API returned HTTP 403: blocked by Cloudflare bot protection (Cf-Mitigated: challenge); retry from a residential/uncensored network or configure an HTTP proxy")
		}
		return nil, 0, fmt.Errorf("digitalplat API returned HTTP 403 (forbidden): %s", digitalPlatErrMessage(body))
	}
	if resp.StatusCode != 200 {
		return nil, 0, fmt.Errorf("digitalplat API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var payload struct {
		Success bool                   `json:"success"`
		Data    interface{}            `json:"data"`
		Meta    map[string]interface{} `json:"meta"`
		Error   interface{}            `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, 0, fmt.Errorf("failed to parse digitalplat response: %w", err)
	}

	if !payload.Success {
		msg := ""
		switch t := payload.Error.(type) {
		case string:
			msg = t
		case map[string]interface{}:
			msg = getStr(lowercaseKeys(t), "message", "detail", "error")
		}
		if msg == "" {
			msg = "unknown error"
		}
		return nil, 0, fmt.Errorf("digitalplat API error: %s", msg)
	}

	items, err := extractDomainItems(payload.Data)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse digitalplat data: %w", err)
	}

	totalPages := getMetaInt(payload.Meta, "last_page", "total_pages", "pages")
	return items, totalPages, nil
}

func extractDomainItems(data interface{}) ([]interface{}, error) {
	switch t := data.(type) {
	case []interface{}:
		return t, nil
	case map[string]interface{}:
		m := lowercaseKeys(t)
		for _, k := range []string{"domains", "items", "data", "list", "records", "results"} {
			if v, ok := m[k]; ok {
				if arr, ok := v.([]interface{}); ok {
					return arr, nil
				}
			}
		}
		return nil, fmt.Errorf("data object has no domain list field")
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected data type %T", data)
	}
}

func getMetaInt(m map[string]interface{}, keys ...string) int {
	m = lowercaseKeys(m)
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case float64:
				return int(t)
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
					return n
				}
			}
		}
	}
	return 0
}

func mapDigitalPlatDomain(item interface{}) *models.Domain {
	m, ok := item.(map[string]interface{})
	if !ok {
		return nil
	}
	m = lowercaseKeys(m)

	name := getStr(m, "name", "domain", "domain_name", "domainname")
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" || !strings.Contains(name, ".") {
		return nil
	}

	d := &models.Domain{
		Name:   name,
		Status: "active",
	}
	if st := getStr(m, "status", "state", "domain_status", "domainstatus"); st != "" {
		d.Status = st
	}
	if t := getTime(m, "expiry_date", "expiration_date", "expires_at", "expirydate", "expiresat", "expirationdate", "expiry"); t != nil {
		d.ExpiryDate = t
	}
	if t := getTime(m, "created_at", "registration_date", "createdat", "created", "registrationdate"); t != nil {
		d.RegistrationDate = t
	}
	if t := getTime(m, "updated_at", "updated_date", "updatedat", "updated"); t != nil {
		d.UpdatedDate = t
	}
	if ns := getStringOrSlice(m, "nameservers", "nameserver", "name_servers", "nserver", "ns"); ns != "" {
		d.Nameservers = ns
	}
	if getBool(m, "auto_renew", "autorenew", "auto_renewal") {
		d.AutoRenew = true
	}
	return d
}

// QueryDigitalPlatWhois queries the DigitalPlat official RDAP server
// (rdap.digitalplat.org/domain/<name>) for WHOIS data.
func QueryDigitalPlatWhois(domain string) (*WhoisInfo, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	base := config.AppConfig.DigitalPlatRDAPURL
	if base == "" {
		base = "https://rdap.digitalplat.org"
	}
	base = strings.TrimRight(base, "/")

	apiURL := fmt.Sprintf("%s/domain/%s", base, url.PathEscape(domain))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create DigitalPlat RDAP request: %w", err)
	}
	digitalPlatBrowserHeaders(req)
	req.Header.Set("Accept", "application/rdap+json, application/json, text/plain, */*")

	client := digitalPlatClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("DigitalPlat RDAP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read DigitalPlat RDAP response: %w", err)
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 403 && strings.Contains(resp.Header.Get("Cf-Mitigated"), "challenge") {
			return nil, fmt.Errorf("DigitalPlat RDAP returned HTTP 403: blocked by Cloudflare bot protection; retry from a residential/uncensored network or configure an HTTP proxy")
		}
		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("DigitalPlat RDAP: domain %s not found", domain)
		}
		return nil, fmt.Errorf("DigitalPlat RDAP returned HTTP %d", resp.StatusCode)
	}

	var rdap struct {
		ObjectClassName string       `json:"objectClassName"`
		Handle          string       `json:"handle"`
		LDHName         string       `json:"ldhName"`
		UnicodeName     string       `json:"unicodeName"`
		Status          []string     `json:"status"`
		Events          []rdapEvent  `json:"events"`
		Nameservers     []rdapNS     `json:"nameservers"`
		Entities        []rdapEntity `json:"entities"`
	}
	if err := json.Unmarshal(body, &rdap); err != nil {
		return nil, fmt.Errorf("failed to parse DigitalPlat RDAP response: %w", err)
	}

	if rdap.ObjectClassName != "" && rdap.ObjectClassName != "domain" {
		return nil, fmt.Errorf("DigitalPlat RDAP returned unexpected object %q", rdap.ObjectClassName)
	}

	info := &WhoisInfo{
		DomainName:  rdap.LDHName,
		Registrar:   rdapEntityVCard(rdap.Entities, "registrar", "fn"),
		RawText:     string(body),
		Nameservers: joinRDAPNameservers(rdap.Nameservers),
		Status:      strings.Join(rdap.Status, ", "),
	}
	if info.DomainName == "" {
		info.DomainName = rdap.Handle
	}
	if info.DomainName == "" {
		info.DomainName = rdap.UnicodeName
	}

	for _, e := range rdap.Events {
		t := parseISODate(e.EventDate)
		if t == nil {
			continue
		}
		switch e.EventAction {
		case "registration":
			info.CreationDate = t
		case "expiration":
			info.ExpiryDate = t
		case "last changed":
			info.UpdatedDate = t
		}
	}

	info.RegistrantName = rdapEntityVCard(rdap.Entities, "registrant", "fn")
	info.RegistrantEmail = rdapEntityVCard(rdap.Entities, "registrant", "email")

	if info.DomainName == "" && info.ExpiryDate == nil && info.RawText == "" {
		return nil, fmt.Errorf("DigitalPlat RDAP returned no usable data for %s", domain)
	}

	return info, nil
}

type rdapEvent struct {
	EventAction string `json:"eventAction"`
	EventDate   string `json:"eventDate"`
}

type rdapNS struct {
	LDHName string `json:"ldhName"`
}

type rdapEntity struct {
	Roles      []string    `json:"roles"`
	VCardArray interface{} `json:"vcardArray"`
}

// rdapEntityVCard extracts a vCard field (e.g. "fn", "email") from the entity
// that has the given role (e.g. "registrar", "registrant").
func rdapEntityVCard(entities []rdapEntity, role, field string) string {
	for _, e := range entities {
		if !stringInSlice(e.Roles, role) {
			continue
		}
		if v, ok := e.VCardArray.([]interface{}); ok && len(v) >= 2 {
			if props, ok := v[1].([]interface{}); ok {
				for _, p := range props {
					arr, ok := p.([]interface{})
					if !ok || len(arr) < 4 {
						continue
					}
					name, _ := arr[0].(string)
					if !strings.EqualFold(name, field) {
						continue
					}
					if s, ok := arr[3].(string); ok {
						if s = strings.TrimSpace(s); s != "" {
							return s
						}
					}
				}
			}
		}
	}
	return ""
}

func joinRDAPNameservers(ns []rdapNS) string {
	if len(ns) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ns))
	for _, n := range ns {
		if s := strings.TrimSpace(n.LDHName); s != "" {
			parts = append(parts, strings.ToLower(s))
		}
	}
	return strings.Join(parts, ",")
}

func stringInSlice(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

func lowercaseKeys(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		lk := strings.ToLower(k)
		switch t := v.(type) {
		case map[string]interface{}:
			out[lk] = lowercaseKeys(t)
		case []interface{}:
			arr := make([]interface{}, len(t))
			for i, e := range t {
				if em, ok := e.(map[string]interface{}); ok {
					arr[i] = lowercaseKeys(em)
				} else {
					arr[i] = e
				}
			}
			out[lk] = arr
		default:
			out[lk] = v
		}
	}
	return out
}

func getStr(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case float64:
				return strconv.FormatFloat(t, 'f', -1, 64)
			case bool:
				return strconv.FormatBool(t)
			}
		}
	}
	return ""
}

func getTime(m map[string]interface{}, keys ...string) *time.Time {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			s := strings.TrimSpace(t)
			if s == "" {
				continue
			}
			if tt := parseISODate(s); tt != nil {
				return tt
			}
			if ts, err := strconv.ParseInt(s, 10, 64); err == nil && ts > 0 {
				return unixToTime(ts)
			}
		case float64:
			if t > 0 {
				return unixToTime(int64(t))
			}
		}
	}
	return nil
}

func unixToTime(ts int64) *time.Time {
	var t time.Time
	if ts >= 1_000_000_000_000 {
		t = time.UnixMilli(ts)
	} else {
		t = time.Unix(ts, 0)
	}
	return &t
}

func getStringOrSlice(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			parts := strings.FieldsFunc(t, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t' })
			var cleaned []string
			for _, p := range parts {
				p = strings.TrimSpace(strings.ToLower(p))
				if p != "" {
					cleaned = append(cleaned, p)
				}
			}
			if len(cleaned) > 0 {
				return strings.Join(cleaned, ",")
			}
		case []interface{}:
			var parts []string
			for _, e := range t {
				if s, ok := e.(string); ok && strings.TrimSpace(s) != "" {
					parts = append(parts, strings.TrimSpace(strings.ToLower(s)))
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, ",")
			}
		}
	}
	return ""
}

func getBool(m map[string]interface{}, keys ...string) bool {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case bool:
				return t
			case string:
				switch strings.ToLower(strings.TrimSpace(t)) {
				case "true", "1", "yes", "on", "enabled":
					return true
				}
			case float64:
				return t == 1
			}
		}
	}
	return false
}
