package services

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"DomainManager/config"
)

// uaWhoisServerDefault is the UANIC whois server (port 43) that backs the
// official dig.ua web interface (https://dig.ua).
const uaWhoisServerDefault = "whois.ua:43"

// IsPPUADomain reports whether the domain is a *.pp.ua free second-level
// domain such as example.pp.ua. The zone name itself (pp.ua) is not counted.
func IsPPUADomain(domain string) bool {
	name := strings.TrimSpace(strings.ToLower(domain))
	if name == "pp.ua" {
		return false
	}
	return strings.HasSuffix(name, ".pp.ua")
}

// uaWhoisServer returns the configured UANIC whois server address.
func uaWhoisServer() string {
	if config.AppConfig != nil && config.AppConfig.UaWhoisServer != "" {
		return config.AppConfig.UaWhoisServer
	}
	return uaWhoisServerDefault
}

// QueryUaWhois queries the UANIC whois server for a .ua zone domain. Since
// *.pp.ua subdomains share the pp.ua zone record, the base second-level domain
// (pp.ua) is queried and the result is attributed to the requested subdomain.
func QueryUaWhois(domain string) (*WhoisInfo, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	query := domain
	if IsPPUADomain(domain) {
		query = "pp.ua"
	}

	server := uaWhoisServer()
	conn, err := net.DialTimeout("tcp", server, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to UANIC whois server %s: %w", server, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))

	if _, err := fmt.Fprintf(conn, "%s\r\n", query); err != nil {
		return nil, fmt.Errorf("failed to send whois query: %w", err)
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read whois response: %w", err)
	}

	raw := sb.String()
	if strings.Contains(raw, "Invalid request.") || strings.Contains(raw, "No entries found") {
		return nil, fmt.Errorf("UANIC whois returned no data for %s", domain)
	}
	return parseUaWhois(domain, raw), nil
}

// parseUaWhois parses the UANIC "key: value" whois format.
func parseUaWhois(domain, raw string) *WhoisInfo {
	info := &WhoisInfo{
		DomainName:  domain,
		WhoisServer: uaWhoisServer(),
		RawText:     raw,
	}
	var nameservers []string
	var statuses []string

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:idx]))
		val := strings.TrimSpace(line[idx+1:])
		if val == "" {
			continue
		}
		switch key {
		case "domain":
			if info.DomainName == "" {
				info.DomainName = val
			}
		case "status":
			// The domain-level status is the first one in the UANIC record;
			// later status lines belong to registrant/contact sections.
			if len(statuses) == 0 {
				statuses = append(statuses, val)
			}
		case "created":
			if t := parseUaWhoisDate(val); t != nil {
				info.CreationDate = t
			}
		case "modified":
			if t := parseUaWhoisDate(val); t != nil {
				info.UpdatedDate = t
			}
		case "expires":
			if t := parseUaWhoisDate(val); t != nil {
				info.ExpiryDate = t
			}
		case "nserver":
			nameservers = append(nameservers, strings.ToLower(strings.Fields(val)[0]))
		case "registrar":
			info.Registrar = val
		case "organization":
			if info.RegistrantOrg == "" {
				info.RegistrantOrg = val
			}
		case "person":
			if info.RegistrantName == "" {
				info.RegistrantName = val
			}
		case "e-mail":
			if info.RegistrantEmail == "" {
				info.RegistrantEmail = val
			}
		case "url":
			if info.RegistrarURL == "" {
				info.RegistrarURL = val
			}
		case "country":
			if info.RegistrantCountry == "" {
				info.RegistrantCountry = val
			}
		}
	}

	if len(nameservers) > 0 {
		info.Nameservers = strings.Join(nameservers, ",")
	}
	if len(statuses) > 0 {
		info.Status = strings.Join(statuses, ", ")
	}
	return info
}

// parseUaWhoisDate parses UANIC date strings such as "2008-01-01 10:35:38+02".
func parseUaWhoisDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	formats := []string{
		"2006-01-02 15:04:05-07",
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
