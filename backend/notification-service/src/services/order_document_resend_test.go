package services

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/pos/notification-service/src/models"
)

type orderDocumentResendRepo struct {
	notification *models.Notification
}

func (r *orderDocumentResendRepo) Create(ctx context.Context, notification *models.Notification) error {
	notification.ID = "notification-1"
	r.notification = notification
	return nil
}

func (r *orderDocumentResendRepo) HasSentOrderNotification(ctx context.Context, tenantID, transactionID string) (bool, error) {
	return false, nil
}

func (r *orderDocumentResendRepo) UpdateStatus(ctx context.Context, id string, status models.NotificationStatus, sentAt, failedAt *time.Time, errorMsg *string) error {
	r.notification.Status = status
	r.notification.SentAt = sentAt
	r.notification.FailedAt = failedAt
	r.notification.ErrorMsg = errorMsg
	return nil
}

func (r *orderDocumentResendRepo) GetNotificationHistory(filters map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}

func (r *orderDocumentResendRepo) CountNotifications(filters map[string]interface{}) (int, error) {
	return 0, nil
}

func (r *orderDocumentResendRepo) GetByID(id string) (*models.Notification, error) {
	return nil, sql.ErrNoRows
}

func (r *orderDocumentResendRepo) Update(notification *models.Notification) error {
	r.notification = notification
	return nil
}

type orderDocumentResendEmailProvider struct {
	to      string
	subject string
	body    string
}

func (p *orderDocumentResendEmailProvider) Send(to, subject, body string, isHTML bool) error {
	p.to = to
	p.subject = subject
	p.body = body
	return nil
}

func TestHandleOrderDocumentResend(t *testing.T) {
	repo := &orderDocumentResendRepo{}
	emailProvider := &orderDocumentResendEmailProvider{}
	service := &NotificationService{
		repo:          repo,
		emailProvider: emailProvider,
		templates: map[string]*template.Template{
			"order_invoice": template.Must(template.New("order_invoice").Parse(`{{.OrderReference}} {{.CustomerEmail}} {{if .ShowPaidWatermark}}PAID{{else}}INVOICE{{end}} {{range .Items}}{{.ProductName}} {{end}}`)),
		},
		frontendURL: "http://localhost:3000",
	}

	event := models.NotificationEvent{
		EventID:   "event-1",
		EventType: "order.document.resend",
		TenantID:  "tenant-1",
		Data: map[string]interface{}{
			"document_type":   "receipt",
			"order_id":        "order-1",
			"order_reference": "GO-0001",
			"customer_name":   "Customer",
			"customer_email":  "customer@example.com",
			"delivery_type":   "pickup",
			"subtotal_amount": float64(30000),
			"total_amount":    float64(30000),
			"created_at":      "2026-05-19T10:00:00Z",
			"paid_at":         "2026-05-19T10:05:00Z",
			"items": []interface{}{
				map[string]interface{}{
					"product_name": "Coffee",
					"quantity":     float64(2),
					"unit_price":   float64(15000),
					"total_price":  float64(30000),
				},
			},
		},
	}

	if err := service.handleOrderDocumentResend(context.Background(), event); err != nil {
		t.Fatalf("handleOrderDocumentResend() error = %v", err)
	}

	if repo.notification == nil {
		t.Fatal("expected notification record to be created")
	}
	if repo.notification.Status != models.NotificationStatusSent {
		t.Fatalf("status = %s, want sent", repo.notification.Status)
	}
	if emailProvider.to != "customer@example.com" {
		t.Fatalf("email sent to %q, want customer@example.com", emailProvider.to)
	}
	if !strings.Contains(emailProvider.body, "PAID") || !strings.Contains(emailProvider.body, "Coffee") {
		t.Fatalf("rendered email body missing receipt content: %s", emailProvider.body)
	}
	if repo.notification.Metadata["event_type"] != "order.document.resend" {
		t.Fatalf("metadata event_type = %v", repo.notification.Metadata["event_type"])
	}
}
