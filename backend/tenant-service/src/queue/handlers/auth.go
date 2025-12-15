package handlers

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"github.com/pos/tenant-service/src/config"
	"github.com/pos/tenant-service/src/queue"

	"github.com/pos/tenant-service/src/services"
)

type AuthConsumer struct {
	consumer      *queue.KafkaConsumer
	tenantService *services.TenantService
}

func NewAuthConsumer(tenantService *services.TenantService) *AuthConsumer {
	return &AuthConsumer{
		consumer:      queue.NewKafkaConsumer(config.KAFKA_BROKERS, "auth-service", []string{queue.TOPIC_OWNER_VERIFIED}),
		tenantService: tenantService,
	}
}

func (a *AuthConsumer) StartAuthConsumer(ctx context.Context) {
	a.consumer.Start(ctx, a.handler)
}

func (a *AuthConsumer) Stop() {
	a.consumer.Stop()
}

func (a *AuthConsumer) handler(ctx context.Context, msg kafka.Message) error {
	eventData := msg.Value
	switch msg.Topic {
	case queue.TOPIC_OWNER_VERIFIED:
		var event struct {
			TenantID string `json:"tenant_id"`
		}
		if err := json.Unmarshal(eventData, &event); err != nil {
			return err
		}
		return a.tenantService.HandleOwnerVerifiedEvent(ctx, event.TenantID)
	default:
		// Unknown topic
	}
	return nil
}
