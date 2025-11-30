package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/pos/tenant-service/src/models"
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
	tenantRepo *repository.TenantRepository
	db         *sql.DB
}

func NewTenantService(db *sql.DB) *TenantService {
	return &TenantService{
		tenantRepo: repository.NewTenantRepository(db),
		db:         db,
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

	// Create owner user in the same transaction
	ownerUserID, verificationToken, err := s.createOwnerUser(ctx, tx, tenant.ID, req.Email, string(hashedPassword), req.FirstName, req.LastName)
	if err != nil {
		return nil, fmt.Errorf("failed to create owner user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send verification email (async, non-blocking)
	go s.sendVerificationEmail(tenant.ID, ownerUserID, req.Email, req.FirstName, verificationToken)

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

	// Generate verification token (valid for 24 hours)
	verificationToken := generateVerificationToken()
	expiresAt := time.Now().Add(24 * time.Hour)

	query := `
		INSERT INTO users (
			tenant_id, email, password_hash, role, status, 
			first_name, last_name, email_verified, 
			verification_token, verification_token_expires_at
		)
		VALUES ($1, $2, $3, 'owner', 'active', $4, $5, false, $6, $7)
		RETURNING id
	`
	
	var userID string
	err = tx.QueryRowContext(ctx, query, tenantID, email, hashedPassword, firstName, lastName, verificationToken, expiresAt).Scan(&userID)
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

func (s *TenantService) sendVerificationEmail(tenantID, userID, email, firstName, verificationToken string) {
	// Get notification service URL from environment
	notificationURL := os.Getenv("NOTIFICATION_SERVICE_URL")
	if notificationURL == "" {
		notificationURL = "http://localhost:8085"
	}

	// Build verification URL
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", frontendURL, verificationToken)

	payload := map[string]interface{}{
		"tenant_id":        tenantID,
		"user_id":          userID,
		"email":            email,
		"first_name":       firstName,
		"type":             "email_verification",
		"verification_url": verificationURL,
		"verification_token": verificationToken,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Failed to marshal verification email payload: %v\n", err)
		return
	}

	resp, err := http.Post(notificationURL+"/send", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to send verification email: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		fmt.Printf("Verification email request failed with status: %d\n", resp.StatusCode)
	} else {
		fmt.Printf("Verification email sent successfully to %s\n", email)
	}
}

func (s *TenantService) GetBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	return s.tenantRepo.FindBySlug(ctx, slug)
}

func (s *TenantService) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	return s.tenantRepo.FindByID(ctx, id)
}
