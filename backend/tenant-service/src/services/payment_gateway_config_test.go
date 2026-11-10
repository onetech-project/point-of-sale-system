package services

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/pos/tenant-service/src/repository"
)

// prefixEncryptor "encrypts" by prefixing the column context, so tests can
// check which value went to which column.
type prefixEncryptor struct{}

func (prefixEncryptor) Encrypt(_ context.Context, p string) (string, error) { return "enc:" + p, nil }
func (prefixEncryptor) EncryptWithContext(_ context.Context, p, c string) (string, error) {
	return "enc:" + c + ":" + p, nil
}
func (prefixEncryptor) Decrypt(_ context.Context, c string) (string, error) {
	return strings.TrimPrefix(c, "enc:"), nil
}
func (prefixEncryptor) DecryptWithContext(_ context.Context, c, ctxName string) (string, error) {
	return strings.TrimPrefix(c, "enc:"+ctxName+":"), nil
}
func (prefixEncryptor) EncryptBatch(_ context.Context, p []string) ([]string, error) { return p, nil }
func (prefixEncryptor) DecryptBatch(_ context.Context, c []string) ([]string, error) { return c, nil }

var tenantConfigColumns = []string{
	"tenant_id", "enabled_delivery_types", "service_area", "delivery_fee_config", "auto_calculate_fees",
	"midtrans_server_key", "midtrans_client_key", "midtrans_merchant_id", "midtrans_environment",
	"payment_gateway", "manjo_client_id", "manjo_client_secret",
	"manjo_merchant_id", "manjo_partner_id", "manjo_environment", "created_at", "updated_at",
}

type storedConfig struct {
	midtransServerKey, midtransClientKey string
	gateway                              string
	manjoClientID, manjoSecret           string
	manjoMerchantID                      string
}

func expectGetConfig(mock sqlmock.Sqlmock, c storedConfig) {
	enc := func(column, v string) string {
		if v == "" {
			return ""
		}
		return "enc:tenant_config:" + column + ":" + v
	}
	if c.gateway == "" {
		c.gateway = "midtrans"
	}
	mock.ExpectQuery(`FROM tenant_configs`).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows(tenantConfigColumns).AddRow(
			"tenant-1", "{pickup}", []byte(`{}`), []byte(`{}`), false,
			enc("midtrans_server_key", c.midtransServerKey), enc("midtrans_client_key", c.midtransClientKey), "", "sandbox",
			c.gateway, c.manjoClientID, enc("manjo_client_secret", c.manjoSecret),
			c.manjoMerchantID, "", "sandbox", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z",
		))
}

func newTestConfigService(t *testing.T) (*TenantConfigService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	repo := repository.NewTenantConfigRepository(db, prefixEncryptor{}, nil)
	return NewTenantConfigService(repo, db), mock
}

// anyArgs matches n arbitrary query arguments.
func anyArgs(n int) []driver.Value {
	args := make([]driver.Value, n)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	return args
}

func TestSetDefaultPaymentGatewayRejectsUnconfiguredGateway(t *testing.T) {
	svc, mock := newTestConfigService(t)
	expectGetConfig(mock, storedConfig{midtransServerKey: "srv", midtransClientKey: "cli"})

	err := svc.SetDefaultPaymentGateway(context.Background(), "tenant-1", "manjo")
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestSetDefaultPaymentGatewayRejectsUnknownGateway(t *testing.T) {
	svc, _ := newTestConfigService(t)
	err := svc.SetDefaultPaymentGateway(context.Background(), "tenant-1", "xendit")
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestSetDefaultPaymentGatewaySwitchesToConfiguredManjo(t *testing.T) {
	svc, mock := newTestConfigService(t)
	expectGetConfig(mock, storedConfig{
		midtransServerKey: "srv", midtransClientKey: "cli",
		manjoClientID: "EP1", manjoSecret: "secret", manjoMerchantID: "MT1",
	})
	args := anyArgs(15)
	args[9] = "manjo" // payment_gateway
	mock.ExpectExec(`UPDATE tenant_configs`).WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.SetDefaultPaymentGateway(context.Background(), "tenant-1", "manjo"); err != nil {
		t.Fatalf("SetDefaultPaymentGateway returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUpdateManjoConfigKeepsStoredSecretsWhenBlank(t *testing.T) {
	svc, mock := newTestConfigService(t)
	expectGetConfig(mock, storedConfig{
		manjoClientID: "EP1", manjoSecret: "old-secret", manjoMerchantID: "MT1",
	})
	args := anyArgs(15)
	args[10] = "EP2"                                              // manjo_client_id
	args[11] = "enc:tenant_config:manjo_client_secret:old-secret" // re-encrypted stored secret
	args[12] = "MT2"
	mock.ExpectExec(`UPDATE tenant_configs`).WithArgs(args...).WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.UpdateManjoConfig(context.Background(), &UpdateManjoConfigRequest{
		TenantID: "tenant-1", ClientID: "EP2", MerchantID: "MT2", Environment: "sandbox",
	})
	if err != nil {
		t.Fatalf("UpdateManjoConfig returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUpdateManjoConfigRequiresSecretsOnFirstSave(t *testing.T) {
	svc, mock := newTestConfigService(t)
	expectGetConfig(mock, storedConfig{})

	err := svc.UpdateManjoConfig(context.Background(), &UpdateManjoConfigRequest{
		TenantID: "tenant-1", ClientID: "EP1", MerchantID: "MT1", Environment: "sandbox",
	})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("err = %v, want ValidationError for missing client secret", err)
	}
}

func TestGetManjoCredentialsFallsBackToClientIDForPartnerID(t *testing.T) {
	svc, mock := newTestConfigService(t)
	expectGetConfig(mock, storedConfig{
		manjoClientID: "EP1", manjoSecret: "secret", manjoMerchantID: "MT1",
	})

	creds, err := svc.GetManjoCredentials(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("GetManjoCredentials returned error: %v", err)
	}
	if creds.PartnerID != "EP1" || creds.ClientSecret != "secret" || !creds.IsConfigured {
		t.Fatalf("unexpected credentials: %+v", creds)
	}
}

func TestGetManjoConfigNeverReturnsSecrets(t *testing.T) {
	svc, mock := newTestConfigService(t)
	expectGetConfig(mock, storedConfig{
		manjoClientID: "EP1", manjoSecret: "secret", manjoMerchantID: "MT1",
	})

	config, err := svc.GetManjoConfig(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("GetManjoConfig returned error: %v", err)
	}
	if !config.HasClientSecret || !config.IsConfigured {
		t.Fatalf("unexpected config: %+v", config)
	}
}
