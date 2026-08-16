package services

import "testing"

func TestParseAliyunPage(t *testing.T) {
	body := []byte(`{"Data":{"Domain":[{"DomainName":"a.com","ExpirationDate":"2027-01-02 00:00:00","AutoRenewEnabled":true},{"DomainName":"b.com"}]},"TotalItemNum":2}`)
	domains, total, err := parseAliyunPage(body)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(domains) != 2 {
		t.Fatalf("len(domains) = %d, want 2", len(domains))
	}
	if domains[0].Name != "a.com" {
		t.Errorf("name = %q", domains[0].Name)
	}
	if domains[0].ExpiryDate == nil {
		t.Error("expiry_date not parsed")
	}
	if !domains[0].AutoRenew {
		t.Error("auto_renew not parsed")
	}
}

func TestParseAliyunPageError(t *testing.T) {
	body := []byte(`{"Code":"InvalidAccessKeyId.NotFound","Message":"bad key"}`)
	if _, _, err := parseAliyunPage(body); err == nil {
		t.Error("expected error for API error body")
	}
}

func TestParseTencentPage(t *testing.T) {
	body := []byte(`{"Response":{"TotalCount":1,"DomainSet":[{"DomainName":"t.com","ExpirationDate":"2026-08-01 00:00:00","AutoRenew":1}]}}`)
	domains, total, err := parseTencentPage(body)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(domains) != 1 {
		t.Fatalf("total=%d len=%d", total, len(domains))
	}
	if domains[0].Name != "t.com" || !domains[0].AutoRenew {
		t.Errorf("name=%q autoRenew=%v", domains[0].Name, domains[0].AutoRenew)
	}
	if domains[0].ExpiryDate == nil {
		t.Error("expiry not parsed")
	}
}

func TestParseHuaweiPage(t *testing.T) {
	body := []byte(`{"total_count":2,"domains":[{"domain_name":"h.com","domain_status":"ACTIVE","expiration_date":"2027-03-01T00:00:00Z"},{"domain_name":"h.net"}]}`)
	domains, total, err := parseHuaweiPage(body)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(domains) != 2 {
		t.Fatalf("total=%d len=%d", total, len(domains))
	}
	if domains[0].Name != "h.com" || domains[0].Status != "active" {
		t.Errorf("name=%q status=%q", domains[0].Name, domains[0].Status)
	}
	if domains[0].ExpiryDate == nil {
		t.Error("expiry not parsed")
	}
}

func TestParseNamecheapPage(t *testing.T) {
	body := []byte(`<ApiResponse Status="OK"><CommandResponse><DomainGetListResult Domain="api.namecheap.com" Total="2" TotalDomains="3" Page="1" PageSize="100"><Domain Name="a.com" ID="1"/><Domain Name="b.net" ID="2"/><Domain Name="c.org" ID="3"/></DomainGetListResult></CommandResponse></ApiResponse>`)
	domains, total, err := parseNamecheapPage(body)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(domains) != 3 {
		t.Fatalf("len = %d, want 3", len(domains))
	}
	if domains[0].Name != "a.com" {
		t.Errorf("first = %q", domains[0].Name)
	}
}

func TestParseNamecheapPageError(t *testing.T) {
	body := []byte(`<ApiResponse Status="ERROR"><Errors><Error Number="2014156">Invalid API Key</Error></Errors></ApiResponse>`)
	if _, _, err := parseNamecheapPage(body); err == nil {
		t.Error("expected error for ERROR status")
	}
}
