package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/point-of-sale-system/order-service/src/models"
)

// ErrRefundAlreadyExists means the payment already has a refund that is
// processing or succeeded.
var ErrRefundAlreadyExists = errors.New("payment already has an active refund")

// PaymentRefundRepository stores refunds of guest order payments.
type PaymentRefundRepository struct {
	db *sql.DB
}

func NewPaymentRefundRepository(db *sql.DB) *PaymentRefundRepository {
	return &PaymentRefundRepository{db: db}
}

// Create inserts a refund. A second active refund of the same payment is
// rejected by a partial unique index and returned as ErrRefundAlreadyExists,
// so two concurrent refund requests cannot both reach the gateway.
func (r *PaymentRefundRepository) Create(ctx context.Context, refund *models.PaymentRefund) error {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO payment_refunds (
			payment_transaction_id, order_id, tenant_id, payment_gateway,
			partner_refund_no, amount, reason, status,
			requested_by_user_id, requested_by_name
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`,
		refund.PaymentTransactionID, refund.OrderID, refund.TenantID, refund.PaymentGateway,
		refund.PartnerRefundNo, refund.Amount, refund.Reason, refund.Status,
		refund.RequestedByUserID, refund.RequestedByName,
	).Scan(&refund.ID, &refund.CreatedAt, &refund.UpdatedAt)

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" && pqErr.Constraint == "uq_payment_refunds_active_per_payment" {
		return ErrRefundAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}
	return nil
}

// UpdateResult records the gateway's answer to a refund.
func (r *PaymentRefundRepository) UpdateResult(ctx context.Context, refund *models.PaymentRefund) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payment_refunds
		SET status = $2, gateway_refund_no = $3, failure_reason = $4,
		    response_payload = $5, updated_at = NOW()
		WHERE id = $1`,
		refund.ID, refund.Status, refund.GatewayRefundNo, refund.FailureReason, nullableJSON(refund.ResponsePayload),
	)
	if err != nil {
		return fmt.Errorf("failed to update refund: %w", err)
	}
	return nil
}

// ListByOrderID returns an order's refunds, newest first.
func (r *PaymentRefundRepository) ListByOrderID(ctx context.Context, orderID string) ([]models.PaymentRefund, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, payment_transaction_id, order_id, tenant_id, payment_gateway,
		       partner_refund_no, gateway_refund_no, amount, reason, status,
		       failure_reason, requested_by_user_id, requested_by_name, created_at, updated_at
		FROM payment_refunds
		WHERE order_id = $1
		ORDER BY created_at DESC`, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to list refunds: %w", err)
	}
	defer rows.Close()

	refunds := []models.PaymentRefund{}
	for rows.Next() {
		var refund models.PaymentRefund
		if err := rows.Scan(
			&refund.ID, &refund.PaymentTransactionID, &refund.OrderID, &refund.TenantID, &refund.PaymentGateway,
			&refund.PartnerRefundNo, &refund.GatewayRefundNo, &refund.Amount, &refund.Reason, &refund.Status,
			&refund.FailureReason, &refund.RequestedByUserID, &refund.RequestedByName, &refund.CreatedAt, &refund.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan refund: %w", err)
		}
		refunds = append(refunds, refund)
	}
	return refunds, rows.Err()
}

func nullableJSON(raw []byte) interface{} {
	if len(raw) == 0 {
		return nil
	}
	return raw
}
