package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/pos/user-service/src/models"
)

var (
	ErrOnboardingMissingAuth  = errors.New("onboarding auth headers are required")
	ErrOnboardingInvalidTour  = errors.New("onboarding tour does not match role")
	ErrOnboardingInvalidInput = errors.New("invalid onboarding request")
)

type OnboardingService struct {
	db *sql.DB
}

type OnboardingIdentity struct {
	TenantID string
	UserID   string
	Role     string
}

func NewOnboardingService(db *sql.DB) *OnboardingService {
	return &OnboardingService{db: db}
}

func (s *OnboardingService) GetProgress(ctx context.Context, identity OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error) {
	if err := validateOnboardingRequest(identity, tourKey, tourVersion); err != nil {
		return nil, err
	}

	var completedAt time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT completed_at
		FROM user_onboarding_progress
		WHERE tenant_id = $1
		  AND user_id = $2
		  AND tour_key = $3
		  AND tour_version = $4
	`, identity.TenantID, identity.UserID, tourKey, tourVersion).Scan(&completedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return &models.OnboardingProgress{Completed: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch onboarding progress: %w", err)
	}

	return &models.OnboardingProgress{
		Completed:   true,
		CompletedAt: &completedAt,
	}, nil
}

func (s *OnboardingService) Complete(ctx context.Context, identity OnboardingIdentity, tourKey string, tourVersion int) (*models.OnboardingProgress, error) {
	if err := validateOnboardingRequest(identity, tourKey, tourVersion); err != nil {
		return nil, err
	}

	var completedAt time.Time
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO user_onboarding_progress (tenant_id, user_id, tour_key, tour_version, role, completed_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, tour_key, tour_version)
		DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			role = EXCLUDED.role,
			completed_at = user_onboarding_progress.completed_at
		RETURNING completed_at
	`, identity.TenantID, identity.UserID, tourKey, tourVersion, identity.Role).Scan(&completedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to complete onboarding: %w", err)
	}

	return &models.OnboardingProgress{
		Completed:   true,
		CompletedAt: &completedAt,
	}, nil
}

func validateOnboardingRequest(identity OnboardingIdentity, tourKey string, tourVersion int) error {
	if identity.TenantID == "" || identity.UserID == "" || identity.Role == "" {
		return ErrOnboardingMissingAuth
	}
	if tourKey == "" || tourVersion <= 0 {
		return ErrOnboardingInvalidInput
	}
	if expectedTourKey(identity.Role) != tourKey {
		return ErrOnboardingInvalidTour
	}
	return nil
}

func expectedTourKey(role string) string {
	switch role {
	case string(models.RoleOwner):
		return "owner-onboarding"
	case string(models.RoleManager):
		return "manager-onboarding"
	case string(models.RoleCashier):
		return "cashier-onboarding"
	default:
		return ""
	}
}
