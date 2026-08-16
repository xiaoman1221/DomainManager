package services

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"strings"
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
	if config.From == "" {
		config.From = config.Username
	}

	toAddrs := splitAndTrim(config.To, ",")
	if len(toAddrs) == 0 {
		return fmt.Errorf("at least one recipient email is required")
	}

	addr := net.JoinHostPort(config.SMTPHost, config.SMTPPort)
	tlsConfig := &tls.Config{ServerName: config.SMTPHost}

	// Port 465 uses implicit TLS; other ports use plain text + STARTTLS.
	var conn net.Conn
	var err error
	if config.SMTPPort == "465" {
		conn, err = tls.Dial("tcp", addr, tlsConfig)
	} else {
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to start SMTP session: %w", err)
	}
	defer client.Close()

	if config.SMTPPort != "465" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}

	if config.Username != "" {
		auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	if err := client.Mail(config.From); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	for _, to := range toAddrs {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("SMTP RCPT TO %s failed: %w", to, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open SMTP data connection: %w", err)
	}

	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s",
		config.From, strings.Join(toAddrs, ","), encodedSubject, body)
	if _, err := w.Write([]byte(msg)); err != nil {
		w.Close()
		return fmt.Errorf("failed to write email body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to finish email: %w", err)
	}
	client.Quit()

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
