package services

import (
	"DomainManager/database"
	"DomainManager/models"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CertimateService struct {
	BaseURL  string
	Username string
	Password string
	Token    string
}

type CertimateCertListResponse struct {
	Page       int                   `json:"page"`
	PerPage    int                   `json:"perPage"`
	TotalItems int                   `json:"totalItems"`
	Items      []CertimateCertRecord `json:"items"`
}

type CertimateCertRecord struct {
	ID                 string `json:"id"`
	Source             string `json:"source"`
	SubjectAltNames    string `json:"subjectAltNames"`
	SerialNumber       string `json:"serialNumber"`
	Certificate        string `json:"certificate"`
	PrivateKey         string `json:"privateKey"`
	IssuerOrg          string `json:"issuerOrg"`
	IssuerCertificate  string `json:"issuerCertificate"`
	KeyAlgorithm       string `json:"keyAlgorithm"`
	SignatureAlgorithm string `json:"signatureAlgorithm"`
	ValidityNotBefore  string `json:"validityNotBefore"`
	ValidityNotAfter   string `json:"validityNotAfter"`
	Created            string `json:"created"`
	Updated            string `json:"updated"`
}

func NewCertimateService(config models.CertimateConfig) *CertimateService {
	return &CertimateService{
		BaseURL:  config.URL,
		Username: config.Username,
		Password: config.Password,
		Token:    config.Token,
	}
}

// certimateAuthResponse is the PocketBase superuser auth response.
type certimateAuthResponse struct {
	Token string `json:"token"`
}

// ensureToken returns a valid PocketBase superuser token. When no token is
// cached it logs in with the configured username/password
// (POST /api/collections/_superusers/auth-with-password).
func (s *CertimateService) ensureToken() (string, error) {
	if s.Token != "" {
		return s.Token, nil
	}
	if s.Username == "" || s.Password == "" {
		return "", errors.New("Certimate 未配置登录账号密码")
	}

	body, err := json.Marshal(map[string]string{
		"identity": s.Username,
		"password": s.Password,
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode Certimate login: %w", err)
	}

	apiURL := strings.TrimRight(s.BaseURL, "/") + "/api/collections/_superusers/auth-with-password"
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to call Certimate login API: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read Certimate login response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Certimate login failed (status %d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var auth certimateAuthResponse
	if err := json.Unmarshal(data, &auth); err != nil {
		return "", fmt.Errorf("failed to parse Certimate login response: %w", err)
	}
	if auth.Token == "" {
		return "", errors.New("Certimate login did not return a token")
	}
	s.Token = auth.Token
	return s.Token, nil
}

// getJSON performs an authenticated GET and returns the response body. On a
// 401 the cached token is dropped and the request is retried once after a fresh
// login (the token may have expired since it was cached).
func (s *CertimateService) getJSON(apiURL string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	for attempt := 0; attempt < 2; attempt++ {
		token, err := s.ensureToken()
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to call Certimate API: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response: %w", readErr)
		}

		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			s.Token = "" // force re-authentication
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Certimate API returned status %d: %s", resp.StatusCode, string(body))
		}
		return body, nil
	}

	return nil, errors.New("Certimate API authorization failed")
}

func (s *CertimateService) ListCertificates(page, perPage int) (*CertimateCertListResponse, error) {
	apiURL := fmt.Sprintf("%s/api/collections/certificate/records?page=%d&perPage=%d", s.BaseURL, page, perPage)

	body, err := s.getJSON(apiURL)
	if err != nil {
		return nil, err
	}

	var result CertimateCertListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (s *CertimateService) GetCertificate(certID string) (*CertimateCertRecord, error) {
	apiURL := fmt.Sprintf("%s/api/collections/certificate/records/%s", s.BaseURL, url.PathEscape(certID))

	body, err := s.getJSON(apiURL)
	if err != nil {
		return nil, err
	}

	var result CertimateCertRecord
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (s *CertimateService) SyncCertificates(userID uint) (int, error) {
	page := 1
	perPage := 100
	total := 0

	for {
		result, err := s.ListCertificates(page, perPage)
		if err != nil {
			return total, err
		}

		for _, cert := range result.Items {
			notBefore := parseCertimateTime(cert.ValidityNotBefore)
			notAfter := parseCertimateTime(cert.ValidityNotAfter)
			isExpired := notAfter != nil && notAfter.Before(time.Now())
			status := "active"
			if isExpired {
				status = "expired"
			}

			domain := extractPrimaryDomain(cert.SubjectAltNames)

			values := map[string]interface{}{
				"domain":              domain,
				"issuer":              cert.IssuerOrg,
				"serial_number":       cert.SerialNumber,
				"not_before":          notBefore,
				"not_after":           notAfter,
				"subject_alt_names":   cert.SubjectAltNames,
				"key_algorithm":       cert.KeyAlgorithm,
				"signature_algorithm": cert.SignatureAlgorithm,
				"is_expired":          isExpired,
				"source":              "certimate",
				"certificate":         cert.Certificate,
				"private_key":         cert.PrivateKey,
				"status":              status,
			}

			var existing models.Certificate
			result_db := database.DB.Where("user_id = ? AND certimate_id = ?", userID, cert.ID).First(&existing)
			if result_db.Error == nil {
				// Map-based update so zero values (e.g. is_expired=false after
				// renewal) are written correctly.
				if err := database.DB.Model(&existing).Updates(values).Error; err != nil {
					return total, fmt.Errorf("failed to update certificate %s: %w", cert.ID, err)
				}
			} else {
				dbCert := models.Certificate{
					UserID:             userID,
					CertimateID:        cert.ID,
					Domain:             domain,
					Issuer:             cert.IssuerOrg,
					SerialNumber:       cert.SerialNumber,
					NotBefore:          notBefore,
					NotAfter:           notAfter,
					SubjectAltNames:    cert.SubjectAltNames,
					KeyAlgorithm:       cert.KeyAlgorithm,
					SignatureAlgorithm: cert.SignatureAlgorithm,
					IsExpired:          isExpired,
					Source:             "certimate",
					Certificate:        cert.Certificate,
					PrivateKey:         cert.PrivateKey,
					Status:             status,
				}
				if err := database.DB.Create(&dbCert).Error; err != nil {
					return total, fmt.Errorf("failed to create certificate %s: %w", cert.ID, err)
				}
			}
			total++
		}

		if len(result.Items) < perPage {
			break
		}
		page++
	}

	return total, nil
}

func extractPrimaryDomain(san string) string {
	if san == "" {
		return ""
	}
	// SAN entries may be separated by ";" or "," and may include wildcard
	// prefixes (e.g. "*.example.com") or "DNS:" labels.
	for _, d := range splitAndTrim(san, ";") {
		for _, part := range splitAndTrim(d, ",") {
			name := strings.ToLower(strings.TrimSpace(part))
			name = strings.TrimPrefix(name, "*.")
			name = strings.TrimPrefix(name, "dns:")
			name = strings.Trim(name, ".")
			if name != "" && strings.Contains(name, ".") {
				return name
			}
		}
	}
	return strings.ToLower(strings.TrimSpace(san))
}

// parseCertimateTime parses the time formats Certimate can return.
// It returns nil instead of a zero time when parsing fails so certificates
// with unknown dates are not mislabelled as expired.
func parseCertimateTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}

func splitAndTrim(s, sep string) []string {
	result := []string{}
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	result := []string{}
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
