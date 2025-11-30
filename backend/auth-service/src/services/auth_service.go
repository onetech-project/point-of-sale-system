package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pos/auth-service/src/models"
	"github.com/pos/auth-service/src/repository"
	"golang.org/x/crypto/bcrypt"
)

type EventPublisher interface {
	PublishUserLogin(ctx context.Context, tenantID, userID, email, name, ipAddress, userAgent string) error
}

type AuthService struct {
	db              *sql.DB
	sessionRepo     *repository.SessionRepository
	sessionManager  *SessionManager
	jwtService      *JWTService
	rateLimiter     *RateLimiter
	eventPublisher  EventPublisher
}

func NewAuthService(
	db *sql.DB,
	sessionManager *SessionManager,
	jwtService *JWTService,
	rateLimiter *RateLimiter,
	eventPublisher EventPublisher,
) *AuthService {
	return &AuthService{
		db:             db,
		sessionRepo:    repository.NewSessionRepository(db),
		sessionManager: sessionManager,
		jwtService:     jwtService,
		rateLimiter:    rateLimiter,
		eventPublisher: eventPublisher,
	}
}

// Login authenticates a user and creates a session
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest, ipAddress, userAgent string) (*models.LoginResponse, string, error) {
	fmt.Printf("DEBUG: Login attempt for email: %s\n", req.Email)
	
	// First, find the tenant for this email
	tenantID, err := s.getTenantIDByEmail(ctx, req.Email)
	if err != nil {
		fmt.Printf("DEBUG: Failed to get tenant ID: %v\n", err)
		return nil, "", fmt.Errorf("failed to lookup tenant: %w", err)
	}
	
	fmt.Printf("DEBUG: Found tenant ID: %s\n", tenantID)
	
	if tenantID == "" {
		fmt.Printf("DEBUG: No tenant found for email\n")
		return nil, "", ErrInvalidCredentials
	}

	// Check rate limit
	allowed, _, err := s.rateLimiter.CheckLoginLimit(ctx, req.Email, tenantID)
	if err != nil {
		return nil, "", fmt.Errorf("rate limit check failed: %w", err)
	}

	if !allowed {
		retryAfter, _ := s.rateLimiter.GetRemainingTime(ctx, req.Email, tenantID)
		return nil, "", &RateLimitError{
			RetryAfter: retryAfter,
		}
	}

	// Query user from database
	fmt.Printf("DEBUG: Querying user with email=%s, tenant_id=%s\n", req.Email, tenantID)
	user, err := s.getUserByEmailAndTenant(ctx, req.Email, tenantID)
	if err != nil {
		fmt.Printf("DEBUG: Error querying user: %v\n", err)
		// Increment failed attempts
		s.rateLimiter.IncrementLoginAttempts(ctx, req.Email, tenantID)
		return nil, "", fmt.Errorf("authentication failed: %w", err)
	}

	if user == nil {
		fmt.Printf("DEBUG: User not found\n")
		// Increment failed attempts
		s.rateLimiter.IncrementLoginAttempts(ctx, req.Email, tenantID)
		return nil, "", ErrInvalidCredentials
	}

	fmt.Printf("DEBUG: User found - ID: %s, Status: %s, Hash length: %d\n", user.ID, user.Status, len(user.PasswordHash))

	// Verify password
	fmt.Printf("DEBUG: Comparing password (input length: %d)\n", len(req.Password))
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		fmt.Printf("DEBUG: Password comparison failed: %v\n", err)
		// Increment failed attempts
		s.rateLimiter.IncrementLoginAttempts(ctx, req.Email, tenantID)
		return nil, "", ErrInvalidCredentials
	}

	fmt.Printf("DEBUG: Password verification successful!\n")

	// Check user status
	if user.Status != "active" {
		return nil, "", &UserStatusError{Status: user.Status}
	}

	// Reset rate limit on successful authentication
	s.rateLimiter.ResetLoginAttempts(ctx, req.Email, tenantID)

	// Create session in Redis
	sessionID, err := s.sessionManager.Create(ctx, user.ID, user.TenantID, user.Email, user.Role, user.FirstName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}

	// Create session audit record in PostgreSQL
	session := &models.Session{
		SessionID: sessionID,
		TenantID:  user.TenantID,
		UserID:    user.ID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
	}

	err = s.sessionRepo.Create(ctx, session)
	if err != nil {
		// Non-fatal error - session still works from Redis
		// Log error but don't fail the login
		fmt.Printf("Warning: failed to create session audit record: %v\n", err)
	}

	// Generate JWT token
	token, err := s.jwtService.Generate(sessionID, user.ID, user.TenantID, user.Email, user.Role)
	if err != nil {
		// Cleanup session
		s.sessionManager.Delete(ctx, sessionID)
		return nil, "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Update last login time
	s.updateLastLogin(ctx, user.ID)

	// Publish login event for notification
	if s.eventPublisher != nil {
		name := user.FirstName
		if user.LastName != "" {
			name += " " + user.LastName
		}
		go func() {
			if err := s.eventPublisher.PublishUserLogin(context.Background(), user.TenantID, user.ID, user.Email, name, ipAddress, userAgent); err != nil {
				fmt.Printf("Warning: failed to publish login event: %v\n", err)
			}
		}()
	}

	response := &models.LoginResponse{
		User: models.UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			TenantID:  user.TenantID,
			Role:      user.Role,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Locale:    user.Locale,
		},
		Message: "Login successful",
	}

	return response, token, nil
}

// ValidateSession validates a session and returns session data
func (s *AuthService) ValidateSession(ctx context.Context, sessionID string) (*models.SessionData, error) {
	// Check if session exists in Redis
	sessionData, err := s.sessionManager.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if sessionData == nil {
		return nil, ErrSessionNotFound
	}

	// Renew session TTL (sliding window)
	err = s.sessionManager.Renew(ctx, sessionID)
	if err != nil {
		// Non-fatal error - session still valid
		fmt.Printf("Warning: failed to renew session TTL: %v\n", err)
	}

	return sessionData, nil
}

// Logout terminates a session
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	// Delete from Redis
	err := s.sessionManager.Delete(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	// Mark as terminated in PostgreSQL
	err = s.sessionRepo.Delete(ctx, sessionID)
	if err != nil {
		// Non-fatal error - session already deleted from Redis
		fmt.Printf("Warning: failed to mark session as terminated in database: %v\n", err)
	}

	return nil
}

// TerminateSession is an alias for Logout
func (s *AuthService) TerminateSession(ctx context.Context, sessionID string) error {
	return s.Logout(ctx, sessionID)
}

// Internal helper methods

type User struct {
	ID           string
	TenantID     string
	Email        string
	PasswordHash string
	Role         string
	Status       string
	FirstName    string
	LastName     string
	Locale       string
}

func (s *AuthService) getUserByEmailAndTenant(ctx context.Context, email, tenantID string) (*User, error) {
	fmt.Printf("DEBUG: getUserByEmailAndTenant called - email=%s, tenant_id=%s\n", email, tenantID)
	
	// Set tenant context for RLS policy
	setContextSQL := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
	fmt.Printf("DEBUG: Setting tenant context: %s\n", setContextSQL)
	_, err := s.db.ExecContext(ctx, setContextSQL)
	if err != nil {
		fmt.Printf("DEBUG: Failed to set tenant context: %v\n", err)
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		SELECT id, tenant_id, email, password_hash, role, status, first_name, last_name, locale
		FROM users
		WHERE email = $1 AND tenant_id = $2
	`

	fmt.Printf("DEBUG: Executing query...\n")
	user := &User{}
	var firstName, lastName sql.NullString

	err = s.db.QueryRowContext(ctx, query, email, tenantID).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&firstName,
		&lastName,
		&user.Locale,
	)

	if err == sql.ErrNoRows {
		fmt.Printf("DEBUG: No rows returned from query\n")
		return nil, nil
	}

	if err != nil {
		fmt.Printf("DEBUG: Query error: %v\n", err)
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	fmt.Printf("DEBUG: User retrieved successfully\n")

	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}

	return user, nil
}

func (s *AuthService) getTenantIDByEmail(ctx context.Context, email string) (string, error) {
	query := `
		SELECT tenant_id
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	var tenantID string
	err := s.db.QueryRowContext(ctx, query, email).Scan(&tenantID)

	if err == sql.ErrNoRows {
		return "", nil // User not found
	}

	if err != nil {
		return "", fmt.Errorf("failed to query tenant: %w", err)
	}

	return tenantID, nil
}

func (s *AuthService) updateLastLogin(ctx context.Context, userID string) {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		// Non-fatal error
		fmt.Printf("Warning: failed to update last login time: %v\n", err)
	}
}

// Custom errors

var (
	ErrInvalidCredentials = fmt.Errorf("invalid email or password")
	ErrSessionNotFound    = fmt.Errorf("session not found")
)

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("too many login attempts, retry after %v", e.RetryAfter)
}

type UserStatusError struct {
	Status string
}

func (e *UserStatusError) Error() string {
	return fmt.Sprintf("user account is %s", e.Status)
}
