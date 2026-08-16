package services

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"DomainManager/models"
)

type RenewalPriceResult struct {
	Domain   string  `json:"domain"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
	Source   string  `json:"source"`
	Error    string  `json:"error,omitempty"`
}

func QueryRenewalPrice(domain string, registrar models.Registrar) (*RenewalPriceResult, error) {
	switch registrar.Type {
	case "aliyun", "aliyun_intl":
		return queryAliyunRenewalPrice(domain, registrar)
	case "tencent", "tencent_intl":
		return queryTencentRenewalPrice(domain, registrar)
	default:
		return &RenewalPriceResult{
			Domain: domain,
			Error:  fmt.Sprintf("registrar type %q does not support renewal price query", registrar.Type),
		}, nil
	}
}

func queryAliyunRenewalPrice(domain string, registrar models.Registrar) (*RenewalPriceResult, error) {
	apiKey := registrar.APIKey
	apiSecret := registrar.APISecret
	if apiKey == "" || apiSecret == "" {
		return &RenewalPriceResult{Domain: domain, Error: "aliyun API key/secret not configured"}, nil
	}

	baseURL := "https://domain.aliyuncs.com"
	if registrar.Type == "aliyun_intl" {
		baseURL = "https://domain.aliyuncs.com"
	}

	params := map[string]string{
		"Action":           "CheckDomain",
		"Version":          "2018-01-29",
		"Format":           "JSON",
		"DomainName":       domain,
		"FeeCommand":       "renew",
		"FeePeriod":        "1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   fmt.Sprintf("%d", time.Now().UnixNano()),
		"SignatureVersion": "1.0",
		"AccessKeyId":      apiKey,
	}

	signedParams := signAliyunRequest(params, apiSecret)
	query := url.Values{}
	for k, v := range signedParams {
		query.Set(k, v)
	}

	apiURL := baseURL + "/?" + query.Encode()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return &RenewalPriceResult{Domain: domain, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &RenewalPriceResult{Domain: domain, Error: err.Error()}, nil
	}

	if resp.StatusCode != 200 {
		return &RenewalPriceResult{Domain: domain, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}, nil
	}

	if len(body) == 0 || body[0] != '{' {
		return &RenewalPriceResult{Domain: domain, Error: "API returned non-JSON (domain may not support renewal)"}, nil
	}

	var apiErr struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Code != "" {
		return &RenewalPriceResult{Domain: domain, Error: fmt.Sprintf("[%s] %s", apiErr.Code, apiErr.Message)}, nil
	}

	var result struct {
		RequestId  string `json:"RequestId"`
		DomainName string `json:"DomainName"`
		Avail      int    `json:"Avail"`
		Price      int64  `json:"Price"`
		Premium    bool   `json:"Premium"`
		Reason     string `json:"Reason"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return &RenewalPriceResult{Domain: domain, Error: fmt.Sprintf("parse error: %s", string(body[:min(len(body), 200)]))}, nil
	}

	// CheckDomain returns Avail==1 when the domain is available for *registration*;
	// for renewal of an owned domain that is not meaningful, so we rely on Price.
	if result.Price == 0 {
		reason := result.Reason
		if reason == "" {
			reason = "price data not available for this domain"
		}
		return &RenewalPriceResult{Domain: domain, Error: reason}, nil
	}

	price := float64(result.Price) / 100.0
	return &RenewalPriceResult{
		Domain:   domain,
		Price:    price,
		Currency: "CNY",
		Source:   "registrar",
	}, nil
}

func queryTencentRenewalPrice(domain string, registrar models.Registrar) (*RenewalPriceResult, error) {
	secretID := registrar.APIKey
	secretKey := registrar.APISecret
	if secretID == "" || secretKey == "" {
		return &RenewalPriceResult{Domain: domain, Error: "tencent API secret_id/secret_key not configured"}, nil
	}

	host := "domain.tencentcloudapi.com"
	if registrar.Type == "tencent_intl" {
		host = "domain.intl.tencentcloudapi.com"
	}

	service := "domain"
	action := "DescribeDomainPriceList"
	version := "2018-08-08"
	timestamp := time.Now().Unix()
	payload := `{"Operation":["renew"],"Year":[1]}`

	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := "content-type:application/json\n" + "host:" + host + "\n"
	signedHeaders := "content-type;host"
	hashedRequestPayload := sha256Hex(payload)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)

	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256Hex(canonicalRequest)
	stringToSign := fmt.Sprintf("TC3-HMAC-SHA256\n%d\n%s\n%s",
		timestamp,
		credentialScope,
		hashedCanonicalRequest)

	secretDate := hmacSHA256(date, "TC3"+secretKey)
	secretService := hmacSHA256(service, secretDate)
	secretSigning := hmacSHA256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(hmacSHA256(stringToSign, secretSigning)))

	authorization := fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		secretID,
		credentialScope,
		signedHeaders,
		signature)

	apiURL := "https://" + host
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(payload))
	if err != nil {
		return &RenewalPriceResult{Domain: domain, Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("Authorization", authorization)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &RenewalPriceResult{Domain: domain, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &RenewalPriceResult{Domain: domain, Error: err.Error()}, nil
	}

	if resp.StatusCode != 200 {
		return &RenewalPriceResult{Domain: domain, Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}, nil
	}

	var apiErr struct {
		Response struct {
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Response.Error.Code != "" {
		return &RenewalPriceResult{Domain: domain, Error: fmt.Sprintf("[%s] %s", apiErr.Response.Error.Code, apiErr.Response.Error.Message)}, nil
	}

	var result struct {
		Response struct {
			RequestId string `json:"RequestId"`
			PriceList []struct {
				Tld       string  `json:"Tld"`
				Operation string  `json:"Operation"`
				Price     float64 `json:"Price"`
				RealPrice float64 `json:"RealPrice"`
				Year      int     `json:"Year"`
			} `json:"PriceList"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return &RenewalPriceResult{Domain: domain, Error: "failed to parse response"}, nil
	}

	// Find matching TLD from the domain name
	dotIdx := strings.LastIndex(domain, ".")
	domainTLD := ""
	if dotIdx != -1 {
		domainTLD = strings.TrimPrefix(domain[dotIdx:], ".")
	}

	for _, p := range result.Response.PriceList {
		if p.Operation == "renew" && p.Year == 1 {
			// Accept either "com" or ".com" form returned by the API.
			if domainTLD != "" && p.Tld != domainTLD && p.Tld != "."+domainTLD {
				continue
			}
			price := p.RealPrice
			if price == 0 {
				price = p.Price
			}
			return &RenewalPriceResult{
				Domain:   domain,
				Price:    price,
				Currency: "CNY",
				Source:   "registrar",
			}, nil
		}
	}

	return &RenewalPriceResult{Domain: domain, Error: "no renewal price found for this TLD"}, nil
}
