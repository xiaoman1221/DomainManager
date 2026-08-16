package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"DomainManager/config"
)

type WhoisInfo struct {
	DomainName        string     `json:"domain_name"`
	Registrar         string     `json:"registrar"`
	RegistrarURL      string     `json:"registrar_url"`
	WhoisServer       string     `json:"whois_server"`
	UpdatedDate       *time.Time `json:"updated_date"`
	CreationDate      *time.Time `json:"creation_date"`
	ExpiryDate        *time.Time `json:"expiry_date"`
	RegistrantName    string     `json:"registrant_name"`
	RegistrantOrg     string     `json:"registrant_org"`
	RegistrantEmail   string     `json:"registrant_email"`
	RegistrantPhone   string     `json:"registrant_phone"`
	RegistrantCountry string     `json:"registrant_country"`
	Nameservers       string     `json:"nameservers"`
	DNSSEC            string     `json:"dnssec"`
	Status            string     `json:"status"`
	RawText           string     `json:"raw_text"`
}

type whoisAPIResponse struct {
	Time   float64 `json:"time"`
	Status bool    `json:"status"`
	Error  string  `json:"error,omitempty"`
	Result *struct {
		Domain         string `json:"domain"`
		Registrar      string `json:"registrar"`
		RegistrarURL   string `json:"registrarURL"`
		IANAID         string `json:"ianaId"`
		WhoisServer    string `json:"whoisServer"`
		UpdatedDate    string `json:"updatedDate"`
		CreationDate   string `json:"creationDate"`
		ExpirationDate string `json:"expirationDate"`
		Status         []struct {
			Status string `json:"status"`
			URL    string `json:"url"`
		} `json:"status"`
		NameServers            []string `json:"nameServers"`
		RegistrantName         string   `json:"registrantName"`
		RegistrantOrganization string   `json:"registrantOrganization"`
		RegistrantProvince     string   `json:"registrantProvince"`
		RegistrantCountry      string   `json:"registrantCountry"`
		RegistrantPhone        string   `json:"registrantPhone"`
		RegistrantEmail        string   `json:"registrantEmail"`
		RawWhoisContent        string   `json:"rawWhoisContent"`
	} `json:"result"`
}

func QueryWhois(domain string) (*WhoisInfo, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	apiBase := config.AppConfig.WhoisAPIURL
	if apiBase == "" {
		apiBase = "https://who.zmh.me"
	}
	apiBase = strings.TrimRight(apiBase, "/")

	apiURL := fmt.Sprintf("%s/api/lookup?query=%s", apiBase, url.QueryEscape(domain))

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("WHOIS API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read WHOIS response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("WHOIS API returned HTTP %d", resp.StatusCode)
	}

	var apiResp whoisAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse WHOIS response: %w", err)
	}

	if !apiResp.Status {
		errMsg := apiResp.Error
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return nil, fmt.Errorf("WHOIS API error: %s", errMsg)
	}

	if apiResp.Result == nil {
		return nil, fmt.Errorf("WHOIS API returned empty result for %s", domain)
	}

	r := apiResp.Result
	info := &WhoisInfo{
		DomainName:        r.Domain,
		Registrar:         r.Registrar,
		RegistrarURL:      r.RegistrarURL,
		WhoisServer:       r.WhoisServer,
		RegistrantName:    r.RegistrantName,
		RegistrantOrg:     r.RegistrantOrganization,
		RegistrantEmail:   r.RegistrantEmail,
		RegistrantPhone:   r.RegistrantPhone,
		RegistrantCountry: r.RegistrantCountry,
		RawText:           r.RawWhoisContent,
	}

	if r.CreationDate != "" {
		info.CreationDate = parseISODate(r.CreationDate)
	}
	if r.UpdatedDate != "" {
		info.UpdatedDate = parseISODate(r.UpdatedDate)
	}
	if r.ExpirationDate != "" {
		info.ExpiryDate = parseISODate(r.ExpirationDate)
	}

	if len(r.NameServers) > 0 {
		var ns []string
		for _, n := range r.NameServers {
			ns = append(ns, strings.ToLower(n))
		}
		info.Nameservers = strings.Join(ns, ",")
	}

	if len(r.Status) > 0 {
		var statuses []string
		for _, s := range r.Status {
			statuses = append(statuses, s.Status)
		}
		info.Status = strings.Join(statuses, ", ")
	}

	return info, nil
}

func parseISODate(s string) *time.Time {
	s = strings.TrimSpace(s)
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}
