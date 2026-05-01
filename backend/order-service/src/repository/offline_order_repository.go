package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/utils"
)

// OfflineOrderRepository handles offline order persistence with PII encryption
// Implements UU PDP compliance for offline customer data protection
// Uses the guest_orders table but with order_type='offline'
type OfflineOrderRepository struct {
	db             *sql.DB
	encryptor      utils.Encryptor
	auditPublisher *utils.AuditPublisher
}

// NewOfflineOrderRepository creates a new repository with dependency injection (for testing)
func NewOfflineOrderRepository(db *sql.DB, encryptor utils.Encryptor, auditPublisher *utils.AuditPublisher) *OfflineOrderRepository {
	return &OfflineOrderRepository{
		db:             db,
		encryptor:      encryptor,
		auditPublisher: auditPublisher,
	}
}

// NewOfflineOrderRepositoryWithVault creates a repository with real VaultClient (for production)
func NewOfflineOrderRepositoryWithVault(db *sql.DB, auditPublisher *utils.AuditPublisher) (*OfflineOrderRepository, error) {
	vaultEncryptor, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize VaultEncryptor: %w", err)
	}
	return NewOfflineOrderRepository(db, vaultEncryptor, auditPublisher), nil
}

// getExecutor returns the appropriate SQL executor (transaction or database)
func (r *OfflineOrderRepository) getExecutor(tx *sql.Tx) interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
} {
	if tx != nil {
		return tx
	}
	return r.db
}

// encryptStringPtrWithContext encrypts a pointer to string with encryption context (handles nil values)
func (r *OfflineOrderRepository) encryptStringPtrWithContext(ctx context.Context, value *string, encryptionContext string) (string, error) {
	if value == nil || *value == "" {
		return "", nil
	}
	return r.encryptor.EncryptWithContext(ctx, *value, encryptionContext)
}

// decryptToStringPtrWithContext decrypts to a pointer to string with encryption context (handles empty values)
func (r *OfflineOrderRepository) decryptToStringPtrWithContext(ctx context.Context, encrypted string, encryptionContext string) (*string, error) {
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := r.encryptor.DecryptWithContext(ctx, encrypted, encryptionContext)
	if err != nil {
		return nil, err
	}
	return &decrypted, nil
}

// CreateOfflineOrder inserts a new offline order with encrypted PII fields
// Encrypts: CustomerName, CustomerPhone, CustomerEmail
// Sets order_type='offline' and requires recorded_by_user_id
func (r *OfflineOrderRepository) CreateOfflineOrder(ctx context.Context, tx *sql.Tx, order *models.GuestOrder) (string, error) {
	// Validate offline order requirements
	if order.RecordedByUserID == nil || *order.RecordedByUserID == "" {
		return "", fmt.Errorf("recorded_by_user_id is required for offline orders")
	}
	if order.OrderType != models.OrderTypeOffline {
		return "", fmt.Errorf("order_type must be 'offline'")
	}

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

	query := `
		INSERT INTO guest_orders (
			tenant_id, order_reference, status, order_type,
			delivery_type, customer_name, customer_phone, customer_email,
			table_number, notes,
			subtotal_amount, delivery_fee, total_amount,
			data_consent_given, consent_method, recorded_by_user_id,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	var orderID string
	executor := r.getExecutor(tx)
	err = executor.QueryRowContext(
		ctx,
		query,
		order.TenantID,
		order.OrderReference,
		order.Status,
		order.OrderType,
		order.DeliveryType,
		encryptedName,
		encryptedPhone,
		encryptedEmail,
		order.TableNumber,
		order.Notes,
		order.SubtotalAmount,
		order.DeliveryFee,
		order.TotalAmount,
		order.DataConsentGiven,
		order.ConsentMethod,
		order.RecordedByUserID,
		time.Now(),
	).Scan(&orderID)

	if err != nil {
		return "", fmt.Errorf("failed to create offline order: %w", err)
	}

	return orderID, nil
}

// GetOfflineOrderByID retrieves an offline order with decrypted PII fields
// Returns error if order is not offline type or doesn't exist
func (r *OfflineOrderRepository) GetOfflineOrderByID(ctx context.Context, orderID string, tenantID string) (*models.GuestOrder, error) {
	query := `
		SELECT 
			id, tenant_id, order_reference, status, order_type,
			delivery_type, customer_name, customer_phone, customer_email,
			table_number, notes,
			subtotal_amount, delivery_fee, total_amount,
			data_consent_given, consent_method, recorded_by_user_id,
			last_modified_by_user_id, last_modified_at,
			created_at, paid_at, completed_at, cancelled_at
		FROM guest_orders
		WHERE id = $1 AND tenant_id = $2 AND order_type = 'offline'
	`

	var order models.GuestOrder
	var encryptedName, encryptedPhone, encryptedEmail string
	var tableNumber, notes sql.NullString
	var consentMethod sql.NullString
	var lastModifiedByUserID sql.NullString
	var lastModifiedAt sql.NullTime
	var paidAt, completedAt, cancelledAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, orderID, tenantID).Scan(
		&order.ID,
		&order.TenantID,
		&order.OrderReference,
		&order.Status,
		&order.OrderType,
		&order.DeliveryType,
		&encryptedName,
		&encryptedPhone,
		&encryptedEmail,
		&tableNumber,
		&notes,
		&order.SubtotalAmount,
		&order.DeliveryFee,
		&order.TotalAmount,
		&order.DataConsentGiven,
		&consentMethod,
		&order.RecordedByUserID,
		&lastModifiedByUserID,
		&lastModifiedAt,
		&order.CreatedAt,
		&paidAt,
		&completedAt,
		&cancelledAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("offline order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query offline order: %w", err)
	}

	// Decrypt PII fields
	decryptedName, err := r.encryptor.DecryptWithContext(ctx, encryptedName, "guest_order:customer_name")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer_name: %w", err)
	}
	order.CustomerName = decryptedName

	decryptedPhone, err := r.encryptor.DecryptWithContext(ctx, encryptedPhone, "guest_order:customer_phone")
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt customer_phone: %w", err)
	}
	order.CustomerPhone = decryptedPhone

	if encryptedEmail != "" {
		decryptedEmail, err := r.encryptor.DecryptWithContext(ctx, encryptedEmail, "guest_order:customer_email")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt customer_email: %w", err)
		}
		order.CustomerEmail = &decryptedEmail
	}

	// Handle nullable fields
	if tableNumber.Valid {
		order.TableNumber = &tableNumber.String
	}
	if notes.Valid {
		order.Notes = &notes.String
	}
	if consentMethod.Valid {
		cm := models.ConsentMethod(consentMethod.String)
		order.ConsentMethod = &cm
	}
	if lastModifiedByUserID.Valid {
		order.LastModifiedByUserID = &lastModifiedByUserID.String
	}
	if lastModifiedAt.Valid {
		order.LastModifiedAt = &lastModifiedAt.Time
	}
	if paidAt.Valid {
		order.PaidAt = &paidAt.Time
	}
	if completedAt.Valid {
		order.CompletedAt = &completedAt.Time
	}
	if cancelledAt.Valid {
		order.CancelledAt = &cancelledAt.Time
	}

	return &order, nil
}

// ListOfflineOrders retrieves offline orders with pagination and filtering
// Supports filtering by status and search by order_reference
func (r *OfflineOrderRepository) ListOfflineOrders(ctx context.Context, tenantID string, filters ListOfflineOrdersFilters) ([]models.GuestOrder, int, error) {
	// Build WHERE clause dynamically based on filters
	whereClause := "WHERE tenant_id = $1 AND order_type = 'offline'"
	args := []interface{}{tenantID}
	argCount := 1

	if filters.Status != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filters.Status)
	}

	if filters.SearchQuery != "" {
		argCount++
		whereClause += fmt.Sprintf(" AND order_reference ILIKE $%d", argCount)
		args = append(args, "%"+filters.SearchQuery+"%")
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM guest_orders %s", whereClause)
	var totalCount int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count offline orders: %w", err)
	}

	// Query with pagination
	query := fmt.Sprintf(`
		SELECT 
			id, tenant_id, order_reference, status, order_type,
			delivery_type, customer_name, customer_phone, customer_email,
			table_number, notes,
			subtotal_amount, delivery_fee, total_amount,
			data_consent_given, consent_method, recorded_by_user_id,
			created_at, paid_at, completed_at
		FROM guest_orders
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argCount+1, argCount+2)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query offline orders: %w", err)
	}
	defer rows.Close()

	var orders []models.GuestOrder
	for rows.Next() {
		var order models.GuestOrder
		var encryptedName, encryptedPhone, encryptedEmail string
		var tableNumber, notes sql.NullString
		var consentMethod sql.NullString
		var paidAt, completedAt sql.NullTime

		err := rows.Scan(
			&order.ID,
			&order.TenantID,
			&order.OrderReference,
			&order.Status,
			&order.OrderType,
			&order.DeliveryType,
			&encryptedName,
			&encryptedPhone,
			&encryptedEmail,
			&tableNumber,
			&notes,
			&order.SubtotalAmount,
			&order.DeliveryFee,
			&order.TotalAmount,
			&order.DataConsentGiven,
			&consentMethod,
			&order.RecordedByUserID,
			&order.CreatedAt,
			&paidAt,
			&completedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan offline order: %w", err)
		}

		// Decrypt PII fields
		decryptedName, err := r.encryptor.DecryptWithContext(ctx, encryptedName, "guest_order:customer_name")
		if err != nil {
			return nil, 0, fmt.Errorf("failed to decrypt customer_name: %w", err)
		}
		order.CustomerName = decryptedName

		decryptedPhone, err := r.encryptor.DecryptWithContext(ctx, encryptedPhone, "guest_order:customer_phone")
		if err != nil {
			return nil, 0, fmt.Errorf("failed to decrypt customer_phone: %w", err)
		}
		order.CustomerPhone = decryptedPhone

		if encryptedEmail != "" {
			decryptedEmail, err := r.encryptor.DecryptWithContext(ctx, encryptedEmail, "guest_order:customer_email")
			if err != nil {
				return nil, 0, fmt.Errorf("failed to decrypt customer_email: %w", err)
			}
			order.CustomerEmail = &decryptedEmail
		}

		// Handle nullable fields
		if tableNumber.Valid {
			order.TableNumber = &tableNumber.String
		}
		if notes.Valid {
			order.Notes = &notes.String
		}
		if consentMethod.Valid {
			cm := models.ConsentMethod(consentMethod.String)
			order.ConsentMethod = &cm
		}
		if paidAt.Valid {
			order.PaidAt = &paidAt.Time
		}
		if completedAt.Valid {
			order.CompletedAt = &completedAt.Time
		}

		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating offline orders: %w", err)
	}

	return orders, totalCount, nil
}

// UpdateOfflineOrder updates an existing offline order with encrypted PII fields
// T074: Implement UpdateOfflineOrder method with field tracking
// Only updates fields that are provided (non-nil/non-empty)
// Encrypts: CustomerName, CustomerPhone, CustomerEmail
func (r *OfflineOrderRepository) UpdateOfflineOrder(ctx context.Context, tx *sql.Tx, orderID string, tenantID string, updates *models.UpdateOfflineOrderRequest, modifiedByUserID string) error {
	// Build dynamic UPDATE query based on provided fields
	query := "UPDATE guest_orders SET last_modified_by_user_id = $1, last_modified_at = $2"
	args := []interface{}{modifiedByUserID, time.Now()}
	argCount := 2

	// Customer info updates (with encryption)
	if updates.CustomerName != nil {
		encryptedName, err := r.encryptor.EncryptWithContext(ctx, *updates.CustomerName, "guest_order:customer_name")
		if err != nil {
			return fmt.Errorf("failed to encrypt customer_name: %w", err)
		}
		argCount++
		query += fmt.Sprintf(", customer_name = $%d", argCount)
		args = append(args, encryptedName)
	}

	if updates.CustomerPhone != nil {
		encryptedPhone, err := r.encryptor.EncryptWithContext(ctx, *updates.CustomerPhone, "guest_order:customer_phone")
		if err != nil {
			return fmt.Errorf("failed to encrypt customer_phone: %w", err)
		}
		argCount++
		query += fmt.Sprintf(", customer_phone = $%d", argCount)
		args = append(args, encryptedPhone)
	}

	if updates.CustomerEmail != nil {
		encryptedEmail, err := r.encryptStringPtrWithContext(ctx, updates.CustomerEmail, "guest_order:customer_email")
		if err != nil {
			return fmt.Errorf("failed to encrypt customer_email: %w", err)
		}
		argCount++
		query += fmt.Sprintf(", customer_email = $%d", argCount)
		args = append(args, encryptedEmail)
	}

	// Non-encrypted field updates
	if updates.DeliveryType != nil {
		argCount++
		query += fmt.Sprintf(", delivery_type = $%d", argCount)
		args = append(args, *updates.DeliveryType)
	}

	if updates.TableNumber != nil {
		argCount++
		query += fmt.Sprintf(", table_number = $%d", argCount)
		args = append(args, *updates.TableNumber)
	}

	if updates.Notes != nil {
		argCount++
		query += fmt.Sprintf(", notes = $%d", argCount)
		args = append(args, *updates.Notes)
	}

	if updates.DeliveryFee != nil {
		argCount++
		query += fmt.Sprintf(", delivery_fee = $%d", argCount)
		args = append(args, *updates.DeliveryFee)
	}

	// WHERE clause
	argCount++
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, orderID)

	argCount++
	query += fmt.Sprintf(" AND tenant_id = $%d", argCount)
	args = append(args, tenantID)

	argCount++
	query += fmt.Sprintf(" AND order_type = $%d", argCount)
	args = append(args, models.OrderTypeOffline)

	// Execute update
	executor := r.getExecutor(tx)
	result, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update offline order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("offline order not found or no changes made")
	}

	return nil
}

// UpdateOrderItems replaces order items for an offline order and recalculates totals
// T075: Implement UpdateOrderItems method
// Deletes existing items and inserts new ones within a transaction
func (r *OfflineOrderRepository) UpdateOrderItems(ctx context.Context, tx *sql.Tx, orderID string, tenantID string, items []models.OrderItemInput) (int, int, error) {
	// Delete existing order items
	deleteQuery := `
		DELETE FROM order_items 
		WHERE order_id = $1 
		AND order_id IN (SELECT id FROM guest_orders WHERE tenant_id = $2)
	`
	executor := r.getExecutor(tx)
	_, err := executor.ExecContext(ctx, deleteQuery, orderID, tenantID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to delete existing order items: %w", err)
	}

	// Insert new order items and calculate totals
	insertQuery := `
		INSERT INTO order_items (
			order_id, product_id, product_name, quantity, unit_price, total_price
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	subtotalAmount := 0
	for _, item := range items {
		totalPrice := item.Quantity * item.UnitPrice
		_, err := executor.ExecContext(
			ctx,
			insertQuery,
			orderID,
			item.ProductID,
			item.ProductName,
			item.Quantity,
			item.UnitPrice,
			totalPrice,
		)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to insert order item: %w", err)
		}
		subtotalAmount += totalPrice
	}

	// Get current delivery fee to calculate total
	var deliveryFee int
	feeQuery := "SELECT delivery_fee FROM guest_orders WHERE id = $1 AND tenant_id = $2"
	err = executor.QueryRowContext(ctx, feeQuery, orderID, tenantID).Scan(&deliveryFee)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get delivery fee: %w", err)
	}

	totalAmount := subtotalAmount + deliveryFee

	// Update order totals
	updateQuery := `
		UPDATE guest_orders 
		SET subtotal_amount = $1, total_amount = $2, last_modified_at = $3
		WHERE id = $4 AND tenant_id = $5
	`
	_, err = executor.ExecContext(ctx, updateQuery, subtotalAmount, totalAmount, time.Now(), orderID, tenantID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to update order totals: %w", err)
	}

	return subtotalAmount, totalAmount, nil
}

// ListOfflineOrdersFilters holds filter parameters for listing offline orders
type ListOfflineOrdersFilters struct {
	Status      string // Filter by order status (optional)
	SearchQuery string // Search by order_reference (optional)
	Limit       int    // Page size
	Offset      int    // Page offset
}

// SoftDeleteOfflineOrder marks an offline order as deleted without removing it from database
// T091: Soft delete implementation for US4 - Role-Based Deletion
// Sets deleted_at timestamp and deleted_by_user_id for audit trail
// Only allows deletion of PENDING or CANCELLED orders (not PAID/COMPLETE)
func (r *OfflineOrderRepository) SoftDeleteOfflineOrder(ctx context.Context, orderID, tenantID, deletedByUserID string) error {
	query := `
		UPDATE guest_orders 
		SET 
			deleted_at = $1,
			deleted_by_user_id = $2,
			last_modified_at = $3,
			status = 'CANCELLED'
		WHERE id = $4 
			AND tenant_id = $5 
			AND order_type = 'offline'
			AND deleted_at IS NULL
			AND status IN ('PENDING', 'CANCELLED')
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		time.Now(),
		deletedByUserID,
		time.Now(),
		orderID,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete offline order: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Check if order exists but is in non-deletable state
		var status string
		var deletedAt *time.Time
		checkQuery := `
			SELECT status, deleted_at 
			FROM guest_orders 
			WHERE id = $1 AND tenant_id = $2 AND order_type = 'offline'
		`
		err := r.db.QueryRowContext(ctx, checkQuery, orderID, tenantID).Scan(&status, &deletedAt)
		if err == sql.ErrNoRows {
			return fmt.Errorf("offline order not found")
		}
		if err != nil {
			return fmt.Errorf("failed to check order status: %w", err)
		}

		if deletedAt != nil {
			return fmt.Errorf("order already deleted")
		}
		if status == "PAID" || status == "COMPLETE" {
			return fmt.Errorf("cannot delete orders with status %s - only PENDING or CANCELLED orders can be deleted", status)
		}

		return fmt.Errorf("order cannot be deleted")
	}

	return nil
}
