package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRolePoliciesForRoleAwareRoutes(t *testing.T) {
	tests := []struct {
		name         string
		role         Role
		allowedRoles []Role
		wantStatus   int
	}{
		{
			name:         "cashier can read products for offline orders",
			role:         RoleCashier,
			allowedRoles: []Role{RoleOwner, RoleManager, RoleCashier},
			wantStatus:   http.StatusNoContent,
		},
		{
			name:         "cashier cannot mutate products",
			role:         RoleCashier,
			allowedRoles: []Role{RoleOwner, RoleManager},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "cashier can read operational analytics tasks",
			role:         RoleCashier,
			allowedRoles: []Role{RoleOwner, RoleManager, RoleCashier},
			wantStatus:   http.StatusNoContent,
		},
		{
			name:         "cashier cannot read business analytics",
			role:         RoleCashier,
			allowedRoles: []Role{RoleOwner, RoleManager},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "cashier can read billing subscription and invoices",
			role:         RoleCashier,
			allowedRoles: []Role{RoleOwner, RoleManager, RoleCashier},
			wantStatus:   http.StatusNoContent,
		},
		{
			name:         "cashier cannot manage billing subscription",
			role:         RoleCashier,
			allowedRoles: []Role{RoleOwner, RoleManager},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "owner can manage tenant Midtrans config",
			role:         RoleOwner,
			allowedRoles: []Role{RoleOwner},
			wantStatus:   http.StatusNoContent,
		},
		{
			name:         "manager cannot manage tenant Midtrans config",
			role:         RoleManager,
			allowedRoles: []Role{RoleOwner},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "manager cannot access owner-only admin tenant routes",
			role:         RoleManager,
			allowedRoles: []Role{RoleOwner},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "owner can manage team members",
			role:         RoleOwner,
			allowedRoles: []Role{RoleOwner, RoleManager},
			wantStatus:   http.StatusNoContent,
		},
		{
			name:         "manager can manage allowed team member routes",
			role:         RoleManager,
			allowedRoles: []Role{RoleOwner, RoleManager},
			wantStatus:   http.StatusNoContent,
		},
		{
			name:         "cashier cannot manage team member routes",
			role:         RoleCashier,
			allowedRoles: []Role{RoleOwner, RoleManager},
			wantStatus:   http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := callRBACMiddleware(t, tt.role, tt.allowedRoles...)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func callRBACMiddleware(t *testing.T, role Role, allowedRoles ...Role) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("role", string(role))

	handler := RBACMiddleware(allowedRoles...)(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	if err := handler(c); err != nil {
		t.Fatalf("RBAC middleware returned error: %v", err)
	}

	return rec
}
