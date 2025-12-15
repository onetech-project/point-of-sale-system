package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/pos/tenant-service/src/config"
	"github.com/pos/tenant-service/src/models"
	"github.com/pos/tenant-service/src/queue"
	"github.com/pos/tenant-service/src/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrTenantExists     = errors.New("tenant with this slug already exists")
	ErrBusinessExists   = errors.New("business name already exists")
	ErrInvalidSlug      = errors.New("invalid slug format")
	ErrUserCreationFail = errors.New("failed to create owner user")
)

type TenantService struct {
	tenantRepo     *repository.TenantRepository
	db             *sql.DB
	eventPublisher *queue.EventPublisher
}

func NewTenantService(db *sql.DB) *TenantService {
	return &TenantService{
		tenantRepo:     repository.NewTenantRepository(db),
		db:             db,
		eventPublisher: queue.NewEventPublisher(config.KAFKA_BROKERS),
	}
}

func (s *TenantService) RegisterTenant(ctx context.Context, req *models.CreateTenantRequest) (*models.Tenant, error) {
	slug := req.Slug
	if slug == "" {
		slug = GenerateSlug(req.BusinessName)
	}

	if !IsValidSlug(slug) {
		return nil, ErrInvalidSlug
	}

	existing, err := s.tenantRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing tenant: %w", err)
	}
	if existing != nil {
		return nil, ErrTenantExists
	}

	tenant := &models.Tenant{
		BusinessName: req.BusinessName,
		Slug:         slug,
		Status:       string(models.TenantStatusActive),
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Create tenant
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	if s.eventPublisher != nil {
		go func() {
			if err := s.eventPublisher.PublishTenantRegistrationSuccess(ctx, queue.TenantRegistrationSuccessEvent{
				TenantID:     tenant.ID,
				Email:        req.Email,
				PasswordHash: string(hashedPassword),
				FirstName:    req.FirstName,
				LastName:     req.LastName,
				Timestamp:    time.Now(),
			}); err != nil {
				fmt.Printf("Warning: failed to publish tenant registration success event: %v\n", err)
			}
		}()
	}

	return tenant, nil
}

func (s *TenantService) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	return s.tenantRepo.FindBySlug(ctx, slug)
}

func (s *TenantService) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	return s.tenantRepo.FindByID(ctx, id)
}

// HandleOwnerVerifiedEvent processes the owner.verified event to update tenant status
func (s *TenantService) HandleOwnerVerifiedEvent(ctx context.Context, tenantID string) error {
	tenant, err := s.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to fetch tenant: %w", err)
	}
	if tenant == nil {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	// Update tenant status from inactive to active
	if tenant.Status == string(models.TenantStatusInactive) {
		tenant.Status = string(models.TenantStatusActive)
		if err := s.tenantRepo.Update(ctx, tenant); err != nil {
			return fmt.Errorf("failed to update tenant status: %w", err)
		}
	}

	return nil
}
