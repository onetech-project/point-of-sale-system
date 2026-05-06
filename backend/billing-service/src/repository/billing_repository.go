package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pos/billing-service/src/models"
)

// BillingRepository handles all database operations for billing.
type BillingRepository struct {
	db *sql.DB
}

// NewBillingRepository creates a new BillingRepository.
func NewBillingRepository(db *sql.DB) *BillingRepository {
	return &BillingRepository{db: db}
}

// scanTenant scans a single tenant row into a Tenant model.
func scanTenant(row interface {
	Scan(dest ...interface{}) error
}) (*models.Tenant, error) {
	t := &models.Tenant{}
	var trialStartedAt, trialEndsAt, subscribedAt, subscriptionEndsAt sql.NullTime
	err := row.Scan(
		&t.ID, &t.BusinessName, &t.SubscriptionPlan, &t.BillingCycle,
		&trialStartedAt, &trialEndsAt, &subscribedAt, &subscriptionEndsAt,
		&t.SubscriptionStatus, &t.StorageQuotaBytes,
	)
	if err != nil {
		return nil, err
	}
	if trialStartedAt.Valid {
		t.TrialStartedAt = &trialStartedAt.Time
	}
	if trialEndsAt.Valid {
		t.TrialEndsAt = &trialEndsAt.Time
	}
	if subscribedAt.Valid {
		t.SubscribedAt = &subscribedAt.Time
	}
	if subscriptionEndsAt.Valid {
		t.SubscriptionEndsAt = &subscriptionEndsAt.Time
	}
	return t, nil
}

const tenantSelectColumns = `
	id, business_name, subscription_plan, billing_cycle,
	trial_started_at, trial_ends_at, subscribed_at, subscription_ends_at,
	subscription_status, storage_quota_bytes`

// GetTenantByID returns the tenant with the given ID.
func (r *BillingRepository) GetTenantByID(ctx context.Context, id string) (*models.Tenant, error) {
	query := fmt.Sprintf(`SELECT %s FROM tenants WHERE id = $1 AND status != 'deleted'`, tenantSelectColumns)
	row := r.db.QueryRowContext(ctx, query, id)
	t, err := scanTenant(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// GetTenantsWithExpiringTrials returns trial tenants whose trial ends within the next daysAhead days.
func (r *BillingRepository) GetTenantsWithExpiringTrials(ctx context.Context, daysAhead int) ([]*models.Tenant, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM tenants
		WHERE subscription_plan = 'trial'
		  AND subscription_status = 'trial'
		  AND trial_ends_at > NOW()
		  AND trial_ends_at <= NOW() + ($1 * INTERVAL '1 day')
		ORDER BY trial_ends_at ASC`, tenantSelectColumns)
	return r.queryTenants(ctx, query, daysAhead)
}

// GetExpiredTrialTenants returns tenants whose trial has ended but status is still 'trial'.
func (r *BillingRepository) GetExpiredTrialTenants(ctx context.Context) ([]*models.Tenant, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM tenants
		WHERE subscription_plan = 'trial'
		  AND subscription_status = 'trial'
		  AND trial_ends_at < NOW()
		ORDER BY trial_ends_at ASC`, tenantSelectColumns)
	return r.queryTenants(ctx, query)
}

// GetExpiredSubscriptionTenants returns active-plan tenants whose subscription has ended but status is still 'active'.
func (r *BillingRepository) GetExpiredSubscriptionTenants(ctx context.Context) ([]*models.Tenant, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM tenants
		WHERE subscription_plan != 'trial'
		  AND subscription_status = 'active'
		  AND subscription_ends_at < NOW()
		ORDER BY subscription_ends_at ASC`, tenantSelectColumns)
	return r.queryTenants(ctx, query)
}

// GetGracePeriodExpiredTenants returns tenants in grace_period whose grace window (graceDays after expiry) has passed.
func (r *BillingRepository) GetGracePeriodExpiredTenants(ctx context.Context, graceDays int) ([]*models.Tenant, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM tenants
		WHERE subscription_status = 'grace_period'
		  AND (
		    (subscription_plan = 'trial'    AND trial_ends_at       < NOW() - ($1 * INTERVAL '1 day'))
		    OR
		    (subscription_plan != 'trial'   AND subscription_ends_at < NOW() - ($1 * INTERVAL '1 day'))
		  )
		ORDER BY id ASC`, tenantSelectColumns)
	return r.queryTenants(ctx, query, graceDays)
}

func (r *BillingRepository) queryTenants(ctx context.Context, query string, args ...interface{}) ([]*models.Tenant, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

// UpdateTenantSubscriptionStatus updates the subscription_status column for a tenant.
func (r *BillingRepository) UpdateTenantSubscriptionStatus(ctx context.Context, tenantID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET subscription_status = $2, updated_at = NOW() WHERE id = $1`,
		tenantID, status,
	)
	return err
}

// UpdateTenantSubscribedAt updates all subscription fields after a successful payment.
func (r *BillingRepository) UpdateTenantSubscribedAt(ctx context.Context, tenantID, plan, billingInterval string, subscriptionEndsAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tenants SET
		  subscription_plan     = $2,
		  billing_cycle         = $3,
		  subscribed_at         = NOW(),
		  subscription_ends_at  = $4,
		  subscription_status   = 'active',
		  updated_at            = NOW()
		WHERE id = $1`,
		tenantID, plan, billingInterval, subscriptionEndsAt,
	)
	return err
}

// UpdateTenantBillingCycle updates the billing_cycle preference for a tenant.
func (r *BillingRepository) UpdateTenantBillingCycle(ctx context.Context, tenantID, billingInterval string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tenants SET billing_cycle = $2, updated_at = NOW() WHERE id = $1`,
		tenantID, billingInterval,
	)
	return err
}

// CreateInvoice inserts a new billing invoice and sets its ID.
func (r *BillingRepository) CreateInvoice(ctx context.Context, inv *models.BillingInvoice) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO billing_invoices
		  (tenant_id, invoice_number, amount_idr, billing_interval, period_start, period_end, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		RETURNING id, created_at, updated_at`,
		inv.TenantID, inv.InvoiceNumber, inv.AmountIDR, inv.BillingInterval,
		inv.PeriodStart, inv.PeriodEnd,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
}

// GetInvoiceByID returns an invoice by its UUID.
func (r *BillingRepository) GetInvoiceByID(ctx context.Context, id string) (*models.BillingInvoice, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, invoice_number, amount_idr, billing_interval,
		       period_start, period_end, status, paid_at,
		       midtrans_order_id, midtrans_payment_url, created_at, updated_at
		FROM billing_invoices WHERE id = $1`, id)
	return scanInvoice(row)
}

// GetInvoicesByTenantID returns all invoices for a tenant, newest first.
func (r *BillingRepository) GetInvoicesByTenantID(ctx context.Context, tenantID string) ([]*models.BillingInvoice, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tenant_id, invoice_number, amount_idr, billing_interval,
		       period_start, period_end, status, paid_at,
		       midtrans_order_id, midtrans_payment_url, created_at, updated_at
		FROM billing_invoices
		WHERE tenant_id = $1
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []*models.BillingInvoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

// GetPendingInvoiceByTenantID returns the most recent pending invoice for a tenant.
func (r *BillingRepository) GetPendingInvoiceByTenantID(ctx context.Context, tenantID string) (*models.BillingInvoice, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, invoice_number, amount_idr, billing_interval,
		       period_start, period_end, status, paid_at,
		       midtrans_order_id, midtrans_payment_url, created_at, updated_at
		FROM billing_invoices
		WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1`, tenantID)
	inv, err := scanInvoice(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return inv, err
}

// UpdateInvoiceStatus updates the status, paid_at, and Midtrans fields of an invoice.
func (r *BillingRepository) UpdateInvoiceStatus(ctx context.Context, id, status string, paidAt *time.Time, midtransOrderID, paymentURL *string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE billing_invoices
		SET status              = $2,
		    paid_at             = $3,
		    midtrans_order_id   = $4,
		    midtrans_payment_url = $5,
		    updated_at          = NOW()
		WHERE id = $1`,
		id, status, paidAt, midtransOrderID, paymentURL,
	)
	return err
}

// CreatePaymentAttempt inserts a new payment attempt record.
func (r *BillingRepository) CreatePaymentAttempt(ctx context.Context, attempt *models.BillingPaymentAttempt) error {
	return r.db.QueryRowContext(ctx, `
		INSERT INTO billing_payment_attempts
		  (invoice_id, tenant_id, amount_idr, midtrans_order_id, payment_method, status, error_msg)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		attempt.InvoiceID, attempt.TenantID, attempt.AmountIDR,
		attempt.MidtransOrderID, attempt.PaymentMethod,
		attempt.Status, attempt.ErrorMsg,
	).Scan(&attempt.ID, &attempt.CreatedAt)
}

// GetInvoiceByMidtransOrderID finds an invoice by its Midtrans order ID.
func (r *BillingRepository) GetInvoiceByMidtransOrderID(ctx context.Context, midtransOrderID string) (*models.BillingInvoice, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tenant_id, invoice_number, amount_idr, billing_interval,
		       period_start, period_end, status, paid_at,
		       midtrans_order_id, midtrans_payment_url, created_at, updated_at
		FROM billing_invoices
		WHERE midtrans_order_id = $1
		LIMIT 1`, midtransOrderID)
	inv, err := scanInvoice(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return inv, err
}

// CountInvoicesThisMonth returns the number of invoices created in the current calendar month.
func (r *BillingRepository) CountInvoicesThisMonth(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM billing_invoices
		WHERE DATE_TRUNC('month', created_at) = DATE_TRUNC('month', NOW())`).Scan(&count)
	return count, err
}

// GetTenantsNearingRenewal returns active-plan tenants whose subscription ends within the next daysAhead days.
func (r *BillingRepository) GetTenantsNearingRenewal(ctx context.Context, daysAhead int) ([]*models.Tenant, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM tenants
		WHERE subscription_plan != 'trial'
		  AND subscription_status = 'active'
		  AND subscription_ends_at BETWEEN NOW() AND NOW() + ($1 * INTERVAL '1 day')
		ORDER BY subscription_ends_at ASC`, tenantSelectColumns)
	return r.queryTenants(ctx, query, daysAhead)
}

// GetTenantOwnerEmail returns the email of the active owner user for a tenant.
// Note: the email may be Vault-encrypted; the notification service handles decryption.
func (r *BillingRepository) GetTenantOwnerEmail(ctx context.Context, tenantID string) (string, error) {
	var email string
	err := r.db.QueryRowContext(ctx, `
		SELECT email FROM users
		WHERE tenant_id = $1 AND role = 'owner' AND status = 'active'
		LIMIT 1`, tenantID).Scan(&email)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return email, err
}

// scanInvoice scans a row into a BillingInvoice model.
func scanInvoice(row interface {
	Scan(dest ...interface{}) error
}) (*models.BillingInvoice, error) {
	inv := &models.BillingInvoice{}
	var paidAt sql.NullTime
	var midtransOrderID, midtransPaymentURL sql.NullString
	err := row.Scan(
		&inv.ID, &inv.TenantID, &inv.InvoiceNumber, &inv.AmountIDR, &inv.BillingInterval,
		&inv.PeriodStart, &inv.PeriodEnd, &inv.Status, &paidAt,
		&midtransOrderID, &midtransPaymentURL, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if paidAt.Valid {
		inv.PaidAt = &paidAt.Time
	}
	if midtransOrderID.Valid {
		inv.MidtransOrderID = &midtransOrderID.String
	}
	if midtransPaymentURL.Valid {
		inv.MidtransPaymentURL = &midtransPaymentURL.String
	}
	return inv, nil
}
