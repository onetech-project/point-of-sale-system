package providers

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/jordan-wright/email"
)

// EmailError represents different types of email sending errors
type EmailError struct {
	Type    EmailErrorType
	Message string
	Err     error
}

type EmailErrorType int

const (
	EmailErrorTypeUnknown EmailErrorType = iota
	EmailErrorTypeConnection
	EmailErrorTypeAuth
	EmailErrorTypeTimeout
	EmailErrorTypeInvalidRecipient
	EmailErrorTypeRateLimited
)

func (e *EmailError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *EmailError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error is transient and can be retried
func (e *EmailError) IsRetryable() bool {
	switch e.Type {
	case EmailErrorTypeConnection, EmailErrorTypeTimeout, EmailErrorTypeRateLimited:
		return true
	default:
		return false
	}
}

type EmailProvider interface {
	Send(to, subject, body string, isHTML bool) error
}

type SMTPEmailProvider struct {
	host          string
	port          string
	username      string
	password      string
	from          string
	retryAttempts int
	retryDelay    time.Duration
}

func NewSMTPEmailProvider() *SMTPEmailProvider {
	retryAttempts := 3
	if attempts := getEnv("SMTP_RETRY_ATTEMPTS", "3"); attempts != "" {
		fmt.Sscanf(attempts, "%d", &retryAttempts)
	}

	return &SMTPEmailProvider{
		host:          getEnv("SMTP_HOST", "localhost"),
		port:          getEnv("SMTP_PORT", "587"),
		username:      getEnv("SMTP_USERNAME", ""),
		password:      getEnv("SMTP_PASSWORD", ""),
		from:          getEnv("SMTP_FROM", "noreply@pos-system.com"),
		retryAttempts: retryAttempts,
		retryDelay:    2 * time.Second,
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

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt <= p.retryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2s, 4s, 8s
			delay := p.retryDelay * time.Duration(1<<uint(attempt-1))
			fmt.Printf("[EMAIL] Retry attempt %d/%d after %v\n", attempt, p.retryAttempts, delay)
			time.Sleep(delay)
		}

		err := e.Send(addr, auth)
		if err == nil {
			if attempt > 0 {
				fmt.Printf("[EMAIL] Successfully sent after %d retries\n", attempt)
			}
			return nil
		}

		lastErr = err
		emailErr := classifyEmailError(err)

		// Don't retry if error is not retryable
		if !emailErr.IsRetryable() {
			fmt.Printf("[EMAIL] Non-retryable error: %v\n", emailErr)
			return emailErr
		}

		fmt.Printf("[EMAIL] Retryable error (attempt %d/%d): %v\n", attempt+1, p.retryAttempts+1, emailErr)
	}

	return &EmailError{
		Type:    EmailErrorTypeUnknown,
		Message: fmt.Sprintf("failed after %d attempts", p.retryAttempts+1),
		Err:     lastErr,
	}
}

// classifyEmailError classifies SMTP errors into specific types
func classifyEmailError(err error) *EmailError {
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "connection refused"), strings.Contains(errStr, "connection reset"):
		return &EmailError{
			Type:    EmailErrorTypeConnection,
			Message: "SMTP connection failed",
			Err:     err,
		}
	case strings.Contains(errStr, "authentication failed"), strings.Contains(errStr, "invalid credentials"):
		return &EmailError{
			Type:    EmailErrorTypeAuth,
			Message: "SMTP authentication failed",
			Err:     err,
		}
	case strings.Contains(errStr, "timeout"), strings.Contains(errStr, "deadline exceeded"):
		return &EmailError{
			Type:    EmailErrorTypeTimeout,
			Message: "SMTP connection timeout",
			Err:     err,
		}
	case strings.Contains(errStr, "invalid recipient"), strings.Contains(errStr, "mailbox unavailable"):
		return &EmailError{
			Type:    EmailErrorTypeInvalidRecipient,
			Message: "Invalid or unavailable recipient",
			Err:     err,
		}
	case strings.Contains(errStr, "rate limit"), strings.Contains(errStr, "too many"):
		return &EmailError{
			Type:    EmailErrorTypeRateLimited,
			Message: "SMTP rate limit exceeded",
			Err:     err,
		}
	default:
		return &EmailError{
			Type:    EmailErrorTypeUnknown,
			Message: "SMTP send failed",
			Err:     err,
		}
	}
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
