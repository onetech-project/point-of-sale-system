package events

import (
	"time"

	"github.com/pos/user-service/src/utils"
)

// DataAccessEvent represents audit event for PII data access
type DataAccessEvent struct {
	TenantID     string                 `json:"tenant_id"`
	ActorType    string                 `json:"actor_type"` // user, system, admin
	ActorID      *string                `json:"actor_id"`
	ActorEmail   *string                `json:"actor_email"` // Encrypted
	SessionID    *string                `json:"session_id"`
	Action       string                 `json:"action"` // READ, ACCESS, EXPORT
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	IPAddress    *string                `json:"ip_address"`
	UserAgent    *string                `json:"user_agent"`
	RequestID    *string                `json:"request_id"`
	Purpose      *string                `json:"purpose"` // Legal basis (UU PDP Article 20)
	Metadata     map[string]interface{} `json:"metadata"`
}

// ToAuditEvent converts DataAccessEvent to generic AuditEvent
func (e *DataAccessEvent) ToAuditEvent() *utils.AuditEvent {
	return &utils.AuditEvent{
		TenantID:     e.TenantID,
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
		Purpose:      e.Purpose,
		Metadata:     e.Metadata,
	}
}

// ConsentEvent represents audit event for consent grant/revoke
type ConsentEvent struct {
	TenantID      string                 `json:"tenant_id"`
	SubjectType   string                 `json:"subject_type"` // tenant, guest
	SubjectID     *string                `json:"subject_id"`   // User ID or guest order ID
	ActorType     string                 `json:"actor_type"`   // user, guest
	ActorID       *string                `json:"actor_id"`
	ActorEmail    *string                `json:"actor_email"` // Encrypted
	SessionID     *string                `json:"session_id"`
	Action        string                 `json:"action"` // CONSENT_GRANT, CONSENT_REVOKE
	PurposeCode   string                 `json:"purpose_code"`
	PolicyVersion string                 `json:"policy_version"`
	ConsentMethod string                 `json:"consent_method"` // registration, checkout, settings_update
	IPAddress     *string                `json:"ip_address"`
	UserAgent     *string                `json:"user_agent"`
	RequestID     *string                `json:"request_id"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ToAuditEvent converts ConsentEvent to generic AuditEvent
func (e *ConsentEvent) ToAuditEvent() *utils.AuditEvent {
	metadata := e.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["purpose_code"] = e.PurposeCode
	metadata["policy_version"] = e.PolicyVersion
	metadata["consent_method"] = e.ConsentMethod
	metadata["subject_type"] = e.SubjectType

	resourceID := e.SubjectID
	if resourceID == nil {
		emptyStr := ""
		resourceID = &emptyStr
	}

	return &utils.AuditEvent{
		TenantID:     e.TenantID,
		ActorType:    e.ActorType,
		ActorID:      e.ActorID,
		ActorEmail:   e.ActorEmail,
		SessionID:    e.SessionID,
		Action:       e.Action,
		ResourceType: "consent",
		ResourceID:   *resourceID,
		IPAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		RequestID:    e.RequestID,
		Metadata:     metadata,
	}
}

// DeletionEvent represents audit event for data deletion/anonymization
type DeletionEvent struct {
	TenantID      string                 `json:"tenant_id"`
	ActorType     string                 `json:"actor_type"` // user, system, admin
	ActorID       *string                `json:"actor_id"`
	ActorEmail    *string                `json:"actor_email"` // Encrypted
	SessionID     *string                `json:"session_id"`
	Action        string                 `json:"action"` // DELETE, ANONYMIZE
	ResourceType  string                 `json:"resource_type"`
	ResourceID    string                 `json:"resource_id"`
	DeletionType  string                 `json:"deletion_type"`  // soft, hard
	RetentionDays *int                   `json:"retention_days"` // Grace period before hard delete
	IPAddress     *string                `json:"ip_address"`
	UserAgent     *string                `json:"user_agent"`
	RequestID     *string                `json:"request_id"`
	BeforeValue   map[string]interface{} `json:"before_value"` // Encrypted PII snapshot
	Metadata      map[string]interface{} `json:"metadata"`
}

// ToAuditEvent converts DeletionEvent to generic AuditEvent
func (e *DeletionEvent) ToAuditEvent() *utils.AuditEvent {
	metadata := e.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["deletion_type"] = e.DeletionType
	if e.RetentionDays != nil {
		metadata["retention_days"] = *e.RetentionDays
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
		Metadata:     metadata,
	}
}
