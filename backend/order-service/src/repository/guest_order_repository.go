package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/utils"
)

// GuestOrderRepository handles guest order persistence with PII encryption
// Implements UU PDP compliance for customer data protection
type GuestOrderRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewGuestOrderRepository creates a new repository with dependency injection (for testing)
func NewGuestOrderRepository(db *sql.DB, encryptor utils.Encryptor) *GuestOrderRepository {
	return &GuestOrderRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// NewGuestOrderRepositoryWithVault creates a repository with real VaultClient (for production)
func NewGuestOrderRepositoryWithVault(db *sql.DB) (*GuestOrderRepository, error) {
	vaultEncryptor, err := utils.NewVaultEncryptor()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultEncryptor: %w", err)
	}
	return NewGuestOrderRepository(db, vaultEncryptor), nil
}

// encryptStringPtr encrypts a pointer to string (handles nil values)
func (r *GuestOrderRepository) encryptStringPtr(ctx context.Context, value *string) (string, error) {
	if value == nil || *value == "" {
		return "", nil
	}
	return r.encryptor.Encrypt(ctx, *value)
}

// decryptToStringPtr decrypts to a pointer to string (handles empty values)
func (r *GuestOrderRepository) decryptToStringPtr(ctx context.Context, encrypted string) (*string, error) {
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := r.encryptor.Decrypt(ctx, encrypted)
	if err != nil {
		return nil, err
	}
	return &decrypted, nil
}

// Create inserts a new guest order with encrypted PII fields
// Encrypts: CustomerName, CustomerPhone, CustomerEmail, IPAddress, UserAgent
func (r *GuestOrderRepository) Create(ctx context.Context, tx *sql.Tx, order *models.GuestOrder) (string, error) {
	// Encrypt PII fields
	encryptedName, err := r.encryptor.Encrypt(ctx, order.CustomerName)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt customer_name: %w", err)
	}

	encryptedPhone, err := r.encryptor.Encrypt(ctx, order.CustomerPhone)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt customer_phone: %w", err)
	}

	encryptedEmail, err := r.encryptStringPtr(ctx, order.CustomerEmail)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt customer_email: %w", err)
	}

	encryptedIPAddress, err := r.encryptStringPtr(ctx, order.IPAddress)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt ip_address: %w", err)
	}

	encryptedUserAgent, err := r.encryptStringPtr(ctx, order.UserAgent)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt user_agent: %w", err)
	}

	query := `
		INSERT INTO guest_orders (
			tenant_id, session_id, order_reference, status,
			delivery_type, customer_name, customer_phone, customer_email,
			table_number, notes,
			subtotal_amount, delivery_fee, total_amount,
			ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	var orderID string
	executor := r.getExecutor(tx)
	err = executor.QueryRowContext(
		ctx,
		query,
		order.TenantID,
		order.SessionID,
		order.OrderReference,
		order.Status,
		order.DeliveryType,
		encryptedName,
		encryptedPhone,
		encryptedEmail,
		order.TableNumber,
		order.Notes,
		order.SubtotalAmount,
		order.DeliveryFee,
		order.TotalAmount,
		encryptedIPAddress,
		encryptedUserAgent,
	).Scan(&orderID)

	return orderID, err
}

// GetByReference retrieves a guest order by order_reference with decrypted PII
func (r *GuestOrderRepository) GetByReference(ctx context.Context, tenantID, orderReference string) (*models.GuestOrder, error) {
	query := `
		SELECT 
			id, order_reference, tenant_id, session_id, status,
			subtotal_amount, delivery_fee, total_amount,
			customer_name, customer_phone, customer_email,
			delivery_type, table_number, notes,
			created_at, paid_at, completed_at, cancelled_at,
			ip_address, user_agent,
			is_anonymized, anonymized_at
		FROM guest_orders
		WHERE tenant_id = $1 AND order_reference = $2
	`

	var order models.GuestOrder
	var encryptedName, encryptedPhone, encryptedEmail string
	var encryptedIPAddress, encryptedUserAgent sql.NullString

	err := r.db.QueryRowContext(ctx, query, tenantID, orderReference).Scan(
		&order.ID,
		&order.OrderReference,
		&order.TenantID,
		&order.SessionID,
		&order.Status,
		&order.SubtotalAmount,
		&order.DeliveryFee,
		&order.TotalAmount,
		&encryptedName,
		&encryptedPhone,
		&encryptedEmail,
		&order.DeliveryType,
		&order.TableNumber,
		&order.Notes,
		&order.CreatedAt,
		&order.PaidAt,
		&order.CompletedAt,
		&order.CancelledAt,
		&encryptedIPAddress,
		&encryptedUserAgent,
		&order.IsAnonymized,
		&order.AnonymizedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found: %s", orderReference)
		}
		return nil, err
	}

	// Decrypt PII fields
	order.CustomerName, err = r.encryptor.Decrypt(ctx, encryptedName)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer_name: %w", err)
	}

	order.CustomerPhone, err = r.encryptor.Decrypt(ctx, encryptedPhone)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer_phone: %w", err)
	}

	order.CustomerEmail, err = r.decryptToStringPtr(ctx, encryptedEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer_email: %w", err)
	}

	if encryptedIPAddress.Valid {
		order.IPAddress, err = r.decryptToStringPtr(ctx, encryptedIPAddress.String)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt ip_address: %w", err)
		}
	}

	if encryptedUserAgent.Valid {
		order.UserAgent, err = r.decryptToStringPtr(ctx, encryptedUserAgent.String)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt user_agent: %w", err)
		}
	}

	return &order, nil
}

// UpdateStatus updates the order status and related timestamps
func (r *GuestOrderRepository) UpdateStatus(ctx context.Context, orderID string, status models.OrderStatus) error {
	query := `
		UPDATE guest_orders
		SET status = $1,
			paid_at = CASE WHEN $1 = 'paid' THEN CURRENT_TIMESTAMP ELSE paid_at END,
			completed_at = CASE WHEN $1 = 'completed' THEN CURRENT_TIMESTAMP ELSE completed_at END,
			cancelled_at = CASE WHEN $1 = 'cancelled' THEN CURRENT_TIMESTAMP ELSE cancelled_at END
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, orderID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("order not found: %s", orderID)
	}

	return nil
}

// MarkAnonymized marks an order as anonymized (for UU PDP compliance - right to erasure)
func (r *GuestOrderRepository) MarkAnonymized(ctx context.Context, orderID string) error {
	query := `
		UPDATE guest_orders
		SET is_anonymized = true,
			anonymized_at = CURRENT_TIMESTAMP,
			customer_name = 'ANONYMIZED',
			customer_phone = 'ANONYMIZED',
			customer_email = NULL
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, orderID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("order not found: %s", orderID)
	}

	return nil
}

// getExecutor returns the appropriate SQL executor (transaction or database)
func (r *GuestOrderRepository) getExecutor(tx *sql.Tx) interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if tx != nil {
		return tx
	}
	return r.db
}
