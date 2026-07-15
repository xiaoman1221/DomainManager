package services

import (
	"DomainManager/database"
	"DomainManager/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type CertimateService struct {
	BaseURL string
	Token   string
}

type CertimateCertListResponse struct {
	Page      int                      `json:"page"`
	PerPage   int                      `json:"perPage"`
	TotalItems int                    `json:"totalItems"`
	Items     []CertimateCertRecord    `json:"items"`
}

type CertimateCertRecord struct {
	ID                string   `json:"id"`
	Source            string   `json:"source"`
	SubjectAltNames   string   `json:"subjectAltNames"`
	SerialNumber      string   `json:"serialNumber"`
	Certificate       string   `json:"certificate"`
	PrivateKey        string   `json:"privateKey"`
	IssuerOrg         string   `json:"issuerOrg"`
	IssuerCertificate string   `json:"issuerCertificate"`
	KeyAlgorithm      string   `json:"keyAlgorithm"`
	SignatureAlgorithm string  `json:"signatureAlgorithm"`
	ValidityNotBefore string   `json:"validityNotBefore"`
	ValidityNotAfter  string   `json:"validityNotAfter"`
	Created           string   `json:"created"`
	Updated           string   `json:"updated"`
}

func NewCertimateService(config models.CertimateConfig) *CertimateService {
	return &CertimateService{
		BaseURL: config.URL,
		Token:   config.Token,
	}
}

func (s *CertimateService) ListCertificates(page, perPage int) (*CertimateCertListResponse, error) {
	apiURL := fmt.Sprintf("%s/api/collections/certificate/records?page=%d&perPage=%d", s.BaseURL, page, perPage)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Certimate API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Certimate API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result CertimateCertListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (s *CertimateService) GetCertificate(certID string) (*CertimateCertRecord, error) {
	apiURL := fmt.Sprintf("%s/api/collections/certificate/records/%s", s.BaseURL, url.PathEscape(certID))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Certimate API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Certimate API returned status %d: %s", resp.StatusCode, string(body))
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
			notBefore, _ := time.Parse(time.RFC3339, cert.ValidityNotBefore)
			notAfter, _ := time.Parse(time.RFC3339, cert.ValidityNotAfter)
			isExpired := notAfter.Before(time.Now())

			domain := extractPrimaryDomain(cert.SubjectAltNames)

			dbCert := models.Certificate{
				UserID:          userID,
				CertimateID:     cert.ID,
				Domain:          domain,
				Issuer:          cert.IssuerOrg,
				SerialNumber:    cert.SerialNumber,
				NotBefore:       &notBefore,
				NotAfter:        &notAfter,
				SubjectAltNames: cert.SubjectAltNames,
				KeyAlgorithm:    cert.KeyAlgorithm,
				SignatureAlgorithm: cert.SignatureAlgorithm,
				IsExpired:       isExpired,
				Source:          "certimate",
				Certificate:     cert.Certificate,
				PrivateKey:      cert.PrivateKey,
				Status:          map[bool]string{true: "expired", false: "active"}[isExpired],
			}

			var existing models.Certificate
			result_db := database.DB.Where("user_id = ? AND certimate_id = ?", userID, cert.ID).First(&existing)
			if result_db.Error == nil {
				database.DB.Model(&existing).Updates(dbCert)
			} else {
				database.DB.Create(&dbCert)
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
	domains := []string{}
	for _, d := range splitAndTrim(san, ";") {
		if d != "" {
			domains = append(domains, d)
		}
	}
	if len(domains) > 0 {
		return domains[0]
	}
	return san
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
