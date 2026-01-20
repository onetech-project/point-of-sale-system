package events

import (
	"time"

	"github.com/pos/auth-service/src/utils"
)

// AuthenticationEvent represents audit event for login attempts
type AuthenticationEvent struct {
	TenantID      string                 `json:"tenant_id"`
	ActorType     string                 `json:"actor_type"` // user, admin
	ActorID       *string                `json:"actor_id"`
	ActorEmail    *string                `json:"actor_email"`  // Encrypted
	Action        string                 `json:"action"`       // LOGIN_SUCCESS, LOGIN_FAILURE
	LoginMethod   string                 `json:"login_method"` // password, oauth
	FailureReason *string                `json:"failure_reason"`
	IPAddress     *string                `json:"ip_address"`
	UserAgent     *string                `json:"user_agent"`
	RequestID     *string                `json:"request_id"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ToAuditEvent converts AuthenticationEvent to generic AuditEvent
func (e *AuthenticationEvent) ToAuditEvent() *utils.AuditEvent {
	metadata := e.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["login_method"] = e.LoginMethod
	if e.FailureReason != nil {
		metadata["failure_reason"] = *e.FailureReason
	}

	resourceID := ""
	if e.ActorID != nil {
		resourceID = *e.ActorID
	}

	return &utils.AuditEvent{
		TenantID:     e.TenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    e.ActorType,
		ActorID:      e.ActorID,
		ActorEmail:   e.ActorEmail,
		Action:       e.Action,
		ResourceType: "authentication",
		ResourceID:   resourceID,
		IPAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		RequestID:    e.RequestID,
		Metadata:     metadata,
	}
}

// SessionEvent represents audit event for session lifecycle
type SessionEvent struct {
	TenantID     string                 `json:"tenant_id"`
	ActorType    string                 `json:"actor_type"` // user, admin
	ActorID      *string                `json:"actor_id"`
	ActorEmail   *string                `json:"actor_email"` // Encrypted
	SessionID    *string                `json:"session_id"`
	Action       string                 `json:"action"` // SESSION_CREATED, SESSION_EXPIRED, SESSION_REVOKED
	ExpiresAt    *time.Time             `json:"expires_at"`
	RevokeReason *string                `json:"revoke_reason"`
	IPAddress    *string                `json:"ip_address"`
	UserAgent    *string                `json:"user_agent"`
	RequestID    *string                `json:"request_id"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ToAuditEvent converts SessionEvent to generic AuditEvent
func (e *SessionEvent) ToAuditEvent() *utils.AuditEvent {
	metadata := e.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if e.ExpiresAt != nil {
		metadata["expires_at"] = e.ExpiresAt.Format(time.RFC3339)
	}
	if e.RevokeReason != nil {
		metadata["revoke_reason"] = *e.RevokeReason
	}

	sessionIDStr := ""
	if e.SessionID != nil {
		sessionIDStr = *e.SessionID
	}

	return &utils.AuditEvent{
		TenantID:     e.TenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    e.ActorType,
		ActorID:      e.ActorID,
		ActorEmail:   e.ActorEmail,
		SessionID:    e.SessionID,
		Action:       e.Action,
		ResourceType: "session",
		ResourceID:   sessionIDStr,
		IPAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		RequestID:    e.RequestID,
		Metadata:     metadata,
	}
}
