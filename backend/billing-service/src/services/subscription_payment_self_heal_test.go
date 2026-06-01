package services

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/pos/billing-service/src/models"
)

func TestUpgradeSubscriptionCancelsZeroAmountPendingInvoiceAndCreatesFreshPayment(t *testing.T) {
	t.Setenv("PLAN_MONTHLY_PRICE_IDR", "299000")
	t.Setenv("PLAN_ANNUAL_DISCOUNT_PCT", "20")

	repo := newFakeBillingRepository()
	paymentSvc := &fakeSnapPaymentCreator{}
	tenant := &models.Tenant{ID: "tenant-1", BusinessName: "Cafe", BillingCycle: "monthly"}
	badInvoice := testPendingInvoice("invoice-zero", tenant.ID, "monthly", 0)
	repo.putTenant(tenant)
	repo.putInvoice(badInvoice)
	repo.pendingByTenantInterval[pendingIntervalKey(tenant.ID, "monthly")] = badInvoice

	svc := &SubscriptionService{repo: repo, paymentSvc: paymentSvc}

	got, err := svc.UpgradeSubscription(context.Background(), tenant.ID, "monthly")
	if err != nil {
		t.Fatalf("UpgradeSubscription returned error: %v", err)
	}

	if badInvoice.Status != "cancelled" {
		t.Fatalf("old invoice status = %q, want cancelled", badInvoice.Status)
	}
	if len(repo.createdInvoices) != 1 {
		t.Fatalf("created invoices = %d, want 1", len(repo.createdInvoices))
	}
	if repo.createdInvoices[0].AmountIDR != 299000 {
		t.Fatalf("new invoice amount = %d, want 299000", repo.createdInvoices[0].AmountIDR)
	}
	if got.InvoiceID != repo.createdInvoices[0].ID {
		t.Fatalf("response invoice ID = %q, want new invoice %q", got.InvoiceID, repo.createdInvoices[0].ID)
	}
	if paymentSvc.calls != 1 {
		t.Fatalf("payment calls = %d, want 1", paymentSvc.calls)
	}
}

func TestSwitchBillingCycleCancelsZeroAmountPendingInvoiceAndCreatesFreshPayment(t *testing.T) {
	t.Setenv("PLAN_MONTHLY_PRICE_IDR", "100000")
	t.Setenv("PLAN_ANNUAL_DISCOUNT_PCT", "20")

	repo := newFakeBillingRepository()
	paymentSvc := &fakeSnapPaymentCreator{}
	tenant := &models.Tenant{ID: "tenant-1", BusinessName: "Cafe", BillingCycle: "monthly"}
	badInvoice := testPendingInvoice("invoice-zero", tenant.ID, "annual", 0)
	repo.putTenant(tenant)
	repo.putInvoice(badInvoice)
	repo.pendingByTenantInterval[pendingIntervalKey(tenant.ID, "annual")] = badInvoice

	svc := &SubscriptionService{repo: repo, paymentSvc: paymentSvc}

	got, err := svc.SwitchBillingCycle(context.Background(), tenant.ID, "annual")
	if err != nil {
		t.Fatalf("SwitchBillingCycle returned error: %v", err)
	}

	if badInvoice.Status != "cancelled" {
		t.Fatalf("old invoice status = %q, want cancelled", badInvoice.Status)
	}
	if len(repo.createdInvoices) != 1 {
		t.Fatalf("created invoices = %d, want 1", len(repo.createdInvoices))
	}
	if got.AmountIDR != 960000 {
		t.Fatalf("cycle switch amount = %d, want 960000", got.AmountIDR)
	}
	if got.InvoiceID != repo.createdInvoices[0].ID {
		t.Fatalf("response invoice ID = %q, want new invoice %q", got.InvoiceID, repo.createdInvoices[0].ID)
	}
}

func TestInitiatePaymentCancelsZeroAmountInvoiceAndCreatesFreshPayment(t *testing.T) {
	t.Setenv("PLAN_MONTHLY_PRICE_IDR", "299000")
	t.Setenv("PLAN_ANNUAL_DISCOUNT_PCT", "20")

	repo := newFakeBillingRepository()
	paymentSvc := &fakeSnapPaymentCreator{}
	tenant := &models.Tenant{ID: "tenant-1", BusinessName: "Cafe", BillingCycle: "monthly"}
	badInvoice := testPendingInvoice("invoice-zero", tenant.ID, "monthly", 0)
	repo.putTenant(tenant)
	repo.putInvoice(badInvoice)

	svc := &SubscriptionService{repo: repo, paymentSvc: paymentSvc}

	got, err := svc.InitiatePayment(context.Background(), tenant.ID, badInvoice.ID)
	if err != nil {
		t.Fatalf("InitiatePayment returned error: %v", err)
	}

	if badInvoice.Status != "cancelled" {
		t.Fatalf("old invoice status = %q, want cancelled", badInvoice.Status)
	}
	if len(repo.createdInvoices) != 1 {
		t.Fatalf("created invoices = %d, want 1", len(repo.createdInvoices))
	}
	if got.AmountIDR != 299000 {
		t.Fatalf("payment amount = %d, want 299000", got.AmountIDR)
	}
	if got.InvoiceID == badInvoice.ID {
		t.Fatalf("response reused zero-amount invoice %q", got.InvoiceID)
	}
}

func TestInitiatePaymentDoesNotRewriteNonPendingZeroAmountInvoice(t *testing.T) {
	repo := newFakeBillingRepository()
	tenant := &models.Tenant{ID: "tenant-1", BusinessName: "Cafe", BillingCycle: "monthly"}
	paidInvoice := testPendingInvoice("invoice-paid", tenant.ID, "monthly", 0)
	paidInvoice.Status = "paid"
	repo.putTenant(tenant)
	repo.putInvoice(paidInvoice)

	svc := &SubscriptionService{repo: repo, paymentSvc: &fakeSnapPaymentCreator{}}

	_, err := svc.InitiatePayment(context.Background(), tenant.ID, paidInvoice.ID)
	if err == nil {
		t.Fatal("expected error for paid invoice")
	}
	if paidInvoice.Status != "paid" {
		t.Fatalf("paid invoice status = %q, want paid", paidInvoice.Status)
	}
	if len(repo.statusUpdates) != 0 {
		t.Fatalf("status updates = %d, want 0", len(repo.statusUpdates))
	}
	if len(repo.createdInvoices) != 0 {
		t.Fatalf("created invoices = %d, want 0", len(repo.createdInvoices))
	}
}

func TestUpgradeSubscriptionDoesNotCreateInvoiceForInvalidComputedAmount(t *testing.T) {
	tests := []struct {
		name            string
		billingInterval string
		monthlyPrice    string
		annualDiscount  string
	}{
		{
			name:            "zero monthly price",
			billingInterval: "monthly",
			monthlyPrice:    "0",
			annualDiscount:  "20",
		},
		{
			name:            "negative monthly price",
			billingInterval: "monthly",
			monthlyPrice:    "-1000",
			annualDiscount:  "20",
		},
		{
			name:            "annual discount makes annual amount zero",
			billingInterval: "annual",
			monthlyPrice:    "100000",
			annualDiscount:  "100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PLAN_MONTHLY_PRICE_IDR", tt.monthlyPrice)
			t.Setenv("PLAN_ANNUAL_DISCOUNT_PCT", tt.annualDiscount)

			repo := newFakeBillingRepository()
			paymentSvc := &fakeSnapPaymentCreator{}
			tenant := &models.Tenant{ID: "tenant-1", BusinessName: "Cafe", BillingCycle: "monthly"}
			repo.putTenant(tenant)

			svc := &SubscriptionService{repo: repo, paymentSvc: paymentSvc}

			_, err := svc.UpgradeSubscription(context.Background(), tenant.ID, tt.billingInterval)
			if err == nil {
				t.Fatal("expected invalid amount error")
			}
			if len(repo.createdInvoices) != 0 {
				t.Fatalf("created invoices = %d, want 0", len(repo.createdInvoices))
			}
			if paymentSvc.calls != 0 {
				t.Fatalf("payment calls = %d, want 0", paymentSvc.calls)
			}
		})
	}
}

type fakeBillingRepository struct {
	tenants                 map[string]*models.Tenant
	invoices                map[string]*models.BillingInvoice
	pendingByTenant         map[string]*models.BillingInvoice
	pendingByTenantInterval map[string]*models.BillingInvoice
	createdInvoices         []*models.BillingInvoice
	paymentAttempts         []*models.BillingPaymentAttempt
	statusUpdates           []fakeInvoiceStatusUpdate
}

type fakeInvoiceStatusUpdate struct {
	id     string
	status string
}

func newFakeBillingRepository() *fakeBillingRepository {
	return &fakeBillingRepository{
		tenants:                 map[string]*models.Tenant{},
		invoices:                map[string]*models.BillingInvoice{},
		pendingByTenant:         map[string]*models.BillingInvoice{},
		pendingByTenantInterval: map[string]*models.BillingInvoice{},
	}
}

func (f *fakeBillingRepository) putTenant(tenant *models.Tenant) {
	f.tenants[tenant.ID] = tenant
}

func (f *fakeBillingRepository) putInvoice(inv *models.BillingInvoice) {
	f.invoices[inv.ID] = inv
	if inv.Status == "pending" {
		f.pendingByTenant[inv.TenantID] = inv
		f.pendingByTenantInterval[pendingIntervalKey(inv.TenantID, inv.BillingInterval)] = inv
	}
}

func (f *fakeBillingRepository) GetTenantByID(_ context.Context, id string) (*models.Tenant, error) {
	return f.tenants[id], nil
}

func (f *fakeBillingRepository) GetPendingInvoiceByTenantID(_ context.Context, tenantID string) (*models.BillingInvoice, error) {
	inv := f.pendingByTenant[tenantID]
	if inv == nil || inv.Status != "pending" {
		return nil, nil
	}
	return inv, nil
}

func (f *fakeBillingRepository) UpdateTenantBillingCycle(_ context.Context, tenantID, billingInterval string) error {
	if tenant := f.tenants[tenantID]; tenant != nil {
		tenant.BillingCycle = billingInterval
	}
	return nil
}

func (f *fakeBillingRepository) CancelPendingInvoicesByTenantIDExceptInterval(_ context.Context, tenantID, billingInterval string) error {
	for _, inv := range f.invoices {
		if inv.TenantID == tenantID && inv.Status == "pending" && inv.BillingInterval != billingInterval {
			inv.Status = "cancelled"
		}
	}
	return nil
}

func (f *fakeBillingRepository) GetPendingInvoiceByTenantIDAndInterval(_ context.Context, tenantID, billingInterval string) (*models.BillingInvoice, error) {
	inv := f.pendingByTenantInterval[pendingIntervalKey(tenantID, billingInterval)]
	if inv == nil || inv.Status != "pending" {
		return nil, nil
	}
	return inv, nil
}

func (f *fakeBillingRepository) UpdateInvoiceStatus(_ context.Context, id, status string, paidAt *time.Time, midtransOrderID, paymentURL *string, paymentLinkExpiresAt *time.Time) error {
	inv := f.invoices[id]
	if inv == nil {
		return errors.New("invoice not found")
	}
	inv.Status = status
	inv.PaidAt = paidAt
	if midtransOrderID != nil {
		inv.MidtransOrderID = midtransOrderID
	}
	if paymentURL != nil {
		inv.MidtransPaymentURL = paymentURL
	}
	if paymentLinkExpiresAt != nil {
		inv.PaymentLinkExpiresAt = paymentLinkExpiresAt
	}
	f.statusUpdates = append(f.statusUpdates, fakeInvoiceStatusUpdate{id: id, status: status})
	return nil
}

func (f *fakeBillingRepository) CreateInvoice(_ context.Context, inv *models.BillingInvoice) error {
	inv.ID = fmt.Sprintf("invoice-new-%d", len(f.createdInvoices)+1)
	inv.CreatedAt = time.Now().UTC()
	inv.UpdatedAt = inv.CreatedAt
	f.createdInvoices = append(f.createdInvoices, inv)
	f.putInvoice(inv)
	return nil
}

func (f *fakeBillingRepository) GetTenantOwnerEmail(_ context.Context, _ string) (string, error) {
	return "owner@example.com", nil
}

func (f *fakeBillingRepository) GetInvoiceByID(_ context.Context, id string) (*models.BillingInvoice, error) {
	return f.invoices[id], nil
}

func (f *fakeBillingRepository) GetInvoicesByTenantID(_ context.Context, tenantID string) ([]*models.BillingInvoice, error) {
	var invoices []*models.BillingInvoice
	for _, inv := range f.invoices {
		if inv.TenantID == tenantID {
			invoices = append(invoices, inv)
		}
	}
	return invoices, nil
}

func (f *fakeBillingRepository) GetPaymentAttemptsByInvoiceID(_ context.Context, _, _ string) ([]*models.BillingPaymentAttempt, error) {
	return nil, nil
}

func (f *fakeBillingRepository) GetInvoiceByMidtransOrderID(_ context.Context, midtransOrderID string) (*models.BillingInvoice, error) {
	for _, inv := range f.invoices {
		if inv.MidtransOrderID != nil && *inv.MidtransOrderID == midtransOrderID {
			return inv, nil
		}
	}
	return nil, nil
}

func (f *fakeBillingRepository) CreatePaymentAttempt(_ context.Context, attempt *models.BillingPaymentAttempt) error {
	attempt.ID = fmt.Sprintf("attempt-%d", len(f.paymentAttempts)+1)
	attempt.CreatedAt = time.Now().UTC()
	f.paymentAttempts = append(f.paymentAttempts, attempt)
	return nil
}

func (f *fakeBillingRepository) UpdateTenantSubscribedAt(_ context.Context, tenantID, plan, billingInterval string, subscriptionEndsAt time.Time) error {
	tenant := f.tenants[tenantID]
	if tenant == nil {
		return errors.New("tenant not found")
	}
	tenant.SubscriptionPlan = plan
	tenant.BillingCycle = billingInterval
	tenant.SubscriptionEndsAt = &subscriptionEndsAt
	return nil
}

type fakeSnapPaymentCreator struct {
	calls int
}

func (f *fakeSnapPaymentCreator) CreateSnapPayment(_ context.Context, _ *models.BillingInvoice, _, _ string) (*SnapPaymentResult, error) {
	f.calls++
	return &SnapPaymentResult{
		Token:      fmt.Sprintf("token-%d", f.calls),
		PaymentURL: fmt.Sprintf("https://pay.example.test/%d", f.calls),
		OrderID:    fmt.Sprintf("order-%d", f.calls),
	}, nil
}

func testPendingInvoice(id, tenantID, billingInterval string, amount int) *models.BillingInvoice {
	now := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	return &models.BillingInvoice{
		ID:              id,
		TenantID:        tenantID,
		InvoiceNumber:   id,
		AmountIDR:       amount,
		BillingInterval: billingInterval,
		PeriodStart:     now,
		PeriodEnd:       now.AddDate(0, 1, 0),
		DueAt:           now,
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func pendingIntervalKey(tenantID, billingInterval string) string {
	return tenantID + "|" + billingInterval
}
