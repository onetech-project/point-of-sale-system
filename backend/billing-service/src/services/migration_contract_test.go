package services

import (
	"os"
	"strings"
	"testing"
)

func TestBillingInvoiceAmountConstraintMigrationIsRolloutSafe(t *testing.T) {
	upSQL, err := os.ReadFile("../../../migrations/000074_enforce_positive_billing_invoice_amount.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration: %v", err)
	}
	sql := strings.ToLower(string(upSQL))

	if !strings.Contains(sql, "billing_invoices_amount_idr_positive") {
		t.Fatal("migration must define billing_invoices_amount_idr_positive constraint")
	}
	if !strings.Contains(sql, "check (amount_idr > 0)") {
		t.Fatal("migration must check amount_idr > 0")
	}
	if !strings.Contains(sql, "not valid") {
		t.Fatal("migration must use NOT VALID for production rollout safety")
	}
	if strings.Contains(sql, "validate constraint") {
		t.Fatal("migration must not validate existing production rows during rollout")
	}
}
