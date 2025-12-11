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
	"strings"
	"text/template"
	"time"

	"github.com/pos/notification-service/src/models"
	"github.com/pos/notification-service/src/providers"
	"github.com/pos/notification-service/src/repository"
	"github.com/pos/notification-service/src/utils"
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
		"order_staff_notification.html",
	}

	// Get custom template functions
	funcMap := utils.GetTemplateFuncMap()

	for _, filename := range templateFiles {
		templatePath := filepath.Join(templateDir, filename)
		tmpl, err := template.New(filename).Funcs(funcMap).ParseFiles(templatePath)
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
	case "order.paid":
		return s.handleOrderPaid(ctx, event)
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
		return formatCurrency(amount)
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

// handleOrderPaid processes order.paid events and sends notifications to staff
func (s *NotificationService) handleOrderPaid(ctx context.Context, event models.NotificationEvent) error {
	// Parse the OrderPaidEvent from the generic event data
	eventJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	var orderEvent models.OrderPaidEvent
	if err := json.Unmarshal(eventJSON, &orderEvent); err != nil {
		return fmt.Errorf("failed to unmarshal order.paid event: %w", err)
	}

	// Validate the event
	if err := models.ValidateOrderPaidEvent(&orderEvent); err != nil {
		return fmt.Errorf("invalid order.paid event: %w", err)
	}

	log.Printf("[ORDER_PAID] Processing event for order %s (transaction: %s, tenant: %s)",
		orderEvent.Metadata.OrderID, orderEvent.Metadata.TransactionID, orderEvent.TenantID)

	// Check for duplicate notifications
	alreadySent, err := s.repo.HasSentOrderNotification(ctx, orderEvent.TenantID, orderEvent.Metadata.TransactionID)
	if err != nil {
		log.Printf("[ORDER_PAID] Error checking duplicate: %v", err)
		return fmt.Errorf("failed to check duplicate notification: %w", err)
	}
	if alreadySent {
		// Log detailed duplicate detection for debugging and monitoring
		log.Printf("[DUPLICATE_NOTIFICATION] transaction_id=%s order_id=%s tenant_id=%s payment_method=%s amount=%d - Skipping duplicate notification",
			orderEvent.Metadata.TransactionID,
			orderEvent.Metadata.OrderID,
			orderEvent.TenantID,
			orderEvent.Metadata.PaymentMethod,
			orderEvent.Metadata.TotalAmount)

		// Track duplicate attempts metric
		s.trackMetric("notification.duplicate.prevented", 1, map[string]string{
			"tenant_id":      orderEvent.TenantID,
			"payment_method": orderEvent.Metadata.PaymentMethod,
		})
		return nil
	}

	// Send staff notifications
	if err := s.sendStaffNotifications(ctx, &orderEvent); err != nil {
		log.Printf("[ORDER_PAID] Failed to send staff notifications: %v", err)
		return fmt.Errorf("failed to send staff notifications: %w", err)
	}

	// Send customer receipt if email provided
	if orderEvent.Metadata.CustomerEmail != "" {
		if err := s.sendCustomerReceipt(ctx, &orderEvent); err != nil {
			log.Printf("[ORDER_PAID] Failed to send customer receipt: %v", err)
			// Don't fail the whole operation if customer receipt fails
		}
	}

	log.Printf("[ORDER_PAID] Successfully processed order.paid event for order %s", orderEvent.Metadata.OrderID)
	return nil
}

// queryStaffRecipients gets all staff users who should receive order notifications
func (s *NotificationService) queryStaffRecipients(ctx context.Context, tenantID string) ([]string, error) {
	// This would normally query the user-service API or database
	// For now, we'll use a placeholder implementation
	// In production, this should call user-service's FindStaffWithOrderNotifications endpoint

	// TODO: Implement actual API call to user-service
	// For now, return empty list - this will be implemented when connecting services
	log.Printf("[ORDER_PAID] Querying staff recipients for tenant %s", tenantID)

	// Placeholder: In production, call user-service API
	// GET /api/v1/users/staff-with-notifications?tenant_id={tenantID}

	return []string{}, nil
}

// sendStaffNotifications sends order notification emails to all configured staff members
func (s *NotificationService) sendStaffNotifications(ctx context.Context, orderEvent *models.OrderPaidEvent) error {
	// Query staff recipients
	staffEmails, err := s.queryStaffRecipients(ctx, orderEvent.TenantID)
	if err != nil {
		return fmt.Errorf("failed to query staff recipients: %w", err)
	}

	if len(staffEmails) == 0 {
		log.Printf("[ORDER_PAID] No staff members configured to receive notifications for tenant %s",
			orderEvent.TenantID)
		return nil
	}

	// Convert event to template data
	staffData := convertOrderEventToStaffData(orderEvent)

	// Render template
	body, err := s.renderStaffNotificationTemplate(staffData)
	if err != nil {
		return fmt.Errorf("failed to render staff notification template: %w", err)
	}

	subject := fmt.Sprintf("New Order Paid - %s", orderEvent.Metadata.OrderReference)

	// Send notification to each staff member
	successCount := 0
	for _, email := range staffEmails {
		log.Printf("[ORDER_PAID] Sending notification to staff: %s", email)

		// Create notification metadata
		metadata := map[string]interface{}{
			"event_type":     "order.paid.staff",
			"order_id":       orderEvent.Metadata.OrderID,
			"transaction_id": orderEvent.Metadata.TransactionID,
			"customer_name":  orderEvent.Metadata.CustomerName,
			"total_amount":   orderEvent.Metadata.TotalAmount,
			"payment_method": orderEvent.Metadata.PaymentMethod,
		}

		notification := &models.Notification{
			TenantID:  orderEvent.TenantID,
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Subject:   subject,
			Body:      body,
			Recipient: email,
			Metadata:  metadata,
		}

		if err := s.repo.Create(ctx, notification); err != nil {
			log.Printf("[ORDER_PAID] Failed to create notification record for %s: %v", email, err)
			continue
		}

		if err := s.sendEmail(ctx, notification); err != nil {
			log.Printf("[ORDER_PAID] Failed to send email to %s: %v", email, err)
			continue
		}

		successCount++
	}

	log.Printf("[ORDER_PAID] Successfully sent %d/%d staff notifications", successCount, len(staffEmails))
	return nil
}

// sendCustomerReceipt sends email receipt to customer
func (s *NotificationService) sendCustomerReceipt(ctx context.Context, orderEvent *models.OrderPaidEvent) error {
	// Validate email format
	if !utils.IsValidEmail(orderEvent.Metadata.CustomerEmail) {
		log.Printf("[ORDER_PAID] Invalid email format for customer receipt: %s", orderEvent.Metadata.CustomerEmail)
		return fmt.Errorf("invalid email format: %s", orderEvent.Metadata.CustomerEmail)
	}

	log.Printf("[ORDER_PAID] Sending customer receipt to %s", orderEvent.Metadata.CustomerEmail)

	// Convert event to template data
	customerData := convertOrderEventToCustomerData(orderEvent)

	// Render template
	body, err := s.renderCustomerReceiptTemplate(customerData)
	if err != nil {
		return fmt.Errorf("failed to render customer receipt template: %w", err)
	}

	subject := fmt.Sprintf("Order Receipt - %s", orderEvent.Metadata.OrderReference)

	// Create notification metadata
	metadata := map[string]interface{}{
		"event_type":     "order.paid.customer",
		"order_id":       orderEvent.Metadata.OrderID,
		"transaction_id": orderEvent.Metadata.TransactionID,
		"customer_email": orderEvent.Metadata.CustomerEmail,
		"total_amount":   orderEvent.Metadata.TotalAmount,
	}

	notification := &models.Notification{
		TenantID:  orderEvent.TenantID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: orderEvent.Metadata.CustomerEmail,
		Metadata:  metadata,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification record: %w", err)
	}

	if err := s.sendEmail(ctx, notification); err != nil {
		return fmt.Errorf("failed to send customer receipt: %w", err)
	}

	log.Printf("[ORDER_PAID] Successfully sent customer receipt to %s", orderEvent.Metadata.CustomerEmail)
	return nil
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
	startTime := time.Now()
	err := s.emailProvider.Send(notification.Recipient, notification.Subject, notification.Body, true)
	duration := time.Since(startTime)

	now := time.Now()
	if err != nil {
		// Extract error details if it's an EmailError
		errorMsg := err.Error()
		errorType := "unknown"
		isRetryable := false

		if emailErr, ok := err.(*providers.EmailError); ok {
			errorType = s.getErrorTypeName(emailErr.Type)
			isRetryable = emailErr.IsRetryable()
		}

		notification.Status = models.NotificationStatusFailed
		notification.FailedAt = &now
		notification.ErrorMsg = &errorMsg
		notification.RetryCount++

		// Log detailed error with metrics
		log.Printf("[EMAIL_SEND_FAILED] ID=%d Type=%s Retryable=%v RetryCount=%d Duration=%s Error=%v",
			notification.ID, errorType, isRetryable, notification.RetryCount, duration, err)

		// Update metrics
		s.trackMetric("notification.email.failed", 1, map[string]string{
			"error_type": errorType,
			"retryable":  fmt.Sprintf("%v", isRetryable),
		})
	} else {
		notification.Status = models.NotificationStatusSent
		notification.SentAt = &now

		// Log success with metrics
		log.Printf("[EMAIL_SEND_SUCCESS] ID=%d Duration=%s RetryCount=%d",
			notification.ID, duration, notification.RetryCount)

		// Update metrics
		s.trackMetric("notification.email.sent", 1, map[string]string{
			"retry_count": fmt.Sprintf("%d", notification.RetryCount),
		})
		s.trackMetric("notification.email.duration_ms", duration.Milliseconds(), nil)
	}

	if updateErr := s.repo.UpdateStatus(ctx, notification.ID, notification.Status, notification.SentAt, notification.FailedAt, notification.ErrorMsg); updateErr != nil {
		log.Printf("Failed to update notification status: %v", updateErr)
	}

	return err
}

func (s *NotificationService) getErrorTypeName(errorType providers.EmailErrorType) string {
	switch errorType {
	case providers.EmailErrorTypeConnection:
		return "connection"
	case providers.EmailErrorTypeAuth:
		return "auth"
	case providers.EmailErrorTypeTimeout:
		return "timeout"
	case providers.EmailErrorTypeInvalidRecipient:
		return "invalid_recipient"
	case providers.EmailErrorTypeRateLimited:
		return "rate_limited"
	default:
		return "unknown"
	}
}

// trackMetric tracks notification metrics (placeholder for actual metrics system)
// In production, this would integrate with Prometheus, StatsD, or similar
func (s *NotificationService) trackMetric(name string, value int64, tags map[string]string) {
	tagStr := ""
	if tags != nil && len(tags) > 0 {
		tagPairs := []string{}
		for k, v := range tags {
			tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", k, v))
		}
		tagStr = fmt.Sprintf(" [%s]", strings.Join(tagPairs, ", "))
	}
	log.Printf("[METRIC] %s=%d%s", name, value, tagStr)
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

// SendTestNotification sends a test notification email with sample data
func (s *NotificationService) SendTestNotification(tenantID, recipientEmail, notificationType string) (string, error) {
	ctx := context.Background()

	// Validate email format
	if !utils.IsValidEmail(recipientEmail) {
		return "", fmt.Errorf("invalid email format")
	}

	var subject string
	var body string
	var err error

	switch notificationType {
	case "staff_order_notification":
		subject = "Test: New Order Notification"

		// Create sample staff notification data
		testData := &models.StaffNotificationData{
			OrderID:         "TEST-ORDER-" + time.Now().Format("20060102-150405"),
			OrderReference:  "ORD-TEST-001",
			TransactionID:   "TXN-TEST-" + time.Now().Format("150405"),
			CustomerName:    "Test Customer",
			CustomerEmail:   "test.customer@example.com",
			CustomerPhone:   "+6281234567890",
			DeliveryType:    "delivery",
			DeliveryAddress: "Jl. Sudirman No. 123, Jakarta Pusat",
			SubtotalAmount:  "150.000",
			DeliveryFee:     "15.000",
			TotalAmount:     "165.000",
			PaymentMethod:   "qris",
			PaidAt:          time.Now().Format("2006-01-02 15:04:05"),
			Items: []models.StaffNotificationItem{
				{
					ProductName: "Nasi Goreng Special",
					Quantity:    2,
					UnitPrice:   "50.000",
					TotalPrice:  "100.000",
				},
				{
					ProductName: "Es Teh Manis",
					Quantity:    2,
					UnitPrice:   "10.000",
					TotalPrice:  "20.000",
				},
				{
					ProductName: "Kerupuk",
					Quantity:    3,
					UnitPrice:   "10.000",
					TotalPrice:  "30.000",
				},
			},
		}

		body, err = s.renderStaffNotificationTemplate(testData)
		if err != nil {
			return "", fmt.Errorf("failed to render staff notification template: %w", err)
		}

	case "customer_receipt":
		subject = "Test: Order Receipt"

		// Create sample customer receipt data
		testData := &models.CustomerReceiptData{
			OrderReference:    "ORD-TEST-002",
			CustomerName:      "Test Customer",
			CustomerEmail:     recipientEmail,
			DeliveryType:      "delivery",
			DeliveryAddress:   "Jl. Sudirman No. 123, Jakarta Pusat, DKI Jakarta",
			SubtotalAmount:    "200.000",
			DeliveryFee:       "20.000",
			TotalAmount:       "220.000",
			PaymentMethod:     "qris",
			PaidAt:            time.Now().Format("2006-01-02 15:04:05"),
			ShowPaidWatermark: true,
			Items: []models.CustomerReceiptItem{
				{
					ProductName: "Burger Beef Special",
					Quantity:    2,
					UnitPrice:   "75.000",
					TotalPrice:  "150.000",
				},
				{
					ProductName: "French Fries Large",
					Quantity:    2,
					UnitPrice:   "25.000",
					TotalPrice:  "50.000",
				},
			},
		}

		body, err = s.renderCustomerReceiptTemplate(testData)
		if err != nil {
			return "", fmt.Errorf("failed to render customer receipt template: %w", err)
		}

	default:
		return "", fmt.Errorf("unsupported notification type: %s", notificationType)
	}

	// Create notification record
	metadata := map[string]interface{}{
		"is_test":           true,
		"notification_type": notificationType,
		"sent_at":           time.Now().Format(time.RFC3339),
	}

	notification := &models.Notification{
		TenantID:  tenantID,
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: recipientEmail,
		Metadata:  metadata,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return "", fmt.Errorf("failed to create notification record: %w", err)
	}

	// Send email
	if err := s.sendEmail(ctx, notification); err != nil {
		return notification.ID, fmt.Errorf("failed to send test email: %w", err)
	}

	log.Printf("Test notification sent successfully: type=%s, recipient=%s, notification_id=%s", notificationType, recipientEmail, notification.ID)

	return notification.ID, nil
}

// GetNotificationHistory retrieves notification history with filters and pagination
func (s *NotificationService) GetNotificationHistory(tenantID string, filters map[string]interface{}) (map[string]interface{}, error) {
	// Extract pagination parameters
	page := filters["page"].(int)
	pageSize := filters["page_size"].(int)
	offset := (page - 1) * pageSize

	// Build query filters
	queryFilters := make(map[string]interface{})
	queryFilters["tenant_id"] = tenantID
	queryFilters["limit"] = pageSize
	queryFilters["offset"] = offset

	// Add optional filters
	if orderRef, ok := filters["order_reference"]; ok {
		queryFilters["order_reference"] = orderRef
	}
	if status, ok := filters["status"]; ok {
		queryFilters["status"] = status
	}
	if notifType, ok := filters["type"]; ok {
		queryFilters["type"] = notifType
	}
	if startDate, ok := filters["start_date"]; ok {
		queryFilters["start_date"] = startDate
	}
	if endDate, ok := filters["end_date"]; ok {
		queryFilters["end_date"] = endDate
	}

	// Get notifications from repository
	notifications, err := s.repo.GetNotificationHistory(queryFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification history: %w", err)
	}

	// Get total count for pagination
	totalCount, err := s.repo.CountNotifications(queryFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Calculate total pages
	totalPages := (totalCount + pageSize - 1) / pageSize

	// Build response
	result := map[string]interface{}{
		"notifications": notifications,
		"pagination": map[string]interface{}{
			"current_page": page,
			"page_size":    pageSize,
			"total_items":  totalCount,
			"total_pages":  totalPages,
		},
	}

	return result, nil
}

// ResendNotification resends a failed notification
func (s *NotificationService) ResendNotification(tenantID, notificationID string) (map[string]interface{}, error) {
	// Get notification by ID
	notification, err := s.repo.GetByID(notificationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	// Verify tenant ownership
	if notification.TenantID != tenantID {
		return nil, fmt.Errorf("forbidden")
	}

	// Check if notification already sent
	if notification.Status == "sent" {
		return nil, fmt.Errorf("already sent")
	}

	// Check max retries
	maxRetries := 3
	if notification.RetryCount >= maxRetries {
		return map[string]interface{}{
			"retry_count": notification.RetryCount,
			"max_retries": maxRetries,
		}, fmt.Errorf("max retries exceeded")
	}

	// Increment retry count
	notification.RetryCount++

	// Update notification status to pending
	notification.Status = "pending"
	if err := s.repo.Update(notification); err != nil {
		return nil, fmt.Errorf("failed to update notification: %w", err)
	}

	// Attempt to resend
	ctx := context.Background()
	if err := s.sendEmail(ctx, notification); err != nil {
		// Mark as failed
		notification.Status = "failed"
		errorMsg := err.Error()
		notification.ErrorMsg = &errorMsg
		s.repo.Update(notification)
		return nil, fmt.Errorf("failed to resend notification: %w", err)
	}

	// Mark as sent
	notification.Status = "sent"
	now := time.Now()
	notification.SentAt = &now
	if err := s.repo.Update(notification); err != nil {
		log.Printf("Warning: Failed to update notification status after sending: %v", err)
	}

	log.Printf("Notification resent successfully: notification_id=%s, retry_count=%d", notificationID, notification.RetryCount)

	// Build response
	result := map[string]interface{}{
		"notification_id": notificationID,
		"status":          "sent",
		"sent_at":         notification.SentAt.Format(time.RFC3339),
		"retry_count":     notification.RetryCount,
		"message":         "Notification resent successfully",
	}

	return result, nil
}
