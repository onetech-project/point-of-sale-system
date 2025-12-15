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

type LoginSuccessEvent struct {
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Timestamp time.Time `json:"timestamp"`
}

type RegistrationSuccessEvent struct {
	TenantID          string    `json:"tenant_id"`
	UserID            string    `json:"user_id"`
	Email             string    `json:"email"`
	Name              string    `json:"name"`
	VerificationToken string    `json:"verification_token"`
	Timestamp         time.Time `json:"timestamp"`
}

type LogoutSuccessEvent struct {
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

type ResetPasswordRequestedEvent struct {
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	ResetToken string    `json:"reset_token"`
	Timestamp  time.Time `json:"timestamp"`
}

type PasswordChangedEvent struct {
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

func (p *EventPublisher) PublishUserLoginSuccess(ctx context.Context, event LoginSuccessEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.kafka.Publish(
		ctx,
		TOPIC_LOGIN_SUCCESS,
		uuid.New().String(),
		payload,
	)
}

func (p *EventPublisher) PublishUserRegistrationSuccess(ctx context.Context, event RegistrationSuccessEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.kafka.Publish(
		ctx,
		TOPIC_REGISTRATION_SUCCESS,
		uuid.New().String(),
		payload,
	)
}

func (p *EventPublisher) PublishUserLogoutSuccess(ctx context.Context, event LogoutSuccessEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.kafka.Publish(
		ctx,
		TOPIC_LOGOUT_SUCCESS,
		uuid.New().String(),
		payload,
	)
}

func (p *EventPublisher) PublishResetPasswordRequested(ctx context.Context, event ResetPasswordRequestedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.kafka.Publish(
		ctx,
		TOPIC_RESET_PASSWORD_REQUESTED,
		uuid.New().String(),
		payload,
	)
}

func (p *EventPublisher) PublishPasswordChanged(ctx context.Context, event PasswordChangedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.kafka.Publish(
		ctx,
		TOPIC_PASSWORD_CHANGED,
		uuid.New().String(),
		payload,
	)
}

func (p *EventPublisher) Close() {
	p.kafka.Close()
}
