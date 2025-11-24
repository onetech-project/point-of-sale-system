package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/pos/notification-service/src/models"
	"github.com/pos/notification-service/src/providers"
	"github.com/pos/notification-service/src/repository"
)

type NotificationService struct {
	repo          *repository.NotificationRepository
	emailProvider providers.EmailProvider
	pushProvider  providers.PushProvider
}

func NewNotificationService(db *sql.DB) *NotificationService {
	return &NotificationService{
		repo:          repository.NewNotificationRepository(db),
		emailProvider: providers.NewSMTPEmailProvider(),
		pushProvider:  providers.NewMockPushProvider(),
	}
}

// HandleEvent processes notification events from Kafka
func (s *NotificationService) HandleEvent(ctx context.Context, eventData []byte) error {
	var event models.NotificationEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("Processing event: %s for tenant: %s", event.EventType, event.TenantID)

	switch event.EventType {
	case "user.registered":
		return s.handleUserRegistration(ctx, event)
	case "user.login":
		return s.handleUserLogin(ctx, event)
	case "password.reset_requested":
		return s.handlePasswordResetRequest(ctx, event)
	case "password.changed":
		return s.handlePasswordChanged(ctx, event)
	default:
		log.Printf("Unknown event type: %s", event.EventType)
		return nil
	}
}

func (s *NotificationService) handleUserRegistration(ctx context.Context, event models.NotificationEvent) error {
	email, _ := event.Data["email"].(string)
	name, _ := event.Data["name"].(string)
	verificationToken, _ := event.Data["verification_token"].(string)

	subject := "Welcome! Please verify your email"
	body := s.renderTemplate("registration", map[string]interface{}{
		"Name":  name,
		"Token": verificationToken,
		"URL":   fmt.Sprintf("https://pos-system.com/verify?token=%s", verificationToken),
	})

	notification := &models.Notification{
		TenantID:  event.TenantID,
		UserID:    &event.UserID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  event.Data,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return s.sendEmail(ctx, notification)
}

func (s *NotificationService) handleUserLogin(ctx context.Context, event models.NotificationEvent) error {
	email, _ := event.Data["email"].(string)
	name, _ := event.Data["name"].(string)
	ipAddress, _ := event.Data["ip_address"].(string)
	userAgent, _ := event.Data["user_agent"].(string)

	subject := "New login to your account"
	body := s.renderTemplate("login_alert", map[string]interface{}{
		"Name":      name,
		"IPAddress": ipAddress,
		"UserAgent": userAgent,
		"Time":      time.Now().Format("2006-01-02 15:04:05"),
	})

	notification := &models.Notification{
		TenantID:  event.TenantID,
		UserID:    &event.UserID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  event.Data,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return s.sendEmail(ctx, notification)
}

func (s *NotificationService) handlePasswordResetRequest(ctx context.Context, event models.NotificationEvent) error {
	email, _ := event.Data["email"].(string)
	name, _ := event.Data["name"].(string)
	resetToken, _ := event.Data["reset_token"].(string)

	subject := "Password Reset Request"
	body := s.renderTemplate("password_reset", map[string]interface{}{
		"Name":  name,
		"Token": resetToken,
		"URL":   fmt.Sprintf("https://pos-system.com/reset-password?token=%s", resetToken),
	})

	notification := &models.Notification{
		TenantID:  event.TenantID,
		UserID:    &event.UserID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  event.Data,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return s.sendEmail(ctx, notification)
}

func (s *NotificationService) handlePasswordChanged(ctx context.Context, event models.NotificationEvent) error {
	email, _ := event.Data["email"].(string)
	name, _ := event.Data["name"].(string)

	subject := "Your password has been changed"
	body := s.renderTemplate("password_changed", map[string]interface{}{
		"Name": name,
		"Time": time.Now().Format("2006-01-02 15:04:05"),
	})

	notification := &models.Notification{
		TenantID:  event.TenantID,
		UserID:    &event.UserID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  event.Data,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return s.sendEmail(ctx, notification)
}

func (s *NotificationService) sendEmail(ctx context.Context, notification *models.Notification) error {
	err := s.emailProvider.Send(notification.Recipient, notification.Subject, notification.Body, true)
	
	now := time.Now()
	if err != nil {
		notification.Status = models.NotificationStatusFailed
		notification.FailedAt = &now
		errMsg := err.Error()
		notification.ErrorMsg = &errMsg
		notification.RetryCount++
	} else {
		notification.Status = models.NotificationStatusSent
		notification.SentAt = &now
	}

	if updateErr := s.repo.UpdateStatus(ctx, notification.ID, notification.Status, notification.SentAt, notification.FailedAt, notification.ErrorMsg); updateErr != nil {
		log.Printf("Failed to update notification status: %v", updateErr)
	}

	return err
}

func (s *NotificationService) renderTemplate(templateName string, data map[string]interface{}) string {
	templates := map[string]string{
		"registration": `
<html>
<body>
	<h2>Welcome {{.Name}}!</h2>
	<p>Thank you for registering with our POS system.</p>
	<p>Please verify your email by clicking the link below:</p>
	<p><a href="{{.URL}}">Verify Email</a></p>
	<p>Or copy this link: {{.URL}}</p>
	<p>This link will expire in 24 hours.</p>
</body>
</html>`,
		"login_alert": `
<html>
<body>
	<h2>New Login Detected</h2>
	<p>Hello {{.Name}},</p>
	<p>We detected a new login to your account:</p>
	<ul>
		<li>Time: {{.Time}}</li>
		<li>IP Address: {{.IPAddress}}</li>
		<li>Device: {{.UserAgent}}</li>
	</ul>
	<p>If this wasn't you, please reset your password immediately.</p>
</body>
</html>`,
		"password_reset": `
<html>
<body>
	<h2>Password Reset Request</h2>
	<p>Hello {{.Name}},</p>
	<p>We received a request to reset your password.</p>
	<p>Click the link below to reset your password:</p>
	<p><a href="{{.URL}}">Reset Password</a></p>
	<p>Or copy this link: {{.URL}}</p>
	<p>This link will expire in 1 hour.</p>
	<p>If you didn't request this, please ignore this email.</p>
</body>
</html>`,
		"password_changed": `
<html>
<body>
	<h2>Password Changed Successfully</h2>
	<p>Hello {{.Name}},</p>
	<p>Your password was changed at {{.Time}}.</p>
	<p>If you didn't make this change, please contact support immediately.</p>
</body>
</html>`,
	}

	tmplStr, ok := templates[templateName]
	if !ok {
		return "Template not found"
	}

	tmpl, err := template.New(templateName).Parse(tmplStr)
	if err != nil {
		return fmt.Sprintf("Template error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Template execution error: %v", err)
	}

	return buf.String()
}
