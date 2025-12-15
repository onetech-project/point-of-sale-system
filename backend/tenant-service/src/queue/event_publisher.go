package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventPublisher struct {
	kafka *KafkaProducer
}

func NewEventPublisher(brokers []string) *EventPublisher {
	kafkaProducer := NewKafkaProducer(brokers)
	return &EventPublisher{
		kafka: kafkaProducer,
	}
}

type TenantRegistrationSuccessEvent struct {
	TenantID     string    `json:"tenant_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Timestamp    time.Time `json:"timestamp"`
}

func (p *EventPublisher) PublishTenantRegistrationSuccess(ctx context.Context, event TenantRegistrationSuccessEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.kafka.Publish(
		ctx,
		TOPIC_TENANT_REGISTRATION_SUCCESS,
		uuid.New().String(),
		payload,
	)
}

func (p *EventPublisher) Close() {
	p.kafka.Close()
}
