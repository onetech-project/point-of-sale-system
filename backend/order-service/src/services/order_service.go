package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/queue"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/rs/zerolog/log"
)

// OrderService handles business logic for order management
type OrderService struct {
	db            *sql.DB
	orderRepo     *repository.OrderRepository
	addressRepo   *repository.AddressRepository
	paymentRepo   *repository.PaymentRepository
	kafkaProducer *queue.KafkaProducer
}

// NewOrderService creates a new order service
func NewOrderService(
	db *sql.DB,
	orderRepo *repository.OrderRepository,
	addressRepo *repository.AddressRepository,
	paymentRepo *repository.PaymentRepository,
	kafkaProducer *queue.KafkaProducer,
) *OrderService {
	return &OrderService{
		db:            db,
		orderRepo:     orderRepo,
		addressRepo:   addressRepo,
		paymentRepo:   paymentRepo,
		kafkaProducer: kafkaProducer,
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

	// Publish order.paid event to Kafka if status changed to PAID
	if newStatus == models.OrderStatusPaid {
		if err := s.publishOrderPaidEvent(ctx, order); err != nil {
			log.Error().
				Err(err).
				Str("order_id", orderID).
				Str("order_reference", order.OrderReference).
				Msg("Failed to publish order.paid event to Kafka - notification may not be sent")
			// Don't fail the status update if Kafka publish fails
		}
	}

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

// publishOrderPaidEvent publishes an order.paid event to Kafka for notification service
func (s *OrderService) publishOrderPaidEvent(ctx context.Context, order *models.GuestOrder) error {
	if s.kafkaProducer == nil {
		log.Warn().Msg("Kafka producer not initialized - skipping order.paid event")
		return nil
	}

	// Get order items
	items, err := s.orderRepo.GetOrderItemsByOrderID(ctx, order.ID)
	if err != nil {
		return fmt.Errorf("failed to get order items: %w", err)
	}

	// Get delivery address if applicable
	var deliveryAddress string
	if order.DeliveryType == models.DeliveryTypeDelivery {
		addr, err := s.addressRepo.GetByOrderID(ctx, order.ID)
		if err == nil && addr != nil {
			deliveryAddress = addr.FullAddress
		}
	}

	// Get payment info
	paymentTxn, err := s.paymentRepo.GetPaymentByOrderID(ctx, order.ID)
	if err != nil {
		log.Warn().Err(err).Str("order_id", order.ID).Msg("Failed to get payment info")
	}

	// Extract payment details
	transactionID := ""
	paymentMethod := "unknown"
	if paymentTxn != nil {
		if paymentTxn.MidtransTransactionID != nil {
			transactionID = *paymentTxn.MidtransTransactionID
		}
		if paymentTxn.PaymentType != nil {
			paymentMethod = *paymentTxn.PaymentType
		}
	}

	// Convert order items to event format
	eventItems := make([]map[string]interface{}, len(items))
	for i, item := range items {
		eventItems[i] = map[string]interface{}{
			"product_id":   item.ProductID,
			"product_name": item.ProductName,
			"quantity":     item.Quantity,
			"unit_price":   item.UnitPrice,
			"total_price":  item.TotalPrice,
		}
	}

	// Handle optional customer email
	customerEmail := ""
	if order.CustomerEmail != nil {
		customerEmail = *order.CustomerEmail
	}

	// Handle optional table number
	tableNumber := ""
	if order.TableNumber != nil {
		tableNumber = *order.TableNumber
	}

	// Prepare event payload
	event := map[string]interface{}{
		"event_id":   fmt.Sprintf("order-paid-%s-%d", order.ID, time.Now().Unix()),
		"event_type": "order.paid",
		"tenant_id":  order.TenantID,
		"timestamp":  time.Now().Format(time.RFC3339),
		"metadata": map[string]interface{}{
			"order_id":         order.ID,
			"order_reference":  order.OrderReference,
			"transaction_id":   transactionID,
			"customer_name":    order.CustomerName,
			"customer_phone":   order.CustomerPhone,
			"customer_email":   customerEmail,
			"delivery_type":    order.DeliveryType,
			"delivery_address": deliveryAddress,
			"table_number":     tableNumber,
			"items":            eventItems,
			"subtotal_amount":  order.SubtotalAmount,
			"delivery_fee":     order.DeliveryFee,
			"total_amount":     order.TotalAmount,
			"payment_method":   paymentMethod,
			"paid_at":          order.PaidAt,
		},
	}

	// Publish to Kafka
	key := fmt.Sprintf("order-%s", order.ID)
	if err := s.kafkaProducer.Publish(ctx, key, event); err != nil {
		return fmt.Errorf("failed to publish to Kafka: %w", err)
	}

	log.Info().
		Str("order_id", order.ID).
		Str("order_reference", order.OrderReference).
		Str("transaction_id", transactionID).
		Msg("Published order.paid event to Kafka")

	return nil
}
