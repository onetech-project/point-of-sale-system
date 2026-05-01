package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/observability"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// OfflineOrderService handles business logic for offline order management
// Implements User Story 1: Record Basic Offline Order (MVP)
// Implements User Story 2: Manage Payment Terms and Installments
type OfflineOrderService struct {
	db                     *sql.DB
	offlineOrderRepo       *repository.OfflineOrderRepository
	orderItemRepo          *repository.OrderRepository // Reuse existing order item operations
	paymentRepo            *repository.PaymentRepository
	outboxRepo             *repository.OutboxRepository
	eventPublisher         *EventPublisher
	paymentCalculator      *PaymentCalculator
	tracer                 trace.Tracer // T113: OpenTelemetry tracer
}

// NewOfflineOrderService creates a new offline order service
func NewOfflineOrderService(
	db *sql.DB,
	offlineOrderRepo *repository.OfflineOrderRepository,
	orderItemRepo *repository.OrderRepository,
	paymentRepo *repository.PaymentRepository,
	outboxRepo *repository.OutboxRepository,
	eventPublisher *EventPublisher,
	paymentCalculator *PaymentCalculator,
) *OfflineOrderService {
	return &OfflineOrderService{
		db:                db,
		offlineOrderRepo:  offlineOrderRepo,
		orderItemRepo:     orderItemRepo,
		paymentRepo:       paymentRepo,
		outboxRepo:        outboxRepo,
		eventPublisher:    eventPublisher,
		paymentCalculator: paymentCalculator,
		tracer:            otel.Tracer("offline-order-service"), // T113: Initialize tracer
	}
}

// CreateOfflineOrderRequest represents the request to create an offline order
type CreateOfflineOrderRequest struct {
	TenantID          string                       `json:"tenant_id" validate:"required,uuid"`
	CustomerName      string                       `json:"customer_name" validate:"required,min=2,max=255"`
	CustomerPhone     string                       `json:"customer_phone" validate:"required,min=10,max=20"`
	CustomerEmail     *string                      `json:"customer_email,omitempty" validate:"omitempty,email"`
	DeliveryType      models.DeliveryType          `json:"delivery_type" validate:"required,oneof=pickup delivery dine_in"`
	TableNumber       *string                      `json:"table_number,omitempty"`
	Notes             *string                      `json:"notes,omitempty"`
	Items             []models.CreateOrderItemReq  `json:"items" validate:"required,min=1,dive"`
	DataConsentGiven  bool                         `json:"data_consent_given" validate:"required"`
	ConsentMethod     *models.ConsentMethod        `json:"consent_method" validate:"required_if=DataConsentGiven true"`
	RecordedByUserID  string                       `json:"recorded_by_user_id" validate:"required,uuid"`
	PaymentInfo       *PaymentInfo                 `json:"payment,omitempty"` // US2: Payment terms
}

// PaymentInfo represents payment details for an offline order
type PaymentInfo struct {
	Type                string                `json:"type" validate:"required,oneof=full installment"` // "full" or "installment"
	Amount              *int                  `json:"amount,omitempty"`                                // For full payment
	Method              *models.PaymentMethod `json:"method,omitempty"`                                // For full payment
	DownPaymentAmount   *int                  `json:"down_payment_amount,omitempty"`                   // For installment
	DownPaymentMethod   *models.PaymentMethod `json:"down_payment_method,omitempty"`                   // For installment
	InstallmentCount    int                   `json:"installment_count,omitempty"`                     // Number of installments
	InstallmentAmount   int                   `json:"installment_amount,omitempty"`                    // Amount per installment
	PaymentSchedule     []models.Installment  `json:"payment_schedule,omitempty"`                      // Detailed schedule
}

// CreateOfflineOrder creates a new offline order with full validation
// Implements T030: Business logic validation and T031: Order reference generation
// T112: Records Prometheus metrics for order creation
// T113: Adds OpenTelemetry tracing spans
func (s *OfflineOrderService) CreateOfflineOrder(ctx context.Context, req *CreateOfflineOrderRequest) (*models.GuestOrder, error) {
	// T113: Start trace span
	ctx, span := s.tracer.Start(ctx, "CreateOfflineOrder",
		trace.WithAttributes(
			attribute.String("tenant_id", req.TenantID),
			attribute.String("delivery_type", string(req.DeliveryType)),
			attribute.Bool("has_payment", req.PaymentInfo != nil),
		),
	)
	defer span.End()
	
	// T112: Start timer for order creation duration
	startTime := time.Now()
	
	// Validate data consent requirement for offline orders
	if !req.DataConsentGiven {
		span.RecordError(fmt.Errorf("data consent required"))
		span.SetAttributes(attribute.String("error.type", "validation_error"))
		return nil, fmt.Errorf("data consent is required for offline orders (UU PDP compliance)")
	}

	if req.ConsentMethod == nil {
		span.RecordError(fmt.Errorf("consent method required"))
		span.SetAttributes(attribute.String("error.type", "validation_error"))
		return nil, fmt.Errorf("consent method is required when data consent is given")
	}

	// Begin transaction for atomic operation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Generate order reference (GO-XXXXXX format)
	orderReference := s.generateOrderReference()

	// Calculate totals from items
	var subtotalAmount int
	for _, item := range req.Items {
		subtotalAmount += item.Quantity * item.UnitPrice
	}

	deliveryFee := 0 // Calculate based on delivery type if needed
	totalAmount := subtotalAmount + deliveryFee

	// Create order entity
	order := &models.GuestOrder{
		TenantID:         req.TenantID,
		OrderReference:   orderReference,
		Status:           models.OrderStatusPending,
		OrderType:        models.OrderTypeOffline,
		DeliveryType:     req.DeliveryType,
		CustomerName:     req.CustomerName,
		CustomerPhone:    req.CustomerPhone,
		CustomerEmail:    req.CustomerEmail,
		TableNumber:      req.TableNumber,
		Notes:            req.Notes,
		SubtotalAmount:   subtotalAmount,
		DeliveryFee:      deliveryFee,
		TotalAmount:      totalAmount,
		DataConsentGiven: req.DataConsentGiven,
		ConsentMethod:    req.ConsentMethod,
		RecordedByUserID: &req.RecordedByUserID,
	}

	// Insert order into database
	orderID, err := s.offlineOrderRepo.CreateOfflineOrder(ctx, tx, order)
	if err != nil {
		return nil, fmt.Errorf("failed to create offline order: %w", err)
	}
	order.ID = orderID

	// Insert order items into database
	insertItemQuery := `
		INSERT INTO order_items (order_id, product_id, product_name, quantity, unit_price, total_price)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	for _, item := range req.Items {
		totalPrice := item.Quantity * item.UnitPrice
		_, err := tx.ExecContext(
			ctx,
			insertItemQuery,
			orderID,
			item.ProductID,
			item.ProductName,
			item.Quantity,
			item.UnitPrice,
			totalPrice,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	// T058: Handle payment terms if installment payment is specified
	var paymentTermsID *string
	if req.PaymentInfo != nil && req.PaymentInfo.Type == "installment" {
		// Validate installment payment info
		if len(req.PaymentInfo.PaymentSchedule) == 0 {
			return nil, fmt.Errorf("payment schedule is required for installment payments")
		}
		if req.PaymentInfo.InstallmentCount <= 0 {
			return nil, fmt.Errorf("installment count must be greater than 0")
		}

		// Create payment terms
		paymentTermsReq := &models.CreatePaymentTermsRequest{
			OrderID:            orderID,
			TotalAmount:        totalAmount,
			DownPaymentAmount:  req.PaymentInfo.DownPaymentAmount,
			InstallmentCount:   req.PaymentInfo.InstallmentCount,
			InstallmentAmount:  req.PaymentInfo.InstallmentAmount,
			PaymentSchedule:    req.PaymentInfo.PaymentSchedule,
			CreatedByUserID:    req.RecordedByUserID,
		}

		termsID, err := s.paymentRepo.CreatePaymentTerms(ctx, tx, paymentTermsReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create payment terms: %w", err)
		}
		paymentTermsID = &termsID

		// If down payment was made, record it
		if req.PaymentInfo.DownPaymentAmount != nil && *req.PaymentInfo.DownPaymentAmount > 0 {
			paymentRecordReq := &models.CreatePaymentRecordRequest{
				OrderID:               orderID,
				PaymentTermsID:        paymentTermsID,
				PaymentNumber:         0, // 0 indicates down payment
				AmountPaid:            *req.PaymentInfo.DownPaymentAmount,
				PaymentMethod:         *req.PaymentInfo.DownPaymentMethod,
				RemainingBalanceAfter: totalAmount - *req.PaymentInfo.DownPaymentAmount,
				RecordedByUserID:      req.RecordedByUserID,
			}

			_, err = s.paymentRepo.RecordPayment(ctx, tx, paymentRecordReq)
			if err != nil {
				return nil, fmt.Errorf("failed to record down payment: %w", err)
			}
		}
	} else if req.PaymentInfo != nil && req.PaymentInfo.Type == "full" {
		// Record full payment
		if req.PaymentInfo.Amount == nil || req.PaymentInfo.Method == nil {
			return nil, fmt.Errorf("amount and method are required for full payment")
		}

		paymentRecordReq := &models.CreatePaymentRecordRequest{
			OrderID:               orderID,
			PaymentTermsID:        nil, // No payment terms for full payment
			PaymentNumber:         0,
			AmountPaid:            *req.PaymentInfo.Amount,
			PaymentMethod:         *req.PaymentInfo.Method,
			RemainingBalanceAfter: 0,
			RecordedByUserID:      req.RecordedByUserID,
		}

		_, err = s.paymentRepo.RecordPayment(ctx, tx, paymentRecordReq)
		if err != nil {
			return nil, fmt.Errorf("failed to record payment: %w", err)
		}

		// Update order status to PAID
		// Note: In production, this should use a repository method
		_, err = tx.ExecContext(ctx, "UPDATE guest_orders SET status = $1, paid_at = $2 WHERE id = $3",
			models.OrderStatusPaid, time.Now(), orderID)
		if err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}
		order.Status = models.OrderStatusPaid
	}

	// Publish offline_order.created event to audit trail (T034)
	eventPayload := map[string]interface{}{
		"order_id":         orderID,
		"order_reference":  orderReference,
		"tenant_id":        req.TenantID,
		"customer_name":    req.CustomerName,
		"customer_phone":   req.CustomerPhone,
		"total_amount":     totalAmount,
		"recorded_by_user_id": req.RecordedByUserID,
		"consent_given":    req.DataConsentGiven,
		"consent_method":   req.ConsentMethod,
		"created_at":       time.Now().Format(time.RFC3339),
	}

	eventPayloadJSON, err := json.Marshal(eventPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	eventReq := &models.CreateEventOutboxRequest{
		EventType:    "offline_order.created",
		EventKey:     orderID,
		EventPayload: eventPayloadJSON,
		Topic:        "offline-orders-audit",
	}

	if err := s.eventPublisher.CreateEvent(ctx, tx, eventReq); err != nil {
		log.Error().Err(err).Str("order_id", orderID).Msg("Failed to create audit event")
		// Don't fail the order creation if audit event fails
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "transaction_commit_failed"))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// T113: Record successful operation in span
	span.SetAttributes(
		attribute.String("order_id", orderID),
		attribute.String("order_reference", orderReference),
		attribute.Int("total_amount", totalAmount),
		attribute.String("status", string(order.Status)),
	)
	if req.PaymentInfo != nil {
		span.SetAttributes(
			attribute.String("payment_type", req.PaymentInfo.Type),
		)
	}

	// T112: Record Prometheus metrics for offline order creation
	observability.OfflineOrdersTotal.WithLabelValues(string(order.Status), req.TenantID).Inc()
	observability.OfflineOrderRevenue.WithLabelValues(req.TenantID).Add(float64(totalAmount))
	observability.OfflineOrderCreationDuration.WithLabelValues(req.TenantID).Observe(time.Since(startTime).Seconds())
	
	// T112: Record installment metrics if applicable
	if req.PaymentInfo != nil && req.PaymentInfo.Type == "installment" {
		observability.PaymentInstallmentsTotal.WithLabelValues(req.TenantID, strconv.Itoa(req.PaymentInfo.InstallmentCount)).Inc()
	}

	log.Info().
		Str("order_id", orderID).
		Str("order_reference", orderReference).
		Str("tenant_id", req.TenantID).
		Str("recorded_by", req.RecordedByUserID).
		Msg("Offline order created successfully")

	return order, nil
}

// generateOrderReference generates a unique order reference in GO-XXXXXX format
// Implements T031: Order reference generator
func (s *OfflineOrderService) generateOrderReference() string {
	// Generate 6-digit random number with leading zeros
	randomNum := rand.Intn(1000000)
	return fmt.Sprintf("GO-%06d", randomNum)
}

// GetOfflineOrderByID retrieves an offline order with authorization check
// Implements T032: Authorization ensures user can only access orders from their tenant
func (s *OfflineOrderService) GetOfflineOrderByID(ctx context.Context, orderID string, tenantID string) (*models.GuestOrder, error) {
	order, err := s.offlineOrderRepo.GetOfflineOrderByID(ctx, orderID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get offline order: %w", err)
	}

	if order == nil {
		return nil, fmt.Errorf("offline order not found or access denied")
	}

	// Tenant isolation is enforced at repository level
	// Additional authorization checks could be added here if needed

	return order, nil
}

// GetOrderItemsByOrderID retrieves all items for a specific order.
func (s *OfflineOrderService) GetOrderItemsByOrderID(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	items, err := s.orderItemRepo.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	return items, nil
}

// ListOfflineOrders retrieves offline orders with tenant filtering and pagination
// Implements T033: Tenant filtering ensures users only see their tenant's orders
func (s *OfflineOrderService) ListOfflineOrders(ctx context.Context, tenantID string, filters ListOfflineOrdersFilters) (*ListOfflineOrdersResponse, error) {
	// Set default pagination if not provided
	if filters.Limit == 0 {
		filters.Limit = 20 // Default page size
	}
	if filters.Limit > 100 {
		filters.Limit = 100 // Max page size
	}

	repoFilters := repository.ListOfflineOrdersFilters{
		Status:      filters.Status,
		SearchQuery: filters.SearchQuery,
		Limit:       filters.Limit,
		Offset:      filters.Offset,
	}

	orders, totalCount, err := s.offlineOrderRepo.ListOfflineOrders(ctx, tenantID, repoFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to list offline orders: %w", err)
	}

	log.Debug().
		Str("tenant_id", tenantID).
		Int("count", len(orders)).
		Int("total", totalCount).
		Msg("Retrieved offline orders")

	return &ListOfflineOrdersResponse{
		Orders:     orders,
		TotalCount: totalCount,
		Page:       filters.Offset/filters.Limit + 1,
		PageSize:   filters.Limit,
	}, nil
}

// ListOfflineOrdersFilters holds filter parameters for listing offline orders
type ListOfflineOrdersFilters struct {
	Status      string `json:"status,omitempty"`       // Filter by order status
	SearchQuery string `json:"search_query,omitempty"` // Search by order_reference
	Limit       int    `json:"limit"`                  // Page size
	Offset      int    `json:"offset"`                 // Page offset
}

// ListOfflineOrdersResponse represents the paginated response for listing offline orders
type ListOfflineOrdersResponse struct {
	Orders     []models.GuestOrder `json:"orders"`
	TotalCount int                 `json:"total_count"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
}

// ValidateOrderAccess checks if a user has access to an order (helper for authorization)
func (s *OfflineOrderService) ValidateOrderAccess(ctx context.Context, orderID string, userTenantID string) error {
	order, err := s.offlineOrderRepo.GetOfflineOrderByID(ctx, orderID, userTenantID)
	if err != nil {
		return fmt.Errorf("order access denied: %w", err)
	}
	if order == nil {
		return fmt.Errorf("order not found or access denied")
	}
	return nil
}

// ============================================================================
// User Story 2: Payment Management Methods (T059, T061, T062)
// ============================================================================

// RecordPaymentRequest represents a request to record a payment for an offline order
type RecordPaymentRequest struct {
	OrderID          string                `json:"order_id" validate:"required,uuid"`
	TenantID         string                `json:"tenant_id" validate:"required,uuid"`
	AmountPaid       int                   `json:"amount_paid" validate:"required,min=1"`
	PaymentMethod    models.PaymentMethod  `json:"payment_method" validate:"required"`
	RecordedByUserID string                `json:"recorded_by_user_id" validate:"required,uuid"`
	Notes            *string               `json:"notes,omitempty"`
	ReceiptNumber    *string               `json:"receipt_number,omitempty"`
}

// RecordPayment records a payment for an offline order with validation
// T059: Implement RecordPayment method with validation
// T061: Implement automatic status update on full payment
// T062: Integrate event publishing for payment.received
func (s *OfflineOrderService) RecordPayment(ctx context.Context, req *RecordPaymentRequest) (*models.PaymentRecord, error) {
	// T113: Start trace span
	ctx, span := s.tracer.Start(ctx, "RecordPayment",
		trace.WithAttributes(
			attribute.String("order_id", req.OrderID),
			attribute.String("tenant_id", req.TenantID),
			attribute.Int("amount_paid", req.AmountPaid),
			attribute.String("payment_method", string(req.PaymentMethod)),
		),
	)
	defer span.End()
	
	// Validate order access
	order, err := s.offlineOrderRepo.GetOfflineOrderByID(ctx, req.OrderID, req.TenantID)
	if err != nil || order == nil {
		span.RecordError(fmt.Errorf("order not found or access denied"))
		span.SetAttributes(attribute.String("error.type", "order_not_found"))
		return nil, fmt.Errorf("order not found or access denied")
	}

	// Check if order is already fully paid
	if order.Status == models.OrderStatusPaid || order.Status == models.OrderStatusComplete {
		return nil, fmt.Errorf("order is already fully paid")
	}

	// Get payment terms (if exists)
	paymentTerms, err := s.paymentRepo.GetPaymentTerms(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment terms: %w", err)
	}

	// Get existing payment history
	paymentHistory, err := s.paymentRepo.GetPaymentHistory(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment history: %w", err)
	}

	// Calculate remaining balance
	var currentBalance int
	if paymentTerms != nil {
		currentBalance = paymentTerms.RemainingBalance
	} else {
		// For full payment orders without terms
		totalPaid := s.paymentCalculator.SumPaymentAmounts(paymentHistory)
		currentBalance = order.TotalAmount - totalPaid
	}

	// Validate payment amount
	if err := s.paymentCalculator.ValidatePaymentAmount(req.AmountPaid, currentBalance); err != nil {
		return nil, err
	}

	// Calculate remaining balance after payment
	remainingBalanceAfter, err := s.paymentCalculator.CalculateRemainingBalance(currentBalance, req.AmountPaid)
	if err != nil {
		return nil, err
	}

	// Determine payment number
	paymentNumber := s.paymentCalculator.CalculateNextPaymentNumber(paymentHistory)

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Record payment
	var paymentTermsID *string
	if paymentTerms != nil {
		paymentTermsID = &paymentTerms.ID
	}

	paymentRecordReq := &models.CreatePaymentRecordRequest{
		OrderID:               req.OrderID,
		PaymentTermsID:        paymentTermsID,
		PaymentNumber:         paymentNumber,
		AmountPaid:            req.AmountPaid,
		PaymentMethod:         req.PaymentMethod,
		RemainingBalanceAfter: remainingBalanceAfter,
		RecordedByUserID:      req.RecordedByUserID,
		Notes:                 req.Notes,
		ReceiptNumber:         req.ReceiptNumber,
	}

	paymentRecordID, err := s.paymentRepo.RecordPayment(ctx, tx, paymentRecordReq)
	if err != nil {
		return nil, fmt.Errorf("failed to record payment: %w", err)
	}

	// T061: Update order status if fully paid
	if remainingBalanceAfter == 0 {
		_, err = tx.ExecContext(ctx, "UPDATE guest_orders SET status = $1, paid_at = $2 WHERE id = $3",
			models.OrderStatusPaid, time.Now(), req.OrderID)
		if err != nil {
			return nil, fmt.Errorf("failed to update order status: %w", err)
		}
	}

	// T062: Publish payment.received event
	eventPayload := map[string]interface{}{
		"order_id":                req.OrderID,
		"payment_record_id":       paymentRecordID,
		"payment_number":          paymentNumber,
		"amount_paid":             req.AmountPaid,
		"payment_method":          req.PaymentMethod,
		"remaining_balance_after": remainingBalanceAfter,
		"recorded_by_user_id":     req.RecordedByUserID,
		"is_fully_paid":           remainingBalanceAfter == 0,
		"payment_date":            time.Now().Format(time.RFC3339),
	}

	eventPayloadJSON, err := json.Marshal(eventPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	eventReq := &models.CreateEventOutboxRequest{
		EventType:    "payment.received",
		EventKey:     req.OrderID,
		EventPayload: eventPayloadJSON,
		Topic:        "offline-orders-audit",
	}

	if err := s.eventPublisher.CreateEvent(ctx, tx, eventReq); err != nil {
		log.Error().Err(err).Str("order_id", req.OrderID).Msg("Failed to create payment event")
		// Don't fail the payment if audit event fails
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "transaction_commit_failed"))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// T113: Record successful operation in span
	span.SetAttributes(
		attribute.String("payment_record_id", paymentRecordID),
		attribute.Int("payment_number", paymentNumber),
		attribute.Int("remaining_balance", remainingBalanceAfter),
		attribute.Bool("fully_paid", remainingBalanceAfter == 0),
	)

	// T112: Record payment metric
	observability.OfflineOrderPaymentsTotal.WithLabelValues(req.TenantID, string(req.PaymentMethod)).Inc()

	log.Info().
		Str("order_id", req.OrderID).
		Str("payment_record_id", paymentRecordID).
		Int("amount_paid", req.AmountPaid).
		Int("remaining_balance", remainingBalanceAfter).
		Bool("fully_paid", remainingBalanceAfter == 0).
		Msg("Payment recorded successfully")

	// Return the created payment record
	paymentRecord := &models.PaymentRecord{
		ID:                    paymentRecordID,
		OrderID:               req.OrderID,
		PaymentTermsID:        paymentTermsID,
		PaymentNumber:         paymentNumber,
		AmountPaid:            req.AmountPaid,
		PaymentDate:           time.Now(),
		PaymentMethod:         req.PaymentMethod,
		RemainingBalanceAfter: remainingBalanceAfter,
		RecordedByUserID:      req.RecordedByUserID,
		Notes:                 req.Notes,
		ReceiptNumber:         req.ReceiptNumber,
		CreatedAt:             time.Now(),
	}

	return paymentRecord, nil
}

// GetPaymentHistoryForOrder retrieves payment history for an order
func (s *OfflineOrderService) GetPaymentHistoryForOrder(ctx context.Context, orderID string, tenantID string) ([]models.PaymentRecord, error) {
	// Validate order access
	if err := s.ValidateOrderAccess(ctx, orderID, tenantID); err != nil {
		return nil, err
	}

	return s.paymentRepo.GetPaymentHistory(ctx, orderID)
}

// GetPaymentTermsForOrder retrieves payment terms for an order
func (s *OfflineOrderService) GetPaymentTermsForOrder(ctx context.Context, orderID string, tenantID string) (*models.PaymentTerms, error) {
	// Validate order access
	if err := s.ValidateOrderAccess(ctx, orderID, tenantID); err != nil {
		return nil, err
	}

	return s.paymentRepo.GetPaymentTerms(ctx, orderID)
}

// ============================================================================
// User Story 3: Edit Offline Orders with Audit Trail
// ============================================================================

// UpdateOfflineOrder updates an existing offline order with validation
// T076: Implement UpdateOfflineOrder with validation
// T077: Implement change detection and diff generation
// T078: Implement status constraint check (no edit if PAID)
// T079: Integrate event publishing for offline_order.updated
func (s *OfflineOrderService) UpdateOfflineOrder(ctx context.Context, req *UpdateOfflineOrderRequest) (*models.GuestOrder, error) {
	// T113: Start trace span
	ctx, span := s.tracer.Start(ctx, "UpdateOfflineOrder",
		trace.WithAttributes(
			attribute.String("order_id", req.OrderID),
			attribute.String("tenant_id", req.TenantID),
			attribute.String("modified_by", req.ModifiedByUserID),
		),
	)
	defer span.End()
	
	// T078: Check status constraint - cannot edit orders that are PAID or later
	existingOrder, err := s.offlineOrderRepo.GetOfflineOrderByID(ctx, req.OrderID, req.TenantID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "order_not_found"))
		return nil, fmt.Errorf("failed to get existing order: %w", err)
	}

	// Prevent editing paid/completed/cancelled orders
	if existingOrder.Status != models.OrderStatusPending {
		err := fmt.Errorf("cannot edit order with status %s", existingOrder.Status)
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("error.type", "invalid_status"),
			attribute.String("current_status", string(existingOrder.Status)),
		)
		return nil, fmt.Errorf("cannot edit order with status %s (only PENDING orders can be edited)", existingOrder.Status)
	}

	// Begin transaction for atomic update
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// T077: Generate change diff for audit trail
	changes := s.detectChanges(existingOrder, req)
	if len(changes) == 0 && len(req.ModelUpdates.Items) == 0 {
		return existingOrder, nil // No changes to apply
	}

	// Update order fields if provided
	if len(changes) > 0 {
		err = s.offlineOrderRepo.UpdateOfflineOrder(ctx, tx, req.OrderID, req.TenantID, &req.ModelUpdates, req.ModifiedByUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to update order: %w", err)
		}
	}

	// Update order items if provided
	if len(req.ModelUpdates.Items) > 0 {
		subtotal, total, err := s.offlineOrderRepo.UpdateOrderItems(ctx, tx, req.OrderID, req.TenantID, req.ModelUpdates.Items)
		if err != nil {
			return nil, fmt.Errorf("failed to update order items: %w", err)
		}
		
		// Add totals to change log
		changes["subtotal_amount"] = map[string]interface{}{
			"old": existingOrder.SubtotalAmount,
			"new": subtotal,
		}
		changes["total_amount"] = map[string]interface{}{
			"old": existingOrder.TotalAmount,
			"new": total,
		}
	}

	// T079: Publish offline_order.updated event with change details
	changeDetailsJSON, _ := json.Marshal(changes)
	eventPayload, _ := json.Marshal(map[string]interface{}{
		"order_id":           req.OrderID,
		"tenant_id":          req.TenantID,
		"modified_by_user_id": req.ModifiedByUserID,
		"changes":            string(changeDetailsJSON),
		"modified_at":        time.Now().Unix(),
	})

	err = s.eventPublisher.CreateEvent(ctx, tx, &models.CreateEventOutboxRequest{
		EventType:    "offline_order.updated",
		EventKey:     req.OrderID,
		EventPayload: eventPayload,
		Topic:        "order-events",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to publish update event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "transaction_commit_failed"))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// T113: Record successful operation in span
	span.SetAttributes(
		attribute.Int("change_count", len(changes)),
	)

	// T112: Record update metric
	observability.OfflineOrderUpdatesTotal.WithLabelValues(req.TenantID).Inc()

	log.Info().
		Str("order_id", req.OrderID).
		Str("tenant_id", req.TenantID).
		Str("modified_by", req.ModifiedByUserID).
		Int("change_count", len(changes)).
		Msg("Offline order updated successfully")

	// Return updated order
	return s.offlineOrderRepo.GetOfflineOrderByID(ctx, req.OrderID, req.TenantID)
}

// detectChanges compares existing order with update request and generates change diff
// T077: Implement change detection and diff generation
func (s *OfflineOrderService) detectChanges(existing *models.GuestOrder, req *UpdateOfflineOrderRequest) map[string]interface{} {
	changes := make(map[string]interface{})

	if req.ModelUpdates.CustomerName != nil && *req.ModelUpdates.CustomerName != existing.CustomerName {
		changes["customer_name"] = map[string]interface{}{
			"old": existing.CustomerName,
			"new": *req.ModelUpdates.CustomerName,
		}
	}

	if req.ModelUpdates.CustomerPhone != nil && *req.ModelUpdates.CustomerPhone != existing.CustomerPhone {
		changes["customer_phone"] = map[string]interface{}{
			"old": existing.CustomerPhone,
			"new": *req.ModelUpdates.CustomerPhone,
		}
	}

	if req.ModelUpdates.CustomerEmail != nil {
		oldEmail := ""
		if existing.CustomerEmail != nil {
			oldEmail = *existing.CustomerEmail
		}
		newEmail := *req.ModelUpdates.CustomerEmail
		if oldEmail != newEmail {
			changes["customer_email"] = map[string]interface{}{
				"old": oldEmail,
				"new": newEmail,
			}
		}
	}

	if req.ModelUpdates.DeliveryType != nil && *req.ModelUpdates.DeliveryType != existing.DeliveryType {
		changes["delivery_type"] = map[string]interface{}{
			"old": existing.DeliveryType,
			"new": *req.ModelUpdates.DeliveryType,
		}
	}

	if req.ModelUpdates.TableNumber != nil {
		oldTable := ""
		if existing.TableNumber != nil {
			oldTable = *existing.TableNumber
		}
		newTable := *req.ModelUpdates.TableNumber
		if oldTable != newTable {
			changes["table_number"] = map[string]interface{}{
				"old": oldTable,
				"new": newTable,
			}
		}
	}

	if req.ModelUpdates.Notes != nil {
		oldNotes := ""
		if existing.Notes != nil {
			oldNotes = *existing.Notes
		}
		newNotes := *req.ModelUpdates.Notes
		if oldNotes != newNotes {
			changes["notes"] = map[string]interface{}{
				"old": oldNotes,
				"new": newNotes,
			}
		}
	}

	if req.ModelUpdates.DeliveryFee != nil && *req.ModelUpdates.DeliveryFee != existing.DeliveryFee {
		changes["delivery_fee"] = map[string]interface{}{
			"old": existing.DeliveryFee,
			"new": *req.ModelUpdates.DeliveryFee,
		}
	}

	return changes
}

// UpdateOfflineOrderRequest represents a request to update an offline order
// US3: Edit offline orders with audit trail
type UpdateOfflineOrderRequest struct {
	OrderID          string                         `json:"order_id" validate:"required,uuid"`
	TenantID         string                         `json:"tenant_id" validate:"required,uuid"`
	ModifiedByUserID string                         `json:"modified_by_user_id" validate:"required,uuid"`
	ModelUpdates     models.UpdateOfflineOrderRequest // Actual field updates
}

// DeleteOfflineOrderRequest represents a request to delete an offline order
// T092-T093: Service layer for US4 - Role-Based Deletion
type DeleteOfflineOrderRequest struct {
	OrderID         string `json:"order_id" validate:"required,uuid"`
	TenantID        string `json:"tenant_id" validate:"required,uuid"`
	DeletedByUserID string `json:"deleted_by_user_id" validate:"required,uuid"`
	UserRole        string `json:"user_role"` // T112: User role for metrics
	Reason          string `json:"reason" validate:"required,min=5,max=500"` // Deletion reason for audit
}

// DeleteOfflineOrder soft-deletes an offline order with role validation and audit trail
// T092-T093: Service implementation for US4 - Role-Based Deletion
// Only owner and manager roles can delete orders
// Only PENDING and CANCELLED orders can be deleted (not PAID or COMPLETE)
// Publishes offline_order.deleted event to audit trail
// T113: Adds OpenTelemetry tracing spans
func (s *OfflineOrderService) DeleteOfflineOrder(ctx context.Context, req *DeleteOfflineOrderRequest) error {
	// T113: Start trace span
	ctx, span := s.tracer.Start(ctx, "DeleteOfflineOrder",
		trace.WithAttributes(
			attribute.String("order_id", req.OrderID),
			attribute.String("tenant_id", req.TenantID),
			attribute.String("deleted_by", req.DeletedByUserID),
			attribute.String("user_role", req.UserRole),
		),
	)
	defer span.End()
	
	// Note: Role validation is handled by RequireRole middleware in the handler layer
	// This service assumes the caller has already been authorized

	// Begin transaction to ensure atomic deletion + event creation
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "transaction_begin_failed"))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Perform soft delete in repository
	err = s.offlineOrderRepo.SoftDeleteOfflineOrder(ctx, req.OrderID, req.TenantID, req.DeletedByUserID)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "soft_delete_failed"))
		log.Error().
			Err(err).
			Str("order_id", req.OrderID).
			Str("tenant_id", req.TenantID).
			Str("deleted_by_user_id", req.DeletedByUserID).
			Msg("Failed to soft delete offline order")
		return fmt.Errorf("failed to delete offline order: %w", err)
	}

	// Publish deletion event to audit trail (T093)
	eventPayload := map[string]interface{}{
		"order_id":           req.OrderID,
		"tenant_id":          req.TenantID,
		"deleted_by_user_id": req.DeletedByUserID,
		"deleted_at":         time.Now().Format(time.RFC3339),
		"reason":             req.Reason,
		"event_type":         "offline_order.deleted",
	}

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", req.OrderID).
			Msg("Failed to marshal deletion event payload")
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Create event in outbox using EventPublisher
	eventReq := &models.CreateEventOutboxRequest{
		EventType:    "offline_order.deleted",
		EventKey:     req.OrderID,
		EventPayload: payloadBytes,
		Topic:        "order-events",
	}

	err = s.eventPublisher.CreateEvent(ctx, tx, eventReq)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", req.OrderID).
			Msg("Failed to publish deletion event to outbox")
		return fmt.Errorf("failed to publish deletion event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", "transaction_commit_failed"))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// T113: Record successful operation in span
	span.SetAttributes(
		attribute.String("deletion_reason", req.Reason),
	)

	// T112: Record deletion metric
	userRole := req.UserRole
	if userRole == "" {
		userRole = "unknown"
	}
	observability.OfflineOrderDeletionsTotal.WithLabelValues(req.TenantID, userRole).Inc()

	log.Info().
		Str("order_id", req.OrderID).
		Str("tenant_id", req.TenantID).
		Str("deleted_by_user_id", req.DeletedByUserID).
		Str("reason", req.Reason).
		Msg("Offline order deleted successfully")

	return nil
}
