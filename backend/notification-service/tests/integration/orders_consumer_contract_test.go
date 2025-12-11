package integration

import (
	"context"
	"encoding/json"
	"testing"
	"text/template"
	"time"

	"github.com/pos/notification-service/src/models"
	servicespkg "github.com/pos/notification-service/src/services"
)

type mockRepo struct {
	created bool
}

func (m *mockRepo) Create(ctx context.Context, n *models.Notification) error {
	m.created = true
	n.ID = "test-id"
	n.CreatedAt = time.Now()
	return nil
}

func (m *mockRepo) UpdateStatus(ctx context.Context, id string, status models.NotificationStatus, sentAt, failedAt *time.Time, errorMsg *string) error {
	return nil
}

func (m *mockRepo) FindByID(ctx context.Context, id string) (*models.Notification, error) {
	return nil, nil
}

type mockEmail struct {
	sent bool
}

func (m *mockEmail) Send(to, subject, body string, isHTML bool) error {
	m.sent = true
	return nil
}

// TestOrdersConsumerContract performs a unit-level contract: given a minimal order event,
// the NotificationService should create a Notification record and attempt to send an email.
func TestOrdersConsumerContract(t *testing.T) {
	// Build a minimal order invoice event matching existing handler
	ev := models.NotificationEvent{
		EventType: "order.invoice",
		TenantID:  "tenant-123",
		UserID:    "user-123",
		Data: map[string]interface{}{
			"customer_email":  "buyer@example.com",
			"customer_name":   "Buyer",
			"order_reference": "ORD-1001",
			"total_amount":    50000,
		},
	}

	payload, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	mr := &mockRepo{}
	me := &mockEmail{}

	// Create a service instance for tests with injected mocks
	svc := servicespkg.NewNotificationServiceForTest(mr, me, nil, make(map[string]*template.Template), "http://localhost:3000", nil)

	// Call handler
	if err := svc.HandleEvent(context.Background(), payload); err != nil {
		t.Fatalf("HandleEvent returned error: %v", err)
	}

	if !mr.created {
		t.Fatalf("expected NotificationRepository.Create to be called")
	}
	if !me.sent {
		t.Fatalf("expected EmailProvider.Send to be called")
	}
}
