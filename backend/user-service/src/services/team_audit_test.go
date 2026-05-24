package services

import (
	"context"
	"testing"
	"time"

	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/repository"
	"github.com/pos/user-service/src/utils"
	"github.com/pos/user-service/src/utils/mocks"
)

func TestPublishTeamAuditEvent(t *testing.T) {
	var published *utils.AuditEvent
	publisher := &mocks.MockAuditPublisher{
		PublishFunc: func(ctx context.Context, event *utils.AuditEvent) error {
			published = event
			return nil
		},
	}
	userRepo := repository.NewUserRepository(nil, &mocks.MockEncryptor{
		EncryptFunc: func(ctx context.Context, plaintext string) (string, error) {
			return "enc:" + plaintext, nil
		},
	}, &mocks.NoOpAuditPublisher{})

	publishTeamAuditEvent(
		context.Background(),
		publisher,
		userRepo,
		AuditContext{
			ActorID:    "actor-1",
			ActorEmail: "owner@example.com",
			ActorRole:  "owner",
			IPAddress:  "203.0.113.10",
			UserAgent:  "test-agent",
			RequestID:  "req-1",
		},
		"tenant-1",
		"UPDATE",
		"user",
		"user-1",
		"team.member.role_changed",
		map[string]interface{}{"role": "cashier"},
		map[string]interface{}{"role": "manager"},
		nil,
	)

	if published == nil {
		t.Fatal("expected audit event to be published")
	}
	if published.ActorType != "user" || published.ActorID == nil || *published.ActorID != "actor-1" {
		t.Fatalf("actor = (%s, %v), want user actor-1", published.ActorType, published.ActorID)
	}
	if published.ActorEmail == nil || *published.ActorEmail != "enc:owner@example.com" {
		t.Fatalf("actor email = %v, want encrypted email", published.ActorEmail)
	}
	if published.Action != "UPDATE" || published.ResourceType != "user" || published.ResourceID != "user-1" {
		t.Fatalf("resource/action = %s %s %s", published.Action, published.ResourceType, published.ResourceID)
	}
	if published.Metadata["event_type"] != "team.member.role_changed" {
		t.Fatalf("event_type = %v", published.Metadata["event_type"])
	}
	if published.Metadata["actor_role"] != "owner" {
		t.Fatalf("actor_role = %v", published.Metadata["actor_role"])
	}
	if published.Purpose == nil || *published.Purpose != teamAuditPurpose {
		t.Fatalf("purpose = %v, want %s", published.Purpose, teamAuditPurpose)
	}
}

func TestInvitationAuditEvents(t *testing.T) {
	var published []*utils.AuditEvent
	service := newTestInvitationAuditService(&published)
	ctx := context.Background()
	auditCtx := AuditContext{ActorID: "owner-1", ActorEmail: "owner@example.com", ActorRole: "owner"}
	expiresAt := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	invitation := &models.Invitation{
		ID:        "inv-1",
		TenantID:  "tenant-1",
		Role:      "cashier",
		Status:    models.InvitationPending,
		InvitedBy: "owner-1",
		ExpiresAt: expiresAt,
	}

	service.publishInvitationCreatedAudit(ctx, invitation, auditCtx)
	service.publishInvitationResentAudit(ctx, invitation, expiresAt.Add(-24*time.Hour), auditCtx)
	invitation.Status = models.InvitationRevoked
	service.publishInvitationRevokedAudit(ctx, invitation, models.InvitationPending, auditCtx)
	service.publishInvitationAcceptedAudit(ctx, invitation, "user-1", expiresAt, AuditContext{ActorID: "user-1", ActorEmail: "accepted@example.com"})

	wantTypes := []string{
		"team.invitation.created",
		"team.invitation.resent",
		"team.invitation.revoked",
		"team.invitation.accepted",
	}
	if len(published) != len(wantTypes) {
		t.Fatalf("published events = %d, want %d", len(published), len(wantTypes))
	}
	for i, wantType := range wantTypes {
		if published[i].ResourceType != "invitation" {
			t.Fatalf("event %d resource_type = %s, want invitation", i, published[i].ResourceType)
		}
		if published[i].Metadata["event_type"] != wantType {
			t.Fatalf("event %d type = %v, want %s", i, published[i].Metadata["event_type"], wantType)
		}
		if _, hasToken := published[i].AfterValue["token"]; hasToken {
			t.Fatalf("event %d leaked invitation token in after_value", i)
		}
	}
	if published[0].Action != "CREATE" {
		t.Fatalf("created action = %s, want CREATE", published[0].Action)
	}
	if published[1].BeforeValue["expires_at"] == published[1].AfterValue["expires_at"] {
		t.Fatal("resent audit should record expires_at transition")
	}
	if published[2].AfterValue["status"] != models.InvitationRevoked {
		t.Fatalf("revoked status = %v", published[2].AfterValue["status"])
	}
	if published[3].ActorType != "user" || published[3].ActorID == nil || *published[3].ActorID != "user-1" {
		t.Fatalf("accepted actor = (%s, %v), want user user-1", published[3].ActorType, published[3].ActorID)
	}
	if published[3].AfterValue["accepted_user_id"] != "user-1" {
		t.Fatalf("accepted user id = %v", published[3].AfterValue["accepted_user_id"])
	}
}

func TestTeamMemberAuditEvents(t *testing.T) {
	var published []*utils.AuditEvent
	userService := newTestUserAuditService(&published)
	ctx := context.Background()
	auditCtx := AuditContext{ActorID: "owner-1", ActorRole: "owner"}
	target := &models.User{
		ID:       "user-1",
		TenantID: "tenant-1",
		Role:     string(models.RoleManager),
		Status:   string(models.UserStatusSuspended),
	}

	userService.publishTeamMemberChangeAuditEvents(ctx, target, string(models.RoleCashier), string(models.UserStatusActive), auditCtx)
	managerRole := string(models.RoleManager)
	userService.publishTeamMemberDeniedAudit(ctx, target, &managerRole, nil, AuditContext{ActorID: "manager-1", ActorRole: "manager"})

	if len(published) != 3 {
		t.Fatalf("published events = %d, want 3", len(published))
	}
	if published[0].Metadata["event_type"] != "team.member.role_changed" {
		t.Fatalf("first event type = %v", published[0].Metadata["event_type"])
	}
	if published[0].BeforeValue["role"] != string(models.RoleCashier) || published[0].AfterValue["role"] != string(models.RoleManager) {
		t.Fatalf("role transition = %v -> %v", published[0].BeforeValue["role"], published[0].AfterValue["role"])
	}
	if published[1].Metadata["event_type"] != "team.member.status_changed" {
		t.Fatalf("second event type = %v", published[1].Metadata["event_type"])
	}
	if published[1].BeforeValue["status"] != string(models.UserStatusActive) || published[1].AfterValue["status"] != string(models.UserStatusSuspended) {
		t.Fatalf("status transition = %v -> %v", published[1].BeforeValue["status"], published[1].AfterValue["status"])
	}
	if published[2].Action != "ACCESS" || published[2].Metadata["event_type"] != "team.member.update_denied" {
		t.Fatalf("denied event = %s %v", published[2].Action, published[2].Metadata["event_type"])
	}
	if published[2].Metadata["requested_role"] != string(models.RoleManager) {
		t.Fatalf("requested role = %v", published[2].Metadata["requested_role"])
	}
}

func newTestInvitationAuditService(published *[]*utils.AuditEvent) *InvitationService {
	return &InvitationService{
		userRepo:       newTestAuditUserRepo(),
		auditPublisher: newCollectingAuditPublisher(published),
	}
}

func newTestUserAuditService(published *[]*utils.AuditEvent) *UserService {
	return &UserService{
		userRepo:       newTestAuditUserRepo(),
		auditPublisher: newCollectingAuditPublisher(published),
	}
}

func newTestAuditUserRepo() *repository.UserRepository {
	return repository.NewUserRepository(nil, &mocks.NoOpEncryptor{}, &mocks.NoOpAuditPublisher{})
}

func newCollectingAuditPublisher(published *[]*utils.AuditEvent) *mocks.MockAuditPublisher {
	return &mocks.MockAuditPublisher{
		PublishFunc: func(ctx context.Context, event *utils.AuditEvent) error {
			*published = append(*published, event)
			return nil
		},
	}
}
