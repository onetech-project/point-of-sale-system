package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/middleware"
)

func TestCompletedOrderCorrectionRoutesAreOwnerOnly(t *testing.T) {
	var forwarded []string
	orderService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded = append(forwarded, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer orderService.Close()

	newGateway := func(role middleware.Role) *echo.Echo {
		e := echo.New()
		protected := e.Group("")
		protected.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Set("role", string(role))
				c.Set("tenant_id", "tenant-1")
				return next(c)
			}
		})
		registerAdminOrderRoutes(protected, orderService.URL)
		return e
	}

	const path = "/api/v1/admin/offline-orders/order-1/completed"
	tests := []struct {
		role   middleware.Role
		method string
		want   int
	}{
		{middleware.RoleOwner, http.MethodPatch, http.StatusOK},
		{middleware.RoleOwner, http.MethodDelete, http.StatusOK},
		{middleware.RoleManager, http.MethodPatch, http.StatusForbidden},
		{middleware.RoleManager, http.MethodDelete, http.StatusForbidden},
		{middleware.RoleCashier, http.MethodPatch, http.StatusForbidden},
		{middleware.RoleCashier, http.MethodDelete, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(string(tt.role)+" "+tt.method, func(t *testing.T) {
			forwarded = nil
			rec := httptest.NewRecorder()
			newGateway(tt.role).ServeHTTP(rec, httptest.NewRequest(tt.method, path, nil))

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.want, rec.Body.String())
			}
			if tt.want == http.StatusOK && (len(forwarded) != 1 || forwarded[0] != tt.method+" "+path) {
				t.Fatalf("forwarded = %v, want %s %s", forwarded, tt.method, path)
			}
			if tt.want != http.StatusOK && len(forwarded) != 0 {
				t.Fatalf("forbidden request was forwarded: %v", forwarded)
			}
		})
	}

	// Other offline order routes stay open to managers
	forwarded = nil
	rec := httptest.NewRecorder()
	newGateway(middleware.RoleManager).ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/api/v1/admin/offline-orders/order-1", nil))
	if rec.Code != http.StatusOK || len(forwarded) != 1 {
		t.Fatalf("manager PATCH offline order status = %d, forwarded = %v", rec.Code, forwarded)
	}
}

func TestOrderRefundRouteIsOwnerOnly(t *testing.T) {
	var forwarded []string
	orderService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded = append(forwarded, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer orderService.Close()

	newGateway := func(role middleware.Role) *echo.Echo {
		e := echo.New()
		protected := e.Group("")
		protected.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Set("role", string(role))
				c.Set("tenant_id", "tenant-1")
				return next(c)
			}
		})
		registerAdminOrderRoutes(protected, orderService.URL)
		return e
	}

	const path = "/api/v1/admin/orders/order-1/refund"
	tests := []struct {
		role middleware.Role
		want int
	}{
		{middleware.RoleOwner, http.StatusOK},
		{middleware.RoleManager, http.StatusForbidden},
		{middleware.RoleCashier, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			forwarded = nil
			rec := httptest.NewRecorder()
			newGateway(tt.role).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.want, rec.Body.String())
			}
			if tt.want != http.StatusOK && len(forwarded) != 0 {
				t.Fatalf("forbidden request was forwarded: %v", forwarded)
			}
		})
	}

	// Other order routes stay open to cashiers
	forwarded = nil
	rec := httptest.NewRecorder()
	newGateway(middleware.RoleCashier).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders/order-1", nil))
	if rec.Code != http.StatusOK || len(forwarded) != 1 {
		t.Fatalf("cashier GET order status = %d, forwarded = %v", rec.Code, forwarded)
	}
}
