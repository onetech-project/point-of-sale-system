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

const (
	PaymentLinkTTLMinutes = 15
	PaymentLinkTTL        = PaymentLinkTTLMinutes * time.Minute
)

type SubscriptionCacheInvalidator interface {
	InvalidateSubscriptionStatus(ctx context.Context, tenantID string) error
}

type billingRepository interface {
	GetTenantByID(ctx context.Context, id string) (*models.Tenant, error)
	GetPendingInvoiceByTenantID(ctx context.Context, tenantID string) (*models.BillingInvoice, error)
	UpdateTenantBillingCycle(ctx context.Context, tenantID, billingInterval string) error
	CancelPendingInvoicesByTenantIDExceptInterval(ctx context.Context, tenantID, billingInterval string) error
	GetPendingInvoiceByTenantIDAndInterval(ctx context.Context, tenantID, billingInterval string) (*models.BillingInvoice, error)
	UpdateInvoiceStatus(ctx context.Context, id, status string, paidAt *time.Time, midtransOrderID, paymentURL *string, paymentLinkExpiresAt *time.Time) error
	CreateInvoice(ctx context.Context, inv *models.BillingInvoice) error
	GetTenantOwnerEmail(ctx context.Context, tenantID string) (string, error)
	GetInvoiceByID(ctx context.Context, id string) (*models.BillingInvoice, error)
	GetInvoicesByTenantID(ctx context.Context, tenantID string) ([]*models.BillingInvoice, error)
	GetPaymentAttemptsByInvoiceID(ctx context.Context, tenantID, invoiceID string) ([]*models.BillingPaymentAttempt, error)
	GetInvoiceByMidtransOrderID(ctx context.Context, midtransOrderID string) (*models.BillingInvoice, error)
	CreatePaymentAttempt(ctx context.Context, attempt *models.BillingPaymentAttempt) error
	UpdateTenantSubscribedAt(ctx context.Context, tenantID, plan, billingInterval string, subscriptionEndsAt time.Time) error
}

type snapPaymentCreator interface {
	CreateSnapPayment(ctx context.Context, inv *models.BillingInvoice, tenantEmail, tenantBusinessName string) (*SnapPaymentResult, error)
}

// SubscriptionService contains the core billing business logic.
type SubscriptionService struct {
	db               *sql.DB
	repo             billingRepository
	publisher        *queue.EventPublisher
	paymentSvc       snapPaymentCreator
	cacheInvalidator SubscriptionCacheInvalidator
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

// SetSubscriptionCacheInvalidator wires cache invalidation for status-changing billing events.
func (s *SubscriptionService) SetSubscriptionCacheInvalidator(invalidator SubscriptionCacheInvalidator) {
	s.cacheInvalidator = invalidator
}

// UpgradeResponse is returned by UpgradeSubscription.
type UpgradeResponse struct {
	Invoice       *models.BillingInvoice `json:"invoice"`
	InvoiceID     string                 `json:"invoice_id"`
	InvoiceNumber string                 `json:"invoice_number"`
	AmountIDR     int                    `json:"amount_idr"`
	SnapToken     string                 `json:"snap_token"`
	PaymentURL    string                 `json:"payment_url"`
}

// CycleSwitchResponse is returned when changing billing intervals requires payment.
type CycleSwitchResponse struct {
	Status                 string                 `json:"status"`
	SubscriptionStatus     string                 `json:"subscription_status"`
	CurrentBillingInterval string                 `json:"current_billing_interval"`
	TargetBillingInterval  string                 `json:"billing_interval"`
	SubscriptionEndsAt     *time.Time             `json:"subscription_ends_at,omitempty"`
	Invoice                *models.BillingInvoice `json:"invoice"`
	InvoiceID              string                 `json:"invoice_id"`
	InvoiceNumber          string                 `json:"invoice_number"`
	AmountIDR              int                    `json:"amount_idr"`
	SnapToken              string                 `json:"snap_token"`
	PaymentURL             string                 `json:"payment_url"`
	DueAt                  time.Time              `json:"due_at"`
	PaymentLinkExpiresAt   *time.Time             `json:"payment_link_expires_at,omitempty"`
}

// CycleSwitchLockedError is returned when an active annual subscription cannot
// revert to monthly yet.
type CycleSwitchLockedError struct {
	SubscriptionEndsAt time.Time
}

func (e *CycleSwitchLockedError) Error() string {
	return fmt.Sprintf("yearly subscription can be changed to monthly after %s", e.SubscriptionEndsAt.Format(time.RFC3339))
}

// AsCycleSwitchLockedError extracts the yearly lock error for HTTP mapping.
func AsCycleSwitchLockedError(err error) (*CycleSwitchLockedError, bool) {
	var locked *CycleSwitchLockedError
	if errors.As(err, &locked) {
		return locked, true
	}
	return nil, false
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

	graceDays := GetGracePeriodDaysFromEnv()
	status := computeSubscriptionStatus(tenant, graceDays)
	days := computeDaysRemaining(tenant, status, graceDays)
	retentionCleanupAt := computeRetentionCleanupAt(tenant)
	pendingInvoice, err := s.repo.GetPendingInvoiceByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current invoice: %w", err)
	}
	paymentDueAt := computePaymentDueAt(tenant, pendingInvoice)

	return &models.SubscriptionStatusResponse{
		TenantID:           tenant.ID,
		SubscriptionPlan:   tenant.SubscriptionPlan,
		SubscriptionStatus: status,
		BillingCycle:       tenant.BillingCycle,
		TrialEndsAt:        tenant.TrialEndsAt,
		SubscriptionEndsAt: tenant.SubscriptionEndsAt,
		RetentionStartedAt: tenant.RetentionStartedAt,
		RetentionCleanupAt: retentionCleanupAt,
		DataAnonymizedAt:   tenant.DataAnonymizedAt,
		PaymentDueAt:       paymentDueAt,
		CurrentInvoice:     pendingInvoice,
		IsActive:           status != "expired" && status != "cancelled",
		DaysRemaining:      days,
	}, nil
}

// UpdateBillingCycle updates the tenant's preferred billing cycle.
func (s *SubscriptionService) UpdateBillingCycle(ctx context.Context, tenantID, billingInterval string) error {
	if err := validateBillingInterval(billingInterval); err != nil {
		return err
	}

	tenant, err := s.repo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}
	if tenant == nil {
		return errors.New("tenant not found")
	}
	if tenant.BillingCycle != "" && tenant.BillingCycle != billingInterval {
		return errors.New("billing cycle changes require payment; use the cycle switch endpoint")
	}
	return s.repo.UpdateTenantBillingCycle(ctx, tenantID, billingInterval)
}

// UpgradeSubscription creates an invoice and a Midtrans Snap payment for the tenant.
func (s *SubscriptionService) UpgradeSubscription(ctx context.Context, tenantID, billingInterval string) (*UpgradeResponse, error) {
	if err := validateBillingInterval(billingInterval); err != nil {
		return nil, err
	}

	tenant, err := s.repo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	if tenant == nil {
		return nil, errors.New("tenant not found")
	}

	now := time.Now().UTC()
	return s.createSubscriptionPayment(ctx, tenant, billingInterval, now, computeInvoiceDueAt(tenant, now), true)
}

// SwitchBillingCycle creates a payment-backed invoice for a billing-cycle change.
func (s *SubscriptionService) SwitchBillingCycle(ctx context.Context, tenantID, billingInterval string) (*CycleSwitchResponse, error) {
	if err := validateBillingInterval(billingInterval); err != nil {
		return nil, err
	}

	tenant, err := s.repo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	if tenant == nil {
		return nil, errors.New("tenant not found")
	}

	currentInterval := tenant.BillingCycle
	if currentInterval == "" {
		currentInterval = "monthly"
	}

	now := time.Now().UTC()
	if err := validateBillingCycleSwitch(tenant, currentInterval, billingInterval, now); err != nil {
		return nil, err
	}

	result, err := s.createSubscriptionPayment(ctx, tenant, billingInterval, now, now, true)
	if err != nil {
		return nil, err
	}

	graceDays := GetGracePeriodDaysFromEnv()
	return &CycleSwitchResponse{
		Status:                 "payment_required",
		SubscriptionStatus:     computeSubscriptionStatus(tenant, graceDays),
		CurrentBillingInterval: currentInterval,
		TargetBillingInterval:  billingInterval,
		SubscriptionEndsAt:     tenant.SubscriptionEndsAt,
		Invoice:                result.Invoice,
		InvoiceID:              result.InvoiceID,
		InvoiceNumber:          result.InvoiceNumber,
		AmountIDR:              result.AmountIDR,
		SnapToken:              result.SnapToken,
		PaymentURL:             result.PaymentURL,
		DueAt:                  result.Invoice.DueAt,
		PaymentLinkExpiresAt:   result.Invoice.PaymentLinkExpiresAt,
	}, nil
}

func (s *SubscriptionService) createSubscriptionPayment(ctx context.Context, tenant *models.Tenant, billingInterval string, periodStart, dueAt time.Time, cancelOtherIntervals bool) (*UpgradeResponse, error) {
	if tenant == nil {
		return nil, errors.New("tenant not found")
	}
	if cancelOtherIntervals {
		if err := s.repo.CancelPendingInvoicesByTenantIDExceptInterval(ctx, tenant.ID, billingInterval); err != nil {
			return nil, fmt.Errorf("failed to cancel stale pending invoices: %w", err)
		}
	}

	existing, err := s.repo.GetPendingInvoiceByTenantIDAndInterval(ctx, tenant.ID, billingInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing invoice: %w", err)
	}
	if existing != nil {
		if InvoiceHasInvalidAmount(existing) {
			if err := s.cancelInvalidPendingInvoice(ctx, existing); err != nil {
				return nil, err
			}
			return s.createFreshSubscriptionPayment(ctx, tenant, billingInterval, periodStart, dueAt)
		}
		if isPaymentLinkExpired(existing, time.Now().UTC()) {
			if err := s.repo.UpdateInvoiceStatus(ctx, existing.ID, "expired", nil, nil, nil, nil); err != nil {
				return nil, fmt.Errorf("failed to expire stale pending invoice: %w", err)
			}
		} else if existing.MidtransPaymentURL != nil {
			return upgradeResponseFromInvoice(existing, "", *existing.MidtransPaymentURL), nil
		} else {
			return s.createPaymentForInvoice(ctx, tenant, existing)
		}
	}

	return s.createFreshSubscriptionPayment(ctx, tenant, billingInterval, periodStart, dueAt)
}

func (s *SubscriptionService) createFreshSubscriptionPayment(ctx context.Context, tenant *models.Tenant, billingInterval string, periodStart, dueAt time.Time) (*UpgradeResponse, error) {
	amount, err := ComputeInvoiceAmount(billingInterval)
	if err != nil {
		return nil, err
	}
	periodStart = periodStart.UTC()
	dueAt = dueAt.UTC()
	periodEnd := computePeriodEnd(periodStart, billingInterval)
	invoiceNumber := BuildBillingInvoiceNumber(time.Now().UTC())

	inv := &models.BillingInvoice{
		TenantID:        tenant.ID,
		InvoiceNumber:   invoiceNumber,
		AmountIDR:       amount,
		BillingInterval: billingInterval,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		DueAt:           dueAt,
		Status:          "pending",
	}
	if err := s.repo.CreateInvoice(ctx, inv); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	return s.createPaymentForInvoice(ctx, tenant, inv)
}

func (s *SubscriptionService) cancelInvalidPendingInvoice(ctx context.Context, inv *models.BillingInvoice) error {
	if err := s.repo.UpdateInvoiceStatus(ctx, inv.ID, "cancelled", nil, nil, nil, nil); err != nil {
		return fmt.Errorf("failed to cancel invalid pending invoice: %w", err)
	}
	inv.Status = "cancelled"
	return nil
}

func (s *SubscriptionService) createPaymentForInvoice(ctx context.Context, tenant *models.Tenant, inv *models.BillingInvoice) (*UpgradeResponse, error) {
	ownerEmail, _ := s.repo.GetTenantOwnerEmail(ctx, inv.TenantID)
	snapResult, err := s.paymentSvc.CreateSnapPayment(ctx, inv, ownerEmail, tenant.BusinessName)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Update invoice with Midtrans details.
	paymentLinkExpiresAt := time.Now().UTC().Add(PaymentLinkTTL)
	if err := s.repo.UpdateInvoiceStatus(ctx, inv.ID, "pending", nil, &snapResult.OrderID, &snapResult.PaymentURL, &paymentLinkExpiresAt); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}
	inv.MidtransOrderID = &snapResult.OrderID
	inv.MidtransPaymentURL = &snapResult.PaymentURL
	inv.PaymentLinkExpiresAt = &paymentLinkExpiresAt
	_ = s.recordPaymentAttempt(ctx, inv, &snapResult.OrderID, nil, "pending", nil)

	return upgradeResponseFromInvoice(inv, snapResult.Token, snapResult.PaymentURL), nil
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

	if InvoiceHasInvalidAmount(inv) {
		if err := s.cancelInvalidPendingInvoice(ctx, inv); err != nil {
			return nil, err
		}
		return s.createFreshSubscriptionPayment(ctx, tenant, inv.BillingInterval, inv.PeriodStart, inv.DueAt)
	}

	now := time.Now().UTC()
	if isPaymentLinkExpired(inv, now) {
		if err := s.repo.UpdateInvoiceStatus(ctx, inv.ID, "expired", nil, nil, nil, nil); err != nil {
			return nil, fmt.Errorf("failed to expire stale pending invoice: %w", err)
		}
		return s.createSubscriptionPayment(ctx, tenant, inv.BillingInterval, inv.PeriodStart, inv.DueAt, false)
	}
	if inv.MidtransPaymentURL != nil {
		return upgradeResponseFromInvoice(inv, "", *inv.MidtransPaymentURL), nil
	}

	return s.createPaymentForInvoice(ctx, tenant, inv)
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

// GetInvoicePaymentAttempts returns payment attempts for a tenant invoice.
func (s *SubscriptionService) GetInvoicePaymentAttempts(ctx context.Context, tenantID, invoiceID string) ([]*models.BillingPaymentAttempt, error) {
	inv, err := s.GetInvoice(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetPaymentAttemptsByInvoiceID(ctx, tenantID, inv.ID)
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
		if inv.Status != "pending" {
			return nil
		}

		now := time.Now().UTC()
		if err := s.repo.UpdateInvoiceStatus(ctx, inv.ID, "paid", &now, nil, nil, nil); err != nil {
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
		s.invalidateSubscriptionCache(ctx, inv.TenantID)

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

func (s *SubscriptionService) invalidateSubscriptionCache(ctx context.Context, tenantID string) {
	if s.cacheInvalidator == nil {
		return
	}
	if err := s.cacheInvalidator.InvalidateSubscriptionStatus(ctx, tenantID); err != nil {
		fmt.Printf("Warning: failed to invalidate subscription cache for tenant %s: %v\n", tenantID, err)
	}
}

// CreateNextPeriodInvoice creates a renewal invoice for an active tenant if one doesn't exist yet.
func (s *SubscriptionService) CreateNextPeriodInvoice(ctx context.Context, tenant *models.Tenant) (*models.BillingInvoice, error) {
	// Check for an existing pending invoice.
	existing, err := s.repo.GetPendingInvoiceByTenantID(ctx, tenant.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if !InvoiceHasInvalidAmount(existing) {
			return existing, nil
		}
		if err := s.cancelInvalidPendingInvoice(ctx, existing); err != nil {
			return nil, err
		}
	}

	billingInterval := tenant.BillingCycle
	amount, err := ComputeInvoiceAmount(billingInterval)
	if err != nil {
		return nil, err
	}

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

	invoiceNumber := BuildBillingInvoiceNumber(time.Now().UTC())

	inv := &models.BillingInvoice{
		TenantID:        tenant.ID,
		InvoiceNumber:   invoiceNumber,
		AmountIDR:       amount,
		BillingInterval: billingInterval,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		DueAt:           computeInvoiceDueAt(tenant, periodStart),
		Status:          "pending",
	}
	if err := s.repo.CreateInvoice(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *SubscriptionService) recordPaymentAttempt(ctx context.Context, inv *models.BillingInvoice, orderID, paymentMethod *string, status string, errorMsg *string) error {
	if inv == nil {
		return nil
	}
	attempt := &models.BillingPaymentAttempt{
		InvoiceID:       inv.ID,
		TenantID:        inv.TenantID,
		AmountIDR:       inv.AmountIDR,
		MidtransOrderID: orderID,
		PaymentMethod:   paymentMethod,
		Status:          status,
		ErrorMsg:        errorMsg,
	}
	return s.repo.CreatePaymentAttempt(ctx, attempt)
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

func computeRetentionCleanupAt(tenant *models.Tenant) *time.Time {
	if tenant == nil || tenant.RetentionStartedAt == nil {
		return nil
	}
	cleanupAt := tenant.RetentionStartedAt.AddDate(0, 0, GetRetentionDaysFromEnv())
	return &cleanupAt
}

func computeInvoiceDueAt(tenant *models.Tenant, fallback time.Time) time.Time {
	now := time.Now().UTC()
	if tenant != nil {
		if tenant.SubscriptionPlan == "trial" && tenant.TrialEndsAt != nil && tenant.TrialEndsAt.After(now) {
			return tenant.TrialEndsAt.UTC()
		}
		if tenant.SubscriptionEndsAt != nil && tenant.SubscriptionEndsAt.After(now) {
			return tenant.SubscriptionEndsAt.UTC()
		}
	}
	if fallback.IsZero() {
		return now
	}
	return fallback.UTC()
}

func computePaymentDueAt(tenant *models.Tenant, pendingInvoice *models.BillingInvoice) *time.Time {
	if pendingInvoice != nil {
		dueAt := pendingInvoice.DueAt
		return &dueAt
	}
	if tenant == nil {
		return nil
	}
	if tenant.SubscriptionPlan == "trial" && tenant.TrialEndsAt != nil {
		dueAt := tenant.TrialEndsAt.UTC()
		return &dueAt
	}
	if tenant.SubscriptionEndsAt != nil {
		dueAt := tenant.SubscriptionEndsAt.UTC()
		return &dueAt
	}
	return nil
}

func validateBillingInterval(billingInterval string) error {
	if billingInterval != "monthly" && billingInterval != "annual" {
		return errors.New("billing_interval must be 'monthly' or 'annual'")
	}
	return nil
}

func computePeriodEnd(periodStart time.Time, billingInterval string) time.Time {
	if billingInterval == "annual" {
		return periodStart.AddDate(1, 0, 0)
	}
	return periodStart.AddDate(0, 1, 0)
}

// BuildBillingInvoiceNumber returns an invoice number using the UTC month and
// Unix epoch milliseconds as the unique suffix.
func BuildBillingInvoiceNumber(now time.Time) string {
	now = now.UTC()
	return fmt.Sprintf("INV-%s-%d", now.Format("200601"), now.UnixMilli())
}

func isPaymentLinkExpired(inv *models.BillingInvoice, now time.Time) bool {
	if inv == nil || inv.MidtransPaymentURL == nil {
		return false
	}
	if inv.PaymentLinkExpiresAt == nil {
		return true
	}
	return !inv.PaymentLinkExpiresAt.After(now)
}

func validateBillingCycleSwitch(tenant *models.Tenant, currentInterval, targetInterval string, now time.Time) error {
	if currentInterval == targetInterval {
		return fmt.Errorf("tenant is already using %s billing", targetInterval)
	}
	if currentInterval == "annual" && targetInterval == "monthly" &&
		tenant != nil && tenant.SubscriptionEndsAt != nil && tenant.SubscriptionEndsAt.After(now) {
		return &CycleSwitchLockedError{SubscriptionEndsAt: tenant.SubscriptionEndsAt.UTC()}
	}
	return nil
}

func upgradeResponseFromInvoice(inv *models.BillingInvoice, snapToken, paymentURL string) *UpgradeResponse {
	return &UpgradeResponse{
		Invoice:       inv,
		InvoiceID:     inv.ID,
		InvoiceNumber: inv.InvoiceNumber,
		AmountIDR:     inv.AmountIDR,
		SnapToken:     snapToken,
		PaymentURL:    paymentURL,
	}
}

func verifyMidtransSignature(orderID, statusCode, grossAmount, serverKey, received string) bool {
	raw := orderID + statusCode + grossAmount + serverKey
	sum := sha512.Sum512([]byte(raw))
	computed := fmt.Sprintf("%x", sum)
	return strings.EqualFold(computed, received)
}
