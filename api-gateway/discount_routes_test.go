package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/middleware"
)

func TestDiscountRuleRoutesGoToOrderService(t *testing.T) {
	orderHits := 0
	orderService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orderHits++
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/discount-rules":
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/discount-rules":
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/discount-rules/preview":
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/admin/discount-rules/rule-1":
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/admin/discount-rules/rule-1":
		default:
			t.Fatalf("unexpected order-service request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"order"}`))
	}))
	defer orderService.Close()

	e := echo.New()
	protected := e.Group("")
	protected.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("role", string(middleware.RoleManager))
			c.Set("tenant_id", "tenant-1")
			return next(c)
		}
	})
	adminDiscounts := protected.Group("/api/v1/admin")
	adminDiscounts.Use(middleware.RBACMiddleware(middleware.RoleOwner, middleware.RoleManager))
	adminDiscounts.Any("/discount-rules*", proxyWildcard(orderService.URL))

	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/admin/discount-rules", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/admin/discount-rules", strings.NewReader(`{}`)),
		httptest.NewRequest(http.MethodPost, "/api/v1/admin/discount-rules/preview", strings.NewReader(`{}`)),
		httptest.NewRequest(http.MethodPatch, "/api/v1/admin/discount-rules/rule-1", strings.NewReader(`{}`)),
		httptest.NewRequest(http.MethodDelete, "/api/v1/admin/discount-rules/rule-1", nil),
	}
	for _, req := range requests {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s status = %d, want 200; body=%s", req.Method, req.URL.Path, rec.Code, rec.Body.String())
		}
	}
	if orderHits != len(requests) {
		t.Fatalf("order hits = %d, want %d", orderHits, len(requests))
	}
}
