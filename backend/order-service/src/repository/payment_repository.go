package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// CreatePaymentTransaction creates a new payment transaction record
func (r *PaymentRepository) CreatePaymentTransaction(ctx context.Context, tx *sql.Tx, payment *models.PaymentTransaction) error {
	query := `
		INSERT INTO payment_transactions (
			order_id, midtrans_transaction_id, midtrans_order_id, amount,
			payment_type, transaction_status, fraud_status,
			notification_payload, signature_key, signature_verified,
			qr_code_url, qr_string, expiry_time,
			idempotency_key, notification_received_at, settled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at
	`

	return tx.QueryRowContext(
		ctx,
		query,
		payment.OrderID,
		payment.MidtransTransactionID,
		payment.MidtransOrderID,
		payment.Amount,
		payment.PaymentType,
		payment.TransactionStatus,
		payment.FraudStatus,
		payment.NotificationPayload,
		payment.SignatureKey,
		payment.SignatureVerified,
		payment.QRCodeURL,
		payment.QRString,
		payment.ExpiryTime,
		payment.IdempotencyKey,
		payment.NotificationReceivedAt,
		payment.SettledAt,
	).Scan(&payment.ID, &payment.CreatedAt)
}

// GetPaymentByOrderID retrieves payment transaction by order ID
func (r *PaymentRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (*models.PaymentTransaction, error) {
	query := `
		SELECT id, order_id, midtrans_transaction_id, midtrans_order_id,
			amount, payment_type, transaction_status, fraud_status,
			notification_payload, signature_key, signature_verified,
			qr_code_url, qr_string, expiry_time,
			created_at, notification_received_at, settled_at, idempotency_key
		FROM payment_transactions
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	payment := &models.PaymentTransaction{}
	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.MidtransTransactionID,
		&payment.MidtransOrderID,
		&payment.Amount,
		&payment.PaymentType,
		&payment.TransactionStatus,
		&payment.FraudStatus,
		&payment.NotificationPayload,
		&payment.SignatureKey,
		&payment.SignatureVerified,
		&payment.QRCodeURL,
		&payment.QRString,
		&payment.ExpiryTime,
		&payment.CreatedAt,
		&payment.NotificationReceivedAt,
		&payment.SettledAt,
		&payment.IdempotencyKey,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return payment, err
}

// GetPaymentByTransactionID retrieves payment by Midtrans transaction ID
func (r *PaymentRepository) GetPaymentByTransactionID(ctx context.Context, transactionID string) (*models.PaymentTransaction, error) {
	query := `
		SELECT id, order_id, midtrans_transaction_id, midtrans_order_id,
			amount, payment_type, transaction_status, fraud_status,
			notification_payload, signature_key, signature_verified,
			qr_code_url, qr_string, expiry_time,
			created_at, notification_received_at, settled_at, idempotency_key
		FROM payment_transactions
		WHERE midtrans_transaction_id = $1
	`

	payment := &models.PaymentTransaction{}
	err := r.db.QueryRowContext(ctx, query, transactionID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.MidtransTransactionID,
		&payment.MidtransOrderID,
		&payment.Amount,
		&payment.PaymentType,
		&payment.TransactionStatus,
		&payment.FraudStatus,
		&payment.NotificationPayload,
		&payment.SignatureKey,
		&payment.SignatureVerified,
		&payment.QRCodeURL,
		&payment.QRString,
		&payment.ExpiryTime,
		&payment.CreatedAt,
		&payment.NotificationReceivedAt,
		&payment.SettledAt,
		&payment.IdempotencyKey,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return payment, err
}

// GetPaymentByIdempotencyKey checks if a payment with this idempotency key already exists
func (r *PaymentRepository) GetPaymentByIdempotencyKey(ctx context.Context, idempotencyKey string) (*models.PaymentTransaction, error) {
	query := `
		SELECT id, order_id, midtrans_transaction_id, midtrans_order_id,
			amount, payment_type, transaction_status, fraud_status,
			notification_payload, signature_key, signature_verified,
			qr_code_url, qr_string, expiry_time,
			created_at, notification_received_at, settled_at, idempotency_key
		FROM payment_transactions
		WHERE idempotency_key = $1
	`

	payment := &models.PaymentTransaction{}
	err := r.db.QueryRowContext(ctx, query, idempotencyKey).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.MidtransTransactionID,
		&payment.MidtransOrderID,
		&payment.Amount,
		&payment.PaymentType,
		&payment.TransactionStatus,
		&payment.FraudStatus,
		&payment.NotificationPayload,
		&payment.SignatureKey,
		&payment.SignatureVerified,
		&payment.QRCodeURL,
		&payment.QRString,
		&payment.ExpiryTime,
		&payment.CreatedAt,
		&payment.NotificationReceivedAt,
		&payment.SettledAt,
		&payment.IdempotencyKey,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return payment, err
}

// UpdatePaymentStatus updates the payment transaction status
func (r *PaymentRepository) UpdatePaymentStatus(ctx context.Context, id string, status string, settledAt *time.Time) error {
	query := `
		UPDATE payment_transactions
		SET transaction_status = $2, settled_at = $3
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, status, settledAt)
	return err
}

// UpdatePaymentStatusByTransactionID updates payment by Midtrans transaction ID
func (r *PaymentRepository) UpdatePaymentStatusByTransactionID(ctx context.Context, tx *sql.Tx, transactionID, status string, settledAt *time.Time, notificationPayload json.RawMessage) error {
	query := `
		UPDATE payment_transactions
		SET transaction_status = $2, 
		    settled_at = $3,
		    notification_payload = $4,
		    notification_received_at = NOW()
		WHERE midtrans_transaction_id = $1
	`

	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, transactionID, status, settledAt, notificationPayload)
	} else {
		_, err = r.db.ExecContext(ctx, query, transactionID, status, settledAt, notificationPayload)
	}
	return err
}
