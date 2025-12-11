package services

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"github.com/pos/notification-service/src/models"
	"github.com/pos/notification-service/src/utils"
)

// TemplateService handles email template rendering
type TemplateService struct {
	templates map[string]*template.Template
}

// NewTemplateService creates a new template service
func NewTemplateService() *TemplateService {
	return &TemplateService{
		templates: make(map[string]*template.Template),
	}
}

// LoadTemplate loads a specific template file
func (s *TemplateService) LoadTemplate(name string, path string) error {
	funcMap := utils.GetTemplateFuncMap()
	tmpl, err := template.New(name).Funcs(funcMap).ParseFiles(path)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}
	s.templates[name] = tmpl
	return nil
}

// renderStaffNotificationTemplate renders the staff notification email template
func (s *NotificationService) renderStaffNotificationTemplate(data *models.StaffNotificationData) (string, error) {
	tmpl, ok := s.templates["order_staff_notification"]
	if !ok {
		return "", fmt.Errorf("template not found: order_staff_notification")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return buf.String(), nil
}

// renderCustomerReceiptTemplate renders the customer receipt email template
func (s *NotificationService) renderCustomerReceiptTemplate(data *models.CustomerReceiptData) (string, error) {
	tmpl, ok := s.templates["order_invoice"]
	if !ok {
		return "", fmt.Errorf("template not found: order_invoice")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return buf.String(), nil
}

// convertOrderEventToStaffData converts OrderPaidEvent to StaffNotificationData
func convertOrderEventToStaffData(event *models.OrderPaidEvent) *models.StaffNotificationData {
	items := make([]models.StaffNotificationItem, len(event.Metadata.Items))
	for i, item := range event.Metadata.Items {
		items[i] = models.StaffNotificationItem{
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   utils.FormatCurrency(item.UnitPrice),
			TotalPrice:  utils.FormatCurrency(item.TotalPrice),
		}
	}

	deliveryFee := ""
	if event.Metadata.DeliveryFee > 0 {
		deliveryFee = utils.FormatCurrency(event.Metadata.DeliveryFee)
	}

	return &models.StaffNotificationData{
		OrderID:         event.Metadata.OrderID,
		OrderReference:  event.Metadata.OrderReference,
		TransactionID:   event.Metadata.TransactionID,
		CustomerName:    event.Metadata.CustomerName,
		CustomerEmail:   event.Metadata.CustomerEmail,
		CustomerPhone:   event.Metadata.CustomerPhone,
		DeliveryType:    event.Metadata.DeliveryType,
		DeliveryAddress: event.Metadata.DeliveryAddress,
		TableNumber:     event.Metadata.TableNumber,
		Items:           items,
		SubtotalAmount:  utils.FormatCurrency(event.Metadata.SubtotalAmount),
		DeliveryFee:     deliveryFee,
		TotalAmount:     utils.FormatCurrency(event.Metadata.TotalAmount),
		PaymentMethod:   event.Metadata.PaymentMethod,
		PaidAt:          event.Metadata.PaidAt.Format(time.RFC3339),
	}
}

// convertOrderEventToCustomerData converts OrderPaidEvent to CustomerReceiptData
func convertOrderEventToCustomerData(event *models.OrderPaidEvent) *models.CustomerReceiptData {
	items := make([]models.CustomerReceiptItem, len(event.Metadata.Items))
	for i, item := range event.Metadata.Items {
		items[i] = models.CustomerReceiptItem{
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   utils.FormatCurrency(item.UnitPrice),
			TotalPrice:  utils.FormatCurrency(item.TotalPrice),
		}
	}

	deliveryFee := ""
	if event.Metadata.DeliveryFee > 0 {
		deliveryFee = utils.FormatCurrency(event.Metadata.DeliveryFee)
	}

	return &models.CustomerReceiptData{
		OrderReference:    event.Metadata.OrderReference,
		CustomerName:      event.Metadata.CustomerName,
		CustomerEmail:     event.Metadata.CustomerEmail,
		DeliveryType:      event.Metadata.DeliveryType,
		DeliveryAddress:   event.Metadata.DeliveryAddress,
		TableNumber:       event.Metadata.TableNumber,
		Items:             items,
		SubtotalAmount:    utils.FormatCurrency(event.Metadata.SubtotalAmount),
		DeliveryFee:       deliveryFee,
		TotalAmount:       utils.FormatCurrency(event.Metadata.TotalAmount),
		PaymentMethod:     event.Metadata.PaymentMethod,
		PaidAt:            event.Metadata.PaidAt.Format("2006-01-02 15:04:05"),
		ShowPaidWatermark: true,
	}
}
