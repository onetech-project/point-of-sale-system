package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/rs/zerolog/log"
)

// OrderService handles business logic for order management
type OrderService struct {
	db        *sql.DB
	orderRepo *repository.OrderRepository
}

// NewOrderService creates a new order service
func NewOrderService(db *sql.DB, orderRepo *repository.OrderRepository) *OrderService {
	return &OrderService{
		db:        db,
		orderRepo: orderRepo,
	}
}

// GetOrderByReference retrieves an order by its reference number
func (s *OrderService) GetOrderByReference(ctx context.Context, orderReference string) (*models.GuestOrder, error) {
	return s.orderRepo.GetOrderByReference(ctx, orderReference)
}

// GetOrderByID retrieves an order by its ID
func (s *OrderService) GetOrderByID(ctx context.Context, orderID string) (*models.GuestOrder, error) {
	return s.orderRepo.GetOrderByID(ctx, orderID)
}

// ListOrdersByTenant retrieves orders for a tenant with optional status filter
func (s *OrderService) ListOrdersByTenant(
	ctx context.Context,
	tenantID string,
	status *models.OrderStatus,
	limit, offset int,
) ([]*models.GuestOrder, error) {
	return s.orderRepo.ListOrdersByTenant(ctx, tenantID, status, limit, offset)
}

// UpdateOrderStatus updates order status with validation
// Implements T088: Status transition validation following state machine from research.md
func (s *OrderService) UpdateOrderStatus(
	ctx context.Context,
	orderID string,
	newStatus models.OrderStatus,
) error {
	// Get current order
	order, err := s.orderRepo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		return fmt.Errorf("order not found")
	}

	// Validate status transition
	if !s.isValidTransition(order.Status, newStatus) {
		log.Warn().
			Str("order_id", orderID).
			Str("current_status", string(order.Status)).
			Str("new_status", string(newStatus)).
			Msg("Invalid status transition attempted")
		return fmt.Errorf("invalid status transition from %s to %s", order.Status, newStatus)
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Calculate timestamps based on new status (T089)
	now := time.Now()
	var paidAt, completedAt, cancelledAt *time.Time

	switch newStatus {
	case models.OrderStatusPaid:
		paidAt = &now
	case models.OrderStatusComplete:
		completedAt = &now
	case models.OrderStatusCancelled:
		cancelledAt = &now
	}

	// Update order status
	err = s.orderRepo.UpdateOrderStatus(ctx, tx, orderID, newStatus, paidAt, completedAt, cancelledAt)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info().
		Str("order_id", orderID).
		Str("order_reference", order.OrderReference).
		Str("old_status", string(order.Status)).
		Str("new_status", string(newStatus)).
		Msg("Order status updated successfully")

	return nil
}

// isValidTransition validates state machine transitions
// Implements T088: State machine validation from research.md
//
// Valid transitions:
// PENDING -> PAID (payment webhook)
// PENDING -> CANCELLED (payment failed/expired or admin cancellation)
// PAID -> COMPLETE (admin marks as delivered)
// PAID -> CANCELLED (admin cancellation after payment - requires refund process)
func (s *OrderService) isValidTransition(currentStatus, newStatus models.OrderStatus) bool {
	// Same status is always valid (idempotent updates)
	if currentStatus == newStatus {
		return true
	}

	// Define valid transitions
	validTransitions := map[models.OrderStatus][]models.OrderStatus{
		models.OrderStatusPending: {
			models.OrderStatusPaid,
			models.OrderStatusCancelled,
		},
		models.OrderStatusPaid: {
			models.OrderStatusComplete,
			models.OrderStatusCancelled, // Requires refund handling
		},
		// Terminal states - no transitions allowed
		models.OrderStatusComplete:  {},
		models.OrderStatusCancelled: {},
	}

	allowedTransitions, exists := validTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, allowedStatus := range allowedTransitions {
		if newStatus == allowedStatus {
			return true
		}
	}

	return false
}

// AddOrderNote adds a note to an order (for courier tracking, admin comments, etc.)
func (s *OrderService) AddOrderNote(ctx context.Context, orderID, note, userName string) error {
	// Use provided userName from API Gateway (X-User-Name header)
	// Default to "Admin" if not provided
	createdByName := userName
	if createdByName == "" {
		createdByName = "Admin"
	}

	// Create note record
	orderNote := &models.OrderNote{
		OrderID:       orderID,
		Note:          note,
		CreatedByName: &createdByName,
	}

	err := s.orderRepo.CreateOrderNote(ctx, orderNote)
	if err != nil {
		return fmt.Errorf("failed to create order note: %w", err)
	}

	log.Info().
		Str("order_id", orderID).
		Str("note_id", orderNote.ID).
		Str("note", note).
		Msg("Note added to order and saved to order_notes table")

	return nil
}

// GetOrderItems retrieves all items for a specific order
func (s *OrderService) GetOrderItems(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	return s.orderRepo.GetOrderItemsByOrderID(ctx, orderID)
}

// GetOrderNotes retrieves all notes for a specific order
func (s *OrderService) GetOrderNotes(ctx context.Context, orderID string) ([]*models.OrderNote, error) {
	return s.orderRepo.GetOrderNotesByOrderID(ctx, orderID)
}
