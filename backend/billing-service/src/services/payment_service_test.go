package services

import (
	"strings"
	"testing"
	"time"

	"github.com/pos/billing-service/src/models"
)

func TestBuildSubscriptionItemMatchesInvoiceAmount(t *testing.T) {
	inv := &models.BillingInvoice{
		InvoiceNumber: "INV-202605-000001",
		AmountIDR:     299000,
		PeriodStart:   time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
		PeriodEnd:     time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC),
	}

	item := buildSubscriptionItem(inv, "Monthly")

	if item.Price*int64(item.Qty) != int64(inv.AmountIDR) {
		t.Fatalf("item total = %d, want %d", item.Price*int64(item.Qty), inv.AmountIDR)
	}
	if len([]rune(item.Name)) > midtransItemNameMaxLength {
		t.Fatalf("item name length = %d, want <= %d", len([]rune(item.Name)), midtransItemNameMaxLength)
	}
	if item.Name != "Posku Monthly Subscription" {
		t.Fatalf("item name = %q, want %q", item.Name, "Posku Monthly Subscription")
	}
}

func TestBuildCustomerDetailsOmitsInvalidEmail(t *testing.T) {
	customer := buildCustomerDetails(
		"vault:v1:encrypted-email:ciphertext",
		"Artheon Cafe",
	)

	if customer == nil {
		t.Fatal("expected customer details with business name")
	}
	if customer.Email != "" {
		t.Fatalf("email = %q, want empty invalid encrypted email omitted", customer.Email)
	}
	if customer.FName != "Artheon Cafe" {
		t.Fatalf("first name = %q, want business name", customer.FName)
	}
}

func TestBuildCustomerDetailsNormalizesValidEmail(t *testing.T) {
	customer := buildCustomerDetails(" Owner <owner@example.com> ", "Bechalof.id")

	if customer == nil {
		t.Fatal("expected customer details")
	}
	if customer.Email != "owner@example.com" {
		t.Fatalf("email = %q, want normalized address", customer.Email)
	}
}

func TestTruncateForMidtrans(t *testing.T) {
	value := strings.Repeat("a", midtransItemNameMaxLength+10)

	got := truncateForMidtrans(value, midtransItemNameMaxLength)

	if len([]rune(got)) != midtransItemNameMaxLength {
		t.Fatalf("length = %d, want %d", len([]rune(got)), midtransItemNameMaxLength)
	}
}
