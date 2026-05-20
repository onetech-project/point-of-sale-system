package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetDeliveryConfigAllowsActiveTrialTenant(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, business_name, status, subscription_status FROM tenants WHERE slug = \$1 AND status != 'deleted'`).
		WithArgs("bistro-one").
		WillReturnRows(sqlmock.NewRows([]string{"id", "business_name", "status", "subscription_status"}).AddRow("tenant-1", "Bistro One", "active", "trial"))
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

	svc := NewTenantConfigService(nil, db)
	config, err := svc.GetDeliveryConfig(context.Background(), "bistro-one")
	if err != nil {
		t.Fatalf("GetDeliveryConfig returned error: %v", err)
	}
	if config.TenantID != "tenant-1" || config.TenantName != "Bistro One" || len(config.EnabledDeliveryTypes) != 3 {
		t.Fatalf("unexpected config: %+v", config)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetDeliveryConfigRejectsUnavailableTenants(t *testing.T) {
	tests := []struct {
		name               string
		status             string
		subscriptionStatus string
	}{
		{name: "suspended", status: "suspended", subscriptionStatus: "active"},
		{name: "inactive", status: "inactive", subscriptionStatus: "active"},
		{name: "grace period", status: "active", subscriptionStatus: "grace_period"},
		{name: "expired", status: "active", subscriptionStatus: "expired"},
		{name: "cancelled", status: "active", subscriptionStatus: "cancelled"},
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

			svc := NewTenantConfigService(nil, db)
			config, err := svc.GetDeliveryConfig(context.Background(), "bistro-one")
			if config != nil {
				t.Fatalf("config = %+v, want nil", config)
			}
			var unavailable *TenantUnavailableError
			if !errors.As(err, &unavailable) {
				t.Fatalf("err = %v, want TenantUnavailableError", err)
			}
			if unavailable.Status != tt.status || unavailable.SubscriptionStatus != tt.subscriptionStatus {
				t.Fatalf("unavailable = %+v", unavailable)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet sql expectations: %v", err)
			}
		})
	}
}

func TestGetDeliveryConfigReturnsNotFoundForUnknownOrDeletedTenant(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, business_name, status, subscription_status FROM tenants WHERE slug = \$1 AND status != 'deleted'`).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	svc := NewTenantConfigService(nil, db)
	_, err = svc.GetDeliveryConfig(context.Background(), "missing")
	if !errors.Is(err, ErrTenantNotFound) {
		t.Fatalf("err = %v, want ErrTenantNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
