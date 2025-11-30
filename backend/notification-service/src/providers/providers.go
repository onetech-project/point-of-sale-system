package providers

import (
	"fmt"
	"net/smtp"
	"os"

	"github.com/jordan-wright/email"
)

type EmailProvider interface {
	Send(to, subject, body string, isHTML bool) error
}

type SMTPEmailProvider struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPEmailProvider() *SMTPEmailProvider {
	return &SMTPEmailProvider{
		host:     getEnv("SMTP_HOST", "localhost"),
		port:     getEnv("SMTP_PORT", "587"),
		username: getEnv("SMTP_USERNAME", ""),
		password: getEnv("SMTP_PASSWORD", ""),
		from:     getEnv("SMTP_FROM", "noreply@pos-system.com"),
	}
}

func (p *SMTPEmailProvider) Send(to, subject, body string, isHTML bool) error {
	e := email.NewEmail()
	e.From = p.from
	e.To = []string{to}
	e.Subject = subject

	if isHTML {
		e.HTML = []byte(body)
	} else {
		e.Text = []byte(body)
	}

	// If no SMTP configured, log and return (for development)
	if p.username == "" {
		fmt.Printf("[EMAIL] To: %s, Subject: %s\n%s\n", to, subject, body)
		return nil
	}

	addr := fmt.Sprintf("%s:%s", p.host, p.port)
	auth := smtp.PlainAuth("", p.username, p.password, p.host)

	return e.Send(addr, auth)
}

type PushProvider interface {
	Send(token, title, body string, data map[string]string) error
}

type MockPushProvider struct{}

func NewMockPushProvider() *MockPushProvider {
	return &MockPushProvider{}
}

func (p *MockPushProvider) Send(token, title, body string, data map[string]string) error {
	fmt.Printf("[PUSH] Token: %s, Title: %s, Body: %s, Data: %v\n", token, title, body, data)
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
