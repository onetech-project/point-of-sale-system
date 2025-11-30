package models

import "time"

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail  NotificationType = "email"
	NotificationTypePush   NotificationType = "push"
	NotificationTypeInApp  NotificationType = "in_app"
	NotificationTypeSMS    NotificationType = "sms"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusQueued    NotificationStatus = "queued"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusRetrying  NotificationStatus = "retrying"
)

// Notification represents a notification record
type Notification struct {
	ID          string             `json:"id" db:"id"`
	TenantID    string             `json:"tenant_id" db:"tenant_id"`
	UserID      *string            `json:"user_id,omitempty" db:"user_id"`
	Type        NotificationType   `json:"type" db:"type"`
	Status      NotificationStatus `json:"status" db:"status"`
	Subject     string             `json:"subject,omitempty" db:"subject"`
	Body        string             `json:"body" db:"body"`
	Recipient   string             `json:"recipient" db:"recipient"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	SentAt      *time.Time         `json:"sent_at,omitempty" db:"sent_at"`
	FailedAt    *time.Time         `json:"failed_at,omitempty" db:"failed_at"`
	ErrorMsg    *string            `json:"error_msg,omitempty" db:"error_msg"`
	RetryCount  int                `json:"retry_count" db:"retry_count"`
	CreatedAt   time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" db:"updated_at"`
}

// SendNotificationRequest represents a request to send a notification
type SendNotificationRequest struct {
	Type      NotificationType       `json:"type" validate:"required"`
	Recipient string                 `json:"recipient" validate:"required"`
	Subject   string                 `json:"subject,omitempty"`
	Body      string                 `json:"body" validate:"required"`
	TemplateID string                `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	UserID    *string                `json:"user_id,omitempty"`
}

// NotificationEvent represents a Kafka message for notifications
type NotificationEvent struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"` // registration, login, password_reset, etc
	TenantID  string                 `json:"tenant_id"`
	UserID    string                 `json:"user_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// NotificationResponse represents the API response
type NotificationResponse struct {
	ID        string             `json:"id"`
	Type      NotificationType   `json:"type"`
	Status    NotificationStatus `json:"status"`
	Recipient string             `json:"recipient"`
	CreatedAt time.Time          `json:"created_at"`
}
