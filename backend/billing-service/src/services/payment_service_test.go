package services

import (
	"strings"
	"testing"
	"time"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"

	"github.com/pos/billing-service/src/models"
)

func TestNewPaymentServiceStoresTrimmedWebhookURL(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "server-key")
	t.Setenv("MIDTRANS_ENV", "sandbox")
	t.Setenv("MIDTRANS_WEBHOOK_URL", " https://billing.example.com/api/v1/billing/webhook ")
	t.Setenv("FRONTEND_DOMAIN", " http://localhost:3000/ ")

	svc := NewPaymentService()

	if svc.webhookURL != "https://billing.example.com/api/v1/billing/webhook" {
		t.Fatalf("webhookURL = %q, want trimmed webhook URL", svc.webhookURL)
	}
	if svc.frontendDomain != "http://localhost:3000/" {
		t.Fatalf("frontendDomain = %q, want trimmed frontend domain", svc.frontendDomain)
	}
}

func TestConfigureSnapNotificationsSkipsEmptyWebhookURL(t *testing.T) {
	svc := &PaymentService{}
	var client snap.Client
	client.New("server-key", midtrans.Sandbox)

	svc.configureSnapNotifications(&client)

	if client.Options.PaymentOverrideNotification != nil {
		t.Fatalf("override notification = %q, want nil", *client.Options.PaymentOverrideNotification)
	}
}

func TestConfigureSnapNotificationsSetsOverrideWebhookURL(t *testing.T) {
	webhookURL := "https://billing.example.com/api/v1/billing/webhook"
	svc := &PaymentService{webhookURL: webhookURL}
	var client snap.Client
	client.New("server-key", midtrans.Sandbox)

	svc.configureSnapNotifications(&client)

	if client.Options.PaymentOverrideNotification == nil {
		t.Fatal("expected override notification to be set")
	}
	if *client.Options.PaymentOverrideNotification != webhookURL {
		t.Fatalf("override notification = %q, want %q", *client.Options.PaymentOverrideNotification, webhookURL)
	}
	if client.Options.PaymentAppendNotification != nil {
		t.Fatalf("append notification = %q, want nil", *client.Options.PaymentAppendNotification)
	}
}

func TestBuildSnapExpiryDetailsUsesPaymentLinkTTL(t *testing.T) {
	expiry := buildSnapExpiryDetails()

	if expiry.Duration != int64(PaymentLinkTTLMinutes) {
		t.Fatalf("expiry duration = %d, want %d", expiry.Duration, PaymentLinkTTLMinutes)
	}
	if expiry.Unit != "minute" {
		t.Fatalf("expiry unit = %q, want minute", expiry.Unit)
	}
}

func TestBuildSnapCallbacksSkipsEmptyFrontendDomain(t *testing.T) {
	if callbacks := buildSnapCallbacks(" "); callbacks != nil {
		t.Fatalf("callbacks = %#v, want nil", callbacks)
	}
}

func TestBuildSnapCallbacksSetsFinishRedirectURL(t *testing.T) {
	callbacks := buildSnapCallbacks(" http://localhost:3000/ ")

	if callbacks == nil {
		t.Fatal("expected callbacks to be set")
	}
	if callbacks.Finish != "http://localhost:3000/subscription?payment_return=midtrans" {
		t.Fatalf("finish callback = %q, want subscription return URL", callbacks.Finish)
	}
}

func TestBuildBillingOrderIDUsesFullInvoiceSuffix(t *testing.T) {
	inv := &models.BillingInvoice{
		InvoiceNumber: "INV-202605-1779202312863",
		PeriodStart:   time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC),
	}

	if got := buildBillingOrderID(inv); got != "BILL-202605-1779202312863" {
		t.Fatalf("order ID = %q, want full invoice suffix", got)
	}
}

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
