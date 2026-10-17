package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
)

const (
	ReservationTTL       = 15 * time.Minute
	InventoryCachePrefix = "inventory:"
	InventoryCacheTTL    = 5 * time.Minute

	// OfflineReservationTTL keeps offline order reservations out of the cleanup job's reach:
	// staff-recorded orders hold their stock until they are paid or cancelled, which can take
	// months for installment orders.
	OfflineReservationTTL = 10 * 365 * 24 * time.Hour
)

// ErrInsufficientStock is returned when a product does not have enough unreserved stock
var ErrInsufficientStock = errors.New("insufficient stock")

type InventoryService struct {
	db              *sql.DB
	redisClient     *redis.Client
	reservationRepo *repository.ReservationRepository
}

func NewInventoryService(db *sql.DB, redisClient *redis.Client) *InventoryService {
	return &InventoryService{
		db:              db,
		redisClient:     redisClient,
		reservationRepo: repository.NewReservationRepository(db),
	}
}

// CheckAvailabilityWithLock checks if products are available and locks them for reservation
// Uses SELECT FOR UPDATE to prevent race conditions
func (s *InventoryService) CheckAvailabilityWithLock(ctx context.Context, tx *sql.Tx, tenantID string, items []models.CartItem) error {
	for _, item := range items {
		// Get product quantity with row-level lock
		var currentQuantity int
		query := `
SELECT stock_quantity
FROM products
WHERE tenant_id = $1 AND id = $2
FOR UPDATE
`

		err := tx.QueryRowContext(ctx, query, tenantID, item.ProductID).Scan(&currentQuantity)
		if err == sql.ErrNoRows {
			return fmt.Errorf("product %s not found", item.ProductID)
		}
		if err != nil {
			return fmt.Errorf("failed to check product %s: %w", item.ProductID, err)
		}

		// Calculate available inventory (current - active reservations)
		reserved, err := s.reservationRepo.GetTotalReservedQuantity(ctx, item.ProductID)
		if err != nil {
			return fmt.Errorf("failed to get reservations for product %s: %w", item.ProductID, err)
		}

		available := currentQuantity - reserved
		if available < item.Quantity {
			return fmt.Errorf("insufficient stock for product %s (available: %d, requested: %d)",
				item.ProductName, available, item.Quantity)
		}
	}

	return nil
}

// CreateReservations creates inventory reservations for cart items
func (s *InventoryService) CreateReservations(ctx context.Context, tx *sql.Tx, orderID string, items []models.CartItem) error {
	expiresAt := time.Now().Add(ReservationTTL)

	for _, item := range items {
		reservation := &models.InventoryReservation{
			OrderID:   orderID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Status:    models.ReservationStatusActive,
			ExpiresAt: expiresAt,
		}

		err := s.reservationRepo.CreateReservation(ctx, tx, reservation)
		if err != nil {
			log.Error().Err(err).
				Str("order_id", orderID).
				Str("product_id", item.ProductID).
				Msg("Failed to create reservation")
			return fmt.Errorf("failed to create reservation for product %s: %w", item.ProductID, err)
		}

		log.Info().
			Str("reservation_id", reservation.ID).
			Str("order_id", orderID).
			Str("product_id", item.ProductID).
			Int("quantity", item.Quantity).
			Time("expires_at", expiresAt).
			Msg("Reservation created")
	}

	return nil
}

// ReserveStockForOrder locks the products, verifies availability and reserves the requested
// quantities for an order inside the caller's transaction. Quantities of duplicate products are
// combined, and products are locked in a stable order to avoid deadlocks between concurrent orders.
func (s *InventoryService) ReserveStockForOrder(ctx context.Context, tx *sql.Tx, tenantID, orderID string, items []models.CartItem, expiresAt time.Time) error {
	quantities := make(map[string]int)
	names := make(map[string]string)
	for _, item := range items {
		quantities[item.ProductID] += item.Quantity
		names[item.ProductID] = item.ProductName
	}

	productIDs := make([]string, 0, len(quantities))
	for productID := range quantities {
		productIDs = append(productIDs, productID)
	}
	sort.Strings(productIDs)

	for _, productID := range productIDs {
		requested := quantities[productID]

		var currentQuantity int
		query := `
SELECT stock_quantity
FROM products
WHERE tenant_id = $1 AND id = $2
FOR UPDATE
`
		err := tx.QueryRowContext(ctx, query, tenantID, productID).Scan(&currentQuantity)
		if err == sql.ErrNoRows {
			return fmt.Errorf("product %s not found", names[productID])
		}
		if err != nil {
			return fmt.Errorf("failed to check product %s: %w", productID, err)
		}

		// Read reservations through the transaction so reservations released earlier in it
		// (e.g. when editing an order's items) are no longer counted
		reserved, err := s.reservationRepo.GetTotalReservedQuantityTx(ctx, tx, productID)
		if err != nil {
			return fmt.Errorf("failed to get reservations for product %s: %w", productID, err)
		}

		available := currentQuantity - reserved
		if available < requested {
			if available < 0 {
				available = 0
			}
			return fmt.Errorf("%w for product %s (available: %d, requested: %d)",
				ErrInsufficientStock, names[productID], available, requested)
		}

		reservation := &models.InventoryReservation{
			OrderID:   orderID,
			ProductID: productID,
			Quantity:  requested,
			Status:    models.ReservationStatusActive,
			ExpiresAt: expiresAt,
		}
		if err := s.reservationRepo.CreateReservation(ctx, tx, reservation); err != nil {
			return fmt.Errorf("failed to create reservation for product %s: %w", productID, err)
		}

		log.Info().
			Str("reservation_id", reservation.ID).
			Str("order_id", orderID).
			Str("product_id", productID).
			Int("quantity", requested).
			Time("expires_at", expiresAt).
			Msg("Reservation created")
	}

	return nil
}

// ConvertReservationsTx converts an order's active reservations to permanent allocations inside
// the caller's transaction, decrementing product stock. Already converted or released
// reservations are skipped, so calling it more than once for an order is safe.
func (s *InventoryService) ConvertReservationsTx(ctx context.Context, tx *sql.Tx, orderID string) error {
	reservations, err := s.reservationRepo.GetActiveReservationsByOrderIDTx(ctx, tx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get reservations for order %s: %w", orderID, err)
	}

	for _, reservation := range reservations {
		// Convert reservation status
		if err := s.reservationRepo.ConvertReservation(ctx, tx, reservation.ID); err != nil {
			return fmt.Errorf("failed to convert reservation %s: %w", reservation.ID, err)
		}

		// Decrement product quantity permanently
		query := `
UPDATE products
SET stock_quantity = stock_quantity - $1,
    updated_at = NOW()
WHERE id = $2 AND stock_quantity >= $1
`
		result, err := tx.ExecContext(ctx, query, reservation.Quantity, reservation.ProductID)
		if err != nil {
			return fmt.Errorf("failed to decrement product %s quantity: %w", reservation.ProductID, err)
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to check affected rows: %w", err)
		}

		if rows == 0 {
			return fmt.Errorf("%w for product %s during conversion", ErrInsufficientStock, reservation.ProductID)
		}

		log.Info().
			Str("reservation_id", reservation.ID).
			Str("order_id", orderID).
			Str("product_id", reservation.ProductID).
			Int("quantity", reservation.Quantity).
			Msg("Reservation converted to permanent allocation")
	}

	return nil
}

// ConvertReservationsToPermanent converts reservations to permanent inventory allocation after payment
func (s *InventoryService) ConvertReservationsToPermanent(ctx context.Context, orderID string) error {
	// Begin transaction for atomic conversion
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.ConvertReservationsTx(ctx, tx, orderID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit conversion transaction: %w", err)
	}

	return nil
}

// ReleaseReservationsTx releases an order's active reservations inside the caller's transaction
func (s *InventoryService) ReleaseReservationsTx(ctx context.Context, tx *sql.Tx, orderID string) error {
	released, err := s.reservationRepo.ReleaseActiveReservationsByOrderID(ctx, tx, orderID)
	if err != nil {
		return fmt.Errorf("failed to release reservations for order %s: %w", orderID, err)
	}

	if released > 0 {
		log.Info().
			Str("order_id", orderID).
			Int64("released", released).
			Msg("Reservations released")
	}

	return nil
}

// ReleaseReservations releases reservations (for expired or cancelled orders)
func (s *InventoryService) ReleaseReservations(ctx context.Context, orderID string) error {
	reservations, err := s.reservationRepo.GetReservationsByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get reservations for order %s: %w", orderID, err)
	}

	for _, reservation := range reservations {
		if reservation.Status != models.ReservationStatusActive {
			continue
		}

		if err := s.reservationRepo.ReleaseReservation(ctx, nil, reservation.ID); err != nil {
			log.Error().Err(err).
				Str("reservation_id", reservation.ID).
				Msg("Failed to release reservation")
			continue
		}

		log.Info().
			Str("reservation_id", reservation.ID).
			Str("order_id", orderID).
			Str("product_id", reservation.ProductID).
			Msg("Reservation released")
	}

	return nil
}

// GetAvailableInventory calculates available inventory for a product
func (s *InventoryService) GetAvailableInventory(ctx context.Context, tenantID, productID string) (int, error) {
	// Get current quantity from database
	var currentQuantity int
	query := `SELECT quantity FROM products WHERE tenant_id = $1 AND id = $2`
	err := s.db.QueryRowContext(ctx, query, tenantID, productID).Scan(&currentQuantity)
	if err != nil {
		return 0, fmt.Errorf("failed to get product quantity: %w", err)
	}

	// Get total active reservations
	reserved, err := s.reservationRepo.GetTotalReservedQuantity(ctx, productID)
	if err != nil {
		return 0, fmt.Errorf("failed to get reserved quantity: %w", err)
	}

	available := currentQuantity - reserved
	if available < 0 {
		available = 0
	}

	return available, nil
}
