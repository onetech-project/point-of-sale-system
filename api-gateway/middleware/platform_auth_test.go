package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

func TestPlatformAdminAuthSetsPlatformContext(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	token := signPlatformAuthTestToken(t, "test-secret", PlatformJWTClaims{
		SessionID:       "session-1",
		PlatformAdminID: "admin-1",
		Email:           "owner@example.com",
		Role:            "platform_owner",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/analytics/overview", nil)
	req.AddCookie(&http.Cookie{Name: "platform_auth_token", Value: token})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextRan := false
	next := func(c echo.Context) error {
		nextRan = true
		if got := c.Get("platform_admin_id"); got != "admin-1" {
			t.Fatalf("platform_admin_id = %v, want admin-1", got)
		}
		if got := c.Get("platform_session_id"); got != "session-1" {
			t.Fatalf("platform_session_id = %v, want session-1", got)
		}
		return c.NoContent(http.StatusNoContent)
	}

	if err := PlatformAdminAuth()(next)(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
	if !nextRan {
		t.Fatal("expected platform auth middleware to call next")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestPlatformAdminAuthRejectsMissingCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/analytics/overview", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	next := func(c echo.Context) error {
		t.Fatal("next should not run")
		return nil
	}
	if err := PlatformAdminAuth()(next)(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func signPlatformAuthTestToken(t *testing.T, secret string, claims PlatformJWTClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return token
}
