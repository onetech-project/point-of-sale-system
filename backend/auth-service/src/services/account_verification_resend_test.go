package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/pos/auth-service/src/repository"
)

type fakeVerificationEncryptor struct{}

func (fakeVerificationEncryptor) Encrypt(ctx context.Context, plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (fakeVerificationEncryptor) EncryptWithContext(ctx context.Context, plaintext string, encryptionContext string) (string, error) {
	return "enc:" + encryptionContext + ":" + plaintext, nil
}

func (fakeVerificationEncryptor) Decrypt(ctx context.Context, ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

func (fakeVerificationEncryptor) DecryptWithContext(ctx context.Context, ciphertext string, encryptionContext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"+encryptionContext+":"), nil
}

func (fakeVerificationEncryptor) EncryptBatch(ctx context.Context, plaintexts []string) ([]string, error) {
	encrypted := make([]string, len(plaintexts))
	for i, plaintext := range plaintexts {
		encrypted[i] = "enc:" + plaintext
	}
	return encrypted, nil
}

func (fakeVerificationEncryptor) DecryptBatch(ctx context.Context, ciphertexts []string) ([]string, error) {
	decrypted := make([]string, len(ciphertexts))
	for i, ciphertext := range ciphertexts {
		decrypted[i] = strings.TrimPrefix(ciphertext, "enc:")
	}
	return decrypted, nil
}

type fakeVerificationPublisher struct {
	email string
	name  string
	token string
	err   error
}

func (f *fakeVerificationPublisher) PublishUserLogin(ctx context.Context, tenantID, userID, email, name, ipAddress, userAgent string) error {
	return nil
}

func (f *fakeVerificationPublisher) PublishUserRegistered(ctx context.Context, tenantID, userID, email, name, verificationToken string) error {
	f.email = email
	f.name = name
	f.token = verificationToken
	return f.err
}

func TestResendVerificationEmailReusesUnexpiredToken(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT\s+u\.id,\s+u\.tenant_id,\s+u\.first_name,\s+u\.last_name,\s+u\.verification_token,\s+u\.verification_token_expires_at`).
		WithArgs("enc:user:email:owner@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"tenant_id",
			"first_name",
			"last_name",
			"verification_token",
			"verification_token_expires_at",
		}).AddRow(
			"user-1",
			"tenant-1",
			"enc:user:first_name:Ada",
			"enc:user:last_name:Owner",
			"enc:verification_token:token:existing-token",
			time.Now().Add(time.Hour),
		))
	mock.ExpectCommit()

	publisher := &fakeVerificationPublisher{}
	service := &AuthService{
		accountVerificationRepo: repository.NewAccountVerificationRepository(db, fakeVerificationEncryptor{}),
		eventPublisher:          publisher,
	}

	if err := service.ResendVerificationEmail(context.Background(), " Owner@Example.COM "); err != nil {
		t.Fatalf("ResendVerificationEmail returned error: %v", err)
	}
	if publisher.email != "owner@example.com" || publisher.name != "Ada Owner" || publisher.token != "existing-token" {
		t.Fatalf("unexpected published event: %+v", publisher)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestResendVerificationEmailRefreshesExpiredToken(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT\s+u\.id,\s+u\.tenant_id,\s+u\.first_name,\s+u\.last_name,\s+u\.verification_token,\s+u\.verification_token_expires_at`).
		WithArgs("enc:user:email:owner@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"tenant_id",
			"first_name",
			"last_name",
			"verification_token",
			"verification_token_expires_at",
		}).AddRow(
			"user-1",
			"tenant-1",
			"enc:user:first_name:Ada",
			"enc:user:last_name:Owner",
			"enc:verification_token:token:expired-token",
			time.Now().Add(-time.Hour),
		))
	mock.ExpectExec(`UPDATE users\s+SET verification_token = \$1, verification_token_expires_at = \$2, updated_at = NOW\(\)`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	publisher := &fakeVerificationPublisher{}
	service := &AuthService{
		accountVerificationRepo: repository.NewAccountVerificationRepository(db, fakeVerificationEncryptor{}),
		eventPublisher:          publisher,
	}

	if err := service.ResendVerificationEmail(context.Background(), "owner@example.com"); err != nil {
		t.Fatalf("ResendVerificationEmail returned error: %v", err)
	}
	if publisher.token == "" || publisher.token == "expired-token" {
		t.Fatalf("token = %q, want refreshed token", publisher.token)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestResendVerificationEmailNoopsForUnknownAccount(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT\s+u\.id,\s+u\.tenant_id,\s+u\.first_name,\s+u\.last_name,\s+u\.verification_token,\s+u\.verification_token_expires_at`).
		WithArgs("enc:user:email:missing@example.com").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	publisher := &fakeVerificationPublisher{}
	service := &AuthService{
		accountVerificationRepo: repository.NewAccountVerificationRepository(db, fakeVerificationEncryptor{}),
		eventPublisher:          publisher,
	}

	if err := service.ResendVerificationEmail(context.Background(), "missing@example.com"); err != nil {
		t.Fatalf("ResendVerificationEmail returned error: %v", err)
	}
	if publisher.email != "" {
		t.Fatalf("publisher.email = %s, want empty", publisher.email)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestResendVerificationEmailIgnoresPublisherError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT\s+u\.id,\s+u\.tenant_id,\s+u\.first_name,\s+u\.last_name,\s+u\.verification_token,\s+u\.verification_token_expires_at`).
		WithArgs("enc:user:email:owner@example.com").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"tenant_id",
			"first_name",
			"last_name",
			"verification_token",
			"verification_token_expires_at",
		}).AddRow(
			"user-1",
			"tenant-1",
			"enc:user:first_name:Ada",
			"enc:user:last_name:Owner",
			"enc:verification_token:token:existing-token",
			time.Now().Add(time.Hour),
		))
	mock.ExpectCommit()

	publisher := &fakeVerificationPublisher{err: errors.New("kafka unavailable")}
	service := &AuthService{
		accountVerificationRepo: repository.NewAccountVerificationRepository(db, fakeVerificationEncryptor{}),
		eventPublisher:          publisher,
	}

	if err := service.ResendVerificationEmail(context.Background(), "owner@example.com"); err != nil {
		t.Fatalf("ResendVerificationEmail returned error: %v", err)
	}
	if publisher.token != "existing-token" {
		t.Fatalf("token = %q, want existing-token", publisher.token)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
