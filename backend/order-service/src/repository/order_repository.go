package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/utils"
	"github.com/rs/zerolog/log"
)

// OrderRepository handles database operations for orders
type OrderRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewOrderRepository creates a new order repository with custom encryptor
func NewOrderRepository(db *sql.DB, encryptor utils.Encryptor) *OrderRepository {
	return &OrderRepository{
		db:        db,
		encryptor: encryptor,
	}
}

// NewOrderRepositoryWithVault creates a repository with Vault encryption (production)
func NewOrderRepositoryWithVault(db *sql.DB) (*OrderRepository, error) {
	vaultClient, err := utils.NewVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}
	return NewOrderRepository(db, vaultClient), nil
}

// Helper function to decrypt pointer string fields
func (r *OrderRepository) decryptToStringPtr(ctx context.Context, encrypted string, encContext string) (*string, error) {
	if encrypted == "" {
		return nil, nil
	}
	decrypted, err := r.encryptor.DecryptWithContext(ctx, encrypted, encContext)
	if err != nil {
		return nil, err
	}
	return &decrypted, nil
}

// GetOrderByReference retrieves an order by its reference number
func (r *OrderRepository) GetOrderByReference(ctx context.Context, orderReference string) (*models.GuestOrder, error) {
	query := `
		SELECT od.id, od.order_reference, od.tenant_id, od.status, od.subtotal_amount, od.delivery_fee, od.total_amount,
					od.customer_name, od.customer_phone, od.customer_email, od.delivery_type, od.table_number, od.notes,
					od.created_at, od.paid_at, od.completed_at, od.cancelled_at, od.session_id, od.ip_address, od.user_agent, od.is_anonymized,
					od.anonymized_at, t.slug as tenant_slug
		FROM guest_orders od
		LEFT JOIN tenants t ON od.tenant_id = t.id
		WHERE order_reference = $1
	`

	var order models.GuestOrder
	var encryptedName, encryptedPhone sql.NullString
	var encryptedEmail, encryptedIP, encryptedUA sql.NullString

	err := r.db.QueryRowContext(ctx, query, orderReference).Scan(
		&order.ID,
		&order.OrderReference,
		&order.TenantID,
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
		&order.SessionID,
		&encryptedIP,
		&encryptedUA,
		&order.IsAnonymized,
		&order.AnonymizedAt,
		&order.TenantSlug,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows // Return error instead of (nil, nil)
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("order_reference", orderReference).
			Msg("Failed to get order by reference")
		return nil, err
	}

	// Decrypt PII fields
	if encryptedName.Valid {
		if order.CustomerName, err = r.encryptor.DecryptWithContext(ctx, encryptedName.String, "guest_order:customer_name"); err != nil {
			return nil, fmt.Errorf("failed to decrypt customer_name: %w", err)
		}
	}
	if encryptedPhone.Valid {
		if order.CustomerPhone, err = r.encryptor.DecryptWithContext(ctx, encryptedPhone.String, "guest_order:customer_phone"); err != nil {
			return nil, fmt.Errorf("failed to decrypt customer_phone: %w", err)
		}
	}
	if encryptedEmail.Valid {
		if order.CustomerEmail, err = r.decryptToStringPtr(ctx, encryptedEmail.String, "guest_order:customer_email"); err != nil {
			return nil, fmt.Errorf("failed to decrypt customer_email: %w", err)
		}
	}
	if encryptedIP.Valid && encryptedIP.String != "" {
		if order.IPAddress, err = r.decryptToStringPtr(ctx, encryptedIP.String, "guest_order:ip_address"); err != nil {
			return nil, fmt.Errorf("failed to decrypt ip_address: %w", err)
		}
	}
	if encryptedUA.Valid && encryptedUA.String != "" {
		if order.UserAgent, err = r.decryptToStringPtr(ctx, encryptedUA.String, "guest_order:user_agent"); err != nil {
			return nil, fmt.Errorf("failed to decrypt user_agent: %w", err)
		}
	}

	return &order, nil
}

// GetOrderByID retrieves an order by its ID
func (r *OrderRepository) GetOrderByID(ctx context.Context, orderID string) (*models.GuestOrder, error) {
	query := `
SELECT id, order_reference, tenant_id, status, subtotal_amount, delivery_fee, total_amount,
       customer_name, customer_phone, customer_email, delivery_type, table_number, notes,
       created_at, paid_at, completed_at, cancelled_at, session_id, ip_address, user_agent
FROM guest_orders
WHERE id = $1
`

	var order models.GuestOrder
	var encryptedName, encryptedPhone sql.NullString
	var encryptedEmail, encryptedIP, encryptedUA sql.NullString

	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&order.ID,
		&order.OrderReference,
		&order.TenantID,
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
		&order.SessionID,
		&encryptedIP,
		&encryptedUA,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows // Return error instead of (nil, nil)
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to get order by ID")
		return nil, err
	}

	// Decrypt PII fields
	if encryptedName.Valid {
		if order.CustomerName, err = r.encryptor.DecryptWithContext(ctx, encryptedName.String, "guest_order:customer_name"); err != nil {
			return nil, fmt.Errorf("failed to decrypt customer_name: %w", err)
		}
	}
	if encryptedPhone.Valid {
		if order.CustomerPhone, err = r.encryptor.DecryptWithContext(ctx, encryptedPhone.String, "guest_order:customer_phone"); err != nil {
			return nil, fmt.Errorf("failed to decrypt customer_phone: %w", err)
		}
	}
	if encryptedEmail.Valid {
		if order.CustomerEmail, err = r.decryptToStringPtr(ctx, encryptedEmail.String, "guest_order:customer_email"); err != nil {
			return nil, fmt.Errorf("failed to decrypt customer_email: %w", err)
		}
	}
	if encryptedIP.Valid && encryptedIP.String != "" {
		if order.IPAddress, err = r.decryptToStringPtr(ctx, encryptedIP.String, "guest_order:ip_address"); err != nil {
			return nil, fmt.Errorf("failed to decrypt ip_address: %w", err)
		}
	}
	if encryptedUA.Valid && encryptedUA.String != "" {
		if order.UserAgent, err = r.decryptToStringPtr(ctx, encryptedUA.String, "guest_order:user_agent"); err != nil {
			return nil, fmt.Errorf("failed to decrypt user_agent: %w", err)
		}
	}

	return &order, nil
}

// UpdateOrderStatus updates the order status and corresponding timestamps
func (r *OrderRepository) UpdateOrderStatus(
	ctx context.Context,
	tx *sql.Tx,
	orderID string,
	status models.OrderStatus,
	paidAt, completedAt, cancelledAt *time.Time,
) error {
	query := `
UPDATE guest_orders
SET status = $1,
    paid_at = COALESCE($2, paid_at),
    completed_at = COALESCE($3, completed_at),
    cancelled_at = COALESCE($4, cancelled_at)
WHERE id = $5
`

	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, status, paidAt, completedAt, cancelledAt, orderID)
	} else {
		_, err = r.db.ExecContext(ctx, query, status, paidAt, completedAt, cancelledAt, orderID)
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("status", string(status)).
			Msg("Failed to update order status")
		return err
	}

	log.Info().
		Str("order_id", orderID).
		Str("status", string(status)).
		Msg("Order status updated successfully")

	return nil
}

// UpdateOrderNotes updates the notes field of an order
func (r *OrderRepository) UpdateOrderNotes(ctx context.Context, orderID, notes string) error {
	query := `
UPDATE guest_orders
SET notes = $1
WHERE id = $2
`

	_, err := r.db.ExecContext(ctx, query, notes, orderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to update order notes")
		return err
	}

	log.Info().
		Str("order_id", orderID).
		Msg("Order notes updated successfully")

	return nil
}

// ListOrdersByTenant retrieves orders for a tenant with optional status filter
func (r *OrderRepository) ListOrdersByTenant(
	ctx context.Context,
	tenantID string,
	status *models.OrderStatus,
	limit, offset int,
) ([]*models.GuestOrder, error) {
	query := `
SELECT id, order_reference, tenant_id, status, subtotal_amount, delivery_fee, total_amount,
       customer_name, customer_phone, customer_email, delivery_type, table_number, notes,
       created_at, paid_at, completed_at, cancelled_at, session_id, ip_address, user_agent
FROM guest_orders
WHERE tenant_id = $1
`

	args := []interface{}{tenantID}
	argCount := 1

	if status != nil {
		argCount++
		query += ` AND status = $` + string(rune(argCount+'0'))
		args = append(args, *status)
	}

	query += ` ORDER BY created_at DESC LIMIT $` + string(rune(argCount+1+'0')) + ` OFFSET $` + string(rune(argCount+2+'0'))
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID).
			Msg("Failed to list orders")
		return nil, err
	}
	defer rows.Close()

	orders := []*models.GuestOrder{}
	for rows.Next() {
		var order models.GuestOrder
		var encryptedName, encryptedPhone sql.NullString
		var encryptedEmail, encryptedIP, encryptedUA sql.NullString

		err := rows.Scan(
			&order.ID,
			&order.OrderReference,
			&order.TenantID,
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
			&order.SessionID,
			&encryptedIP,
			&encryptedUA,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan order row")
			return nil, err
		}

		// Decrypt PII fields
		if encryptedName.Valid {
			if order.CustomerName, err = r.encryptor.DecryptWithContext(ctx, encryptedName.String, "guest_order:customer_name"); err != nil {
				log.Error().Err(err).Msg("Failed to decrypt customer_name")
				return nil, fmt.Errorf("failed to decrypt customer_name: %w", err)
			}
		}
		if encryptedPhone.Valid {
			if order.CustomerPhone, err = r.encryptor.DecryptWithContext(ctx, encryptedPhone.String, "guest_order:customer_phone"); err != nil {
				log.Error().Err(err).Msg("Failed to decrypt customer_phone")
				return nil, fmt.Errorf("failed to decrypt customer_phone: %w", err)
			}
		}
		if encryptedEmail.Valid {
			if order.CustomerEmail, err = r.decryptToStringPtr(ctx, encryptedEmail.String, "guest_order:customer_email"); err != nil {
				log.Error().Err(err).Msg("Failed to decrypt customer_email")
				return nil, fmt.Errorf("failed to decrypt customer_email: %w", err)
			}
		}
		if encryptedIP.Valid {
			if order.IPAddress, err = r.decryptToStringPtr(ctx, encryptedIP.String, "guest_order:ip_address"); err != nil {
				log.Error().Err(err).Msg("Failed to decrypt ip_address")
				return nil, fmt.Errorf("failed to decrypt ip_address: %w", err)
			}
		}
		if encryptedUA.Valid {
			if order.UserAgent, err = r.decryptToStringPtr(ctx, encryptedUA.String, "guest_order:user_agent"); err != nil {
				log.Error().Err(err).Msg("Failed to decrypt user_agent")
				return nil, fmt.Errorf("failed to decrypt user_agent: %w", err)
			}
		}

		orders = append(orders, &order)
	}

	if err = rows.Err(); err != nil {
		log.Error().Err(err).Msg("Error iterating order rows")
		return nil, err
	}

	return orders, nil
}

// GetOrderItemsByOrderID retrieves all items for a specific order
func (r *OrderRepository) GetOrderItemsByOrderID(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	query := `
SELECT id, order_id, product_id, product_name, unit_price, quantity, total_price
FROM order_items
WHERE order_id = $1
ORDER BY id
`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		log.Error().Err(err).Str("order_id", orderID).Msg("Failed to query order items")
		return nil, err
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.ProductName,
			&item.UnitPrice,
			&item.Quantity,
			&item.TotalPrice,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan order item row")
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// CreateOrderNote adds a note to an order
func (r *OrderRepository) CreateOrderNote(ctx context.Context, note *models.OrderNote) error {
	query := `
INSERT INTO order_notes (order_id, note, created_by_user_id, created_by_name)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at
`

	err := r.db.QueryRowContext(ctx, query, note.OrderID, note.Note, note.CreatedByUserID, note.CreatedByName).
		Scan(&note.ID, &note.CreatedAt)
	if err != nil {
		log.Error().Err(err).Str("order_id", note.OrderID).Msg("Failed to create order note")
		return err
	}

	log.Info().Str("order_id", note.OrderID).Str("note_id", note.ID).Msg("Order note created")
	return nil
}

// GetOrderNotesByOrderID retrieves all notes for a specific order
func (r *OrderRepository) GetOrderNotesByOrderID(ctx context.Context, orderID string) ([]*models.OrderNote, error) {
	query := `
SELECT id, order_id, note, created_by_user_id, created_by_name, created_at
FROM order_notes
WHERE order_id = $1
ORDER BY created_at DESC
`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		log.Error().Err(err).Str("order_id", orderID).Msg("Failed to query order notes")
		return nil, err
	}
	defer rows.Close()

	var notes []*models.OrderNote
	for rows.Next() {
		var note models.OrderNote
		err := rows.Scan(
			&note.ID,
			&note.OrderID,
			&note.Note,
			&note.CreatedByUserID,
			&note.CreatedByName,
			&note.CreatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan order note row")
			return nil, err
		}
		notes = append(notes, &note)
	}

	return notes, rows.Err()
}
