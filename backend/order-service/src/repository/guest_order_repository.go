package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/utils"
)

// GuestOrderRepository handles guest order persistence with PII encryption
// Implements UU PDP compliance for customer data protection
type GuestOrderRepository struct {
	db             *sql.DB
	encryptor      utils.Encryptor
	auditPublisher *utils.AuditPublisher
}

// NewGuestOrderRepository creates a new repository with dependency injection (for testing)
func NewGuestOrderRepository(db *sql.DB, encryptor utils.Encryptor, auditPublisher *utils.AuditPublisher) *GuestOrderRepository {
	return &GuestOrderRepository{
		db:             db,
		encryptor:      encryptor,
		auditPublisher: auditPublisher,
	}
}

// NewGuestOrderRepositoryWithVault creates a repository with real VaultClient (for production)
func NewGuestOrderRepositoryWithVault(db *sql.DB, auditPublisher *utils.AuditPublisher) (*GuestOrderRepository, error) {
	vaultEncryptor, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultEncryptor: %w", err)
	}
	return NewGuestOrderRepository(db, vaultEncryptor, auditPublisher), nil
}

// encryptStringPtrWithContext encrypts a pointer to string with encryption context (handles nil values)
func (r *GuestOrderRepository) encryptStringPtrWithContext(ctx context.Context, value *string, encryptionContext string) (string, error) {
	if value == nil || *value == "" {
		return "", nil
	}
	return r.encryptor.EncryptWithContext(ctx, *value, encryptionContext)
}

// decryptToStringPtrWithContext decrypts to a pointer to string with encryption context (handles empty values)
func (r *GuestOrderRepository) decryptToStringPtrWithContext(ctx context.Context, encrypted string, encryptionContext string) (*string, error) {
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := r.encryptor.DecryptWithContext(ctx, encrypted, encryptionContext)
	if err != nil {
		return nil, err
	}
	return &decrypted, nil
}

// Create inserts a new guest order with encrypted PII fields
// Encrypts: CustomerName, CustomerPhone, CustomerEmail, IPAddress, UserAgent
func (r *GuestOrderRepository) Create(ctx context.Context, tx *sql.Tx, order *models.GuestOrder) (string, error) {
	// Encrypt PII fields with context
	encryptedName, err := r.encryptor.EncryptWithContext(ctx, order.CustomerName, "guest_order:customer_name")
	if err != nil {
		return "", fmt.Errorf("failed to encrypt customer_name: %w", err)
	}

	encryptedPhone, err := r.encryptor.EncryptWithContext(ctx, order.CustomerPhone, "guest_order:customer_phone")
	if err != nil {
		return "", fmt.Errorf("failed to encrypt customer_phone: %w", err)
	}

	encryptedEmail, err := r.encryptStringPtrWithContext(ctx, order.CustomerEmail, "guest_order:customer_email")
	if err != nil {
		return "", fmt.Errorf("failed to encrypt customer_email: %w", err)
	}

	encryptedIPAddress, err := r.encryptStringPtrWithContext(ctx, order.IPAddress, "guest_order:ip_address")
	if err != nil {
		return "", fmt.Errorf("failed to encrypt ip_address: %w", err)
	}

	encryptedUserAgent, err := r.encryptStringPtrWithContext(ctx, order.UserAgent, "guest_order:user_agent")
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

	if err != nil {
		return "", err
	}

	// T101: Publish GuestOrderCreatedEvent to audit trail
	if r.auditPublisher != nil {
		afterValue := map[string]interface{}{
			"order_reference": order.OrderReference,
			"customer_name":   encryptedName,
			"customer_phone":  encryptedPhone,
			"customer_email":  encryptedEmail,
			"delivery_type":   order.DeliveryType,
			"total_amount":    order.TotalAmount,
		}

		auditEvent := &utils.AuditEvent{
			TenantID:     order.TenantID,
			ActorType:    "guest",
			Action:       "CREATE",
			ResourceType: "guest_order",
			ResourceID:   orderID,
			AfterValue:   afterValue,
			IPAddress:    order.IPAddress,
			UserAgent:    order.UserAgent,
			Metadata: map[string]interface{}{
				"status": order.Status,
			},
		}

		// Use background context with timeout for audit event
		auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := r.auditPublisher.Publish(auditCtx, auditEvent); err != nil {
			fmt.Printf("Failed to publish guest order create audit event: %v\n", err)
		}
	}

	return orderID, nil
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

	// Decrypt PII fields with context
	order.CustomerName, err = r.encryptor.DecryptWithContext(ctx, encryptedName, "guest_order:customer_name")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer_name: %w", err)
	}

	order.CustomerPhone, err = r.encryptor.DecryptWithContext(ctx, encryptedPhone, "guest_order:customer_phone")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer_phone: %w", err)
	}

	order.CustomerEmail, err = r.decryptToStringPtrWithContext(ctx, encryptedEmail, "guest_order:customer_email")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer_email: %w", err)
	}

	if encryptedIPAddress.Valid {
		order.IPAddress, err = r.decryptToStringPtrWithContext(ctx, encryptedIPAddress.String, "guest_order:ip_address")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt ip_address: %w", err)
		}
	}

	if encryptedUserAgent.Valid {
		order.UserAgent, err = r.decryptToStringPtrWithContext(ctx, encryptedUserAgent.String, "guest_order:user_agent")
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
