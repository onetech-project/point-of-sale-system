package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestTenantStatusEnforcerBlocksSuspendedTenant(t *testing.T) {
	enforcer := newTenantStatusTestEnforcer(t, "suspended", "active")

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
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "suspended" {
		t.Fatalf("status payload = %q, want suspended", body["status"])
	}
}

func TestTenantStatusEnforcerAllowsInactiveForOffboardingRoutes(t *testing.T) {
	enforcer := newTenantStatusTestEnforcer(t, "inactive", "active")

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

func TestTenantStatusEnforcerBlocksInactiveTenant(t *testing.T) {
	enforcer := newTenantStatusTestEnforcer(t, "inactive", "active")

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
		t.Fatal("expected inactive tenant to be blocked")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "inactive" {
		t.Fatalf("status payload = %q, want inactive", body["status"])
	}
}

func TestTenantStatusEnforcerPublicAvailability(t *testing.T) {
	tests := []struct {
		name               string
		status             string
		subscriptionStatus string
		wantCode           int
		nextShouldRun      bool
	}{
		{name: "active trial passes", status: "active", subscriptionStatus: "trial", wantCode: http.StatusNoContent, nextShouldRun: true},
		{name: "active subscription passes", status: "active", subscriptionStatus: "active", wantCode: http.StatusNoContent, nextShouldRun: true},
		{name: "suspended blocks", status: "suspended", subscriptionStatus: "active", wantCode: http.StatusForbidden},
		{name: "inactive blocks", status: "inactive", subscriptionStatus: "active", wantCode: http.StatusForbidden},
		{name: "grace period blocks", status: "active", subscriptionStatus: "grace_period", wantCode: http.StatusForbidden},
		{name: "expired blocks", status: "active", subscriptionStatus: "expired", wantCode: http.StatusForbidden},
		{name: "cancelled blocks", status: "active", subscriptionStatus: "cancelled", wantCode: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enforcer := newTenantStatusTestEnforcer(t, tt.status, tt.subscriptionStatus)

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/public/menu/tenant-1/products", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("tenant_id")
			c.SetParamValues("tenant-1")

			nextRan := false
			next := func(c echo.Context) error {
				nextRan = true
				return c.NoContent(http.StatusNoContent)
			}

			if err := enforcer.RequirePublicTenantAvailableParam("tenant_id")(next)(c); err != nil {
				t.Fatalf("middleware returned error: %v", err)
			}
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantCode, rec.Body.String())
			}
			if nextRan != tt.nextShouldRun {
				t.Fatalf("nextRan = %v, want %v", nextRan, tt.nextShouldRun)
			}
			if tt.wantCode == http.StatusForbidden {
				var body map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if body["message"] != "This tenant is currently not available at this moment." {
					t.Fatalf("message = %q", body["message"])
				}
				if body["status"] != tt.status || body["subscription_status"] != tt.subscriptionStatus {
					t.Fatalf("unexpected availability payload: %+v", body)
				}
			}
		})
	}
}

func newTenantStatusTestEnforcer(t *testing.T, status, subscriptionStatus string) *TenantStatusEnforcer {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/tenants/tenant-1/status" {
			t.Fatalf("unexpected tenant status path: %s", r.URL.Path)
		}
		_, _ = fmt.Fprintf(w, `{"status":%q,"subscription_status":%q}`, status, subscriptionStatus)
	}))
	t.Cleanup(server.Close)
	return &TenantStatusEnforcer{
		tenantURL:  server.URL,
		httpClient: server.Client(),
	}
}
