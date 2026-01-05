package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/pos/tenant-service/src/models"
	"github.com/pos/tenant-service/src/queue"
	"github.com/pos/tenant-service/src/repository"
	"github.com/pos/tenant-service/src/utils"
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
	encryptor      utils.Encryptor
}

func NewTenantService(db *sql.DB, eventPublisher *queue.EventPublisher) *TenantService {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize Vault client: %v", err))
	}

	return &TenantService{
		tenantRepo:     repository.NewTenantRepository(db),
		db:             db,
		eventPublisher: eventPublisher,
		encryptor:      vaultClient,
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
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Create tenant
	if err := s.tenantRepo.Create(ctx, tx, tenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create owner user in the same transaction
	ownerUserID, verificationToken, err := s.createOwnerUser(ctx, tx, tenant.ID, req.Email, string(hashedPassword), req.FirstName, req.LastName)
	if err != nil {
		return nil, fmt.Errorf("failed to create owner user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send verification email (async, non-blocking)
	if s.eventPublisher != nil {
		name := strings.TrimSpace(fmt.Sprintf("%s %s", req.FirstName, req.LastName))
		go func() {
			if err := s.eventPublisher.PublishUserRegistered(context.Background(), tenant.ID, ownerUserID, req.Email, name, verificationToken); err != nil {
				fmt.Printf("Warning: failed to publish login event: %v\n", err)
			}
		}()
	}

	return tenant, nil
}

func (s *TenantService) createOwnerUser(ctx context.Context, tx *sql.Tx, tenantID, email, hashedPassword, firstName, lastName string) (string, string, error) {
	// Set tenant context for RLS policy
	// Note: SET LOCAL doesn't support parameterized queries, but tenant_id is a UUID so it's safe
	setContextSQL := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
	_, err := tx.ExecContext(ctx, setContextSQL)
	if err != nil {
		return "", "", fmt.Errorf("failed to set tenant context: %w", err)
	}

	// Encrypt PII fields before storing (FR-009: Field-level encryption)
	encryptedEmail, err := s.encryptor.Encrypt(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt email: %w", err)
	}

	encryptedFirstName, err := s.encryptor.Encrypt(ctx, firstName)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt first_name: %w", err)
	}

	encryptedLastName, err := s.encryptor.Encrypt(ctx, lastName)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt last_name: %w", err)
	}

	// Generate searchable hash for email (T051 - efficient encrypted field search)
	emailHash := utils.HashForSearch(email)

	// Generate verification token (valid for 24 hours)
	verificationToken := generateVerificationToken()
	expiresAt := time.Now().Add(24 * time.Hour)

	query := `
		INSERT INTO users (
			tenant_id, email, email_hash, password_hash, role, status, 
			first_name, last_name, email_verified, 
			verification_token, verification_token_expires_at
		)
		VALUES ($1, $2, $3, $4, 'owner', 'inactive', $5, $6, false, $7, $8)
		RETURNING id
	`

	var userID string
	err = tx.QueryRowContext(ctx, query, tenantID, encryptedEmail, emailHash, hashedPassword, encryptedFirstName, encryptedLastName, verificationToken, expiresAt).Scan(&userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to insert user: %w", err)
	}

	return userID, verificationToken, nil
}

func generateVerificationToken() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func (s *TenantService) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	return s.tenantRepo.FindBySlug(ctx, slug)
}

func (s *TenantService) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	return s.tenantRepo.FindByID(ctx, id)
}
