package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestTenantStatusEnforcerBlocksSuspendedTenant(t *testing.T) {
	enforcer := newTenantStatusTestEnforcer(t, "suspended")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tenant-1")

	nextRan := false
	next := func(c echo.Context) error {
		nextRan = true
		return c.NoContent(http.StatusNoContent)
	}

	if err := enforcer.EnforceTenantAccount()(next)(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
	if nextRan {
		t.Fatal("expected suspended tenant to be blocked")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestTenantStatusEnforcerAllowsInactiveForOffboardingRoutes(t *testing.T) {
	enforcer := newTenantStatusTestEnforcer(t, "inactive")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant/data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("tenant_id", "tenant-1")

	nextRan := false
	next := func(c echo.Context) error {
		nextRan = true
		return c.NoContent(http.StatusNoContent)
	}

	if err := enforcer.AllowTenantStatuses("active", "inactive")(next)(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
	if !nextRan {
		t.Fatal("expected inactive tenant to reach offboarding route")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func newTenantStatusTestEnforcer(t *testing.T, status string) *TenantStatusEnforcer {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/tenants/tenant-1/status" {
			t.Fatalf("unexpected tenant status path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprintf(w, `{"status":%q}`, status)
	}))
	t.Cleanup(server.Close)
	return &TenantStatusEnforcer{
		tenantURL:  server.URL,
		httpClient: server.Client(),
	}
}
