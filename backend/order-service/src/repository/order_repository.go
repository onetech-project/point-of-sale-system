package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/rs/zerolog/log"
)

// OrderRepository handles database operations for orders
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{
		db: db,
	}
}

// GetOrderByReference retrieves an order by its reference number
func (r *OrderRepository) GetOrderByReference(ctx context.Context, orderReference string) (*models.GuestOrder, error) {
	query := `
SELECT id, order_reference, tenant_id, status, subtotal_amount, delivery_fee, total_amount,
       customer_name, customer_phone, delivery_type, table_number, notes,
       created_at, paid_at, completed_at, cancelled_at, session_id, ip_address, user_agent
FROM guest_orders
WHERE order_reference = $1
`

	var order models.GuestOrder
	err := r.db.QueryRowContext(ctx, query, orderReference).Scan(
		&order.ID,
		&order.OrderReference,
		&order.TenantID,
		&order.Status,
		&order.SubtotalAmount,
		&order.DeliveryFee,
		&order.TotalAmount,
		&order.CustomerName,
		&order.CustomerPhone,
		&order.DeliveryType,
		&order.TableNumber,
		&order.Notes,
		&order.CreatedAt,
		&order.PaidAt,
		&order.CompletedAt,
		&order.CancelledAt,
		&order.SessionID,
		&order.IPAddress,
		&order.UserAgent,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Order not found
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("order_reference", orderReference).
			Msg("Failed to get order by reference")
		return nil, err
	}

	return &order, nil
}

// GetOrderByID retrieves an order by its ID
func (r *OrderRepository) GetOrderByID(ctx context.Context, orderID string) (*models.GuestOrder, error) {
	query := `
SELECT id, order_reference, tenant_id, status, subtotal_amount, delivery_fee, total_amount,
       customer_name, customer_phone, delivery_type, table_number, notes,
       created_at, paid_at, completed_at, cancelled_at, session_id, ip_address, user_agent
FROM guest_orders
WHERE id = $1
`

	var order models.GuestOrder
	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&order.ID,
		&order.OrderReference,
		&order.TenantID,
		&order.Status,
		&order.SubtotalAmount,
		&order.DeliveryFee,
		&order.TotalAmount,
		&order.CustomerName,
		&order.CustomerPhone,
		&order.DeliveryType,
		&order.TableNumber,
		&order.Notes,
		&order.CreatedAt,
		&order.PaidAt,
		&order.CompletedAt,
		&order.CancelledAt,
		&order.SessionID,
		&order.IPAddress,
		&order.UserAgent,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Order not found
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to get order by ID")
		return nil, err
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
       customer_name, customer_phone, delivery_type, table_number, notes,
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
		err := rows.Scan(
			&order.ID,
			&order.OrderReference,
			&order.TenantID,
			&order.Status,
			&order.SubtotalAmount,
			&order.DeliveryFee,
			&order.TotalAmount,
			&order.CustomerName,
			&order.CustomerPhone,
			&order.DeliveryType,
			&order.TableNumber,
			&order.Notes,
			&order.CreatedAt,
			&order.PaidAt,
			&order.CompletedAt,
			&order.CancelledAt,
			&order.SessionID,
			&order.IPAddress,
			&order.UserAgent,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan order row")
			return nil, err
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
