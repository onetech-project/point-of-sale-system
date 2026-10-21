package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	customMiddleware "github.com/point-of-sale-system/order-service/src/middleware"
	"github.com/point-of-sale-system/order-service/src/services"
)

func newCompletedOrderRoutes() *echo.Echo {
	e := echo.New()
	noop := func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	requireRole := func(roles ...string) echo.MiddlewareFunc {
		allowed := make([]customMiddleware.Role, len(roles))
		for i, role := range roles {
			allowed[i] = customMiddleware.Role(role)
		}
		return customMiddleware.RequireRole(allowed...)
	}
	// No dependencies are needed: every request below is rejected before the database is reached
	service := services.NewOfflineOrderService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	RegisterOfflineOrderRoutes(e, NewOfflineOrderHandler(service), noop, requireRole, noop)
	return e
}

func TestCompletedOrderCorrectionRoutesAreOwnerOnly(t *testing.T) {
	e := newCompletedOrderRoutes()

	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		for _, role := range []string{"manager", "cashier", ""} {
			t.Run(method+" as "+role, func(t *testing.T) {
				req := httptest.NewRequest(method, "/api/v1/admin/offline-orders/order-1/completed",
					strings.NewReader(`{"reason":"Salah input","acknowledge_accounting_impact":true}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("X-Tenant-ID", "tenant-1")
				req.Header.Set("X-User-ID", "user-1")
				if role != "" {
					req.Header.Set("X-User-Role", role)
				}
				rec := httptest.NewRecorder()

				e.ServeHTTP(rec, req)

				want := http.StatusForbidden
				if role == "" {
					want = http.StatusUnauthorized
				}
				if rec.Code != want {
					t.Fatalf("status = %d, want %d; body=%s", rec.Code, want, rec.Body.String())
				}
			})
		}
	}
}

func TestCompletedOrderCorrectionRequiresAcknowledgement(t *testing.T) {
	e := newCompletedOrderRoutes()

	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/v1/admin/offline-orders/order-1/completed",
				strings.NewReader(`{"reason":"Salah input","customer_name":"Budi"}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			req.Header.Set("X-Tenant-ID", "tenant-1")
			req.Header.Set("X-User-ID", "user-1")
			req.Header.Set("X-User-Role", "Owner")
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "acknowledge_accounting_impact") {
				t.Fatalf("body = %s, want acknowledgement error", rec.Body.String())
			}
		})
	}
}
