package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestGetPublicPlanUsesFallbacks(t *testing.T) {
	t.Setenv("PLAN_MONTHLY_PRICE_IDR", "")
	t.Setenv("PLAN_ANNUAL_DISCOUNT_PCT", "")
	t.Setenv("PLAN_TRIAL_DAYS", "")

	body := callGetPublicPlan(t)

	assertNumber(t, body, "monthly_price_idr", 299000)
	assertNumber(t, body, "annual_discount_pct", 20)
	assertNumber(t, body, "annual_price_idr", 2870400)
	assertNumber(t, body, "trial_days", 7)
}

func TestGetPublicPlanUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("PLAN_MONTHLY_PRICE_IDR", "500000")
	t.Setenv("PLAN_ANNUAL_DISCOUNT_PCT", "25")
	t.Setenv("PLAN_TRIAL_DAYS", "14")

	body := callGetPublicPlan(t)

	assertNumber(t, body, "monthly_price_idr", 500000)
	assertNumber(t, body, "annual_discount_pct", 25)
	assertNumber(t, body, "annual_price_idr", 4500000)
	assertNumber(t, body, "trial_days", 14)
}

func callGetPublicPlan(t *testing.T) map[string]interface{} {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/public/plans", nil)
	rec := httptest.NewRecorder()

	if err := GetPublicPlan(e.NewContext(req, rec)); err != nil {
		t.Fatalf("GetPublicPlan returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return body
}

func assertNumber(t *testing.T, body map[string]interface{}, key string, want float64) {
	t.Helper()

	got, ok := body[key].(float64)
	if !ok {
		t.Fatalf("%s is %T, want number", key, body[key])
	}
	if got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}
