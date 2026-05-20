package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/labstack/echo/v4"
)

func TestLoadTenantHealthScansCommandCenterCounts(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`(?s)SELECT.*registered_tenants.*FROM tenants`).
		WillReturnRows(sqlmock.NewRows([]string{
			"registered_tenants",
			"active_tenants",
			"trial_tenants",
			"grace_period_tenants",
			"will_expire_7",
			"will_expire_14",
			"will_expire_30",
			"expired_tenants",
			"cancelled_tenants",
			"suspended_tenants",
			"inactive_tenants",
			"scheduled_deletes",
		}).AddRow(20, 12, 3, 2, 1, 4, 6, 2, 1, 1, 2, 1))

	svc := &PlatformService{db: db}
	health, err := svc.loadTenantHealth(context.Background())
	if err != nil {
		t.Fatalf("loadTenantHealth returned error: %v", err)
	}
	if health.RegisteredTenants != 20 || health.ActiveTenants != 12 || health.WillExpire14 != 4 || health.ExpiredTenants != 2 {
		t.Fatalf("unexpected health counts: %+v", health)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestLoadRevenueSummaryCountsPaidIncomeSeparately(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 20, 23, 59, 59, 0, time.UTC)

	mock.ExpectQuery(`(?s)SELECT.*FILTER.*FROM billing_invoices`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"paid", "pending", "expired", "overdue"}).AddRow(150000, 70000, 20000, 30000))
	mock.ExpectQuery(`(?s)SELECT COALESCE\(SUM\(amount_idr\), 0\).*WHERE status = 'paid' AND paid_at >=`).
		WithArgs(sqlmock.AnyArg(), start).
		WillReturnRows(sqlmock.NewRows([]string{"previous"}).AddRow(100000))
	mock.ExpectQuery(`(?s)SELECT status, COUNT\(\*\).*GROUP BY status`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).AddRow("paid", 3).AddRow("pending", 2))
	mock.ExpectQuery(`(?s)SELECT billing_interval, COUNT\(\*\).*GROUP BY billing_interval`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"billing_interval", "count"}).AddRow("monthly", 4).AddRow("annual", 1))
	mock.ExpectQuery(`(?s)SELECT DATE_TRUNC\('day', paid_at\).*GROUP BY day`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"day", "sum", "count"}).AddRow(start, 150000, 3))
	mock.ExpectQuery(`(?s)SELECT t.id, t.business_name.*FROM billing_invoices bi.*JOIN tenants`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{"tenant_id", "business_name", "paid", "count"}).AddRow("tenant-1", "Tenant One", 150000, 3))
	mock.ExpectQuery(`(?s)SELECT bi.id, bi.tenant_id.*WHERE bi.status = 'pending' AND bi.due_at < NOW\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"tenant_id",
			"business_name",
			"invoice_number",
			"amount_idr",
			"billing_interval",
			"period_start",
			"period_end",
			"due_at",
			"status",
			"paid_at",
			"midtrans_order_id",
			"midtrans_payment_url",
			"payment_link_expires_at",
			"created_at",
			"updated_at",
		}))

	svc := &PlatformService{db: db}
	summary, err := svc.loadRevenueSummary(context.Background(), start, end)
	if err != nil {
		t.Fatalf("loadRevenueSummary returned error: %v", err)
	}
	if summary.PaidIncomeIDR != 150000 || summary.PendingIncomeIDR != 70000 || summary.OverdueIncomeIDR != 30000 {
		t.Fatalf("unexpected revenue totals: %+v", summary)
	}
	if summary.PeriodDeltaPercent == nil || *summary.PeriodDeltaPercent != 50 {
		t.Fatalf("period delta = %v, want 50", summary.PeriodDeltaPercent)
	}
	if summary.InvoiceCountByStatus["paid"] != 3 || summary.BillingCycleMix["monthly"] != 4 || len(summary.Timeseries) != 1 {
		t.Fatalf("unexpected revenue breakdown: %+v", summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestTenantActionRejectsMissingReasonBeforeDatabaseWork(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/tenants/tenant-1/actions/suspend", bytes.NewBufferString(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-Platform-Admin-ID", "admin-1")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("tenant_id", "action")
	c.SetParamValues("tenant-1", "suspend")

	svc := &PlatformService{}
	if err := svc.TenantAction(c); err != nil {
		t.Fatalf("TenantAction returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestListAuditEventsReturnsImmutableAuditRows(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	occurredAt := time.Date(2026, 5, 20, 3, 5, 7, 0, time.UTC)
	mock.ExpectQuery(`(?s)WITH combined_events AS .*FROM audit_events ae.*LOWER\(ae\.action\) = LOWER\(\$3\)`).
		WithArgs("", "", "", 100, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"type",
			"action",
			"actor_type",
			"resource",
			"reason",
			"before_status",
			"after_status",
			"occurred_at",
			"delete_after",
		}).AddRow("audit", "LOGIN", "user", "Tenant One - authentication:user-1", nil, nil, nil, occurredAt, nil))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/audit-events", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := &PlatformService{db: db}
	if err := svc.ListAuditEvents(c); err != nil {
		t.Fatalf("ListAuditEvents returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Events []ActivityItem `json:"events"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Events) != 1 || body.Events[0].Type != "audit" || body.Events[0].Action != "LOGIN" {
		t.Fatalf("unexpected audit events: %+v", body.Events)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestListAuditEventsUsesCaseInsensitiveActionFilter(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`(?s)LOWER\(tle\.action\) = LOWER\(\$3\).*LOWER\(ae\.action\) = LOWER\(\$3\)`).
		WithArgs("", "", "login", 100, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"type",
			"action",
			"actor_type",
			"resource",
			"reason",
			"before_status",
			"after_status",
			"occurred_at",
			"delete_after",
		}).AddRow("audit", "LOGIN", "user", "authentication:user-1", nil, nil, nil, time.Now().UTC(), nil))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/audit-events?action=login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	svc := &PlatformService{db: db}
	if err := svc.ListAuditEvents(c); err != nil {
		t.Fatalf("ListAuditEvents returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetTenantReturnsEmptyCollectionsWhenOptionalQueriesFail(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)SELECT t\.id, t\.business_name.*FROM tenants t.*WHERE t\.id = \$1`).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"business_name",
			"slug",
			"status",
			"owner_email",
			"subscription_plan",
			"billing_cycle",
			"subscription_status",
			"trial_ends_at",
			"subscription_ends_at",
			"expires_at",
			"storage_used_bytes",
			"storage_quota_bytes",
			"user_count",
			"active_user_count",
			"paid_invoice_total_idr",
			"open_ticket_count",
			"last_active_at",
			"created_at",
			"updated_at",
			"suspended_at",
			"deactivated_at",
			"scheduled_delete_at",
			"status_reason",
		}).AddRow(
			"tenant-1", "Tenant One", "tenant-one", "active", "owner@example.com",
			"starter", "monthly", "active", nil, now, now,
			int64(0), int64(2147483648), int64(1), int64(1), int64(299000), int64(0), now,
			now, now, nil, nil, nil, nil,
		))
	mock.ExpectQuery(`(?s)FROM tenant_lifecycle_events`).WithArgs("tenant-1", 20).WillReturnError(errors.New("optional activity unavailable"))
	mock.ExpectQuery(`(?s)FROM billing_invoices bi`).WithArgs("tenant-1").WillReturnError(errors.New("optional billing unavailable"))
	mock.ExpectQuery(`(?s)FROM support_tickets st`).WithArgs("tenant-1", "").WillReturnError(errors.New("optional tickets unavailable"))
	mock.ExpectQuery(`(?s)FROM platform_tenant_notes`).WithArgs("tenant-1", 20).WillReturnError(errors.New("optional notes unavailable"))

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/tenants/tenant-1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("tenant_id")
	c.SetParamValues("tenant-1")

	svc := &PlatformService{db: db}
	if err := svc.GetTenant(c); err != nil {
		t.Fatalf("GetTenant returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body PlatformTenantDetail
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Activity == nil || body.Tickets == nil || body.Notes == nil || body.Billing.Invoices == nil || body.Billing.PaymentAttempts == nil {
		t.Fatalf("expected empty non-nil collections, got %+v", body)
	}
	if len(body.Activity) != 0 || len(body.Tickets) != 0 || len(body.Notes) != 0 || len(body.Billing.Invoices) != 0 || len(body.Billing.PaymentAttempts) != 0 {
		t.Fatalf("expected empty collections, got %+v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestParseDateRangeSupportsYearToDate(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/analytics/revenue?range=year", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	before := time.Now().UTC()
	start, end, err := parseDateRange(c)
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("parseDateRange returned error: %v", err)
	}
	wantStart := time.Date(before.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(wantStart) {
		t.Fatalf("start = %s, want %s", start, wantStart)
	}
	if end.Before(before) || end.After(after.Add(time.Second)) {
		t.Fatalf("end = %s, want current time between %s and %s", end, before, after)
	}
}

func TestTenantCommandCenterHelpers(t *testing.T) {
	if tenantSortClause("-paid_total") != "paid_invoice_total_idr DESC" {
		t.Fatal("unexpected paid total sort clause")
	}
	if tenantSortClause("unsafe") != "t.created_at DESC" {
		t.Fatal("unsafe sort should fall back to created_at desc")
	}
	if !validSubscriptionPlan("professional") || validSubscriptionPlan("free") {
		t.Fatal("subscription plan validation failed")
	}
	if !validBillingCycle("annual") || validBillingCycle("weekly") {
		t.Fatal("billing cycle validation failed")
	}
	if got := maskEmail("owner@example.com"); got != "ow***@example.com" {
		t.Fatalf("maskEmail = %q", got)
	}
}
