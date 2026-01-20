package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/pos/auth-service/src/models"
	"github.com/pos/auth-service/src/queue"
	"github.com/pos/auth-service/src/repository"
	"github.com/pos/auth-service/src/utils"

	"golang.org/x/crypto/bcrypt"
)

type PasswordResetService struct {
	resetRepo      *repository.PasswordResetRepository
	userDB         *sql.DB
	eventPublisher *queue.EventPublisher
	encryptor      utils.Encryptor
}

func NewPasswordResetService(resetRepo *repository.PasswordResetRepository, userDB *sql.DB, eventPublisher *queue.EventPublisher, encryptor utils.Encryptor) *PasswordResetService {
	return &PasswordResetService{
		resetRepo:      resetRepo,
		userDB:         userDB,
		eventPublisher: eventPublisher,
		encryptor:      encryptor,
	}
}

func (s *PasswordResetService) RequestReset(email string) (string, error) {
	// Encrypt email for database lookup (deterministic encryption)
	ctx := context.Background()
	encryptedEmail, err := s.encryptor.EncryptWithContext(ctx, email, "user:email")
	if err != nil {
		return "", err
	}

	var userID uuid.UUID
	var tenantID uuid.UUID
	var encryptedFirstName string
	var encryptedLastName string
	query := `SELECT id, tenant_id, first_name, last_name FROM users WHERE email = $1`
	err = s.userDB.QueryRow(query, encryptedEmail).Scan(&userID, &tenantID, &encryptedFirstName, &encryptedLastName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	// Decrypt first_name and last_name
	firstName, err := s.encryptor.DecryptWithContext(ctx, encryptedFirstName, "user:first_name")
	if err != nil {
		return "", err
	}
	lastName, err := s.encryptor.DecryptWithContext(ctx, encryptedLastName, "user:last_name")
	if err != nil {
		return "", err
	}

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	recentCount, err := s.resetRepo.CountRecentRequests(userID, tenantID, oneHourAgo)
	if err != nil {
		return "", err
	}
	if recentCount >= 3 {
		return "", errors.New("too many reset requests, please try again later")
	}

	token, err := generateSecureToken(32)
	if err != nil {
		return "", err
	}

	resetToken := &models.PasswordResetToken{
		UserID:    userID,
		TenantID:  tenantID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err = s.resetRepo.Create(resetToken)
	if err != nil {
		return "", err
	}

	// Publish event to notification service
	name := firstName + " " + lastName
	if err := s.eventPublisher.PublishPasswordResetRequested(ctx, tenantID.String(), userID.String(), email, name, token); err != nil {
		// Log error but don't fail the request
		log.Printf("Error publishing password reset event: %v", err)
		return token, nil
	}
	log.Printf("Published password reset event for user: %s", email)

	return token, nil
}

func (s *PasswordResetService) ValidateToken(token string) (*models.PasswordResetToken, error) {
	resetToken, err := s.resetRepo.FindByToken(token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid or expired token")
		}
		return nil, err
	}

	if resetToken.UsedAt.Valid {
		return nil, errors.New("token already used")
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	return resetToken, nil
}

func (s *PasswordResetService) ResetPassword(token, newPassword string) error {
	resetToken, err := s.ValidateToken(token)
	if err != nil {
		return err
	}

	// Get user details for notification
	var encryptedEmail, encryptedFirstName, encryptedLastName string
	query := `SELECT email, first_name, last_name FROM users WHERE id = $1 AND tenant_id = $2`
	err = s.userDB.QueryRow(query, resetToken.UserID, resetToken.TenantID).Scan(&encryptedEmail, &encryptedFirstName, &encryptedLastName)
	if err != nil {
		return err
	}

	// Decrypt email, first_name, and last_name
	ctx := context.Background()
	email, err := s.encryptor.DecryptWithContext(ctx, encryptedEmail, "user:email")
	if err != nil {
		return err
	}
	firstName, err := s.encryptor.DecryptWithContext(ctx, encryptedFirstName, "user:first_name")
	if err != nil {
		return err
	}
	lastName, err := s.encryptor.DecryptWithContext(ctx, encryptedLastName, "user:last_name")
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	updateQuery := `UPDATE users SET password_hash = $1 WHERE id = $2 AND tenant_id = $3`
	_, err = s.userDB.Exec(updateQuery, string(hashedPassword), resetToken.UserID, resetToken.TenantID)
	if err != nil {
		return err
	}

	err = s.resetRepo.MarkAsUsed(resetToken.ID)
	if err != nil {
		return err
	}

	// Publish password changed event
	name := firstName + " " + lastName
	if err := s.eventPublisher.PublishPasswordChanged(ctx, resetToken.TenantID.String(), resetToken.UserID.String(), email, name); err != nil {
		// Log error but don't fail the request
		log.Printf("Error publishing password changed event: %v", err)
	} else {
		log.Printf("Published password changed event for user: %s", email)
	}

	return nil
}

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
