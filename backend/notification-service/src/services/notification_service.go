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
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/pos/notification-service/src/models"
	"github.com/pos/notification-service/src/providers"
	"github.com/pos/notification-service/src/repository"
)

type NotificationService struct {
	repo                          repository.NotificationRepositoryInterface
	emailProvider                 providers.EmailProvider
	pushProvider                  providers.PushProvider
	templates                     map[string]*template.Template
	frontendURL                   string
	redisProvider                 *providers.RedisProvider
	db                            *sql.DB
	orderAggregationWindowSeconds int64
}

func NewNotificationService(db *sql.DB, redisProv *providers.RedisProvider) *NotificationService {
	service := &NotificationService{
		repo:          repository.NewNotificationRepository(db),
		emailProvider: providers.NewSMTPEmailProvider(),
		pushProvider:  providers.NewMockPushProvider(),
		templates:     make(map[string]*template.Template),
		frontendURL:   getEnv("FRONTEND_DOMAIN", "http://localhost:3000"),
		redisProvider: redisProv,
		db:            db,
	}

	// Read aggregation window from env, default to 5 seconds when not set or invalid
	if v := os.Getenv("ORDER_AGG_WINDOW_SECONDS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			service.orderAggregationWindowSeconds = n
		} else {
			service.orderAggregationWindowSeconds = 5
		}
	} else {
		service.orderAggregationWindowSeconds = 5
	}

	// Load all templates
	if err := service.loadTemplates(); err != nil {
		log.Printf("Warning: Failed to load templates: %v", err)
	}

	return service
}

// NewNotificationServiceForTest returns a NotificationService wired with the provided
// dependencies. It's intended for tests so callers can inject mock implementations.
func NewNotificationServiceForTest(repo repository.NotificationRepositoryInterface, emailProv providers.EmailProvider, pushProv providers.PushProvider, templates map[string]*template.Template, frontendURL string, redisProv *providers.RedisProvider) *NotificationService {
	svc := &NotificationService{
		repo:          repo,
		emailProvider: emailProv,
		pushProvider:  pushProv,
		templates:     templates,
		frontendURL:   frontendURL,
		redisProvider: redisProv,
	}
	return svc
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

	// Also unmarshal into a generic map to capture any top-level fields
	// (e.g. events that place payload fields at top-level instead of under `data`).
	var raw map[string]interface{}
	if err := json.Unmarshal(eventData, &raw); err == nil {
		// Remove known envelope keys
		delete(raw, "event_id")
		delete(raw, "event_type")
		delete(raw, "tenant_id")
		delete(raw, "user_id")
		delete(raw, "timestamp")

		// Merge payload into event.Data: prefer explicit `data` if present, otherwise
		// use remaining top-level keys or nested `payload` map.
		if event.Data == nil {
			event.Data = make(map[string]interface{})
		}

		if payloadRaw, ok := raw["payload"]; ok {
			if payloadMap, ok := payloadRaw.(map[string]interface{}); ok {
				for k, v := range payloadMap {
					event.Data[k] = v
				}
			}
			delete(raw, "payload")
		}

		// Merge any other leftover top-level fields into Data
		for k, v := range raw {
			event.Data[k] = v
		}
	}

	// Preserve the original tenant identifier (may be in form `tenant:slug`),
	// then normalize to a UUID where possible.
	originalTenant := event.TenantID
	normalizedTenant, err := s.resolveTenantID(ctx, event.TenantID)
	if err != nil {
		log.Printf("warning: failed to resolve tenant '%s': %v", event.TenantID, err)
		// fall back to the original value
	} else {
		event.TenantID = normalizedTenant
	}

	// Keep the original form available for Redis stream naming (tests expect the
	// stream key to include the original identifier like `tenant:demo`).
	if event.Data == nil {
		event.Data = make(map[string]interface{})
	}
	event.Data["__raw_tenant"] = originalTenant

	log.Printf("Processing event: %s for tenant: %s", event.EventType, event.TenantID)

	// Dedupe: for order-related events use event_records to prevent duplicate processing.
	if strings.HasPrefix(event.EventType, "order.") && event.EventID != "" {
		er := repository.NewEventRepository(s.db)
		exists, err := er.Exists(ctx, event.EventID)
		if err != nil {
			log.Printf("warning: failed to check event_records for %s: %v", event.EventID, err)
		}
		if exists {
			log.Printf("Skipping already-processed event: %s", event.EventID)
			return nil
		}
		// After successful processing we will insert the event record (see below per-case)
		defer func(e models.NotificationEvent) {
			// best-effort: mark event processed
			if e.EventID != "" {
				_ = er.Insert(context.Background(), e.EventID, e.EventType, e.TenantID, e.Data)
			}
		}(event)
	}

	var handlerErr error
	switch event.EventType {
	case "user.registered":
		handlerErr = s.handleUserRegistration(ctx, event)
	case "user.login":
		handlerErr = s.handleUserLogin(ctx, event)
	case "password.reset_requested":
		handlerErr = s.handlePasswordResetRequest(ctx, event)
	case "password.changed":
		handlerErr = s.handlePasswordChanged(ctx, event)
	case "invitation.created":
		handlerErr = s.handleTeamInvitation(ctx, event)
	case "order.invoice":
		handlerErr = s.handleOrderInvoice(ctx, event)
	case "order.paid":
		handlerErr = s.handleOrderPaid(ctx, event)
	case "order.status_updated":
		handlerErr = s.handleOrderStatusUpdated(ctx, event)
	default:
		log.Printf("Unknown event type: %s", event.EventType)
		handlerErr = nil
	}

	if handlerErr == nil {
		// Publish lightweight in-app update for order events
		s.publishInAppEvent(ctx, event)
	}

	return handlerErr
}

// resolveTenantID resolves tenant identifiers that may be supplied as
// `tenant:<slug>` by looking up the tenants table and returning the UUID.
// If the identifier already looks like a UUID or no mapping is found, the
// original identifier is returned without error.
func (s *NotificationService) resolveTenantID(ctx context.Context, tenantIdentifier string) (string, error) {
	if tenantIdentifier == "" {
		return tenantIdentifier, nil
	}

	// If it looks like our stream/key form tenant:<slug>, extract slug
	if strings.HasPrefix(tenantIdentifier, "tenant:") {
		slug := strings.TrimPrefix(tenantIdentifier, "tenant:")
		// Query tenants table for id by slug
		var id string
		err := s.db.QueryRowContext(ctx, "SELECT id FROM tenants WHERE slug = $1 LIMIT 1", slug).Scan(&id)
		if err != nil {
			if err == sql.ErrNoRows {
				// create a tenant record for this slug (development/test convenience)
				insertErr := s.db.QueryRowContext(ctx, "INSERT INTO tenants (slug, business_name) VALUES ($1, $2) RETURNING id", slug, slug).Scan(&id)
				if insertErr != nil {
					return tenantIdentifier, insertErr
				}
				// created tenant record for slug, useful during tests but not logged in normal runs
				return id, nil
			}
			return tenantIdentifier, err
		}
		return id, nil
	}

	// Otherwise assume it's already a proper tenant id (UUID) and return as-is
	return tenantIdentifier, nil
}

// publishInAppEvent publishes a concise event to the tenant Redis stream for SSE clients.
func (s *NotificationService) publishInAppEvent(ctx context.Context, event models.NotificationEvent) {
	if s.redisProvider == nil {
		return
	}
	// Accept multiple event naming conventions: 'order.paid', 'order_paid', etc.
	if !strings.HasPrefix(event.EventType, "order") {
		return
	}

	// For order.paid we use aggregation/coalescing to avoid rush notifications.
	if event.EventType == "order.paid" {
		s.aggregateOrderPaid(ctx, event)
		return
	}

	// Prefer original tenant identifier for stream naming if available
	streamTenant := event.TenantID
	if raw, ok := event.Data["__raw_tenant"].(string); ok && raw != "" {
		streamTenant = raw
	}
	stream := fmt.Sprintf("tenant:%s:stream", streamTenant)
	// Ensure `data` is a JSON string so consumers can unmarshal predictably
	// Wrap event.Data under a `data` key so downstream consumers (and tests)
	// that expect a JSON object with a `data` field can parse it directly.
	wrapper := map[string]interface{}{"data": event.Data}
	var wrapperJSON []byte
	if b, err := json.Marshal(wrapper); err == nil {
		wrapperJSON = b
	} else {
		log.Printf("Failed to marshal wrapper for redis publish: %v", err)
		wrapperJSON = []byte("{\"data\":{}}")
	}

	payload := map[string]interface{}{
		"id":        event.EventID,
		"event":     event.EventType,
		"data":      string(wrapperJSON),
		"timestamp": event.Timestamp.Format(time.RFC3339),
	}

	// Publish to tenant stream (broadcast)
	if _, err := s.redisProvider.PublishToStream(stream, payload); err != nil {
		log.Printf("Failed to publish in-app event to redis stream %s: %v", stream, err)
	}

	// If event targets a specific user, also publish to per-user stream
	var recipientUser string
	if event.UserID != "" {
		recipientUser = event.UserID
	} else if ru, ok := event.Data["recipient_user_id"].(string); ok && ru != "" {
		recipientUser = ru
	} else if ru2, ok := event.Data["user_id"].(string); ok && ru2 != "" {
		recipientUser = ru2
	}
	if recipientUser != "" {
		userStream := fmt.Sprintf("tenant:%s:user:%s:stream", streamTenant, recipientUser)
		if _, err := s.redisProvider.PublishToStream(userStream, payload); err != nil {
			log.Printf("Failed to publish in-app event to user stream %s: %v", userStream, err)
		}
	}
}

// aggregateOrderPaid coalesces rapid order.paid events for a single user into
// a single aggregated notification after a short window.
func (s *NotificationService) aggregateOrderPaid(ctx context.Context, event models.NotificationEvent) {
	if s.redisProvider == nil {
		// fallback to direct publish if no redis
		s.publishInAppEventNoAggregate(ctx, event)
		return
	}

	// Determine recipient user id
	recipientUser := event.UserID
	if recipientUser == "" {
		if ru, ok := event.Data["recipient_user_id"].(string); ok {
			recipientUser = ru
		} else if ru2, ok := event.Data["user_id"].(string); ok {
			recipientUser = ru2
		}
	}

	// If no specific user, publish to tenant stream as broadcast
	streamTenant := event.TenantID
	if raw, ok := event.Data["__raw_tenant"].(string); ok && raw != "" {
		streamTenant = raw
	}

	if recipientUser == "" {
		// no recipient, just publish as normal tenant in-app event
		s.publishInAppEventNoAggregate(ctx, event)
		return
	}

	// Use a redis counter key scoped to tenant+user
	counterKey := fmt.Sprintf("tenant:%s:user:%s:order_paid_count", streamTenant, recipientUser)

	// Increment with TTL equal to aggregation window
	cnt, err := s.redisProvider.IncrWithTTL(ctx, counterKey, s.orderAggregationWindowSeconds)
	if err != nil {
		log.Printf("Failed to increment aggregation counter: %v", err)
		// fallback to publishing directly to user stream
		s.publishInAppEventNoAggregate(ctx, event)
		return
	}

	// If this is the first event in the window, schedule a delayed aggregator
	if cnt == 1 {
		go func() {
			// wait aggregation window then read and publish aggregated notification
			time.Sleep(time.Duration(s.orderAggregationWindowSeconds) * time.Second)
			finalCount, gerr := s.redisProvider.GetInt(context.Background(), counterKey)
			if gerr != nil {
				log.Printf("Failed to read aggregation counter: %v", gerr)
				return
			}
			// delete the counter key
			_ = s.redisProvider.DelKey(context.Background(), counterKey)

			// build aggregated payload
			aggData := map[string]interface{}{}
			aggData["count"] = finalCount
			aggData["summary"] = fmt.Sprintf("You have %d paid order", finalCount)
			aggData["type"] = "order.paid.aggregate"

			wrapper := map[string]interface{}{"data": aggData}
			wrapperJSON, _ := json.Marshal(wrapper)

			payload := map[string]interface{}{
				"id":        event.EventID,
				"event":     "order.paid.aggregate",
				"data":      string(wrapperJSON),
				"timestamp": time.Now().Format(time.RFC3339),
			}

			userStream := fmt.Sprintf("tenant:%s:user:%s:stream", streamTenant, recipientUser)
			if _, perr := s.redisProvider.PublishToStream(userStream, payload); perr != nil {
				log.Printf("Failed to publish aggregated in-app event to user stream %s: %v", userStream, perr)
			}
		}()
	}
}

// publishInAppEventNoAggregate is used as a fallback to publish immediately
func (s *NotificationService) publishInAppEventNoAggregate(ctx context.Context, event models.NotificationEvent) {
	if s.redisProvider == nil {
		return
	}
	streamTenant := event.TenantID
	if raw, ok := event.Data["__raw_tenant"].(string); ok && raw != "" {
		streamTenant = raw
	}
	stream := fmt.Sprintf("tenant:%s:stream", streamTenant)
	wrapper := map[string]interface{}{"data": event.Data}
	var wrapperJSON []byte
	if b, err := json.Marshal(wrapper); err == nil {
		wrapperJSON = b
	} else {
		wrapperJSON = []byte("{\"data\":{}}")
	}
	payload := map[string]interface{}{
		"id":        event.EventID,
		"event":     event.EventType,
		"data":      string(wrapperJSON),
		"timestamp": event.Timestamp.Format(time.RFC3339),
	}
	if _, err := s.redisProvider.PublishToStream(stream, payload); err != nil {
		log.Printf("Failed to publish in-app event to redis stream %s: %v", stream, err)
	}
	// also publish to user stream if recipient user present
	var recipientUser string
	if event.UserID != "" {
		recipientUser = event.UserID
	} else if ru, ok := event.Data["recipient_user_id"].(string); ok {
		recipientUser = ru
	}
	if recipientUser != "" {
		userStream := fmt.Sprintf("tenant:%s:user:%s:stream", streamTenant, recipientUser)
		if _, err := s.redisProvider.PublishToStream(userStream, payload); err != nil {
			log.Printf("Failed to publish in-app event to user stream %s: %v", userStream, err)
		}
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

// handleOrderPaid creates notifications for paid orders: persist email + in-app, publish in-app event, and send email.
func (s *NotificationService) handleOrderPaid(ctx context.Context, event models.NotificationEvent) error {
	// Extract common fields
	orderID, _ := event.Data["order_id"].(string)
	reference, _ := event.Data["reference"].(string)
	total, _ := event.Data["total_amount"].(float64)
	customerEmail, _ := event.Data["customer_email"].(string)

	// Build in-app notification
	inAppSubject := "Order paid"
	inAppBody := fmt.Sprintf("Order %s has been paid (ref=%s, total=%v)", orderID, reference, total)
	inAppNotification := &models.Notification{
		TenantID:  event.TenantID,
		Type:      models.NotificationTypeInApp,
		Status:    models.NotificationStatusPending,
		Subject:   inAppSubject,
		Body:      inAppBody,
		Recipient: "",
		Metadata:  event.Data,
	}

	if err := s.repo.Create(ctx, inAppNotification); err != nil {
		log.Printf("failed to persist in-app notification: %v", err)
	}

	// Publish to Redis stream for SSE clients
	s.publishInAppEvent(ctx, event)

	// Create and send email notification if customer email present
	if customerEmail != "" {
		subject := fmt.Sprintf("Payment received: %s", reference)
		body := fmt.Sprintf("Hi, we received payment for order %s (ref=%s). Total: %v", orderID, reference, total)
		emailNotification := &models.Notification{
			TenantID:  event.TenantID,
			Type:      models.NotificationTypeEmail,
			Status:    models.NotificationStatusPending,
			Subject:   subject,
			Body:      body,
			Recipient: customerEmail,
			Metadata:  event.Data,
		}
		if err := s.repo.Create(ctx, emailNotification); err != nil {
			log.Printf("failed to persist email notification: %v", err)
		} else {
			if err := s.sendEmail(ctx, emailNotification); err != nil {
				log.Printf("sendEmail error: %v", err)
			}
		}
	}

	return nil
}

// handleOrderStatusUpdated handles generic order status changes and emits in-app notifications.
func (s *NotificationService) handleOrderStatusUpdated(ctx context.Context, event models.NotificationEvent) error {
	orderID, _ := event.Data["order_id"].(string)
	status, _ := event.Data["status"].(string)

	subject := "Order status updated"
	body := fmt.Sprintf("Order %s status changed to %s", orderID, status)

	notification := &models.Notification{
		TenantID:  event.TenantID,
		Type:      models.NotificationTypeInApp,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: "",
		Metadata:  event.Data,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		log.Printf("failed to persist status update notification: %v", err)
	}

	// publish to redis stream
	s.publishInAppEvent(ctx, event)

	return nil
}
