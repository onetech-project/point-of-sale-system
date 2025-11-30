package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pos/auth-service/src/models"
	"github.com/pos/auth-service/src/repository"

	"golang.org/x/crypto/bcrypt"
)

type PasswordResetService struct {
	resetRepo *repository.PasswordResetRepository
	userDB    *sql.DB
}

func NewPasswordResetService(resetRepo *repository.PasswordResetRepository, userDB *sql.DB) *PasswordResetService {
	return &PasswordResetService{
		resetRepo: resetRepo,
		userDB:    userDB,
	}
}

func (s *PasswordResetService) RequestReset(email string, tenantIDStr string) (string, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return "", errors.New("invalid tenant ID")
	}

	var userID uuid.UUID
	query := `SELECT id FROM users WHERE email = $1 AND tenant_id = $2`
	err = s.userDB.QueryRow(query, email, tenantID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `UPDATE users SET password = $1 WHERE id = $2 AND tenant_id = $3`
	_, err = s.userDB.Exec(query, string(hashedPassword), resetToken.UserID, resetToken.TenantID)
	if err != nil {
		return err
	}

	return s.resetRepo.MarkAsUsed(resetToken.ID)
}

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
