package events

import "time"

// NotificationEvent represents a Kafka message for notifications
type NotificationEvent struct {
EventID   string                 `json:"event_id"`
EventType string                 `json:"event_type"` // invitation_created, password_reset, etc
TenantID  string                 `json:"tenant_id"`
UserID    string                 `json:"user_id,omitempty"`
Data      map[string]interface{} `json:"data"`
Timestamp time.Time              `json:"timestamp"`
}
