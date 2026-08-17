package handlers

import (
	"encoding/json"
	"testing"

	"DomainManager/config"
	"DomainManager/services"
)

func TestApplyGlobalSMTPDefaults(t *testing.T) {
	config.AppConfig = &config.Config{DBSettings: map[string]string{
		config.SettingSMTPHost:       "smtp.example.com",
		config.SettingSMTPPort:       "465",
		config.SettingSMTPUsername:   "mailer",
		config.SettingSMTPPassword:   "secret",
		config.SettingSMTPFrom:       "noreply@example.com",
		config.SettingSMTPEncryption: "ssl",
	}}

	// Email channel without its own SMTP -> filled from global settings.
	out := applyGlobalSMTPDefaults("email", `{"to":"a@b.com"}`)
	var cfg services.EmailConfig
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("unmarshal merged config: %v", err)
	}
	if cfg.SMTPHost != "smtp.example.com" || cfg.SMTPPort != "465" || cfg.Username != "mailer" || cfg.Password != "secret" {
		t.Fatalf("global SMTP not applied: %+v", cfg)
	}
	if cfg.From != "noreply@example.com" {
		t.Fatalf("from = %q, want noreply@example.com", cfg.From)
	}
	if !cfg.UseTLS {
		t.Fatal("use_tls should be true for ssl encryption")
	}

	// Email channel with its own SMTP -> left untouched.
	out2 := applyGlobalSMTPDefaults("email", `{"smtp_host":"own.example.com","smtp_port":"587","username":"own","to":"a@b.com"}`)
	var cfg2 services.EmailConfig
	if err := json.Unmarshal([]byte(out2), &cfg2); err != nil {
		t.Fatalf("unmarshal own config: %v", err)
	}
	if cfg2.SMTPHost != "own.example.com" || cfg2.SMTPPort != "587" || cfg2.Username != "own" {
		t.Fatalf("channel SMTP was overridden: %+v", cfg2)
	}

	// Non-email channel -> passthrough unchanged.
	if got := applyGlobalSMTPDefaults("bark", `{"key":"x"}`); got != `{"key":"x"}` {
		t.Fatalf("bark config should pass through, got %q", got)
	}
}
