package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
)

func TestGetInternalTenantStatusIncludesSubscriptionStatus(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT\s+t\.status,\s+t\.subscription_status,`).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"status",
			"subscription_status",
			"midtrans_configured",
			"midtrans_environment",
		}).AddRow("active", "grace_period", false, "sandbox"))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/tenants/tenant-1/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("tenant_id")
	c.SetParamValues("tenant-1")

	handler := NewTenantHandler(db)
	if err := handler.GetInternalTenantStatus(c); err != nil {
		t.Fatalf("GetInternalTenantStatus returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["tenant_id"] != "tenant-1" || body["status"] != "active" || body["subscription_status"] != "grace_period" {
		t.Fatalf("unexpected response: %+v", body)
	}
	if body["midtrans_configured"] != false || body["midtrans_environment"] != "sandbox" {
		t.Fatalf("unexpected midtrans response: %+v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
