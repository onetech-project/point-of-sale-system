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

type mockRedis struct {
	published  bool
	lastStream string
}

func (m *mockRedis) PublishToStream(streamName string, fieldValues map[string]interface{}) (string, error) {
	m.published = true
	m.lastStream = streamName
	return "1-0", nil
}

func TestOrderPaidProcessing(t *testing.T) {
	ev := models.NotificationEvent{
		EventType: "order.paid",
		TenantID:  "tenant-xyz",
		UserID:    "user-xyz",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"order_id":       "ord-1",
			"reference":      "REF-1",
			"total_amount":   100000.0,
			"customer_email": "buyer@example.com",
		},
	}

	payload, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	mr := &mockRepo{}
	me := &mockEmail{}

	// Create test service with injected mocks
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
	// Since redisProvider wasn't set on svc, publishInAppEvent should be a no-op
}
