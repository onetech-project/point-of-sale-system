package queue

import (
	"github.com/pos/notification-service/src/services"
)

// NewOrdersConsumer constructs a Kafka consumer configured for the `orders.events` topic
// and routes incoming messages to the provided NotificationService's HandleEvent method.
// This is a small integration wrapper (T008b) to give a clear place to add dedupe and routing logic.
func NewOrdersConsumer(brokers []string, groupID string, svc *services.NotificationService) *KafkaConsumer {
	topic := "orders.events"
	return NewKafkaConsumer(brokers, topic, groupID, svc.HandleEvent)
}
