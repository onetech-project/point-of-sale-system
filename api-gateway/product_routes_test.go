package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/middleware"
)

func TestProductRoutesAllowCashierReadButBlockMutations(t *testing.T) {
	productService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/products" {
			t.Fatalf("path = %s, want /api/v1/products", r.URL.Path)
		}
		if r.URL.Query().Get("include_primary_photo") != "true" {
			t.Fatalf("include_primary_photo query was not forwarded")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer productService.Close()

	e := echo.New()
	protected := e.Group("")
	protected.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("role", string(middleware.RoleCashier))
			c.Set("tenant_id", "tenant-1")
			return next(c)
		}
	})
	registerProductRoutes(protected, productService.URL)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/products?limit=100&include_primary_photo=true", nil)
	getRec := httptest.NewRecorder()
	e.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("cashier GET status = %d, want %d; body=%s", getRec.Code, http.StatusOK, getRec.Body.String())
	}

	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/products", strings.NewReader(`{"name":"Coffee"}`))
	postReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	postRec := httptest.NewRecorder()
	e.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Fatalf("cashier POST status = %d, want %d; body=%s", postRec.Code, http.StatusForbidden, postRec.Body.String())
	}
}
