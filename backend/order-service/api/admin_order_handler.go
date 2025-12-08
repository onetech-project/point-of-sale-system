package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/services"
)

// AdminOrderHandler handles admin order management operations
type AdminOrderHandler struct {
	orderService *services.OrderService
}

// NewAdminOrderHandler creates a new admin order handler
func NewAdminOrderHandler(orderService *services.OrderService) *AdminOrderHandler {
	return &AdminOrderHandler{
		orderService: orderService,
	}
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
	orders, err := h.orderService.ListOrdersByTenant(ctx, tenantID, statusFilter, limit, offset)
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

	// Get tenant ID from JWT claims for authorization
	tenantID := c.QueryParam("tenant_id")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get order
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

	return c.JSON(http.StatusOK, order)
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
}
