package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type fakeAccountVerificationService struct {
	resendEmail string
	resendErr   error
}

func (f *fakeAccountVerificationService) VerifyAccount(ctx context.Context, token string) error {
	return nil
}

func (f *fakeAccountVerificationService) ResendVerificationEmail(ctx context.Context, email string) error {
	f.resendEmail = email
	return f.resendErr
}

func TestResendVerificationRejectsInvalidEmail(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/resend-verification", strings.NewReader(`{"email":"invalid"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	service := &fakeAccountVerificationService{}
	handler := NewAccountVerificationHandler(service)

	if err := handler.ResendVerification(c); err != nil {
		t.Fatalf("ResendVerification returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if service.resendEmail != "" {
		t.Fatalf("resendEmail = %s, want empty", service.resendEmail)
	}
}

func TestResendVerificationReturnsGenericSuccess(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/resend-verification", strings.NewReader(`{"email":" Owner@Example.COM "}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	service := &fakeAccountVerificationService{}
	handler := NewAccountVerificationHandler(service)

	if err := handler.ResendVerification(c); err != nil {
		t.Fatalf("ResendVerification returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if service.resendEmail != "owner@example.com" {
		t.Fatalf("resendEmail = %s, want owner@example.com", service.resendEmail)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["message"] != "If an unverified account exists, a verification email has been sent." {
		t.Fatalf("message = %q", body["message"])
	}
}

func TestResendVerificationReturnsInternalServerError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/resend-verification", strings.NewReader(`{"email":"owner@example.com"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := NewAccountVerificationHandler(&fakeAccountVerificationService{resendErr: errors.New("database unavailable")})

	if err := handler.ResendVerification(c); err != nil {
		t.Fatalf("ResendVerification returned error: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
