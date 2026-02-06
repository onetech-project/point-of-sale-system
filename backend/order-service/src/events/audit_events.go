package events

import (
	"time"

	"github.com/point-of-sale-system/order-service/src/utils"
)

// OrderEvent represents audit event for order operations
type OrderEvent struct {
	TenantID     string                 `json:"tenant_id"`
	ActorType    string                 `json:"actor_type"` // user, guest, system
	ActorID      *string                `json:"actor_id"`
	ActorEmail   *string                `json:"actor_email"` // Encrypted
	SessionID    *string                `json:"session_id"`
	Action       string                 `json:"action"` // CREATE, READ, UPDATE, DELETE
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	OrderStatus  *string                `json:"order_status"`
	TotalAmount  *float64               `json:"total_amount"`
	IPAddress    *string                `json:"ip_address"`
	UserAgent    *string                `json:"user_agent"`
	RequestID    *string                `json:"request_id"`
	BeforeValue  map[string]interface{} `json:"before_value"` // Encrypted PII
	AfterValue   map[string]interface{} `json:"after_value"`  // Encrypted PII
	Metadata     map[string]interface{} `json:"metadata"`
}

// ToAuditEvent converts OrderEvent to generic AuditEvent
func (e *OrderEvent) ToAuditEvent() *utils.AuditEvent {
	metadata := e.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if e.OrderStatus != nil {
		metadata["order_status"] = *e.OrderStatus
	}
	if e.TotalAmount != nil {
		metadata["total_amount"] = *e.TotalAmount
	}

	return &utils.AuditEvent{
		TenantID:     e.TenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    e.ActorType,
		ActorID:      e.ActorID,
		ActorEmail:   e.ActorEmail,
		SessionID:    e.SessionID,
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceID:   e.ResourceID,
		IPAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		RequestID:    e.RequestID,
		BeforeValue:  e.BeforeValue,
		AfterValue:   e.AfterValue,
		Metadata:     metadata,
	}
}

// GuestDataEvent represents audit event for guest order PII access/deletion
type GuestDataEvent struct {
	TenantID       string                 `json:"tenant_id"`
	GuestOrderID   string                 `json:"guest_order_id"`
	OrderReference string                 `json:"order_reference"`
	Action         string                 `json:"action"`      // ACCESS, ANONYMIZE
	VerifiedBy     string                 `json:"verified_by"` // email, phone
	IPAddress      *string                `json:"ip_address"`
	UserAgent      *string                `json:"user_agent"`
	RequestID      *string                `json:"request_id"`
	BeforeValue    map[string]interface{} `json:"before_value"` // Customer PII before anonymization
	Metadata       map[string]interface{} `json:"metadata"`
}

// ToAuditEvent converts GuestDataEvent to generic AuditEvent
func (e *GuestDataEvent) ToAuditEvent() *utils.AuditEvent {
	metadata := e.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["guest_order_id"] = e.GuestOrderID
	metadata["order_reference"] = e.OrderReference
	metadata["verified_by"] = e.VerifiedBy

	return &utils.AuditEvent{
		TenantID:     e.TenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    "guest",
		Action:       e.Action,
		ResourceType: "guest_order",
		ResourceID:   e.GuestOrderID,
		IPAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		RequestID:    e.RequestID,
		BeforeValue:  e.BeforeValue,
		Metadata:     metadata,
	}
}
