package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestAuditContextFromRequest(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPatch, "/team/users/user-1", nil)
	req.Header.Set("X-User-ID", "actor-1")
	req.Header.Set("X-User-Email", "owner@example.com")
	req.Header.Set("X-User-Role", "owner")
	req.Header.Set("X-Forwarded-For", "203.0.113.10")
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Request-ID", "req-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	auditCtx := auditContextFromRequest(c)

	if auditCtx.ActorID != "actor-1" {
		t.Fatalf("ActorID = %s, want actor-1", auditCtx.ActorID)
	}
	if auditCtx.ActorEmail != "owner@example.com" {
		t.Fatalf("ActorEmail = %s, want owner@example.com", auditCtx.ActorEmail)
	}
	if auditCtx.ActorRole != "owner" {
		t.Fatalf("ActorRole = %s, want owner", auditCtx.ActorRole)
	}
	if auditCtx.IPAddress != "203.0.113.10" {
		t.Fatalf("IPAddress = %s, want 203.0.113.10", auditCtx.IPAddress)
	}
	if auditCtx.UserAgent != "test-agent" {
		t.Fatalf("UserAgent = %s, want test-agent", auditCtx.UserAgent)
	}
	if auditCtx.RequestID != "req-1" {
		t.Fatalf("RequestID = %s, want req-1", auditCtx.RequestID)
	}
}
