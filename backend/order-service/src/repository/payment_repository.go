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

// ============================================================================
// Offline Order Payment Methods (T053-T057)
// ============================================================================

// getExecutor returns the appropriate SQL executor (transaction or database)
func (r *PaymentRepository) getExecutor(tx *sql.Tx) interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if tx != nil {
		return tx
	}
	return r.db
}

// CreatePaymentTerms inserts payment terms for an offline order with installments
// Must be called within a transaction alongside order creation
// T054: Implement CreatePaymentTerms method
func (r *PaymentRepository) CreatePaymentTerms(ctx context.Context, tx *sql.Tx, req *models.CreatePaymentTermsRequest) (string, error) {
	// Validate payment structure
	if req.TotalAmount <= 0 {
		return "", models.ErrInvalidTotalAmount
	}
	if req.DownPaymentAmount != nil && *req.DownPaymentAmount >= req.TotalAmount {
		return "", models.ErrDownPaymentTooLarge
	}
	if req.InstallmentCount < 0 {
		return "", models.ErrInvalidInstallmentCount
	}
	if req.InstallmentAmount < 0 {
		return "", models.ErrInvalidInstallmentAmount
	}

	// Calculate initial remaining balance (subtract down payment if exists)
	remainingBalance := req.TotalAmount
	totalPaid := 0
	if req.DownPaymentAmount != nil && *req.DownPaymentAmount > 0 {
		totalPaid = *req.DownPaymentAmount
		remainingBalance = req.TotalAmount - *req.DownPaymentAmount
	}

	query := `
		INSERT INTO payment_terms (
			order_id, total_amount, down_payment_amount,
			installment_count, installment_amount, payment_schedule,
			total_paid, remaining_balance,
			created_at, created_by_user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	var paymentTermsID string
	executor := r.getExecutor(tx)
	err := executor.QueryRowContext(
		ctx,
		query,
		req.OrderID,
		req.TotalAmount,
		req.DownPaymentAmount,
		req.InstallmentCount,
		req.InstallmentAmount,
		models.PaymentSchedule(req.PaymentSchedule),
		totalPaid,
		remainingBalance,
		time.Now(),
		req.CreatedByUserID,
	).Scan(&paymentTermsID)

	if err != nil {
		return "", err
	}

	return paymentTermsID, nil
}

// RecordPayment inserts a payment record and updates payment terms balance
// Must be called within a transaction to ensure atomicity
// T055: Implement RecordPayment method with trigger integration
func (r *PaymentRepository) RecordPayment(ctx context.Context, tx *sql.Tx, req *models.CreatePaymentRecordRequest) (string, error) {
	// Validate payment
	if req.AmountPaid <= 0 {
		return "", models.ErrInvalidPaymentAmount
	}
	if req.RemainingBalanceAfter < 0 {
		return "", models.ErrNegativeBalance
	}

	query := `
		INSERT INTO payment_records (
			order_id, payment_terms_id, payment_number,
			amount_paid, payment_date, payment_method,
			remaining_balance_after, recorded_by_user_id,
			notes, receipt_number, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	var paymentRecordID string
	executor := r.getExecutor(tx)
	err := executor.QueryRowContext(
		ctx,
		query,
		req.OrderID,
		req.PaymentTermsID,
		req.PaymentNumber,
		req.AmountPaid,
		time.Now(),
		req.PaymentMethod,
		req.RemainingBalanceAfter,
		req.RecordedByUserID,
		req.Notes,
		req.ReceiptNumber,
		time.Now(),
	).Scan(&paymentRecordID)

	if err != nil {
		return "", err
	}

	// Update payment terms total_paid and remaining_balance
	if req.PaymentTermsID != nil {
		updateQuery := `
			UPDATE payment_terms
			SET total_paid = total_paid + $1,
			    remaining_balance = remaining_balance - $1
			WHERE id = $2
		`
		_, err = executor.ExecContext(ctx, updateQuery, req.AmountPaid, *req.PaymentTermsID)
		if err != nil {
			return "", err
		}
	}

	return paymentRecordID, nil
}

// GetPaymentHistory retrieves all payment records for an order, ordered by payment date
// T056: Implement GetPaymentHistory method
func (r *PaymentRepository) GetPaymentHistory(ctx context.Context, orderID string) ([]models.PaymentRecord, error) {
	query := `
		SELECT 
			id, order_id, payment_terms_id, payment_number,
			amount_paid, payment_date, payment_method,
			remaining_balance_after, recorded_by_user_id,
			notes, receipt_number, created_at
		FROM payment_records
		WHERE order_id = $1
		ORDER BY payment_date DESC, payment_number ASC
	`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.PaymentRecord
	for rows.Next() {
		var record models.PaymentRecord
		err := rows.Scan(
			&record.ID,
			&record.OrderID,
			&record.PaymentTermsID,
			&record.PaymentNumber,
			&record.AmountPaid,
			&record.PaymentDate,
			&record.PaymentMethod,
			&record.RemainingBalanceAfter,
			&record.RecordedByUserID,
			&record.Notes,
			&record.ReceiptNumber,
			&record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// GetPaymentTerms retrieves payment terms for an order
// T057: Implement GetPaymentTerms method
func (r *PaymentRepository) GetPaymentTerms(ctx context.Context, orderID string) (*models.PaymentTerms, error) {
	query := `
		SELECT 
			id, order_id, total_amount, down_payment_amount,
			installment_count, installment_amount, payment_schedule,
			total_paid, remaining_balance,
			created_at, created_by_user_id
		FROM payment_terms
		WHERE order_id = $1
	`

	var terms models.PaymentTerms
	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&terms.ID,
		&terms.OrderID,
		&terms.TotalAmount,
		&terms.DownPaymentAmount,
		&terms.InstallmentCount,
		&terms.InstallmentAmount,
		&terms.PaymentSchedule,
		&terms.TotalPaid,
		&terms.RemainingBalance,
		&terms.CreatedAt,
		&terms.CreatedByUserID,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No payment terms for this order (full payment order)
	}
	if err != nil {
		return nil, err
	}

	return &terms, nil
}

// GetPaymentTermsByID retrieves payment terms by ID
func (r *PaymentRepository) GetPaymentTermsByID(ctx context.Context, paymentTermsID string) (*models.PaymentTerms, error) {
	query := `
		SELECT 
			id, order_id, total_amount, down_payment_amount,
			installment_count, installment_amount, payment_schedule,
			total_paid, remaining_balance,
			created_at, created_by_user_id
		FROM payment_terms
		WHERE id = $1
	`

	var terms models.PaymentTerms
	err := r.db.QueryRowContext(ctx, query, paymentTermsID).Scan(
		&terms.ID,
		&terms.OrderID,
		&terms.TotalAmount,
		&terms.DownPaymentAmount,
		&terms.InstallmentCount,
		&terms.InstallmentAmount,
		&terms.PaymentSchedule,
		&terms.TotalPaid,
		&terms.RemainingBalance,
		&terms.CreatedAt,
		&terms.CreatedByUserID,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	return &terms, nil
}
