package services

import (
	"crypto/sha512"
	"fmt"
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
