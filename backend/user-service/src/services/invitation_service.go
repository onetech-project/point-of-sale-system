package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/user-service/src/events"
	"github.com/pos/user-service/src/models"
	"github.com/pos/user-service/src/queue"
	"github.com/pos/user-service/src/repository"
	"github.com/pos/user-service/src/utils"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvitationNotFound  = errors.New("invitation not found")
	ErrInvitationExpired   = errors.New("invitation expired")
	ErrInvitationInvalid   = errors.New("invitation invalid")
	ErrEmailAlreadyInvited = errors.New("email already invited")
	ErrEmailAlreadyExists  = errors.New("email already registered")
)

type InvitationService struct {
	invitationRepo *repository.InvitationRepository
	userRepo       *repository.UserRepository
	db             *sql.DB
	eventProducer  *queue.KafkaProducer
}

func NewInvitationService(db *sql.DB, eventProducer *queue.KafkaProducer, auditPublisher utils.AuditPublisherInterface) (*InvitationService, error) {
	userRepo, err := repository.NewUserRepositoryWithVault(db, auditPublisher)
	if err != nil {
		return nil, fmt.Errorf("failed to create user repository: %w", err)
	}

	invitationRepo, err := repository.NewInvitationRepositoryWithVault(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create invitation repository: %w", err)
	}

	return &InvitationService{
		invitationRepo: invitationRepo,
		userRepo:       userRepo,
		db:             db,
		eventProducer:  eventProducer,
	}, nil
}

func (s *InvitationService) Create(ctx context.Context, tenantID, email, role, invitedByID string) (*models.Invitation, error) {
	// Check if email is already registered in this tenant
	existingUser, err := s.userRepo.FindByEmail(ctx, tenantID, email)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Check if there's already a pending invitation
	existingInvitation, err := s.invitationRepo.FindByEmail(ctx, tenantID, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invitation: %w", err)
	}
	if existingInvitation != nil {
		return nil, ErrEmailAlreadyInvited
	}

	// Generate secure token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	invitation := &models.Invitation{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Email:     email,
		Role:      role,
		Token:     token,
		Status:    models.InvitationPending,
		InvitedBy: invitedByID,
		ExpiresAt: now.Add(7 * 24 * time.Hour), // 7 days expiration
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	// Publish invitation event to Kafka for notification service
	if s.eventProducer != nil {
		// Fetch inviter info
		inviter, err := s.userRepo.FindByID(ctx, tenantID, invitedByID)
		inviterName := "Team Member"
		if err == nil && inviter != nil {
			if inviter.FirstName != nil && inviter.LastName != nil {
				inviterName = fmt.Sprintf("%s %s", *inviter.FirstName, *inviter.LastName)
			}
		}

		// Fetch tenant info
		var tenantName string
		err = s.db.QueryRowContext(ctx, "SELECT business_name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)
		if err != nil {
			tenantName = "the team"
		}

		event := &events.NotificationEvent{
			EventID:   uuid.New().String(),
			EventType: "invitation.created",
			TenantID:  tenantID,
			UserID:    invitedByID,
			Data: map[string]interface{}{
				"invitation_id":    invitation.ID,
				"email":            email,
				"role":             role,
				"token":            token,
				"invitation_token": token,
				"expires_at":       invitation.ExpiresAt.Format(time.RFC3339),
				"invited_by":       invitedByID,
				"inviter_name":     inviterName,
				"tenant_name":      tenantName,
			},
			Timestamp: now,
		}

		// Send event to Kafka (non-blocking, log error if failed)
		if err := s.eventProducer.Publish(ctx, invitation.ID, event); err != nil {
			// Log the error but don't fail the invitation creation
			fmt.Printf("Warning: failed to publish invitation event: %v\n", err)
		}
	}

	return invitation, nil
}

func (s *InvitationService) List(ctx context.Context, tenantID string) ([]*models.Invitation, error) {
	invitations, err := s.invitationRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}

	// Update expired invitations
	now := time.Now()
	for _, inv := range invitations {
		if inv.Status == models.InvitationPending && inv.ExpiresAt.Before(now) {
			s.invitationRepo.UpdateStatus(ctx, inv.ID, models.InvitationExpired)
			inv.Status = models.InvitationExpired
		}
	}

	return invitations, nil
}

func (s *InvitationService) Accept(ctx context.Context, token, firstName, lastName, password string, consents []string, ipAddress, userAgent string) (*models.User, error) {
	// Find invitation by token
	invitation, err := s.invitationRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to find invitation: %w", err)
	}
	if invitation == nil {
		return nil, ErrInvitationNotFound
	}

	// Validate invitation
	if invitation.Status != models.InvitationPending {
		return nil, ErrInvitationInvalid
	}
	if invitation.ExpiresAt.Before(time.Now()) {
		s.invitationRepo.UpdateStatus(ctx, invitation.ID, models.InvitationExpired)
		return nil, ErrInvitationExpired
	}

	// Check if email is already registered in this tenant
	existingUser, err := s.userRepo.FindByEmail(ctx, invitation.TenantID, invitation.Email)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	now := time.Now()
	user := &models.User{
		ID:           uuid.New().String(),
		TenantID:     invitation.TenantID,
		Email:        invitation.Email,
		PasswordHash: string(hashedPassword),
		Role:         invitation.Role,
		FirstName:    &firstName,
		LastName:     &lastName,
		Status:       string(models.UserStatusActive),
		Locale:       "en",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Mark invitation as accepted
	if err := s.invitationRepo.MarkAccepted(ctx, invitation.ID); err != nil {
		return nil, fmt.Errorf("failed to mark invitation as accepted: %w", err)
	}

	// Publish ConsentGrantedEvent to Kafka (async, after user creation)
	if s.eventProducer != nil {
		go func() {
			// Required consents for tenant users (implicit)
			requiredConsents := []string{"operational", "third_party_midtrans"}
			
			consentEvent := events.ConsentGrantedEvent{
				EventID:          uuid.New().String(),
				EventType:        "consent.granted",
				TenantID:         user.TenantID,
				SubjectType:      "tenant",
				SubjectID:        user.ID,
				ConsentMethod:    "registration", // Invitation acceptance is similar to registration
				PolicyVersion:    "1.0.0",
				Consents:         consents,          // Optional consents provided by user
				RequiredConsents: requiredConsents,  // Required consents (implicit)
				Metadata: events.ConsentMetadata{
					IPAddress: ipAddress,
					UserAgent: userAgent,
					SessionID: nil,
					RequestID: "", // TODO: Extract from context
				},
				Timestamp: time.Now(),
			}

			if err := s.eventProducer.Publish(context.Background(), user.ID, consentEvent); err != nil {
				fmt.Printf("Warning: failed to publish consent event: %v\n", err)
			}
		}()
	}

	return user, nil
}

// Resend resends an invitation
func (s *InvitationService) Resend(ctx context.Context, tenantID, invitationID, resendByID string) (*models.Invitation, error) {
	// Find invitation
	invitation, err := s.invitationRepo.FindByID(ctx, invitationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find invitation: %w", err)
	}
	if invitation == nil {
		return nil, ErrInvitationNotFound
	}

	// Verify tenant ownership
	if invitation.TenantID != tenantID {
		return nil, ErrInvitationNotFound
	}

	// Can only resend pending invitations
	if invitation.Status != models.InvitationPending {
		return nil, errors.New("can only resend pending invitations")
	}

	// Generate new token and extend expiration
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	invitation.Token = token
	invitation.ExpiresAt = now.Add(7 * 24 * time.Hour) // 7 days expiration
	invitation.UpdatedAt = now

	if err := s.invitationRepo.UpdateToken(ctx, invitation.ID, token, invitation.ExpiresAt); err != nil {
		return nil, fmt.Errorf("failed to update invitation token: %w", err)
	}

	// Publish resend event to Kafka
	if s.eventProducer != nil {
		// Fetch inviter info
		inviter, err := s.userRepo.FindByID(ctx, tenantID, resendByID)
		inviterName := "Team Member"
		if err == nil && inviter != nil {
			if inviter.FirstName != nil && inviter.LastName != nil {
				inviterName = fmt.Sprintf("%s %s", *inviter.FirstName, *inviter.LastName)
			}
		}

		// Fetch tenant info
		var tenantName string
		err = s.db.QueryRowContext(ctx, "SELECT business_name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)
		if err != nil {
			tenantName = "the team"
		}

		event := &events.NotificationEvent{
			EventID:   uuid.New().String(),
			EventType: "invitation.created", // Reuse same template
			TenantID:  tenantID,
			UserID:    resendByID,
			Data: map[string]interface{}{
				"invitation_id":    invitation.ID,
				"email":            invitation.Email,
				"role":             invitation.Role,
				"token":            token,
				"invitation_token": token,
				"expires_at":       invitation.ExpiresAt.Format(time.RFC3339),
				"invited_by":       resendByID,
				"inviter_name":     inviterName,
				"tenant_name":      tenantName,
			},
			Timestamp: now,
		}

		if err := s.eventProducer.Publish(ctx, invitation.ID, event); err != nil {
			fmt.Printf("Warning: failed to publish resend invitation event: %v\n", err)
		}
	}

	return invitation, nil
}

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
