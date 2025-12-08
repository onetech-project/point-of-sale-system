package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	templates     map[string]*template.Template
	frontendURL   string
}

func NewNotificationService(db *sql.DB) *NotificationService {
	service := &NotificationService{
		repo:          repository.NewNotificationRepository(db),
		emailProvider: providers.NewSMTPEmailProvider(),
		pushProvider:  providers.NewMockPushProvider(),
		templates:     make(map[string]*template.Template),
		frontendURL:   getEnv("FRONTEND_DOMAIN", "http://localhost:3000"),
	}

	// Load all templates
	if err := service.loadTemplates(); err != nil {
		log.Printf("Warning: Failed to load templates: %v", err)
	}

	return service
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (s *NotificationService) loadTemplates() error {
	templateDir := getEnv("TEMPLATE_DIR", "./templates")

	templateFiles := []string{
		"registration.html",
		"login_alert.html",
		"password_reset.html",
		"password_changed.html",
		"team_invitation.html",
		"order_invoice.html",
	}

	for _, filename := range templateFiles {
		templatePath := filepath.Join(templateDir, filename)
		tmpl, err := template.ParseFiles(templatePath)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", filename, err)
		}

		// Store template with name without extension
		templateName := filename[:len(filename)-5] // Remove .html
		s.templates[templateName] = tmpl
		log.Printf("Loaded template: %s", templateName)
	}

	return nil
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
	case "invitation.created":
		return s.handleTeamInvitation(ctx, event)
	case "order.invoice":
		return s.handleOrderInvoice(ctx, event)
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
		"URL":   fmt.Sprintf("%s/verify?token=%s", s.frontendURL, verificationToken),
	})

	// Add event_type to metadata
	metadata := event.Data
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event_type"] = event.EventType

	notification := &models.Notification{
		TenantID:  event.TenantID,
		UserID:    &event.UserID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  metadata,
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

	// Add event_type to metadata
	metadata := event.Data
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event_type"] = event.EventType

	notification := &models.Notification{
		TenantID:  event.TenantID,
		UserID:    &event.UserID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  metadata,
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
		"URL":   fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, resetToken),
	})

	// Add event_type to metadata
	metadata := event.Data
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event_type"] = event.EventType

	notification := &models.Notification{
		TenantID:  event.TenantID,
		UserID:    &event.UserID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  metadata,
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

	// Add event_type to metadata
	metadata := event.Data
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event_type"] = event.EventType

	notification := &models.Notification{
		TenantID:  event.TenantID,
		UserID:    &event.UserID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  metadata,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return s.sendEmail(ctx, notification)
}

func (s *NotificationService) handleTeamInvitation(ctx context.Context, event models.NotificationEvent) error {
	email, _ := event.Data["email"].(string)
	inviterName, _ := event.Data["inviter_name"].(string)
	tenantName, _ := event.Data["tenant_name"].(string)
	role, _ := event.Data["role"].(string)
	invitationToken, _ := event.Data["invitation_token"].(string)

	subject := fmt.Sprintf("You're invited to join %s", tenantName)
	body := s.renderTemplate("team_invitation", map[string]interface{}{
		"InviterName": inviterName,
		"TenantName":  tenantName,
		"Role":        role,
		"URL":         fmt.Sprintf("%s/accept-invitation?token=%s", s.frontendURL, invitationToken),
	})

	// Add event_type to metadata
	metadata := event.Data
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event_type"] = event.EventType

	notification := &models.Notification{
		TenantID:  event.TenantID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  metadata,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return s.sendEmail(ctx, notification)
}

func (s *NotificationService) handleOrderInvoice(ctx context.Context, event models.NotificationEvent) error {
	email, _ := event.Data["customer_email"].(string)
	customerName, _ := event.Data["customer_name"].(string)
	orderReference, _ := event.Data["order_reference"].(string)
	deliveryType, _ := event.Data["delivery_type"].(string)
	createdAt, _ := event.Data["created_at"].(string)

	// Convert amounts from interface{} to numbers
	subtotalAmount := 0
	deliveryFee := 0
	totalAmount := 0

	if val, ok := event.Data["subtotal_amount"].(float64); ok {
		subtotalAmount = int(val)
	}
	if val, ok := event.Data["delivery_fee"].(float64); ok {
		deliveryFee = int(val)
	}
	if val, ok := event.Data["total_amount"].(float64); ok {
		totalAmount = int(val)
	}

	// Parse items
	type InvoiceItem struct {
		ProductName string
		Quantity    int
		UnitPrice   int
		TotalPrice  int
	}

	items := []InvoiceItem{}
	if itemsData, ok := event.Data["items"].([]interface{}); ok {
		for _, itemInterface := range itemsData {
			if itemMap, ok := itemInterface.(map[string]interface{}); ok {
				item := InvoiceItem{
					ProductName: itemMap["product_name"].(string),
				}
				if qty, ok := itemMap["quantity"].(float64); ok {
					item.Quantity = int(qty)
				}
				if price, ok := itemMap["unit_price"].(float64); ok {
					item.UnitPrice = int(price)
				}
				if total, ok := itemMap["total_price"].(float64); ok {
					item.TotalPrice = int(total)
				}
				items = append(items, item)
			}
		}
	}

	// Format currency helper
	formatIDR := func(amount int) string {
		return fmt.Sprintf("%s", formatCurrency(amount))
	}

	// Prepare template data
	templateData := map[string]interface{}{
		"OrderReference": orderReference,
		"CustomerName":   customerName,
		"CustomerEmail":  email,
		"DeliveryType":   deliveryType,
		"CreatedAt":      createdAt,
		"SubtotalAmount": formatIDR(subtotalAmount),
		"DeliveryFee":    formatIDR(deliveryFee),
		"TotalAmount":    formatIDR(totalAmount),
		"Items":          items,
		"OrderURL":       fmt.Sprintf("%s/orders/%s", s.frontendURL, orderReference),
	}

	// Render items with formatted prices
	formattedItems := make([]map[string]interface{}, len(items))
	for i, item := range items {
		formattedItems[i] = map[string]interface{}{
			"ProductName": item.ProductName,
			"Quantity":    item.Quantity,
			"UnitPrice":   formatIDR(item.UnitPrice),
			"TotalPrice":  formatIDR(item.TotalPrice),
		}
	}
	templateData["Items"] = formattedItems

	subject := fmt.Sprintf("Order Invoice - %s", orderReference)
	body := s.renderTemplate("order_invoice", templateData)

	// Add event_type to metadata
	metadata := event.Data
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event_type"] = event.EventType

	notification := &models.Notification{
		TenantID:  event.TenantID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  metadata,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return s.sendEmail(ctx, notification)
}

// formatCurrency formats an amount in IDR currency
func formatCurrency(amount int) string {
	// Simple formatting for Indonesian Rupiah
	if amount < 0 {
		return fmt.Sprintf("-%s", formatCurrency(-amount))
	}

	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		return str
	}

	// Add thousand separators
	var result string
	for i, c := range str {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}

	return result
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
	tmpl, ok := s.templates[templateName]
	if !ok {
		log.Printf("Template not found: %s", templateName)
		return fmt.Sprintf("Template '%s' not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("Template execution error for %s: %v", templateName, err)
		return fmt.Sprintf("Template execution error: %v", err)
	}

	return buf.String()
}
