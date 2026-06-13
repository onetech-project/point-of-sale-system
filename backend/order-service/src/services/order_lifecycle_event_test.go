package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
)

func TestLifecycleOutboxEventPayloadShape(t *testing.T) {
	payload, err := json.Marshal(map[string]interface{}{
		"event_id":        "order.fulfilled:order-1",
		"event_type":      "order.fulfilled",
		"tenant_id":       "tenant-1",
		"order_id":        "order-1",
		"order_reference": "ORD-001",
		"order_type":      models.OrderTypeOnline,
		"old_status":      models.OrderStatusPaid,
		"new_status":      models.OrderStatusComplete,
		"occurred_at":     time.Date(2026, 6, 11, 10, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		"items": []map[string]interface{}{
			{
				"order_item_id": "item-1",
				"product_id":    "product-1",
				"product_name":  "Nasi Goreng",
				"quantity":      2,
				"unit_price":    25000,
				"total_price":   50000,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded["event_type"] != "order.fulfilled" {
		t.Fatalf("unexpected event type: %v", decoded["event_type"])
	}
	if decoded["event_id"] != "order.fulfilled:order-1" {
		t.Fatalf("unexpected event id: %v", decoded["event_id"])
	}
	items, ok := decoded["items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("expected one item in payload")
	}
}
