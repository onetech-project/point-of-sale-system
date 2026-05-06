package services

import (
	"context"
	"crypto/sha512"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pos/billing-service/src/models"
	"github.com/pos/billing-service/src/queue"
	"github.com/pos/billing-service/src/repository"
	"github.com/pos/billing-service/src/utils"
)

// SubscriptionService contains the core billing business logic.
type SubscriptionService struct {
	db         *sql.DB
	repo       *repository.BillingRepository
	publisher  *queue.EventPublisher
	paymentSvc *PaymentService
}

// NewSubscriptionService creates a SubscriptionService.
func NewSubscriptionService(db *sql.DB, repo *repository.BillingRepository, publisher *queue.EventPublisher, paymentSvc *PaymentService) *SubscriptionService {
	return &SubscriptionService{
		db:         db,
		repo:       repo,
		publisher:  publisher,
		paymentSvc: paymentSvc,
	}
}

// UpgradeResponse is returned by UpgradeSubscription.
type UpgradeResponse struct {
	InvoiceID     string `json:"invoice_id"`
	InvoiceNumber string `json:"invoice_number"`
	AmountIDR     int    `json:"amount_idr"`
	SnapToken     string `json:"snap_token"`
	PaymentURL    string `json:"payment_url"`
}

// MidtransWebhookPayload represents the Midtrans payment notification body.
type MidtransWebhookPayload struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	MerchantID        string `json:"merchant_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status,omitempty"`
}

// GetSubscriptionStatus computes and returns the current subscription status for a tenant.
func (s *SubscriptionService) GetSubscriptionStatus(ctx context.Context, tenantID string) (*models.SubscriptionStatusResponse, error) {
	tenant, err := s.repo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	if tenant == nil {
		return nil, errors.New("tenant not found")
	}

	graceDays := utils.GetEnvInt("PLAN_GRACE_PERIOD_DAYS", 7)
	status := computeSubscriptionStatus(tenant, graceDays)
	days := computeDaysRemaining(tenant, status, graceDays)

	return &models.SubscriptionStatusResponse{
		TenantID:           tenant.ID,
		SubscriptionPlan:   tenant.SubscriptionPlan,
		SubscriptionStatus: status,
		BillingCycle:       tenant.BillingCycle,
		TrialEndsAt:        tenant.TrialEndsAt,
		SubscriptionEndsAt: tenant.SubscriptionEndsAt,
		IsActive:           status != "expired" && status != "cancelled",
		DaysRemaining:      days,
	}, nil
}

// UpdateBillingCycle updates the tenant's preferred billing cycle.
func (s *SubscriptionService) UpdateBillingCycle(ctx context.Context, tenantID, billingInterval string) error {
	if billingInterval != "monthly" && billingInterval != "annual" {
		return errors.New("billing_interval must be 'monthly' or 'annual'")
	}
	return s.repo.UpdateTenantBillingCycle(ctx, tenantID, billingInterval)
}

// UpgradeSubscription creates an invoice and a Midtrans Snap payment for the tenant.
func (s *SubscriptionService) UpgradeSubscription(ctx context.Context, tenantID, billingInterval string) (*UpgradeResponse, error) {
	if billingInterval != "monthly" && billingInterval != "annual" {
		return nil, errors.New("billing_interval must be 'monthly' or 'annual'")
	}

	tenant, err := s.repo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	if tenant == nil {
		return nil, errors.New("tenant not found")
	}

	// Check for an existing pending invoice to avoid duplicates.
	existing, err := s.repo.GetPendingInvoiceByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invoice: %w", err)
	}
	if existing != nil && existing.MidtransPaymentURL != nil {
		// Return the existing payment URL.
		snapToken := ""
		return &UpgradeResponse{
			InvoiceID:     existing.ID,
			InvoiceNumber: existing.InvoiceNumber,
			AmountIDR:     existing.AmountIDR,
			SnapToken:     snapToken,
			PaymentURL:    *existing.MidtransPaymentURL,
		}, nil
	}

	amount := computeAmount(billingInterval)
	now := time.Now().UTC()
	periodStart := now
	var periodEnd time.Time
	if billingInterval == "annual" {
		periodEnd = now.AddDate(1, 0, 0)
	} else {
		periodEnd = now.AddDate(0, 1, 0)
	}

	count, err := s.repo.CountInvoicesThisMonth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count invoices: %w", err)
	}
	invoiceNumber := fmt.Sprintf("INV-%s-%06d", now.Format("200601"), count+1)

	inv := &models.BillingInvoice{
		TenantID:        tenantID,
		InvoiceNumber:   invoiceNumber,
		AmountIDR:       amount,
		BillingInterval: billingInterval,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		Status:          "pending",
	}
	if err := s.repo.CreateInvoice(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	ownerEmail, _ := s.repo.GetTenantOwnerEmail(ctx, tenantID)

	snapResult, err := s.paymentSvc.CreateSnapPayment(ctx, inv, ownerEmail, tenant.BusinessName)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Update invoice with Midtrans details.
	if err := s.repo.UpdateInvoiceStatus(ctx, inv.ID, "pending", nil, &snapResult.OrderID, &snapResult.PaymentURL); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	return &UpgradeResponse{
		InvoiceID:     inv.ID,
		InvoiceNumber: inv.InvoiceNumber,
		AmountIDR:     inv.AmountIDR,
		SnapToken:     snapResult.Token,
		PaymentURL:    snapResult.PaymentURL,
	}, nil
}

// InitiatePayment creates a new Midtrans Snap payment for an existing pending invoice.
func (s *SubscriptionService) InitiatePayment(ctx context.Context, tenantID, invoiceID string) (*UpgradeResponse, error) {
	inv, err := s.repo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	if inv == nil || inv.TenantID != tenantID {
		return nil, errors.New("invoice not found")
	}
	if inv.Status != "pending" {
		return nil, fmt.Errorf("invoice status is '%s', payment can only be initiated for pending invoices", inv.Status)
	}

	tenant, err := s.repo.GetTenantByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return nil, errors.New("tenant not found")
	}

	ownerEmail, _ := s.repo.GetTenantOwnerEmail(ctx, tenantID)
	snapResult, err := s.paymentSvc.CreateSnapPayment(ctx, inv, ownerEmail, tenant.BusinessName)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	if err := s.repo.UpdateInvoiceStatus(ctx, inv.ID, "pending", nil, &snapResult.OrderID, &snapResult.PaymentURL); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	return &UpgradeResponse{
		InvoiceID:     inv.ID,
		InvoiceNumber: inv.InvoiceNumber,
		AmountIDR:     inv.AmountIDR,
		SnapToken:     snapResult.Token,
		PaymentURL:    snapResult.PaymentURL,
	}, nil
}

// GetInvoices returns all invoices for a tenant.
func (s *SubscriptionService) GetInvoices(ctx context.Context, tenantID string) ([]*models.BillingInvoice, error) {
	return s.repo.GetInvoicesByTenantID(ctx, tenantID)
}

// GetInvoice returns a specific invoice, verifying it belongs to the tenant.
func (s *SubscriptionService) GetInvoice(ctx context.Context, tenantID, invoiceID string) (*models.BillingInvoice, error) {
	inv, err := s.repo.GetInvoiceByID(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil || inv.TenantID != tenantID {
		return nil, errors.New("invoice not found")
	}
	return inv, nil
}

// HandlePaymentWebhook processes an inbound Midtrans payment notification.
func (s *SubscriptionService) HandlePaymentWebhook(ctx context.Context, payload MidtransWebhookPayload) error {
	serverKey := utils.GetEnv("MIDTRANS_SERVER_KEY")
	if !verifyMidtransSignature(payload.OrderID, payload.StatusCode, payload.GrossAmount, serverKey, payload.SignatureKey) {
		return errors.New("invalid signature")
	}

	inv, err := s.repo.GetInvoiceByMidtransOrderID(ctx, payload.OrderID)
	if err != nil {
		return fmt.Errorf("failed to find invoice: %w", err)
	}
	if inv == nil {
		return fmt.Errorf("invoice not found for order_id: %s", payload.OrderID)
	}

	paymentMethod := payload.PaymentType
	attempt := &models.BillingPaymentAttempt{
		InvoiceID:       inv.ID,
		TenantID:        inv.TenantID,
		AmountIDR:       inv.AmountIDR,
		MidtransOrderID: &payload.OrderID,
		PaymentMethod:   &paymentMethod,
		Status:          "pending",
	}

	switch payload.TransactionStatus {
	case "settlement", "capture":
		attempt.Status = "completed"
		_ = s.repo.CreatePaymentAttempt(ctx, attempt)

		if inv.Status == "paid" {
			return nil // idempotent
		}

		now := time.Now().UTC()
		if err := s.repo.UpdateInvoiceStatus(ctx, inv.ID, "paid", &now, nil, nil); err != nil {
			return fmt.Errorf("failed to update invoice: %w", err)
		}

		var subscriptionEndsAt time.Time
		if inv.BillingInterval == "annual" {
			subscriptionEndsAt = inv.PeriodEnd
		} else {
			subscriptionEndsAt = inv.PeriodEnd
		}

		if err := s.repo.UpdateTenantSubscribedAt(ctx, inv.TenantID, "starter", inv.BillingInterval, subscriptionEndsAt); err != nil {
			return fmt.Errorf("failed to update tenant subscription: %w", err)
		}

		ownerEmail, _ := s.repo.GetTenantOwnerEmail(ctx, inv.TenantID)
		tenant, _ := s.repo.GetTenantByID(ctx, inv.TenantID)
		tenantName := ""
		if tenant != nil {
			tenantName = tenant.BusinessName
		}
		_ = s.publisher.PublishInvoicePaid(ctx, inv.TenantID, ownerEmail, tenantName,
			inv.InvoiceNumber, inv.AmountIDR, inv.BillingInterval, inv.PeriodStart, inv.PeriodEnd)

	case "deny", "cancel", "expire":
		errMsg := fmt.Sprintf("payment %s", payload.TransactionStatus)
		attempt.Status = "failed"
		attempt.ErrorMsg = &errMsg
		_ = s.repo.CreatePaymentAttempt(ctx, attempt)

		ownerEmail, _ := s.repo.GetTenantOwnerEmail(ctx, inv.TenantID)
		tenant, _ := s.repo.GetTenantByID(ctx, inv.TenantID)
		tenantName := ""
		if tenant != nil {
			tenantName = tenant.BusinessName
		}
		_ = s.publisher.PublishInvoicePaymentFailed(ctx, inv.TenantID, ownerEmail, tenantName,
			inv.InvoiceNumber, errMsg)
	}

	return nil
}

// CreateNextPeriodInvoice creates a renewal invoice for an active tenant if one doesn't exist yet.
func (s *SubscriptionService) CreateNextPeriodInvoice(ctx context.Context, tenant *models.Tenant) (*models.BillingInvoice, error) {
	// Check for an existing pending invoice.
	existing, err := s.repo.GetPendingInvoiceByTenantID(ctx, tenant.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	billingInterval := tenant.BillingCycle
	amount := computeAmount(billingInterval)

	var periodStart, periodEnd time.Time
	if tenant.SubscriptionEndsAt != nil {
		periodStart = *tenant.SubscriptionEndsAt
	} else {
		periodStart = time.Now().UTC()
	}
	if billingInterval == "annual" {
		periodEnd = periodStart.AddDate(1, 0, 0)
	} else {
		periodEnd = periodStart.AddDate(0, 1, 0)
	}

	count, err := s.repo.CountInvoicesThisMonth(ctx)
	if err != nil {
		return nil, err
	}
	invoiceNumber := fmt.Sprintf("INV-%s-%06d", time.Now().UTC().Format("200601"), count+1)

	inv := &models.BillingInvoice{
		TenantID:        tenant.ID,
		InvoiceNumber:   invoiceNumber,
		AmountIDR:       amount,
		BillingInterval: billingInterval,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		Status:          "pending",
	}
	if err := s.repo.CreateInvoice(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// --- helpers ---

func computeSubscriptionStatus(tenant *models.Tenant, graceDays int) string {
	now := time.Now()
	grace := time.Duration(graceDays) * 24 * time.Hour

	if tenant.SubscriptionPlan == "trial" {
		if tenant.TrialEndsAt == nil || tenant.TrialEndsAt.After(now) {
			return "trial"
		}
		if tenant.TrialEndsAt.After(now.Add(-grace)) {
			return "grace_period"
		}
		return "expired"
	}

	// Paid plan
	if tenant.SubscriptionEndsAt == nil || tenant.SubscriptionEndsAt.After(now) {
		return "active"
	}
	if tenant.SubscriptionEndsAt.After(now.Add(-grace)) {
		return "grace_period"
	}
	return "expired"
}

func computeDaysRemaining(tenant *models.Tenant, status string, graceDays int) int {
	now := time.Now()
	grace := time.Duration(graceDays) * 24 * time.Hour

	switch status {
	case "trial":
		if tenant.TrialEndsAt != nil {
			return max(0, int(time.Until(*tenant.TrialEndsAt).Hours()/24))
		}
	case "active":
		if tenant.SubscriptionEndsAt != nil {
			return max(0, int(time.Until(*tenant.SubscriptionEndsAt).Hours()/24))
		}
	case "grace_period":
		if tenant.SubscriptionPlan == "trial" && tenant.TrialEndsAt != nil {
			graceEnd := tenant.TrialEndsAt.Add(grace)
			return max(0, int(graceEnd.Sub(now).Hours()/24))
		}
		if tenant.SubscriptionEndsAt != nil {
			graceEnd := tenant.SubscriptionEndsAt.Add(grace)
			return max(0, int(graceEnd.Sub(now).Hours()/24))
		}
	}
	return 0
}

func computeAmount(billingInterval string) int {
	monthly := utils.GetEnvInt("PLAN_MONTHLY_PRICE_IDR", 299000)
	if billingInterval == "annual" {
		discountPct := utils.GetEnvInt("PLAN_ANNUAL_DISCOUNT_PCT", 20)
		annual := monthly * 12
		discount := annual * discountPct / 100
		return annual - discount
	}
	return monthly
}

func verifyMidtransSignature(orderID, statusCode, grossAmount, serverKey, received string) bool {
	raw := orderID + statusCode + grossAmount + serverKey
	sum := sha512.Sum512([]byte(raw))
	computed := fmt.Sprintf("%x", sum)
	return strings.EqualFold(computed, received)
}
