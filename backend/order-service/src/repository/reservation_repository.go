package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
)

type ReservationRepository struct {
	db *sql.DB
}

func NewReservationRepository(db *sql.DB) *ReservationRepository {
	return &ReservationRepository{db: db}
}

// CreateReservation creates a new inventory reservation
func (r *ReservationRepository) CreateReservation(ctx context.Context, reservation *models.InventoryReservation) error {
	query := `
INSERT INTO inventory_reservations (
order_id, product_id, quantity, status, expires_at, released_at
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at
`

	return r.db.QueryRowContext(
		ctx,
		query,
		reservation.OrderID,
		reservation.ProductID,
		reservation.Quantity,
		reservation.Status,
		reservation.ExpiresAt,
		reservation.ReleasedAt,
	).Scan(&reservation.ID, &reservation.CreatedAt)
}

// GetReservationsByOrderID retrieves all reservations for an order
func (r *ReservationRepository) GetReservationsByOrderID(ctx context.Context, orderID string) ([]*models.InventoryReservation, error) {
	query := `
SELECT id, order_id, product_id, quantity, status,
   created_at, expires_at, released_at
FROM inventory_reservations
WHERE order_id = $1
ORDER BY created_at DESC
`

	rows, err := r.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []*models.InventoryReservation
	for rows.Next() {
		reservation := &models.InventoryReservation{}
		err := rows.Scan(
			&reservation.ID,
			&reservation.OrderID,
			&reservation.ProductID,
			&reservation.Quantity,
			&reservation.Status,
			&reservation.CreatedAt,
			&reservation.ExpiresAt,
			&reservation.ReleasedAt,
		)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, reservation)
	}

	return reservations, rows.Err()
}

// GetReservationByID retrieves a specific reservation
func (r *ReservationRepository) GetReservationByID(ctx context.Context, id string) (*models.InventoryReservation, error) {
	query := `
SELECT id, order_id, product_id, quantity, status,
   created_at, expires_at, released_at
FROM inventory_reservations
WHERE id = $1
`

	reservation := &models.InventoryReservation{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&reservation.ID,
		&reservation.OrderID,
		&reservation.ProductID,
		&reservation.Quantity,
		&reservation.Status,
		&reservation.CreatedAt,
		&reservation.ExpiresAt,
		&reservation.ReleasedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return reservation, err
}

// UpdateReservationStatus updates the status and released time
func (r *ReservationRepository) UpdateReservationStatus(ctx context.Context, id string, status models.ReservationStatus, releasedAt *time.Time) error {
	query := `
UPDATE inventory_reservations
SET status = $2, released_at = $3
WHERE id = $1
`

	_, err := r.db.ExecContext(ctx, query, id, status, releasedAt)
	return err
}

// DeleteReservation removes a reservation
func (r *ReservationRepository) DeleteReservation(ctx context.Context, id string) error {
	query := `DELETE FROM inventory_reservations WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetExpiredReservations retrieves all expired active reservations
func (r *ReservationRepository) GetExpiredReservations(ctx context.Context) ([]*models.InventoryReservation, error) {
	query := `
		SELECT id, order_id, product_id, quantity, status,
			   created_at, expires_at, released_at
		FROM inventory_reservations
		WHERE status = 'active' AND expires_at < NOW()
		ORDER BY expires_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []*models.InventoryReservation
	for rows.Next() {
		reservation := &models.InventoryReservation{}
		err := rows.Scan(
			&reservation.ID,
			&reservation.OrderID,
			&reservation.ProductID,
			&reservation.Quantity,
			&reservation.Status,
			&reservation.CreatedAt,
			&reservation.ExpiresAt,
			&reservation.ReleasedAt,
		)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, reservation)
	}

	return reservations, rows.Err()
}

// GetTotalReservedQuantity returns the total reserved quantity for a product
func (r *ReservationRepository) GetTotalReservedQuantity(ctx context.Context, productID string) (int, error) {
	query := `
		SELECT COALESCE(SUM(quantity), 0)
		FROM inventory_reservations
		WHERE product_id = $1 AND status = 'active'
	`

	var total int
	err := r.db.QueryRowContext(ctx, query, productID).Scan(&total)
	return total, err
}

// ConvertReservation converts a reservation to "converted" status
func (r *ReservationRepository) ConvertReservation(ctx context.Context, tx *sql.Tx, id string) error {
	query := `
		UPDATE inventory_reservations
		SET status = 'converted'
		WHERE id = $1
	`

	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, id)
	} else {
		_, err = r.db.ExecContext(ctx, query, id)
	}
	return err
}

// ReleaseReservation releases a reservation
func (r *ReservationRepository) ReleaseReservation(ctx context.Context, tx *sql.Tx, id string) error {
	now := time.Now()
	query := `
		UPDATE inventory_reservations
		SET status = 'released', released_at = $2
		WHERE id = $1
	`

	var err error
	if tx != nil {
		_, err = tx.ExecContext(ctx, query, id, now)
	} else {
		_, err = r.db.ExecContext(ctx, query, id, now)
	}
	return err
}
