package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestPublicPlansRouteProxiesToBillingService(t *testing.T) {
	var sawBillingRequest bool
	var gotMethod, gotPath string
	billingService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawBillingRequest = true
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"monthly_price_idr":500000,"annual_discount_pct":25,"annual_price_idr":4500000,"trial_days":14}`))
	}))
	defer billingService.Close()

	e := echo.New()
	registerPublicPlansRoute(e.Group(""), billingService.URL)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/plans", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !sawBillingRequest {
		t.Fatal("billing service did not receive request")
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/public/plans" {
		t.Fatalf("path = %s, want /public/plans", gotPath)
	}
	if got := rec.Body.String(); got != `{"monthly_price_idr":500000,"annual_discount_pct":25,"annual_price_idr":4500000,"trial_days":14}` {
		t.Fatalf("body = %s", got)
	}
}
