package services

import (
	"bytes"
	"fmt"
	"text/template"

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
	items := make([]models.StaffNotificationItem, len(event.Data.Items))
	for i, item := range event.Data.Items {
		items[i] = models.StaffNotificationItem{
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   utils.FormatCurrency(item.UnitPrice),
			TotalPrice:  utils.FormatCurrency(item.TotalPrice),
		}
	}

	deliveryFee := ""
	if event.Data.DeliveryFee > 0 {
		deliveryFee = utils.FormatCurrency(event.Data.DeliveryFee)
	}

	return &models.StaffNotificationData{
		OrderID:         event.Data.OrderID,
		OrderReference:  event.Data.OrderReference,
		TransactionID:   event.Data.TransactionID,
		CustomerName:    event.Data.CustomerName,
		CustomerEmail:   event.Data.CustomerEmail,
		CustomerPhone:   event.Data.CustomerPhone,
		DeliveryType:    event.Data.DeliveryType,
		DeliveryAddress: event.Data.DeliveryAddress,
		TableNumber:     event.Data.TableNumber,
		Items:           items,
		SubtotalAmount:  utils.FormatCurrency(event.Data.SubtotalAmount),
		DeliveryFee:     deliveryFee,
		TotalAmount:     utils.FormatCurrency(event.Data.TotalAmount),
		PaymentMethod:   event.Data.PaymentMethod,
		PaidAt:          event.Data.PaidAt.Format("02 January 2006 15:04"),
		CreatedAt:       event.Data.CreatedAt.Format("02 January 2006 15:04"),
	}
}

// convertOrderEventToCustomerData converts OrderPaidEvent to CustomerReceiptData
func convertOrderEventToCustomerData(event *models.OrderPaidEvent, frontendURL string) *models.CustomerReceiptData {
	items := make([]models.CustomerReceiptItem, len(event.Data.Items))
	for i, item := range event.Data.Items {
		items[i] = models.CustomerReceiptItem{
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   utils.FormatCurrency(item.UnitPrice),
			TotalPrice:  utils.FormatCurrency(item.TotalPrice),
		}
	}

	deliveryFee := ""
	if event.Data.DeliveryFee > 0 {
		deliveryFee = utils.FormatCurrency(event.Data.DeliveryFee)
	}

	return &models.CustomerReceiptData{
		OrderReference:    event.Data.OrderReference,
		CustomerName:      event.Data.CustomerName,
		CustomerEmail:     event.Data.CustomerEmail,
		DeliveryType:      event.Data.DeliveryType,
		DeliveryAddress:   event.Data.DeliveryAddress,
		TableNumber:       event.Data.TableNumber,
		Items:             items,
		SubtotalAmount:    utils.FormatCurrency(event.Data.SubtotalAmount),
		DeliveryFee:       deliveryFee,
		TotalAmount:       utils.FormatCurrency(event.Data.TotalAmount),
		PaymentMethod:     event.Data.PaymentMethod,
		PaidAt:            event.Data.PaidAt.Format("02 January 2006 15:04"),
		CreatedAt:         event.Data.CreatedAt.Format("02 January 2006 15:04"),
		OrderURL:          fmt.Sprintf("%s/orders/%s", frontendURL, event.Data.OrderReference),
		ShowPaidWatermark: true,
	}
}
