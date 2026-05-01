package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/services"
)

// OfflineOrderHandler handles offline order management operations
// Implements User Story 1: Record Basic Offline Order (MVP)
// Implements User Story 2: Manage Payment Terms and Installments
type OfflineOrderHandler struct {
	offlineOrderService *services.OfflineOrderService
}

// NewOfflineOrderHandler creates a new offline order handler
func NewOfflineOrderHandler(offlineOrderService *services.OfflineOrderService) *OfflineOrderHandler {
	return &OfflineOrderHandler{
		offlineOrderService: offlineOrderService,
	}
}

// CreateOfflineOrder handles POST /offline-orders
// Implements T036: Create offline order with request validation
func (h *OfflineOrderHandler) CreateOfflineOrder(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from header (injected by API Gateway)
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get user ID from header (injected by API Gateway from JWT)
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "user authentication is required",
		})
	}

	// Parse request body
	var req services.CreateOfflineOrderRequest
	if err := c.Bind(&req); err != nil {
		log.Warn().Err(err).Msg("Failed to bind request body")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Override tenant_id and recorded_by_user_id from headers
	req.TenantID = tenantID
	req.RecordedByUserID = userID

	// Create offline order
	order, err := h.offlineOrderService.CreateOfflineOrder(ctx, &req)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID).
			Str("user_id", userID).
			Msg("Failed to create offline order")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create offline order",
		})
	}

	log.Info().
		Str("order_id", order.ID).
		Str("order_reference", order.OrderReference).
		Str("tenant_id", tenantID).
		Msg("Offline order created successfully")

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"order": order,
	})
}

// ListOfflineOrders handles GET /offline-orders
// Implements T037: List offline orders with query filters
func (h *OfflineOrderHandler) ListOfflineOrders(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from header
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Parse query parameters
	filters := services.ListOfflineOrdersFilters{
		Status:      c.QueryParam("status"),      // Optional: filter by status
		SearchQuery: c.QueryParam("search"),      // Optional: search by order_reference
		Limit:       20,                          // Default limit
		Offset:      0,                           // Default offset
	}

	// Parse pagination
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		limit, err := strconv.Atoi(limitParam)
		if err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		offset, err := strconv.Atoi(offsetParam)
		if err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Validate status if provided
	if filters.Status != "" {
		validStatuses := map[string]bool{
			"PENDING":   true,
			"PAID":      true,
			"COMPLETE":  true,
			"CANCELLED": true,
		}
		if !validStatuses[filters.Status] {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid status. Must be: PENDING, PAID, COMPLETE, or CANCELLED",
			})
		}
	}

	// Get offline orders
	result, err := h.offlineOrderService.ListOfflineOrders(ctx, tenantID, filters)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID).
			Msg("Failed to list offline orders")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve offline orders",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"orders":      result.Orders,
		"total_count": result.TotalCount,
		"page":        result.Page,
		"page_size":   result.PageSize,
	})
}

// GetOfflineOrderByID handles GET /offline-orders/:id
// Implements T038: Get single offline order by ID
func (h *OfflineOrderHandler) GetOfflineOrderByID(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from header
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get order ID from path parameter
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "order_id is required",
		})
	}

	// Get offline order
	order, err := h.offlineOrderService.GetOfflineOrderByID(ctx, orderID, tenantID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("tenant_id", tenantID).
			Msg("Failed to get offline order")
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Offline order not found or access denied",
		})
	}

	// Get order items (T038 extension: include items in detail response)
	items, err := h.offlineOrderService.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		log.Warn().
			Err(err).
			Str("order_id", orderID).
			Msg("Failed to get order items, returning order without items")
		items = []models.OrderItem{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"order": order,
		"items": items,
	})
}

// ============================================================================
// User Story 2: Payment Management Handlers (T064, T065)
// ============================================================================

// RecordPayment handles POST /offline-orders/:id/payments
// T064: Implement POST handler for recording payments
func (h *OfflineOrderHandler) RecordPayment(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from header
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get user ID from header
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "user authentication is required",
		})
	}

	// Get order ID from path parameter
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "order_id is required",
		})
	}

	// Parse request body
	var req services.RecordPaymentRequest
	if err := c.Bind(&req); err != nil {
		log.Warn().Err(err).Msg("Failed to bind request body")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Override fields from path and headers
	req.OrderID = orderID
	req.TenantID = tenantID
	req.RecordedByUserID = userID

	// Record payment
	paymentRecord, err := h.offlineOrderService.RecordPayment(ctx, &req)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("tenant_id", tenantID).
			Msg("Failed to record payment")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	log.Info().
		Str("order_id", orderID).
		Str("payment_record_id", paymentRecord.ID).
		Int("amount_paid", paymentRecord.AmountPaid).
		Bool("fully_paid", paymentRecord.RemainingBalanceAfter == 0).
		Msg("Payment recorded successfully")

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"payment": paymentRecord,
	})
}

// GetPaymentHistory handles GET /offline-orders/:id/payments
// T065: Implement GET handler for payment history
func (h *OfflineOrderHandler) GetPaymentHistory(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from header
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get order ID from path parameter
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "order_id is required",
		})
	}

	// Get payment history
	payments, err := h.offlineOrderService.GetPaymentHistoryForOrder(ctx, orderID, tenantID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("tenant_id", tenantID).
			Msg("Failed to get payment history")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve payment history",
		})
	}

	// Get payment terms (if exists)
	paymentTerms, err := h.offlineOrderService.GetPaymentTermsForOrder(ctx, orderID, tenantID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("tenant_id", tenantID).
			Msg("Failed to get payment terms")
		// Don't fail, just return payments without terms
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"payments":      payments,
		"payment_terms": paymentTerms,
	})
}

// ============================================================================
// User Story 3: Edit Offline Orders with Audit Trail
// ============================================================================

// UpdateOfflineOrder handles PATCH /offline-orders/:id
// T080: Implement PATCH /offline-orders/{id} handler
// T081: Add request validation for update operations
func (h *OfflineOrderHandler) UpdateOfflineOrder(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from header
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get user ID from header
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "user authentication is required",
		})
	}

	// Get order ID from path parameter
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "order_id is required",
		})
	}

	// Parse request body
	var modelUpdates models.UpdateOfflineOrderRequest
	if err := c.Bind(&modelUpdates); err != nil {
		log.Warn().Err(err).Msg("Failed to bind request body")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// T081: Validate update request
	if modelUpdates.CustomerName != nil && len(*modelUpdates.CustomerName) < 2 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "customer_name must be at least 2 characters",
		})
	}

	if modelUpdates.CustomerPhone != nil && len(*modelUpdates.CustomerPhone) < 10 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "customer_phone must be at least 10 characters",
		})
	}

	if modelUpdates.CustomerEmail != nil && *modelUpdates.CustomerEmail != "" {
		// Basic email validation
		if !strings.Contains(*modelUpdates.CustomerEmail, "@") {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "customer_email must be a valid email address",
			})
		}
	}

	// Build service request
	req := &services.UpdateOfflineOrderRequest{
		OrderID:          orderID,
		TenantID:         tenantID,
		ModifiedByUserID: userID,
		ModelUpdates:     modelUpdates,
	}

	// Call service to update order
	updatedOrder, err := h.offlineOrderService.UpdateOfflineOrder(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("tenant_id", tenantID).
			Str("user_id", userID).
			Msg("Failed to update offline order")
		
		// Check for specific error types
		if strings.Contains(err.Error(), "cannot edit order with status") {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": err.Error(),
			})
		}
		
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update offline order",
		})
	}

	log.Info().
		Str("order_id", orderID).
		Str("tenant_id", tenantID).
		Msg("Offline order updated successfully")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"order": updatedOrder,
	})
}

// DeleteOfflineOrder handles DELETE /offline-orders/:id
// T094: Delete offline order with reason (US4 - Role-Based Deletion)
// Only owners and managers can delete orders (enforced by RequireRole middleware)
// Only PENDING and CANCELLED orders can be deleted
func (h *OfflineOrderHandler) DeleteOfflineOrder(c echo.Context) error {
	ctx := c.Request().Context()

	// Get tenant ID from header
	tenantID := c.Request().Header.Get("X-Tenant-ID")
	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "tenant_id is required",
		})
	}

	// Get user ID from header
	userID := c.Request().Header.Get("X-User-ID")
	if userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "user authentication is required",
		})
	}

	// T112: Get user role from header for metrics
	userRole := c.Request().Header.Get("X-User-Role")

	// Get order ID from URL
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "order_id is required",
		})
	}

	// Get deletion reason from query parameter (required)
	reason := c.QueryParam("reason")
	if reason == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "deletion reason is required (query parameter: ?reason=...)",
		})
	}

	// Validate reason length
	if len(reason) < 5 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "deletion reason must be at least 5 characters",
		})
	}
	if len(reason) > 500 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "deletion reason must not exceed 500 characters",
		})
	}

	// Call service to delete order
	req := &services.DeleteOfflineOrderRequest{
		OrderID:         orderID,
		TenantID:        tenantID,
		DeletedByUserID: userID,
		UserRole:        userRole, // T112: Pass user role for metrics
		Reason:          reason,
	}

	err := h.offlineOrderService.DeleteOfflineOrder(ctx, req)
	if err != nil {
		// Check for specific error types
		if strings.Contains(err.Error(), "not found") {
			log.Warn().
				Err(err).
				Str("order_id", orderID).
				Str("tenant_id", tenantID).
				Str("user_id", userID).
				Msg("Offline order not found during delete")
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "offline order not found",
			})
		}
		if strings.Contains(err.Error(), "cannot delete orders with status") {
			log.Warn().
				Err(err).
				Str("order_id", orderID).
				Str("tenant_id", tenantID).
				Str("user_id", userID).
				Msg("Offline order delete blocked by status")
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": err.Error(),
			})
		}
		if strings.Contains(err.Error(), "already deleted") {
			// DELETE should be idempotent: repeated deletes return success.
			log.Info().
				Str("order_id", orderID).
				Str("tenant_id", tenantID).
				Str("user_id", userID).
				Msg("Offline order already deleted")
			return c.JSON(http.StatusOK, map[string]interface{}{
				"message": "offline order already deleted",
				"order_id": orderID,
			})
		}

		log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("tenant_id", tenantID).
			Str("user_id", userID).
			Msg("Failed to delete offline order")

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete offline order",
		})
	}

	log.Info().
		Str("order_id", orderID).
		Str("tenant_id", tenantID).
		Str("deleted_by_user_id", userID).
		Msg("Offline order deleted successfully")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "offline order deleted successfully",
		"order_id": orderID,
	})
}

// ============================================================================
// Route Registration
// ============================================================================

// RegisterOfflineOrderRoutes registers HTTP routes for offline orders
// Implements T039: Register routes with JWT middleware
// T063: Extended to support installment payment in POST /offline-orders
// T064-T065: Payment management routes
// T080: Edit offline order route
// T095: Delete route with role-based access control (US4)
// T110: Rate limiting middleware applied to all offline order routes
func RegisterOfflineOrderRoutes(e *echo.Echo, handler *OfflineOrderHandler, jwtMiddleware echo.MiddlewareFunc, requireRoleMiddleware func(...string) echo.MiddlewareFunc, rateLimitMiddleware echo.MiddlewareFunc) {
	// Offline order routes (all require authentication and rate limiting)
	// T110: Apply rate limiting to prevent abuse of offline order operations
	offlineOrders := e.Group("/api/v1/admin/offline-orders", jwtMiddleware, rateLimitMiddleware)
	
	// US1: Basic offline order operations
	offlineOrders.POST("", handler.CreateOfflineOrder)           // T063: Create new offline order (supports installment)
	offlineOrders.GET("", handler.ListOfflineOrders)             // List offline orders with filters
	offlineOrders.GET("/:id", handler.GetOfflineOrderByID)       // Get single offline order
	
	// US2: Payment management
	offlineOrders.POST("/:id/payments", handler.RecordPayment)    // T064: Record a payment
	offlineOrders.GET("/:id/payments", handler.GetPaymentHistory) // T065: Get payment history
	
	// US3: Edit offline orders
	offlineOrders.PATCH("/:id", handler.UpdateOfflineOrder)       // T080: Update offline order
	
	// US4: Delete offline orders (owner and manager only)
	// T095: Apply RequireRole middleware to DELETE route
	offlineOrders.DELETE("/:id", handler.DeleteOfflineOrder, requireRoleMiddleware("owner", "manager"))
	
	log.Info().Msg("Offline order routes registered successfully with rate limiting")
}