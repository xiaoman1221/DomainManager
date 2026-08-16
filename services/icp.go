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

type ICPInfo struct {
	Domain       string     `json:"domain"`
	ICPNumber    string     `json:"icp_number"`
	OwnerName    string     `json:"owner_name"`
	OwnerType    string     `json:"owner_type"`
	VerifyStatus string     `json:"verify_status"`
	FilingDate   *time.Time `json:"filing_date"`
	ServiceName  string     `json:"service_name"`
	ServiceURL   string     `json:"service_url"`
}

type icpAPIResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Params struct {
		Total int `json:"total"`
		List  []struct {
			Domain           string `json:"domain"`
			DomainID         int    `json:"domainId"`
			MainLicence      string `json:"mainLicence"`
			NatureName       string `json:"natureName"`
			ServiceLicence   string `json:"serviceLicence"`
			UnitName         string `json:"unitName"`
			UpdateRecordTime string `json:"updateRecordTime"`
			ContentTypeName  string `json:"contentTypeName"`
			LimitAccess      string `json:"limitAccess"`
			ServiceName      string `json:"serviceName"`
			LeaderName       string `json:"leaderName"`
		} `json:"list"`
	} `json:"params"`
	Success bool `json:"success"`
}

func QueryICP(domain string) (*ICPInfo, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	apiBase := config.AppConfig.ICPAPIURL
	if apiBase == "" {
		apiBase = "http://127.0.0.1:16181"
	}
	apiBase = strings.TrimRight(apiBase, "/")

	apiURL := fmt.Sprintf("%s/query/web?search=%s", apiBase, url.QueryEscape(domain))

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("ICP API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ICP response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ICP API returned HTTP %d", resp.StatusCode)
	}

	var apiResp icpAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse ICP response: %w", err)
	}

	if !apiResp.Success || apiResp.Code != 200 {
		return nil, fmt.Errorf("ICP API error: %s", apiResp.Msg)
	}

	if apiResp.Params.Total == 0 || len(apiResp.Params.List) == 0 {
		return &ICPInfo{
			Domain:       domain,
			VerifyStatus: "未备案",
		}, nil
	}

	d := apiResp.Params.List[0]
	info := &ICPInfo{
		Domain:       d.Domain,
		ICPNumber:    d.MainLicence,
		OwnerName:    d.UnitName,
		OwnerType:    d.NatureName,
		VerifyStatus: "已备案",
		ServiceName:  d.ServiceName,
	}

	if d.UpdateRecordTime != "" {
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, f := range formats {
			if t, err := time.Parse(f, d.UpdateRecordTime); err == nil {
				info.FilingDate = &t
				break
			}
		}
	}

	return info, nil
}
