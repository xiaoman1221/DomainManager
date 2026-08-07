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
	case "digitalplat":
		return fetchDigitalPlatDomains(registrar)
	default:
		return nil, fmt.Errorf("registrar type %q does not support automatic domain fetching, please import domains manually", registrar.Type)
	}
}

func fetchAliyunDomains(registrar models.Registrar, baseURL string) ([]models.Domain, error) {
	apiKey := registrar.APIKey
	apiSecret := registrar.APISecret
	if apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("aliyun API key/secret not configured")
	}

	params := map[string]string{
		"Action":           "QueryDomainList",
		"Version":          "2018-01-29",
		"Format":           "JSON",
		"RegionId":         "cn-hangzhou",
		"PageNum":          "1",
		"PageSize":         "50",
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
	return doRegistrarFetch(apiURL, "aliyun", parseAliyunResponse)
}

func fetchTencentDomains(registrar models.Registrar, host string) ([]models.Domain, error) {
	secretID := registrar.APIKey
	secretKey := registrar.APISecret
	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("tencent API secret_id/secret_key not configured")
	}

	service := "domain"
	action := "DescribeDomainNameList"
	version := "2018-08-08"
	timestamp := time.Now().Unix()
	payload := `{"Offset":0,"Limit":50}`

	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")

	// step 1: build canonical request
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

	// step 2: build string to sign
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256Hex(canonicalRequest)
	stringToSign := fmt.Sprintf("TC3-HMAC-SHA256\n%d\n%s\n%s",
		timestamp,
		credentialScope,
		hashedCanonicalRequest)

	// step 3: sign
	secretDate := hmacSHA256(date, "TC3"+secretKey)
	secretService := hmacSHA256(service, secretDate)
	secretSigning := hmacSHA256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(hmacSHA256(stringToSign, secretSigning)))

	// step 4: build authorization
	authorization := fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		secretID,
		credentialScope,
		signedHeaders,
		signature)

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

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tencent API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tencent response: %w", err)
	}

	log.Printf("[tencent] HTTP %d, body length=%d", resp.StatusCode, len(body))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("tencent API returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	return parseTencentResponse(body, "tencent")
}

func fetchCloudflareDomains(registrar models.Registrar) ([]models.Domain, error) {
	token := registrar.APIKey
	if token == "" {
		return nil, fmt.Errorf("cloudflare API token not configured")
	}

	apiURL := "https://api.cloudflare.com/client/v4/zones?status=active&per_page=50"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

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
			Name      string `json:"name"`
			Status    string `json:"status"`
			CreatedOn string `json:"created_on"`
		} `json:"result"`
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

	var domains []models.Domain
	for _, z := range result.Result {
		domains = append(domains, models.Domain{
			Name:        z.Name,
			Registrar:   "Cloudflare",
			Status:      "active",
			Nameservers: "",
		})
	}
	return domains, nil
}

func fetchGoDaddyDomains(registrar models.Registrar) ([]models.Domain, error) {
	key := registrar.APIKey
	secret := registrar.APISecret
	if key == "" || secret == "" {
		return nil, fmt.Errorf("godaddy API key/secret not configured")
	}

	apiURL := "https://api.godaddy.com/v1/domains?status=ACTIVE&limit=50"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "sso-key "+key+":"+secret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("godaddy API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

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

	var result []models.Domain
	for _, d := range domains {
		dm := models.Domain{
			Name:      d.Name,
			Registrar: "GoDaddy",
			Status:    "active",
		}
		if d.Expiry != "" {
			if t, err := time.Parse("2006-01-02T15:04:05Z", d.Expiry); err == nil {
				dm.ExpiryDate = &t
			}
		}
		if len(d.NameServers) > 0 {
			dm.Nameservers = strings.Join(d.NameServers, ",")
		}
		result = append(result, dm)
	}
	return result, nil
}

func fetchNamecheapDomains(registrar models.Registrar) ([]models.Domain, error) {
	user := registrar.APIKey
	apiKey := registrar.APISecret
	username := registrar.APIExtra
	if user == "" || apiKey == "" {
		return nil, fmt.Errorf("namecheap API credentials not configured")
	}

	params := url.Values{}
	params.Set("ApiUser", user)
	params.Set("ApiKey", apiKey)
	params.Set("UserName", username)
	params.Set("Command", "namecheap.domains.getList")
	params.Set("ClientIp", "127.0.0.1")
	params.Set("Page", "1")
	params.Set("PageSize", "50")

	apiURL := "https://api.namecheap.com/xml.response?" + params.Encode()
	return doRegistrarFetch(apiURL, "namecheap", parseNamecheapResponse)
}

func doRegistrarFetch(apiURL, registrarType string, parser func([]byte, string) ([]models.Domain, error)) ([]models.Domain, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("%s API request failed: %w", registrarType, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s failed to read response: %w", registrarType, err)
	}

	log.Printf("[%s] HTTP %d, body length=%d", registrarType, resp.StatusCode, len(body))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s API returned HTTP %d: %s", registrarType, resp.StatusCode, truncate(string(body), 300))
	}

	return parser(body, registrarType)
}

func parseAliyunResponse(body []byte, _ string) ([]models.Domain, error) {
	var apiErr struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Code != "" {
		return nil, fmt.Errorf("aliyun API error [%s]: %s", apiErr.Code, apiErr.Message)
	}

	var result struct {
		Data struct {
			Domain []struct {
				DomainName       string `json:"DomainName"`
				ExpirationDate   string `json:"ExpirationDate"`
				RegistrationDate string `json:"RegistrationDate"`
				InstanceId       string `json:"InstanceId"`
				DomainStatus     string `json:"DomainStatus"`
				Ccompany         string `json:"Ccompany"`
				AutoRenewEnabled bool   `json:"AutoRenewEnabled"`
				Remark           string `json:"Remark"`
			} `json:"Domain"`
		} `json:"Data"`
		TotalItemNum int    `json:"TotalItemNum"`
		RequestId    string `json:"RequestId"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse aliyun response: %w", err)
	}

	log.Printf("[aliyun] parsed %d/%d domains, requestId=%s", len(result.Data.Domain), result.TotalItemNum, result.RequestId)

	var domains []models.Domain
	for _, d := range result.Data.Domain {
		dm := models.Domain{
			Name:      d.DomainName,
			Registrar: "阿里云（万网）",
			Status:    "active",
			Note:      d.Remark,
		}
		if d.ExpirationDate != "" {
			formats := []string{"2006-01-02 15:04:05", "2006-01-02"}
			for _, f := range formats {
				if t, err := time.Parse(f, d.ExpirationDate); err == nil {
					dm.ExpiryDate = &t
					break
				}
			}
		}
		if d.RegistrationDate != "" {
			formats := []string{"2006-01-02 15:04:05", "2006-01-02"}
			for _, f := range formats {
				if t, err := time.Parse(f, d.RegistrationDate); err == nil {
					dm.RegistrationDate = &t
					break
				}
			}
		}
		dm.AutoRenew = d.AutoRenewEnabled
		domains = append(domains, dm)
	}
	return domains, nil
}

func parseTencentResponse(body []byte, _ string) ([]models.Domain, error) {
	var apiErr struct {
		Response struct {
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Response.Error.Code != "" {
		return nil, fmt.Errorf("tencent API error [%s]: %s", apiErr.Response.Error.Code, apiErr.Response.Error.Message)
	}

	var result struct {
		Response struct {
			RequestID  string `json:"RequestId"`
			TotalCount int    `json:"TotalCount"`
			DomainSet  []struct {
				DomainName     string `json:"DomainName"`
				ExpirationDate string `json:"ExpirationDate"`
				CreationDate   string `json:"CreationDate"`
				DomainId       string `json:"DomainId"`
				AutoRenew      int    `json:"AutoRenew"`
				BuyStatus      string `json:"BuyStatus"`
				CodeTld        string `json:"CodeTld"`
				Tld            string `json:"Tld"`
				IsPremium      bool   `json:"IsPremium"`
			} `json:"DomainSet"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse tencent response: %w", err)
	}

	log.Printf("[tencent] parsed %d/%d domains, requestId=%s", len(result.Response.DomainSet), result.Response.TotalCount, result.Response.RequestID)

	var domains []models.Domain
	for _, d := range result.Response.DomainSet {
		dm := models.Domain{
			Name:      d.DomainName,
			Registrar: "腾讯云",
			Status:    "active",
			AutoRenew: d.AutoRenew == 1,
		}
		if d.ExpirationDate != "" {
			formats := []string{"2006-01-02 15:04:05", "2006-01-02"}
			for _, f := range formats {
				if t, err := time.Parse(f, d.ExpirationDate); err == nil {
					dm.ExpiryDate = &t
					break
				}
			}
		}
		if d.CreationDate != "" {
			formats := []string{"2006-01-02 15:04:05", "2006-01-02"}
			for _, f := range formats {
				if t, err := time.Parse(f, d.CreationDate); err == nil {
					dm.RegistrationDate = &t
					break
				}
			}
		}
		domains = append(domains, dm)
	}
	return domains, nil
}

func parseNamecheapResponse(body []byte, _ string) ([]models.Domain, error) {
	bodyStr := string(body)

	if strings.Contains(bodyStr, "Status=\"ERROR\"") || strings.Contains(bodyStr, "<Error>") {
		msgStart := strings.Index(bodyStr, "<Error>")
		if msgStart != -1 {
			msgEnd := strings.Index(bodyStr[msgStart:], "</Error>")
			if msgEnd != -1 {
				return nil, fmt.Errorf("namecheap API error: %s", bodyStr[msgStart+7:msgStart+msgEnd])
			}
		}
		return nil, fmt.Errorf("namecheap API returned an error")
	}

	if !strings.Contains(bodyStr, "DomainGetListResult") {
		return nil, fmt.Errorf("unexpected namecheap response: missing DomainGetListResult")
	}

	var domains []models.Domain
	start := strings.Index(bodyStr, "Name=\"")
	for start != -1 {
		start += 6
		end := strings.Index(bodyStr[start:], "\"")
		if end == -1 {
			break
		}
		name := bodyStr[start : start+end]
		if strings.Contains(name, ".") {
			domains = append(domains, models.Domain{
				Name:      name,
				Registrar: "Namecheap",
				Status:    "active",
			})
		}
		next := strings.Index(bodyStr[start+end:], "Name=\"")
		if next == -1 {
			break
		}
		start = start + end + next
	}

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
	return xml[startIdx : startIdx+endIdx]
}
