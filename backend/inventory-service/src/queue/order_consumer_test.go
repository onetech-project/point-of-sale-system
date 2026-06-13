package queue

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/segmentio/kafka-go"
)

type fakeOrderProcessor struct {
	processed bool
	event     models.OrderLifecycleEvent
}

func (p *fakeOrderProcessor) ProcessFulfilledOrder(_ context.Context, event models.OrderLifecycleEvent) error {
	p.processed = true
	p.event = event
	return nil
}

func TestOrderConsumerProcessesFulfilledEvent(t *testing.T) {
	tenantID := uuid.New()
	orderID := uuid.New()
	processor := &fakeOrderProcessor{}
	consumer := &OrderConsumer{processor: processor}

	err := consumer.processMessage(context.Background(), kafka.Message{
		Headers: []kafka.Header{{Key: "event-type", Value: []byte(models.OrderFulfilledEventType)}},
		Value: []byte(`{
			"event_id":"order.fulfilled:` + orderID.String() + `",
			"tenant_id":"` + tenantID.String() + `",
			"order_id":"` + orderID.String() + `",
			"items":[]
		}`),
	})
	if err != nil {
		t.Fatalf("processMessage error: %v", err)
	}
	if !processor.processed {
		t.Fatal("fulfilled event was not processed")
	}
	if processor.event.EventType != models.OrderFulfilledEventType {
		t.Fatalf("event type = %q, want %q", processor.event.EventType, models.OrderFulfilledEventType)
	}
}

func TestOrderConsumerSkipsNonFulfilledEvent(t *testing.T) {
	processor := &fakeOrderProcessor{}
	consumer := &OrderConsumer{processor: processor}

	err := consumer.processMessage(context.Background(), kafka.Message{
		Headers: []kafka.Header{{Key: "event-type", Value: []byte(models.OrderCancelledEventType)}},
		Value:   []byte(`{"event_type":"order.cancelled"}`),
	})
	if err != nil {
		t.Fatalf("processMessage error: %v", err)
	}
	if processor.processed {
		t.Fatal("non-fulfilled event should not be processed")
	}
}
