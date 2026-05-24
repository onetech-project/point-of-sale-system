package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pos/user-service/src/repository"
	"github.com/pos/user-service/src/utils"
)

const teamAuditPurpose = "team_management"

type AuditContext struct {
	ActorID    string
	ActorEmail string
	ActorRole  string
	IPAddress  string
	UserAgent  string
	RequestID  string
}

func publishTeamAuditEvent(
	ctx context.Context,
	publisher utils.AuditPublisherInterface,
	userRepo *repository.UserRepository,
	auditCtx AuditContext,
	tenantID string,
	action string,
	resourceType string,
	resourceID string,
	eventType string,
	beforeValue map[string]interface{},
	afterValue map[string]interface{},
	metadata map[string]interface{},
) {
	if publisher == nil {
		return
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["event_type"] = eventType
	if auditCtx.ActorRole != "" {
		metadata["actor_role"] = auditCtx.ActorRole
	}

	purpose := teamAuditPurpose
	event := &utils.AuditEvent{
		TenantID:     tenantID,
		Timestamp:    time.Now().UTC(),
		ActorType:    "user",
		ActorID:      stringPtrIfNotEmpty(auditCtx.ActorID),
		ActorEmail:   encryptedActorEmail(ctx, userRepo, auditCtx.ActorEmail),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		IPAddress:    stringPtrIfNotEmpty(auditCtx.IPAddress),
		UserAgent:    stringPtrIfNotEmpty(auditCtx.UserAgent),
		RequestID:    stringPtrIfNotEmpty(auditCtx.RequestID),
		Purpose:      &purpose,
		BeforeValue:  beforeValue,
		AfterValue:   afterValue,
		Metadata:     metadata,
	}

	if event.ActorID == nil {
		event.ActorType = "guest"
	}

	if err := publisher.Publish(ctx, event); err != nil {
		fmt.Printf("Failed to publish team audit event %s: %v\n", eventType, err)
	}
}

func encryptedActorEmail(ctx context.Context, userRepo *repository.UserRepository, email string) *string {
	if userRepo == nil || email == "" {
		return nil
	}

	encrypted, err := userRepo.EncryptFieldWithContext(ctx, email, "user:email")
	if err != nil || encrypted == "" {
		return nil
	}

	return &encrypted
}

func stringPtrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
