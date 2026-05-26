package services

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestOnboardingGetProgressIncomplete(t *testing.T) {
	db, mock := newOnboardingMockDB(t)
	defer db.Close()
	service := NewOnboardingService(db)

	identity := OnboardingIdentity{TenantID: "tenant-1", UserID: "user-1", Role: "owner"}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT completed_at")).
		WithArgs(identity.TenantID, identity.UserID, "owner-onboarding", 1).
		WillReturnError(sql.ErrNoRows)

	progress, err := service.GetProgress(context.Background(), identity, "owner-onboarding", 1)
	if err != nil {
		t.Fatalf("GetProgress returned error: %v", err)
	}
	if progress.Completed {
		t.Fatal("progress.Completed = true, want false")
	}
	assertOnboardingExpectations(t, mock)
}

func TestOnboardingGetProgressCompleted(t *testing.T) {
	db, mock := newOnboardingMockDB(t)
	defer db.Close()
	service := NewOnboardingService(db)

	completedAt := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	identity := OnboardingIdentity{TenantID: "tenant-1", UserID: "user-1", Role: "manager"}
	rows := sqlmock.NewRows([]string{"completed_at"}).AddRow(completedAt)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT completed_at")).
		WithArgs(identity.TenantID, identity.UserID, "manager-onboarding", 1).
		WillReturnRows(rows)

	progress, err := service.GetProgress(context.Background(), identity, "manager-onboarding", 1)
	if err != nil {
		t.Fatalf("GetProgress returned error: %v", err)
	}
	if !progress.Completed {
		t.Fatal("progress.Completed = false, want true")
	}
	if progress.CompletedAt == nil || !progress.CompletedAt.Equal(completedAt) {
		t.Fatalf("CompletedAt = %v, want %v", progress.CompletedAt, completedAt)
	}
	assertOnboardingExpectations(t, mock)
}

func TestOnboardingComplete(t *testing.T) {
	db, mock := newOnboardingMockDB(t)
	defer db.Close()
	service := NewOnboardingService(db)

	completedAt := time.Date(2026, 5, 26, 11, 0, 0, 0, time.UTC)
	identity := OnboardingIdentity{TenantID: "tenant-1", UserID: "user-1", Role: "cashier"}
	rows := sqlmock.NewRows([]string{"completed_at"}).AddRow(completedAt)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO user_onboarding_progress")).
		WithArgs(identity.TenantID, identity.UserID, "cashier-onboarding", 1, identity.Role).
		WillReturnRows(rows)

	progress, err := service.Complete(context.Background(), identity, "cashier-onboarding", 1)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if !progress.Completed {
		t.Fatal("progress.Completed = false, want true")
	}
	if progress.CompletedAt == nil || !progress.CompletedAt.Equal(completedAt) {
		t.Fatalf("CompletedAt = %v, want %v", progress.CompletedAt, completedAt)
	}
	assertOnboardingExpectations(t, mock)
}

func TestOnboardingRejectsMissingAuthAndRoleMismatch(t *testing.T) {
	db, mock := newOnboardingMockDB(t)
	defer db.Close()
	service := NewOnboardingService(db)

	_, err := service.GetProgress(context.Background(), OnboardingIdentity{TenantID: "tenant-1", Role: "owner"}, "owner-onboarding", 1)
	if !errors.Is(err, ErrOnboardingMissingAuth) {
		t.Fatalf("missing auth error = %v, want %v", err, ErrOnboardingMissingAuth)
	}

	_, err = service.Complete(context.Background(), OnboardingIdentity{TenantID: "tenant-1", UserID: "user-1", Role: "cashier"}, "owner-onboarding", 1)
	if !errors.Is(err, ErrOnboardingInvalidTour) {
		t.Fatalf("tour mismatch error = %v, want %v", err, ErrOnboardingInvalidTour)
	}

	assertOnboardingExpectations(t, mock)
}

func newOnboardingMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return db, mock
}

func assertOnboardingExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
