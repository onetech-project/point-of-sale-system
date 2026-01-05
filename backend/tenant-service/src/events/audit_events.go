package events

import (
	"time"

	"github.com/pos/tenant-service/src/utils"
)

// TenantConfigEvent represents audit event for tenant configuration changes
type TenantConfigEvent struct {
	TenantID    string                 `json:"tenant_id"`
	ActorType   string                 `json:"actor_type"` // user, admin, system
	ActorID     *string                `json:"actor_id"`
	ActorEmail  *string                `json:"actor_email"` // Encrypted
	SessionID   *string                `json:"session_id"`
	Action      string                 `json:"action"`      // CREATE, UPDATE, DELETE
	ConfigType  string                 `json:"config_type"` // payment_gateway, business_profile, etc.
	IPAddress   *string                `json:"ip_address"`
	UserAgent   *string                `json:"user_agent"`
	RequestID   *string                `json:"request_id"`
	BeforeValue map[string]interface{} `json:"before_value"` // Encrypted sensitive config
	AfterValue  map[string]interface{} `json:"after_value"`  // Encrypted sensitive config
	Metadata    map[string]interface{} `json:"metadata"`
}

// ToAuditEvent converts TenantConfigEvent to generic AuditEvent
func (e *TenantConfigEvent) ToAuditEvent() *utils.AuditEvent {
	metadata := e.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["config_type"] = e.ConfigType

	return &utils.AuditEvent{
		TenantID:     e.TenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    e.ActorType,
		ActorID:      e.ActorID,
		ActorEmail:   e.ActorEmail,
		SessionID:    e.SessionID,
		Action:       e.Action,
		ResourceType: "tenant_config",
		ResourceID:   e.TenantID,
		IPAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		RequestID:    e.RequestID,
		BeforeValue:  e.BeforeValue,
		AfterValue:   e.AfterValue,
		Metadata:     metadata,
	}
}
