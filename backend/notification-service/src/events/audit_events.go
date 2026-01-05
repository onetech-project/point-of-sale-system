package events

import (
	"time"

	"github.com/pos/notification-service/src/utils"
)

// NotificationEvent represents audit event for notification sending
type NotificationEvent struct {
	TenantID         string                 `json:"tenant_id"`
	ActorType        string                 `json:"actor_type"` // system (automated notifications)
	ActorID          *string                `json:"actor_id"`
	SessionID        *string                `json:"session_id"`
	Action           string                 `json:"action"`            // NOTIFICATION_SENT, NOTIFICATION_FAILED
	NotificationType string                 `json:"notification_type"` // email, sms, push
	RecipientID      *string                `json:"recipient_id"`      // User ID or guest order ID
	RecipientEmail   *string                `json:"recipient_email"`   // Encrypted
	Template         string                 `json:"template"`          // Template name
	Status           string                 `json:"status"`            // sent, failed, queued
	FailureReason    *string                `json:"failure_reason"`
	RequestID        *string                `json:"request_id"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ToAuditEvent converts NotificationEvent to generic AuditEvent
func (e *NotificationEvent) ToAuditEvent() *utils.AuditEvent {
	metadata := e.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["notification_type"] = e.NotificationType
	metadata["template"] = e.Template
	metadata["status"] = e.Status
	if e.FailureReason != nil {
		metadata["failure_reason"] = *e.FailureReason
	}

	resourceID := ""
	if e.RecipientID != nil {
		resourceID = *e.RecipientID
	}

	return &utils.AuditEvent{
		TenantID:     e.TenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    e.ActorType,
		ActorID:      e.ActorID,
		ActorEmail:   e.RecipientEmail,
		SessionID:    e.SessionID,
		Action:       e.Action,
		ResourceType: "notification",
		ResourceID:   resourceID,
		RequestID:    e.RequestID,
		Metadata:     metadata,
	}
}
