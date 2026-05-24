package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestResolveProxyPathReplacesNamedParams(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/invitations/inv-1/revoke", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("inv-1")

	path, query := resolveProxyPath(c, "/invitations/:id/revoke")

	if path != "/invitations/inv-1/revoke" {
		t.Fatalf("path = %s, want /invitations/inv-1/revoke", path)
	}
	if query != "" {
		t.Fatalf("query = %s, want empty", query)
	}
}

func TestResolveProxyPathPreservesEmbeddedQuery(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tenant/users/user-1?force=true", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	path, query := resolveProxyPath(c, "/api/v1/users/user-1?force=true")

	if path != "/api/v1/users/user-1" {
		t.Fatalf("path = %s, want /api/v1/users/user-1", path)
	}
	if query != "force=true" {
		t.Fatalf("query = %s, want force=true", query)
	}
}
