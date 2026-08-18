package services

import (
	"testing"

	"DomainManager/config"
)

func TestIsPPUADomain(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"example.pp.ua", true},
		{"test.PP.UA", true},
		{"pp.ua", false},
		{"example.com", false},
		{"example.com.ua", false},
	}
	for _, c := range cases {
		if got := IsPPUADomain(c.in); got != c.want {
			t.Fatalf("IsPPUADomain(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseUaWhois(t *testing.T) {
	config.AppConfig = &config.Config{UaWhoisServer: "whois.ua:43"}
	raw := `% comment line
domain:           pp.ua
dom-public:       NO
nserver:          ns1.uadns.com
nserver:          ns2.uadns.com
status:           ok
created:          2008-01-01 10:35:38+02
modified:         2025-12-28 16:06:20+02
expires:          2035-01-01 10:35:38+02
source:           UAEPP
registrar:        ua.drs
organization:     Service Online LLC
url:              http://drs.ua
country:          UA
person:           not published
e-mail:           not published
`
	info := parseUaWhois("test.pp.ua", raw)
	if info.Registrar != "ua.drs" {
		t.Fatalf("registrar = %q, want ua.drs", info.Registrar)
	}
	if info.Nameservers != "ns1.uadns.com,ns2.uadns.com" {
		t.Fatalf("nameservers = %q", info.Nameservers)
	}
	if info.ExpiryDate == nil || info.ExpiryDate.Year() != 2035 {
		t.Fatalf("expiry = %v, want 2035", info.ExpiryDate)
	}
	if info.CreationDate == nil || info.CreationDate.Year() != 2008 {
		t.Fatalf("creation = %v, want 2008", info.CreationDate)
	}
	if info.Status != "ok" {
		t.Fatalf("status = %q, want ok", info.Status)
	}
	if info.DomainName != "test.pp.ua" {
		t.Fatalf("domain name = %q, want test.pp.ua", info.DomainName)
	}
}

func TestParseUaWhoisDate(t *testing.T) {
	if parseUaWhoisDate("2008-01-01 10:35:38+02") == nil {
		t.Fatal("expected offset date to parse")
	}
	if parseUaWhoisDate("garbage") != nil {
		t.Fatal("expected garbage to fail")
	}
}
