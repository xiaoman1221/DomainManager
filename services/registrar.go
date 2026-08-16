package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"DomainManager/models"
)

func FetchDomainsFromRegistrar(registrar models.Registrar) ([]models.Domain, error) {
	switch registrar.Type {
	case "aliyun":
		return fetchAliyunDomains(registrar, "https://domain.aliyuncs.com")
	case "aliyun_intl":
		return fetchAliyunDomains(registrar, "https://domain.aliyuncs.com")
	case "tencent":
		return fetchTencentDomains(registrar, "domain.tencentcloudapi.com")
	case "tencent_intl":
		return fetchTencentDomains(registrar, "domain.intl.tencentcloudapi.com")
	case "huawei":
		return fetchHuaweiDomains(registrar)
	case "cloudflare":
		return fetchCloudflareDomains(registrar)
	case "godaddy":
		return fetchGoDaddyDomains(registrar)
	case "namecheap":
		return fetchNamecheapDomains(registrar)
	case "dynadot":
		return fetchDynadotDomains(registrar)
	case "namesilo":
		return fetchNameSiloDomains(registrar)
	case "porkbun":
		return fetchPorkbunDomains(registrar)
	case "amazon":
		return fetchRoute53Domains(registrar)
	case "digitalplat":
		return fetchDigitalPlatDomains(registrar)
	default:
		return nil, fmt.Errorf("registrar type %q does not support automatic domain fetching, please import domains manually", registrar.Type)
	}
}

// fetchAliyunDomains lists all domains by paging through QueryDomainList
// (the API caps PageSize at 100; without paging only the first 50 would be
// returned).
func fetchAliyunDomains(registrar models.Registrar, baseURL string) ([]models.Domain, error) {
	apiKey := registrar.APIKey
	apiSecret := registrar.APISecret
	if apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("aliyun API key/secret not configured")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	const pageSize = 100
	var all []models.Domain
	total := 0
	pageNum := 1
	for {
		params := map[string]string{
			"Action":           "QueryDomainList",
			"Version":          "2018-01-29",
			"Format":           "JSON",
			"RegionId":         "cn-hangzhou",
			"PageNum":          strconv.Itoa(pageNum),
			"PageSize":         strconv.Itoa(pageSize),
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

		resp, err := client.Get(apiURL)
		if err != nil {
			return nil, fmt.Errorf("aliyun API request failed: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("aliyun failed to read response: %w", readErr)
		}
		log.Printf("[aliyun] page %d: HTTP %d, body length=%d", pageNum, resp.StatusCode, len(body))
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("aliyun API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
		}

		domains, pageTotal, err := parseAliyunPage(body)
		if err != nil {
			return nil, err
		}
		all = append(all, domains...)
		total = pageTotal
		if len(domains) == 0 || len(all) >= total {
			break
		}
		pageNum++
		if pageNum > 100 {
			break
		}
	}

	log.Printf("[aliyun] parsed %d/%d domains", len(all), total)
	return all, nil
}

func parseAliyunPage(body []byte) ([]models.Domain, int, error) {
	var apiErr struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Code != "" {
		return nil, 0, fmt.Errorf("aliyun API error [%s]: %s", apiErr.Code, apiErr.Message)
	}

	var result struct {
		Data struct {
			Domain []struct {
				DomainName       string `json:"DomainName"`
				ExpirationDate   string `json:"ExpirationDate"`
				RegistrationDate string `json:"RegistrationDate"`
				DomainStatus     string `json:"DomainStatus"`
				AutoRenewEnabled bool   `json:"AutoRenewEnabled"`
				Remark           string `json:"Remark"`
			} `json:"Domain"`
		} `json:"Data"`
		TotalItemNum int `json:"TotalItemNum"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("failed to parse aliyun response: %w", err)
	}

	var domains []models.Domain
	for _, d := range result.Data.Domain {
		dm := models.Domain{
			Name:      d.DomainName,
			Registrar: "阿里云（万网）",
			Status:    "active",
			Note:      d.Remark,
			AutoRenew: d.AutoRenewEnabled,
		}
		if t := parseISODate(d.ExpirationDate); t != nil {
			dm.ExpiryDate = t
		}
		if t := parseISODate(d.RegistrationDate); t != nil {
			dm.RegistrationDate = t
		}
		domains = append(domains, dm)
	}
	return domains, result.TotalItemNum, nil
}

// fetchTencentDomains lists all domains by paging through
// DescribeDomainNameList (Limit max is 100; without paging only the first 50
// would be returned).
func fetchTencentDomains(registrar models.Registrar, host string) ([]models.Domain, error) {
	secretID := registrar.APIKey
	secretKey := registrar.APISecret
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("tencent API secret_id/secret_key not configured")
	}

	service := "domain"
	action := "DescribeDomainNameList"
	version := "2018-08-08"

	client := &http.Client{Timeout: 15 * time.Second}
	const limit = 100
	var all []models.Domain
	total := 0
	offset := 0
	for {
		payload := fmt.Sprintf(`{"Offset":%d,"Limit":%d}`, offset, limit)
		timestamp := time.Now().Unix()
		date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

		// TC3-HMAC-SHA256 signing
		httpRequestMethod := "POST"
		canonicalURI := "/"
		canonicalQueryString := ""
		canonicalHeaders := "content-type:application/json\n" + "host:" + host + "\n"
		signedHeaders := "content-type;host"
		hashedRequestPayload := sha256Hex(payload)
		canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
			httpRequestMethod, canonicalURI, canonicalQueryString,
			canonicalHeaders, signedHeaders, hashedRequestPayload)

		credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
		hashedCanonicalRequest := sha256Hex(canonicalRequest)
		stringToSign := fmt.Sprintf("TC3-HMAC-SHA256\n%d\n%s\n%s",
			timestamp, credentialScope, hashedCanonicalRequest)

		secretDate := hmacSHA256(date, "TC3"+secretKey)
		secretService := hmacSHA256(service, secretDate)
		secretSigning := hmacSHA256("tc3_request", secretService)
		signature := hex.EncodeToString([]byte(hmacSHA256(stringToSign, secretSigning)))
		authorization := fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
			secretID, credentialScope, signedHeaders, signature)

		apiURL := "https://" + host
		req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Host", host)
		req.Header.Set("X-TC-Action", action)
		req.Header.Set("X-TC-Version", version)
		req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
		req.Header.Set("Authorization", authorization)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("tencent API request failed: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read tencent response: %w", readErr)
		}
		log.Printf("[tencent] offset=%d HTTP %d, body length=%d", offset, resp.StatusCode, len(body))
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("tencent API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
		}

		domains, pageTotal, err := parseTencentPage(body)
		if err != nil {
			return nil, err
		}
		all = append(all, domains...)
		total = pageTotal
		offset += limit
		if len(domains) == 0 || offset >= total {
			break
		}
	}

	log.Printf("[tencent] parsed %d/%d domains", len(all), total)
	return all, nil
}

func parseTencentPage(body []byte) ([]models.Domain, int, error) {
	var apiErr struct {
		Response struct {
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Response.Error.Code != "" {
		return nil, 0, fmt.Errorf("tencent API error [%s]: %s", apiErr.Response.Error.Code, apiErr.Response.Error.Message)
	}

	var result struct {
		Response struct {
			RequestID  string `json:"RequestId"`
			TotalCount int    `json:"TotalCount"`
			DomainSet  []struct {
				DomainName     string `json:"DomainName"`
				ExpirationDate string `json:"ExpirationDate"`
				CreationDate   string `json:"CreationDate"`
				AutoRenew      int    `json:"AutoRenew"`
			} `json:"DomainSet"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("failed to parse tencent response: %w", err)
	}

	var domains []models.Domain
	for _, d := range result.Response.DomainSet {
		dm := models.Domain{
			Name:      d.DomainName,
			Registrar: "腾讯云",
			Status:    "active",
			AutoRenew: d.AutoRenew == 1,
		}
		if t := parseISODate(d.ExpirationDate); t != nil {
			dm.ExpiryDate = t
		}
		if t := parseISODate(d.CreationDate); t != nil {
			dm.RegistrationDate = t
		}
		domains = append(domains, dm)
	}
	return domains, result.Response.TotalCount, nil
}

// fetchCloudflareDomains pages through the zones API (per_page max 100);
// only the first page was fetched before.
func fetchCloudflareDomains(registrar models.Registrar) ([]models.Domain, error) {
	token := registrar.APIKey
	if token == "" {
		return nil, fmt.Errorf("cloudflare API token not configured")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	var all []models.Domain
	page := 1
	for {
		apiURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?status=active&per_page=100&page=%d", page)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("cloudflare API request failed: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		log.Printf("[cloudflare] page %d: HTTP %d, body length=%d", page, resp.StatusCode, len(body))
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("cloudflare API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
		}

		var result struct {
			Success bool `json:"success"`
			Errors  []struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"errors"`
			Result []struct {
				Name string `json:"name"`
			} `json:"result"`
			ResultInfo struct {
				Page       int `json:"page"`
				TotalPages int `json:"total_pages"`
			} `json:"result_info"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse cloudflare response: %w", err)
		}
		if !result.Success {
			msg := "unknown error"
			if len(result.Errors) > 0 {
				msg = result.Errors[0].Message
			}
			return nil, fmt.Errorf("cloudflare API error: %s", msg)
		}

		for _, z := range result.Result {
			all = append(all, models.Domain{Name: z.Name, Registrar: "Cloudflare", Status: "active"})
		}
		if len(result.Result) == 0 || page >= result.ResultInfo.TotalPages {
			break
		}
		page++
	}

	log.Printf("[cloudflare] parsed %d zones", len(all))
	return all, nil
}

// fetchGoDaddyDomains pages through the v1 domains API using the
// X-Pagination-Total header (only the first 50 were fetched before).
func fetchGoDaddyDomains(registrar models.Registrar) ([]models.Domain, error) {
	key := registrar.APIKey
	secret := registrar.APISecret
	if key == "" || secret == "" {
		return nil, fmt.Errorf("godaddy API key/secret not configured")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	const limit = 100
	var all []models.Domain
	offset := 0
	for {
		apiURL := fmt.Sprintf("https://api.godaddy.com/v1/domains?status=ACTIVE&limit=%d&offset=%d", limit, offset)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "sso-key "+key+":"+secret)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("godaddy API request failed: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		totalHeader := resp.Header.Get("X-Pagination-Total")
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		log.Printf("[godaddy] offset=%d HTTP %d, body length=%d", offset, resp.StatusCode, len(body))
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("godaddy API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
		}

		var domains []struct {
			Name        string   `json:"name"`
			Status      string   `json:"status"`
			Expiry      string   `json:"expiration"`
			NameServers []string `json:"nameServers"`
		}
		if err := json.Unmarshal(body, &domains); err != nil {
			return nil, fmt.Errorf("failed to parse godaddy response: %w", err)
		}

		for _, d := range domains {
			dm := models.Domain{Name: d.Name, Registrar: "GoDaddy", Status: "active"}
			if t := parseISODate(d.Expiry); t != nil {
				dm.ExpiryDate = t
			}
			if len(d.NameServers) > 0 {
				dm.Nameservers = strings.Join(d.NameServers, ",")
			}
			all = append(all, dm)
		}

		total, _ := strconv.Atoi(totalHeader)
		offset += len(domains)
		if len(domains) == 0 || (total > 0 && offset >= total) {
			break
		}
	}

	log.Printf("[godaddy] parsed %d domains", len(all))
	return all, nil
}

// fetchNamecheapDomains pages through namecheap.domains.getList
// (only the first 50 were fetched before).
func fetchNamecheapDomains(registrar models.Registrar) ([]models.Domain, error) {
	user := registrar.APIKey
	apiKey := registrar.APISecret
	username := registrar.APIExtra
	if user == "" || apiKey == "" {
		return nil, fmt.Errorf("namecheap API credentials not configured")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	const pageSize = 100
	var all []models.Domain
	total := 0
	page := 1
	for {
		params := url.Values{}
		params.Set("ApiUser", user)
		params.Set("ApiKey", apiKey)
		params.Set("UserName", username)
		params.Set("Command", "namecheap.domains.getList")
		params.Set("ClientIp", "127.0.0.1")
		params.Set("Page", strconv.Itoa(page))
		params.Set("PageSize", strconv.Itoa(pageSize))

		apiURL := "https://api.namecheap.com/xml.response?" + params.Encode()
		resp, err := client.Get(apiURL)
		if err != nil {
			return nil, fmt.Errorf("namecheap API request failed: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("namecheap failed to read response: %w", readErr)
		}
		log.Printf("[namecheap] page %d: HTTP %d, body length=%d", page, resp.StatusCode, len(body))
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("namecheap API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
		}

		domains, pageTotal, err := parseNamecheapPage(body)
		if err != nil {
			return nil, err
		}
		all = append(all, domains...)
		total = pageTotal
		if len(domains) == 0 || len(all) >= total {
			break
		}
		page++
	}

	log.Printf("[namecheap] parsed %d/%d domains", len(all), total)
	return all, nil
}

func parseNamecheapPage(body []byte) ([]models.Domain, int, error) {
	bodyStr := string(body)

	if strings.Contains(bodyStr, "Status=\"ERROR\"") || strings.Contains(bodyStr, "<Error>") {
		msgStart := strings.Index(bodyStr, "<Error>")
		if msgStart != -1 {
			msgEnd := strings.Index(bodyStr[msgStart:], "</Error>")
			if msgEnd != -1 {
				return nil, 0, fmt.Errorf("namecheap API error: %s", bodyStr[msgStart+7:msgStart+msgEnd])
			}
		}
		return nil, 0, fmt.Errorf("namecheap API returned an error")
	}

	// Only look inside the domain result block to avoid matching unrelated
	// attributes such as UserName.
	block := extractXMLBlock(bodyStr, "DomainGetListResult")
	if block == "" {
		return nil, 0, fmt.Errorf("unexpected namecheap response: missing DomainGetListResult")
	}

	total := 0
	if tagStart := strings.Index(bodyStr, "<DomainGetListResult"); tagStart != -1 {
		if tagEnd := strings.Index(bodyStr[tagStart:], ">"); tagEnd != -1 {
			openTag := bodyStr[tagStart : tagStart+tagEnd]
			if v := extractXMLAttr(openTag, "TotalDomains"); v != "" {
				if n, err := strconv.Atoi(v); err == nil {
					total = n
				}
			}
		}
	}

	var domains []models.Domain
	pos := 0
	for {
		start := strings.Index(block[pos:], "<Domain")
		if start == -1 {
			break
		}
		start += pos
		end := strings.Index(block[start:], ">")
		if end == -1 {
			break
		}
		tag := block[start : start+end]
		pos = start + end

		name := extractXMLAttr(tag, "Name")
		if name == "" || !strings.Contains(name, ".") {
			continue
		}
		domains = append(domains, models.Domain{
			Name:      name,
			Registrar: "Namecheap",
			Status:    "active",
		})
	}

	return domains, total, nil
}

// fetchPorkbunDomains uses the Porkbun JSON API
// (POST https://api.porkbun.com/api/json/v3/domain/listAll).
// APIKey = apikey, APISecret = secretapikey.
func fetchPorkbunDomains(registrar models.Registrar) ([]models.Domain, error) {
	apiKey := registrar.APIKey
	secretKey := registrar.APISecret
	if apiKey == "" || secretKey == "" {
		return nil, fmt.Errorf("porkbun API key/secret not configured")
	}

	jsonData, err := json.Marshal(map[string]string{"apikey": apiKey, "secretapikey": secretKey})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal porkbun request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.porkbun.com/api/json/v3/domain/listAll", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("porkbun API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("porkbun failed to read response: %w", err)
	}
	log.Printf("[porkbun] HTTP %d, body length=%d", resp.StatusCode, len(body))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("porkbun API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Domains []struct {
			Domain     string `json:"domain"`
			Status     string `json:"status"`
			CreateDate string `json:"createDate"`
			ExpireDate string `json:"expireDate"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse porkbun response: %w", err)
	}
	if result.Status != "SUCCESS" {
		return nil, fmt.Errorf("porkbun API error: %s", result.Message)
	}

	var domains []models.Domain
	for _, d := range result.Domains {
		dm := models.Domain{Name: d.Domain, Registrar: "Porkbun", Status: "active"}
		if st := strings.ToLower(d.Status); st != "" && st != "active" {
			dm.Status = st
		}
		if t := parseISODate(d.ExpireDate); t != nil {
			dm.ExpiryDate = t
		}
		if t := parseISODate(d.CreateDate); t != nil {
			dm.RegistrationDate = t
		}
		domains = append(domains, dm)
	}
	log.Printf("[porkbun] parsed %d domains", len(domains))
	return domains, nil
}

// fetchHuaweiDomains lists Huawei Cloud domains via
// GET /v2/domains (SDK-HMAC-SHA256 signing).
// APIKey = AccessKey, APISecret = SecretKey, APIEndpoint overrides the host.
func fetchHuaweiDomains(registrar models.Registrar) ([]models.Domain, error) {
	ak := registrar.APIKey
	sk := registrar.APISecret
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("huawei AccessKey/SecretKey not configured")
	}

	base := strings.TrimRight(strings.TrimSpace(registrar.APIEndpoint), "/")
	if base == "" {
		base = "https://domain.myhuaweicloud.com"
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid huawei API endpoint: %w", err)
	}
	host := u.Host

	client := &http.Client{Timeout: 15 * time.Second}
	const limit = 100
	var all []models.Domain
	total := 0
	offset := 0
	for {
		canonicalQuery := fmt.Sprintf("limit=%d&offset=%d", limit, offset)
		xSdkDate := time.Now().UTC().Format("20060102T150405Z")
		canonicalURI := "/v2/domains"

		canonicalHeaders := "host:" + host + "\n" + "x-sdk-date:" + xSdkDate + "\n"
		signedHeaders := "host;x-sdk-date"
		payloadHash := sha256Hex("")
		canonicalRequest := fmt.Sprintf("GET\n%s\n%s\n%s\n%s\n%s",
			canonicalURI, canonicalQuery, canonicalHeaders, signedHeaders, payloadHash)
		stringToSign := "SDK-HMAC-SHA256\n" + xSdkDate + "\n" + sha256Hex(canonicalRequest)
		signature := hex.EncodeToString([]byte(hmacSHA256(stringToSign, sk)))
		authorization := fmt.Sprintf("SDK-HMAC-SHA256 Access=%s, SignedHeaders=%s, Signature=%s", ak, signedHeaders, signature)

		apiURL := base + canonicalURI + "?" + canonicalQuery
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Host", host)
		req.Header.Set("X-Sdk-Date", xSdkDate)
		req.Header.Set("Authorization", authorization)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("huawei API request failed: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("huawei failed to read response: %w", readErr)
		}
		log.Printf("[huawei] offset=%d HTTP %d, body length=%d", offset, resp.StatusCode, len(body))
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("huawei API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
		}

		domains, pageTotal, err := parseHuaweiPage(body)
		if err != nil {
			return nil, err
		}
		all = append(all, domains...)
		total = pageTotal
		offset += limit
		if len(domains) == 0 || offset >= total {
			break
		}
	}

	log.Printf("[huawei] parsed %d/%d domains", len(all), total)
	return all, nil
}

func parseHuaweiPage(body []byte) ([]models.Domain, int, error) {
	var apiErr struct {
		ErrorCode string `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.ErrorCode != "" {
		return nil, 0, fmt.Errorf("huawei API error [%s]: %s", apiErr.ErrorCode, apiErr.ErrorMsg)
	}

	var result struct {
		TotalCount int `json:"total_count"`
		Domains    []struct {
			DomainName     string `json:"domain_name"`
			DomainStatus   string `json:"domain_status"`
			ExpirationDate string `json:"expiration_date"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("failed to parse huawei response: %w", err)
	}

	var domains []models.Domain
	for _, d := range result.Domains {
		dm := models.Domain{Name: d.DomainName, Registrar: "华为云", Status: "active"}
		if st := strings.ToLower(d.DomainStatus); st != "" && st != "active" {
			dm.Status = st
		}
		if t := parseISODate(d.ExpirationDate); t != nil {
			dm.ExpiryDate = t
		}
		domains = append(domains, dm)
	}
	return domains, result.TotalCount, nil
}

// fetchRoute53Domains lists AWS Route53 domains via
// GET https://route53domains.us-east-1.amazonaws.com/2014-05-21/domains
// (SigV4 signing). APIKey = Access Key ID, APISecret = Secret Access Key.
func fetchRoute53Domains(registrar models.Registrar) ([]models.Domain, error) {
	accessKey := registrar.APIKey
	secretKey := registrar.APISecret
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("aws access key/secret not configured")
	}

	region := "us-east-1"
	service := "route53domains"
	host := "route53domains." + region + ".amazonaws.com"
	path := "/2014-05-21/domains"

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	payloadHash := sha256Hex("")
	canonicalQuery := ""
	canonicalHeaders := "host:" + host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + amzDate + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := "GET\n" + path + "\n" + canonicalQuery + "\n" +
		canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHash

	credentialScope := dateStamp + "/" + region + "/" + service + "/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + credentialScope + "\n" + sha256Hex(canonicalRequest)

	kDate := hmacSHA256(dateStamp, "AWS4"+secretKey)
	kRegion := hmacSHA256(region, kDate)
	kService := hmacSHA256(service, kRegion)
	kSigning := hmacSHA256("aws4_request", kService)
	signature := hex.EncodeToString([]byte(hmacSHA256(stringToSign, kSigning)))
	authorization := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature)

	apiURL := "https://" + host + path
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Host", host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("Authorization", authorization)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("route53 API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("route53 failed to read response: %w", err)
	}
	log.Printf("[route53] HTTP %d, body length=%d", resp.StatusCode, len(body))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("route53 API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var result struct {
		Domains []struct {
			DomainName string  `json:"DomainName"`
			AutoRenew  bool    `json:"AutoRenew"`
			Expiry     float64 `json:"Expiry"`
		} `json:"Domains"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse route53 response: %w", err)
	}

	var domains []models.Domain
	for _, d := range result.Domains {
		dm := models.Domain{Name: d.DomainName, Registrar: "AWS Route53", Status: "active", AutoRenew: d.AutoRenew}
		if d.Expiry > 0 {
			t := time.Unix(int64(d.Expiry), 0)
			dm.ExpiryDate = &t
		}
		domains = append(domains, dm)
	}
	log.Printf("[route53] parsed %d domains", len(domains))
	return domains, nil
}

func signAliyunRequest(params map[string]string, secret string) map[string]string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var canonicalParts []string
	for _, k := range keys {
		canonicalParts = append(canonicalParts, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalQueryString := strings.Join(canonicalParts, "&")

	stringToSign := "GET&%2F&" + percentEncode(canonicalQueryString)

	mac := hmac.New(sha1.New, []byte(secret+"&"))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	params["Signature"] = signature
	return params
}

func sha256Hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func hmacSHA256(s, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(s))
	return string(h.Sum(nil))
}

func percentEncode(s string) string {
	s = url.QueryEscape(s)
	s = strings.ReplaceAll(s, "+", "%20")
	s = strings.ReplaceAll(s, "*", "%2A")
	return s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func fetchDynadotDomains(registrar models.Registrar) ([]models.Domain, error) {
	apiKey := strings.TrimSpace(registrar.APIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("dynadot API key not configured")
	}

	// Dynadot listDomains uses Legacy XML API v3 — no HMAC signing needed
	apiURL := "https://api.dynadot.com/api3.xml?key=" + url.QueryEscape(apiKey) + "&command=list_domain"

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("dynadot API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dynadot failed to read response: %w", err)
	}

	log.Printf("[dynadot] HTTP %d, body length=%d", resp.StatusCode, len(body))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("dynadot API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 500))
	}

	bodyStr := string(body)

	// Check for API error in XML response
	if strings.Contains(bodyStr, "<SuccessCode>-1</SuccessCode>") {
		errMsg := extractXMLValue(bodyStr, "Error")
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return nil, fmt.Errorf("dynadot API error: %s", errMsg)
	}

	// Parse XML: ListDomainInfoResponse > ListDomainInfoContent > DomainInfoList > DomainInfo > Domain[]
	domainBlock := extractXMLBlock(bodyStr, "DomainInfoList")
	if domainBlock == "" {
		log.Printf("[dynadot] no domains found (empty DomainInfoList)")
		return nil, nil
	}

	var domains []models.Domain
	// Each domain is wrapped in <Domain>...</Domain>
	for {
		startIdx := strings.Index(domainBlock, "<Domain>")
		if startIdx == -1 {
			break
		}
		endIdx := strings.Index(domainBlock[startIdx:], "</Domain>")
		if endIdx == -1 {
			break
		}
		dBlock := domainBlock[startIdx : startIdx+endIdx+len("</Domain>")]
		domainBlock = domainBlock[startIdx+endIdx:]

		name := extractXMLValue(dBlock, "Name")
		if name == "" {
			continue
		}

		domain := models.Domain{Name: name, Status: "active"}

		// Expiration date (Unix timestamp in milliseconds)
		if expStr := extractXMLValue(dBlock, "Expiration"); expStr != "" {
			if ts, err := strconv.ParseInt(expStr, 10, 64); err == nil {
				t := time.UnixMilli(ts)
				domain.ExpiryDate = &t
			}
		}

		// Nameservers
		var nsList []string
		for i := 0; i < 10; i++ {
			ns := extractXMLValue(dBlock, fmt.Sprintf("Ns%d", i))
			if ns == "" {
				break
			}
			nsList = append(nsList, ns)
		}
		if len(nsList) > 0 {
			domain.Nameservers = strings.Join(nsList, ",")
		}

		domains = append(domains, domain)
	}

	log.Printf("[dynadot] parsed %d domains", len(domains))
	return domains, nil
}

func fetchNameSiloDomains(registrar models.Registrar) ([]models.Domain, error) {
	apiKey := registrar.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("namesilo API key not configured")
	}

	apiURL := fmt.Sprintf("https://api.namesilo.com/api/listDomains?version=1&type=xml&key=%s", apiKey)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("namesilo API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("namesilo failed to read response: %w", err)
	}

	log.Printf("[namesilo] HTTP %d, body length=%d", resp.StatusCode, len(body))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("namesilo API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	bodyStr := string(body)

	// NameSilo returns XML, parse it simply
	if strings.Contains(bodyStr, "<code>300</code>") {
		return nil, fmt.Errorf("namesilo API authentication failed")
	}

	var domains []models.Domain
	// Simple XML parsing for NameSilo domain list
	// Each domain block is between <domain> and </domain>
	for {
		startIdx := strings.Index(bodyStr, "<domain>")
		if startIdx == -1 {
			break
		}
		endIdx := strings.Index(bodyStr[startIdx:], "</domain>")
		if endIdx == -1 {
			break
		}
		block := bodyStr[startIdx : startIdx+endIdx+len("</domain>")]
		bodyStr = bodyStr[startIdx+endIdx:]

		name := extractXMLValue(block, "domain")
		if name == "" {
			continue
		}

		domain := models.Domain{Name: name, Status: "active"}

		if expiry := extractXMLValue(block, "expires"); expiry != "" {
			// NameSilo format: "2027-01-15 10:30:00"
			if t, err := time.Parse("2006-01-02 15:04:05", expiry); err == nil {
				domain.ExpiryDate = &t
			}
		}
		if created := extractXMLValue(block, "created"); created != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", created); err == nil {
				domain.CreationDate = &t
			}
		}
		if ns := extractXMLValue(block, "nameservers"); ns != "" {
			domain.Nameservers = strings.ReplaceAll(ns, ";", ",")
		}

		domains = append(domains, domain)
	}

	log.Printf("[namesilo] parsed %d domains", len(domains))
	return domains, nil
}

func extractXMLValue(xml, tag string) string {
	startTag := "<" + tag + ">"
	endTag := "</" + tag + ">"
	startIdx := strings.Index(xml, startTag)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(startTag)
	endIdx := strings.Index(xml[startIdx:], endTag)
	if endIdx == -1 {
		return ""
	}
	return strings.TrimSpace(xml[startIdx : startIdx+endIdx])
}

func extractXMLBlock(xml, tag string) string {
	startTag := "<" + tag
	endTag := "</" + tag + ">"

	// Find <tag followed by a boundary (> or whitespace) so tags that carry
	// attributes (e.g. <DomainGetListResult Domain="..." ...>) also match.
	startIdx := strings.Index(xml, startTag)
	for startIdx != -1 {
		after := startIdx + len(startTag)
		if after >= len(xml) || xml[after] == '>' || xml[after] == ' ' ||
			xml[after] == '\t' || xml[after] == '\n' || xml[after] == '\r' {
			break
		}
		next := strings.Index(xml[after:], startTag)
		if next == -1 {
			startIdx = -1
			break
		}
		startIdx = after + next
	}
	if startIdx == -1 {
		return ""
	}

	openEnd := strings.Index(xml[startIdx:], ">")
	if openEnd == -1 {
		return ""
	}
	contentStart := startIdx + openEnd + 1

	endIdx := strings.Index(xml[contentStart:], endTag)
	if endIdx == -1 {
		return ""
	}
	return xml[contentStart : contentStart+endIdx]
}

// extractXMLAttr returns the value of a quoted attribute (name="value") inside an
// XML open tag, or an empty string when absent.
func extractXMLAttr(tag, attr string) string {
	marker := attr + "="
	idx := strings.Index(tag, marker)
	if idx == -1 {
		return ""
	}
	// skip the opening quote, then find the closing quote
	start := idx + len(marker) + 1
	end := strings.Index(tag[start:], "\"")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(tag[start : start+end])
}
