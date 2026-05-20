package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
	"github.com/pos/tenant-service/src/services"
)

func TestGetPublicTenantConfigAvailability(t *testing.T) {
	tests := []struct {
		name               string
		status             string
		subscriptionStatus string
		wantCode           int
		wantMessage        string
	}{
		{name: "active trial returns config", status: "active", subscriptionStatus: "trial", wantCode: http.StatusOK},
		{name: "suspended returns unavailable", status: "suspended", subscriptionStatus: "active", wantCode: http.StatusForbidden, wantMessage: "This tenant is currently not available at this moment."},
		{name: "inactive returns unavailable", status: "inactive", subscriptionStatus: "active", wantCode: http.StatusForbidden, wantMessage: "This tenant is currently not available at this moment."},
		{name: "grace returns unavailable", status: "active", subscriptionStatus: "grace_period", wantCode: http.StatusForbidden, wantMessage: "This tenant is currently not available at this moment."},
		{name: "expired returns unavailable", status: "active", subscriptionStatus: "expired", wantCode: http.StatusForbidden, wantMessage: "This tenant is currently not available at this moment."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
			if err != nil {
				t.Fatalf("failed to create sql mock: %v", err)
			}
			defer db.Close()

			mock.ExpectQuery(`SELECT id, business_name, status, subscription_status FROM tenants WHERE slug = \$1 AND status != 'deleted'`).
				WithArgs("bistro-one").
				WillReturnRows(sqlmock.NewRows([]string{"id", "business_name", "status", "subscription_status"}).AddRow("tenant-1", "Bistro One", tt.status, tt.subscriptionStatus))
			if tt.wantCode == http.StatusOK {
				mock.ExpectQuery(`SELECT delivery_enabled, pickup_enabled, dine_in_enabled,`).
					WithArgs("tenant-1").
					WillReturnRows(sqlmock.NewRows([]string{
						"delivery_enabled",
						"pickup_enabled",
						"dine_in_enabled",
						"default_delivery_fee",
						"min_order_amount",
						"estimated_prep_time",
						"charge_delivery_fee",
					}))
			}

			rec := callGetPublicTenantConfig(t, db, "bistro-one")
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantCode, rec.Body.String())
			}
			if tt.wantMessage != "" {
				var body map[string]string
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if body["message"] != tt.wantMessage || body["status"] != tt.status || body["subscription_status"] != tt.subscriptionStatus {
					t.Fatalf("unexpected body: %+v", body)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet sql expectations: %v", err)
			}
		})
	}
}

func TestGetPublicTenantConfigReturnsNotFoundForUnknownTenant(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, business_name, status, subscription_status FROM tenants WHERE slug = \$1 AND status != 'deleted'`).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	rec := callGetPublicTenantConfig(t, db, "missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func callGetPublicTenantConfig(t *testing.T, db *sql.DB, slug string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/public/tenants/"+slug+"/config", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("tenant_slug")
	c.SetParamValues(slug)

	handler := &TenantConfigHandler{configService: services.NewTenantConfigService(nil, db)}
	if err := handler.GetPublicTenantConfig(c); err != nil {
		t.Fatalf("GetPublicTenantConfig returned error: %v", err)
	}
	return rec
}
