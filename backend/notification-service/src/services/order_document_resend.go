package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pos/notification-service/src/models"
	"github.com/pos/notification-service/src/utils"
)

func (s *NotificationService) handleOrderDocumentResend(ctx context.Context, event models.NotificationEvent) error {
	email, _ := event.Data["customer_email"].(string)
	if !utils.IsValidEmail(email) {
		return fmt.Errorf("valid customer_email is required for order document resend")
	}

	documentType, _ := event.Data["document_type"].(string)
	documentType = strings.ToLower(strings.TrimSpace(documentType))
	if documentType != "invoice" && documentType != "receipt" {
		return fmt.Errorf("document_type must be invoice or receipt")
	}

	orderReference, _ := event.Data["order_reference"].(string)
	customerName, _ := event.Data["customer_name"].(string)
	deliveryType, _ := event.Data["delivery_type"].(string)
	tableNumber, _ := event.Data["table_number"].(string)
	paymentMethod, _ := event.Data["payment_method"].(string)
	createdAt := formatDocumentTime(event.Data["created_at"])
	paidAt := formatDocumentTime(event.Data["paid_at"])

	subtotalAmount := toInt(event.Data["subtotal_amount"])
	deliveryFee := toInt(event.Data["delivery_fee"])
	totalAmount := toInt(event.Data["total_amount"])

	items := parseOrderDocumentItems(event.Data["items"])
	deliveryFeeStr := ""
	if deliveryFee > 0 {
		deliveryFeeStr = utils.FormatCurrencyIDR(deliveryFee)
	}

	templateData := map[string]interface{}{
		"OrderReference":    orderReference,
		"CustomerName":      customerName,
		"CustomerEmail":     email,
		"DeliveryType":      deliveryType,
		"TableNumber":       tableNumber,
		"SubtotalAmount":    utils.FormatCurrencyIDR(subtotalAmount),
		"DeliveryFee":       deliveryFeeStr,
		"TotalAmount":       utils.FormatCurrencyIDR(totalAmount),
		"Items":             items,
		"PaymentMethod":     paymentMethod,
		"PaidAt":            paidAt,
		"CreatedAt":         createdAt,
		"OrderURL":          fmt.Sprintf("%s/orders/%s", s.frontendURL, orderReference),
		"ShowPaidWatermark": documentType == "receipt",
	}

	subjectLabel := "Invoice"
	if documentType == "receipt" {
		subjectLabel = "Receipt"
	}
	subject := fmt.Sprintf("Order %s - %s", subjectLabel, orderReference)
	body := s.renderTemplate("order_invoice", templateData)

	metadata := event.Data
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event_type"] = event.EventType

	notification := &models.Notification{
		TenantID:  event.TenantID,
		UserID:    stringPtrOrNil(event.UserID),
		Type:      models.NotificationTypeEmail,
		Status:    models.NotificationStatusPending,
		Subject:   subject,
		Body:      body,
		Recipient: email,
		Metadata:  metadata,
	}

	if err := s.repo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create order document resend notification: %w", err)
	}

	log.Printf("[ORDER_DOCUMENT_RESEND] Sending %s for order %s to %s", documentType, orderReference, email)
	return s.sendEmail(ctx, notification)
}

func parseOrderDocumentItems(value interface{}) []map[string]interface{} {
	rawItems, ok := value.([]interface{})
	if !ok {
		return []map[string]interface{}{}
	}

	items := make([]map[string]interface{}, 0, len(rawItems))
	for _, rawItem := range rawItems {
		itemMap, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}
		productName, _ := itemMap["product_name"].(string)
		unitPrice := toInt(itemMap["unit_price"])
		totalPrice := toInt(itemMap["total_price"])
		items = append(items, map[string]interface{}{
			"ProductName": productName,
			"Quantity":    toInt(itemMap["quantity"]),
			"UnitPrice":   utils.FormatCurrencyIDR(unitPrice),
			"TotalPrice":  utils.FormatCurrencyIDR(totalPrice),
		})
	}

	return items
}

func formatDocumentTime(value interface{}) string {
	raw, _ := value.(string)
	if raw == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.Format("02 January 2006 15:04")
	}
	return raw
}

func toInt(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	case jsonNumber:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return 0
	}
}

type jsonNumber interface {
	Int64() (int64, error)
}

func stringPtrOrNil(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
