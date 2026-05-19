package services

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pos/billing-service/src/models"
)

func TestComputeSubscriptionStatus(t *testing.T) {
	now := time.Now()
	graceDays := 7

	tests := []struct {
		name   string
		tenant *models.Tenant
		want   string
	}{
		{
			name: "trial is active before trial end",
			tenant: &models.Tenant{
				SubscriptionPlan: "trial",
				TrialEndsAt:      ptrTime(now.Add(48 * time.Hour)),
			},
			want: "trial",
		},
		{
			name: "trial enters grace after trial end",
			tenant: &models.Tenant{
				SubscriptionPlan: "trial",
				TrialEndsAt:      ptrTime(now.Add(-48 * time.Hour)),
			},
			want: "grace_period",
		},
		{
			name: "trial expires after grace window",
			tenant: &models.Tenant{
				SubscriptionPlan: "trial",
				TrialEndsAt:      ptrTime(now.Add(-8 * 24 * time.Hour)),
			},
			want: "expired",
		},
		{
			name: "paid plan is active before subscription end",
			tenant: &models.Tenant{
				SubscriptionPlan:   "starter",
				SubscriptionEndsAt: ptrTime(now.Add(72 * time.Hour)),
			},
			want: "active",
		},
		{
			name: "paid plan enters grace after subscription end",
			tenant: &models.Tenant{
				SubscriptionPlan:   "starter",
				SubscriptionEndsAt: ptrTime(now.Add(-24 * time.Hour)),
			},
			want: "grace_period",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeSubscriptionStatus(tt.tenant, graceDays)
			if got != tt.want {
				t.Fatalf("computeSubscriptionStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComputeDaysRemaining(t *testing.T) {
	now := time.Now()
	graceDays := 7
	trialEndsAt := now.Add(49 * time.Hour)
	subscriptionEndsAt := now.Add(73 * time.Hour)
	graceTrialEndsAt := now.Add(-24 * time.Hour)

	tests := []struct {
		name   string
		tenant *models.Tenant
		status string
		min    int
		max    int
	}{
		{
			name: "trial days remaining",
			tenant: &models.Tenant{
				SubscriptionPlan: "trial",
				TrialEndsAt:      &trialEndsAt,
			},
			status: "trial",
			min:    1,
			max:    2,
		},
		{
			name: "active paid days remaining",
			tenant: &models.Tenant{
				SubscriptionPlan:   "starter",
				SubscriptionEndsAt: &subscriptionEndsAt,
			},
			status: "active",
			min:    2,
			max:    3,
		},
		{
			name: "grace period days remaining",
			tenant: &models.Tenant{
				SubscriptionPlan: "trial",
				TrialEndsAt:      &graceTrialEndsAt,
			},
			status: "grace_period",
			min:    5,
			max:    6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeDaysRemaining(tt.tenant, tt.status, graceDays)
			if got < tt.min || got > tt.max {
				t.Fatalf("computeDaysRemaining() = %d, want between %d and %d", got, tt.min, tt.max)
			}
		})
	}
}

func TestComputeAmount(t *testing.T) {
	t.Setenv("PLAN_MONTHLY_PRICE_IDR", "500000")
	t.Setenv("PLAN_ANNUAL_DISCOUNT_PCT", "25")

	if got := computeAmount("monthly"); got != 500000 {
		t.Fatalf("monthly amount = %d, want 500000", got)
	}
	if got := computeAmount("annual"); got != 4500000 {
		t.Fatalf("annual amount = %d, want 4500000", got)
	}
}

func TestComputePeriodEnd(t *testing.T) {
	start := time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)

	if got := computePeriodEnd(start, "monthly"); !got.Equal(start.AddDate(0, 1, 0)) {
		t.Fatalf("monthly period end = %s, want %s", got, start.AddDate(0, 1, 0))
	}
	if got := computePeriodEnd(start, "annual"); !got.Equal(start.AddDate(1, 0, 0)) {
		t.Fatalf("annual period end = %s, want %s", got, start.AddDate(1, 0, 0))
	}
}

func TestBuildBillingInvoiceNumberUsesUnixMilliseconds(t *testing.T) {
	now := time.Date(2026, 5, 19, 8, 9, 10, 123*int(time.Millisecond), time.UTC)

	got := BuildBillingInvoiceNumber(now)

	if got != "INV-202605-1779178150123" {
		t.Fatalf("invoice number = %q, want Unix millisecond suffix", got)
	}
}

func TestValidateBillingCycleSwitch(t *testing.T) {
	now := time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)
	activeAnnualEnd := now.Add(24 * time.Hour)
	expiredAnnualEnd := now.Add(-24 * time.Hour)

	tests := []struct {
		name            string
		tenant          *models.Tenant
		currentInterval string
		targetInterval  string
		wantLocked      bool
		wantErr         string
	}{
		{
			name:            "monthly can switch to annual immediately",
			tenant:          &models.Tenant{SubscriptionEndsAt: &activeAnnualEnd},
			currentInterval: "monthly",
			targetInterval:  "annual",
		},
		{
			name:            "active annual cannot switch to monthly before end",
			tenant:          &models.Tenant{SubscriptionEndsAt: &activeAnnualEnd},
			currentInterval: "annual",
			targetInterval:  "monthly",
			wantLocked:      true,
		},
		{
			name:            "expired annual can switch to monthly",
			tenant:          &models.Tenant{SubscriptionEndsAt: &expiredAnnualEnd},
			currentInterval: "annual",
			targetInterval:  "monthly",
		},
		{
			name:            "same interval is rejected",
			tenant:          &models.Tenant{SubscriptionEndsAt: &activeAnnualEnd},
			currentInterval: "monthly",
			targetInterval:  "monthly",
			wantErr:         "already using monthly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBillingCycleSwitch(tt.tenant, tt.currentInterval, tt.targetInterval, now)
			if tt.wantLocked {
				var locked *CycleSwitchLockedError
				if !errors.As(err, &locked) {
					t.Fatalf("expected CycleSwitchLockedError, got %v", err)
				}
				if !locked.SubscriptionEndsAt.Equal(activeAnnualEnd) {
					t.Fatalf("locked until = %s, want %s", locked.SubscriptionEndsAt, activeAnnualEnd)
				}
				return
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsPaymentLinkExpired(t *testing.T) {
	now := time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)
	url := "https://pay.example.test"
	future := now.Add(time.Hour)
	past := now.Add(-time.Second)

	tests := []struct {
		name string
		inv  *models.BillingInvoice
		want bool
	}{
		{
			name: "invoice without payment url is not expired",
			inv:  &models.BillingInvoice{},
			want: false,
		},
		{
			name: "payment url without expiry is expired",
			inv:  &models.BillingInvoice{MidtransPaymentURL: &url},
			want: true,
		},
		{
			name: "future payment link is reusable",
			inv:  &models.BillingInvoice{MidtransPaymentURL: &url, PaymentLinkExpiresAt: &future},
			want: false,
		},
		{
			name: "past payment link is expired",
			inv:  &models.BillingInvoice{MidtransPaymentURL: &url, PaymentLinkExpiresAt: &past},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPaymentLinkExpired(tt.inv, now); got != tt.want {
				t.Fatalf("isPaymentLinkExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaymentLinkTTLIsFifteenMinutes(t *testing.T) {
	if PaymentLinkTTL != 15*time.Minute {
		t.Fatalf("PaymentLinkTTL = %s, want 15m", PaymentLinkTTL)
	}
}

func TestComputeRetentionCleanupAt(t *testing.T) {
	startedAt := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	got := computeRetentionCleanupAt(&models.Tenant{RetentionStartedAt: &startedAt})
	if got == nil {
		t.Fatal("expected retention cleanup date")
	}
	want := startedAt.AddDate(0, 0, SubscriptionRetentionDays)
	if !got.Equal(want) {
		t.Fatalf("retention cleanup = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}

	if got := computeRetentionCleanupAt(&models.Tenant{}); got != nil {
		t.Fatalf("expected nil cleanup date without retention start, got %s", got.Format(time.RFC3339))
	}
}

func TestComputeInvoiceDueAt(t *testing.T) {
	now := time.Now().UTC()
	trialEnd := now.Add(48 * time.Hour)
	subscriptionEnd := now.Add(72 * time.Hour)
	fallback := now.Add(24 * time.Hour)

	if got := computeInvoiceDueAt(&models.Tenant{SubscriptionPlan: "trial", TrialEndsAt: &trialEnd}, fallback); !got.Equal(trialEnd) {
		t.Fatalf("trial invoice due_at = %s, want %s", got, trialEnd)
	}
	if got := computeInvoiceDueAt(&models.Tenant{SubscriptionPlan: "starter", SubscriptionEndsAt: &subscriptionEnd}, fallback); !got.Equal(subscriptionEnd) {
		t.Fatalf("subscription invoice due_at = %s, want %s", got, subscriptionEnd)
	}
	expiredTrial := now.Add(-24 * time.Hour)
	if got := computeInvoiceDueAt(&models.Tenant{SubscriptionPlan: "trial", TrialEndsAt: &expiredTrial}, fallback); !got.Equal(fallback) {
		t.Fatalf("expired invoice due_at = %s, want fallback %s", got, fallback)
	}
}

func TestComputePaymentDueAtPrefersPendingInvoice(t *testing.T) {
	trialEnd := time.Now().UTC().Add(48 * time.Hour)
	invoiceDue := time.Now().UTC().Add(24 * time.Hour)
	got := computePaymentDueAt(
		&models.Tenant{SubscriptionPlan: "trial", TrialEndsAt: &trialEnd},
		&models.BillingInvoice{DueAt: invoiceDue},
	)
	if got == nil || !got.Equal(invoiceDue) {
		t.Fatalf("payment due_at = %v, want invoice due_at %s", got, invoiceDue)
	}
}

func TestVerifyMidtransSignature(t *testing.T) {
	orderID := "BILL-202605-000001"
	statusCode := "200"
	grossAmount := "299000.00"
	serverKey := "test-server-key"
	raw := orderID + statusCode + grossAmount + serverKey
	sum := sha512.Sum512([]byte(raw))
	signature := fmt.Sprintf("%x", sum)

	if !verifyMidtransSignature(orderID, statusCode, grossAmount, serverKey, signature) {
		t.Fatal("expected valid Midtrans signature")
	}
	if verifyMidtransSignature(orderID, statusCode, grossAmount, serverKey, "invalid") {
		t.Fatal("expected invalid Midtrans signature to be rejected")
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
