package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type NotificationService struct{}

type BarkPayload struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Group     string `json:"group"`
	Sound     string `json:"sound"`
	IsArchive int    `json:"isArchive"`
}

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type BarkConfig struct {
	Server string `json:"server"`
	Key    string `json:"key"`
	Group  string `json:"group"`
}

type EmailConfig struct {
	SMTPHost string `json:"smtp_host"`
	SMTPPort string `json:"smtp_port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"`
	UseTLS   bool   `json:"use_tls"`
}

type WebhookConfig struct {
	URL    string `json:"url"`
	Method string `json:"method"`
}

func NewNotificationService() *NotificationService {
	return &NotificationService{}
}

func (s *NotificationService) SendBark(config BarkConfig, title, body string) error {
	server := config.Server
	if server == "" {
		server = "https://api.day.app"
	}

	payload := BarkPayload{
		Title:     title,
		Body:      body,
		Group:     config.Group,
		Sound:     "bell",
		IsArchive: 1,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal bark payload: %w", err)
	}

	apiURL := fmt.Sprintf("%s/%s", server, config.Key)
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send bark notification: %w", err)
	}
	defer resp.Body.Close()

	bodyResp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("bark API returned status %d: %s", resp.StatusCode, string(bodyResp))
	}

	return nil
}

func (s *NotificationService) SendTelegram(config TelegramConfig, title, body string) error {
	if config.BotToken == "" || config.ChatID == "" {
		return fmt.Errorf("telegram bot token and chat_id are required")
	}

	text := fmt.Sprintf("*%s*\n\n%s", title, body)
	msg := TelegramMessage{
		ChatID:    config.ChatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram message: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.BotToken)
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}
	defer resp.Body.Close()

	bodyResp, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode, string(bodyResp))
	}

	return nil
}

func (s *NotificationService) SendWebhook(config WebhookConfig, title, body string) error {
	if config.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	method := config.Method
	if method == "" {
		method = "POST"
	}

	payload := map[string]string{
		"title": title,
		"body":  body,
		"time":  time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest(method, config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyResp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(bodyResp))
	}

	return nil
}

func (s *NotificationService) SendEmail(config EmailConfig, subject, body string) error {
	if config.SMTPHost == "" || config.SMTPPort == "" {
		return fmt.Errorf("SMTP host and port are required")
	}

	// Simple email sending via net/smtp
	addr := fmt.Sprintf("%s:%s", config.SMTPHost, config.SMTPPort)

	toAddrs := splitAndTrim(config.To, ",")
	if len(toAddrs) == 0 {
		return fmt.Errorf("at least one recipient email is required")
	}

	// Build raw email
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		config.From, config.To, subject, body)

	// For now, just return nil - actual SMTP sending requires net/smtp
	// In production, this should use net/smtp or a library
	_ = addr
	_ = msg

	return nil
}

func (s *NotificationService) SendByType(notificationType, configStr, title, body string) error {
	switch notificationType {
	case "bark":
		var config BarkConfig
		if err := json.Unmarshal([]byte(configStr), &config); err != nil {
			return fmt.Errorf("invalid bark config: %w", err)
		}
		return s.SendBark(config, title, body)

	case "telegram":
		var config TelegramConfig
		if err := json.Unmarshal([]byte(configStr), &config); err != nil {
			return fmt.Errorf("invalid telegram config: %w", err)
		}
		return s.SendTelegram(config, title, body)

	case "email":
		var config EmailConfig
		if err := json.Unmarshal([]byte(configStr), &config); err != nil {
			return fmt.Errorf("invalid email config: %w", err)
		}
		return s.SendEmail(config, title, body)

	case "webhook":
		var config WebhookConfig
		if err := json.Unmarshal([]byte(configStr), &config); err != nil {
			return fmt.Errorf("invalid webhook config: %w", err)
		}
		return s.SendWebhook(config, title, body)

	default:
		return fmt.Errorf("unsupported notification type: %s", notificationType)
	}
}
