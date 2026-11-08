package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/services"
)

type orderService interface {
	ListOrdersByTenant(ctx context.Context, tenantID string, status *models.OrderStatus, orderType models.OrderTypeFilter, limit, offset int) ([]*models.GuestOrder, error)
	GetOrderByID(ctx context.Context, orderID string) (*models.GuestOrder, error)
	GetOrderItems(ctx context.Context, orderID string) ([]models.OrderItem, error)
	GetOrderNotes(ctx context.Context, orderID string) ([]*models.OrderNote, error)
	UpdateOrderStatus(ctx context.Context, orderID string, newStatus models.OrderStatus) error
	AddOrderNote(ctx context.Context, orderID, note, userName string) error
}

// paymentDetailsService exposes an order's payment and refunds to admins.
type paymentDetailsService interface {
	GetPaymentDetails(ctx context.Context, order *models.GuestOrder) (*services.PaymentSummary, []models.PaymentRefund, error)
	RefundOrderPayment(ctx context.Context, order *models.GuestOrder, req services.RefundRequest) (*models.PaymentRefund, error)
}

// AdminOrderHandler handles admin order management operations
type AdminOrderHandler struct {
	orderService   orderService
	paymentService paymentDetailsService
}

// NewAdminOrderHandler creates a new admin order handler
func NewAdminOrderHandler(orderService orderService) *AdminOrderHandler {
	return &AdminOrderHandler{
		orderService: orderService,
	}
}

// WithPaymentService enables payment details and refunds.
func (h *AdminOrderHandler) WithPaymentService(paymentService paymentDetailsService) *AdminOrderHandler {
	h.paymentService = paymentService
	return h
}

// ListOrders handles GET /admin/orders
// Implements T090, T092, T093: List orders with tenant scoping and status filtering
func (h *AdminOrderHandler) ListOrders(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from header (T092: tenant-scoped filtering)
	// API Gateway injects X-Tenant-ID from authenticated session
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get status filter (T093: status filter query parameter)
	statusParam := c.QueryParam("status")
	var statusFilter *models.OrderStatus
	if statusParam != "" {
		status := models.OrderStatus(statusParam)
		// Validate status
		validStatuses := map[models.OrderStatus]bool{
			models.OrderStatusPending:   true,
			models.OrderStatusPaid:      true,
			models.OrderStatusComplete:  true,
			models.OrderStatusCancelled: true,
		}
		if !validStatuses[status] {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid status. Must be: PENDING, PAID, COMPLETE, or CANCELLED",
			})
		}
		statusFilter = &status
	}

	orderTypeFilter := models.OrderTypeFilter(c.QueryParam("order_type"))
	if orderTypeFilter == "" {
		orderTypeFilter = models.OrderTypeFilterOnline
	}
	validOrderTypes := map[models.OrderTypeFilter]bool{
		models.OrderTypeFilterAll:     true,
		models.OrderTypeFilterOnline:  true,
		models.OrderTypeFilterOffline: true,
	}
	if !validOrderTypes[orderTypeFilter] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid order_type. Must be: all, online, or offline",
		})
	}

	// Pagination
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20 // Default limit
	}

	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if offset < 0 {
		offset = 0
	}

	// Get orders
	orders, err := h.orderService.ListOrdersByTenant(ctx, tenantID, statusFilter, orderTypeFilter, limit, offset)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID).
			Msg("Failed to list orders")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve orders",
		})
	}

	// Fetch items and latest note for each order
	ordersWithItems := make([]map[string]interface{}, 0, len(orders))
	for _, order := range orders {
		items, err := h.orderService.GetOrderItems(ctx, order.ID)
		if err != nil {
			log.Warn().Err(err).Str("order_id", order.ID).Msg("Failed to fetch order items")
			items = []models.OrderItem{} // Empty array on error
		}

		// Get latest note only
		notes, notesErr := h.orderService.GetOrderNotes(ctx, order.ID)
		var latestNote *models.OrderNote
		if notesErr == nil && len(notes) > 0 {
			latestNote = notes[0] // Already sorted by created_at DESC
		}

		ordersWithItems = append(ordersWithItems, map[string]interface{}{
			"order":       order,
			"items":       items,
			"latest_note": latestNote,
		})
	}

	log.Info().
		Str("tenant_id", tenantID).
		Str("order_type", string(orderTypeFilter)).
		Int("count", len(orders)).
		Msg("Orders retrieved successfully")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"orders": ordersWithItems,
		"pagination": map[string]int{
			"limit":  limit,
			"offset": offset,
			"count":  len(orders),
		},
	})
}

// GetOrder handles GET /admin/orders/:id
// Implements T090: Get order details
func (h *AdminOrderHandler) GetOrder(c echo.Context) error {
	ctx := c.Request().Context()
	orderID := c.Param("id")

	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "order_id is required",
		})
	}

	// API Gateway injects X-Tenant-ID from the authenticated session.
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get order
	order, err := h.orderService.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Order not found",
			})
		}

		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to get order")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve order",
		})
	}

	if order == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Order not found",
		})
	}

	// Verify tenant ownership (T092: tenant-scoped filtering)
	if order.TenantID != tenantID {
		log.Warn().
			Str("order_id", orderID).
			Str("order_tenant_id", order.TenantID).
			Str("requested_tenant_id", tenantID).
			Msg("Unauthorized access attempt to order from different tenant")
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	items, err := h.orderService.GetOrderItems(ctx, order.ID)
	if err != nil {
		log.Warn().Err(err).Str("order_id", order.ID).Msg("Failed to fetch order items")
		items = []models.OrderItem{}
	}

	notes, notesErr := h.orderService.GetOrderNotes(ctx, order.ID)
	var latestNote *models.OrderNote
	if notesErr == nil && len(notes) > 0 {
		latestNote = notes[0]
	}

	response := map[string]interface{}{
		"order":       order,
		"items":       items,
		"latest_note": latestNote,
	}
	if h.paymentService != nil {
		payment, refunds, err := h.paymentService.GetPaymentDetails(ctx, order)
		if err != nil {
			log.Warn().Err(err).Str("order_id", order.ID).Msg("Failed to fetch payment details")
		} else {
			response["payment"] = payment
			response["refunds"] = refunds
		}
	}

	return c.JSON(http.StatusOK, response)
}

// RefundOrderRequest is the body of a refund request.
type RefundOrderRequest struct {
	Reason string `json:"reason"`
}

// RefundOrder handles POST /admin/orders/:id/refund. It fully refunds the
// order's QRIS payment. The API gateway restricts it to owners; the role is
// checked again here in case the service is reached directly.
func (h *AdminOrderHandler) RefundOrder(c echo.Context) error {
	ctx := c.Request().Context()
	orderID := c.Param("id")
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if orderID == "" || tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "order_id and tenant_id are required"})
	}
	if !strings.EqualFold(c.Request().Header.Get("X-User-Role"), "owner") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Only owners can refund payments"})
	}
	if h.paymentService == nil {
		return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Refunds are not available"})
	}

	var req RefundOrderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" || len([]rune(req.Reason)) > 256 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "reason is required (max 256 characters)"})
	}

	order, err := h.orderService.GetOrderByID(ctx, orderID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Error().Err(err).Str("order_id", orderID).Msg("Failed to get order for refund")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve order"})
	}
	if order == nil || order.TenantID != tenantID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
	}

	userName := c.Request().Header.Get("X-User-Name")
	if userName == "" {
		userName = c.Request().Header.Get("X-User-Email")
	}

	refund, err := h.paymentService.RefundOrderPayment(ctx, order, services.RefundRequest{
		OrderID:  order.ID,
		TenantID: tenantID,
		Reason:   req.Reason,
		UserID:   c.Request().Header.Get("X-User-ID"),
		UserName: userName,
	})

	var gatewayErr *services.RefundGatewayError
	switch {
	case errors.Is(err, services.ErrRefundInProgress):
		return c.JSON(http.StatusConflict, map[string]string{"error": "This payment already has a refund in progress or completed"})
	case errors.Is(err, services.ErrRefundUnsupported):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Refunds are only available for payments made through Manjo"})
	case errors.Is(err, services.ErrRefundNotAllowed):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Only paid orders with a settled payment can be refunded"})
	case errors.As(err, &gatewayErr):
		return c.JSON(http.StatusBadGateway, map[string]interface{}{"error": gatewayErr.Message, "refund": refund})
	case err != nil:
		log.Error().Err(err).Str("order_id", orderID).Msg("Failed to refund order payment")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to refund payment"})
	}

	if refund.Status == models.RefundStatusProcessing {
		return c.JSON(http.StatusAccepted, map[string]interface{}{
			"message": "Refund is being processed by the payment gateway",
			"refund":  refund,
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Payment refunded",
		"refund":  refund,
	})
}

// UpdateOrderStatusRequest represents the request to update order status
type UpdateOrderStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=PENDING PAID COMPLETE CANCELLED"`
}

// UpdateOrderStatus handles PATCH /admin/orders/:id/status
// Implements T090: Update order status with validation
func (h *AdminOrderHandler) UpdateOrderStatus(c echo.Context) error {
	ctx := c.Request().Context()
	orderID := c.Param("id")

	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "order_id is required",
		})
	}

	// Get tenant ID from header (API Gateway injects from session)
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Parse request
	var req UpdateOrderStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Verify tenant ownership
	order, err := h.orderService.GetOrderByID(ctx, orderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to get order")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve order",
		})
	}

	if order == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Order not found",
		})
	}

	if order.TenantID != tenantID {
		log.Warn().
			Str("order_id", orderID).
			Str("order_tenant_id", order.TenantID).
			Str("requested_tenant_id", tenantID).
			Msg("Unauthorized status update attempt")
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	// Update status with validation
	newStatus := models.OrderStatus(req.Status)
	err = h.orderService.UpdateOrderStatus(ctx, orderID, newStatus)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("new_status", req.Status).
			Msg("Failed to update order status")

		// Check if it's a validation error
		if err.Error() == "order not found" {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Order not found",
			})
		}

		if err.Error()[:27] == "invalid status transition" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update order status",
		})
	}

	log.Info().
		Str("order_id", orderID).
		Str("order_reference", order.OrderReference).
		Str("new_status", req.Status).
		Msg("Order status updated by admin")

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Order status updated successfully",
		"status":  req.Status,
	})
}

// AddOrderNoteRequest represents the request to add a note to an order
type AddOrderNoteRequest struct {
	Note string `json:"note" validate:"required,min=1,max=1000"`
}

// AddOrderNote handles POST /admin/orders/:id/notes
// Implements T090: Add notes/comments for courier tracking
func (h *AdminOrderHandler) AddOrderNote(c echo.Context) error {
	ctx := c.Request().Context()
	orderID := c.Param("id")

	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "order_id is required",
		})
	}

	// Get tenant ID from header (API Gateway injects from session)
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Parse request
	var req AddOrderNoteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Verify tenant ownership
	order, err := h.orderService.GetOrderByID(ctx, orderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to get order")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve order",
		})
	}

	if order == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Order not found",
		})
	}

	if order.TenantID != tenantID {
		log.Warn().
			Str("order_id", orderID).
			Str("order_tenant_id", order.TenantID).
			Str("requested_tenant_id", tenantID).
			Msg("Unauthorized note addition attempt")
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Access denied",
		})
	}

	// Get user name from header (set by API Gateway from JWT email)
	userName := c.Request().Header.Get("X-User-Name")
	if userName == "" {
		// Fallback to email if name not available
		userName = c.Request().Header.Get("X-User-Email")
	}

	// Add note
	err = h.orderService.AddOrderNote(ctx, orderID, req.Note, userName)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to add order note")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to add note",
		})
	}

	log.Info().
		Str("order_id", orderID).
		Str("order_reference", order.OrderReference).
		Msg("Note added to order by admin")

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Note added successfully",
	})
}

// RegisterRoutes registers admin order routes
// Implements T091: JWT authentication middleware will be added to these routes
func (h *AdminOrderHandler) RegisterRoutes(e *echo.Echo) {
	// Admin routes - will require JWT authentication middleware
	// TODO: Add JWT middleware here once auth-service integration is complete
	admin := e.Group("/api/v1/admin/orders")
	// admin.Use(middleware.JWTAuth()) // To be implemented

	admin.GET("", h.ListOrders)
	admin.GET("/:id", h.GetOrder)
	admin.PATCH("/:id/status", h.UpdateOrderStatus)
	admin.POST("/:id/notes", h.AddOrderNote)
	admin.POST("/:id/refund", h.RefundOrder)
}
