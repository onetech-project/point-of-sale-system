package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/point-of-sale-system/order-service/src/events"
	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/point-of-sale-system/order-service/src/services"
	"github.com/point-of-sale-system/order-service/src/utils"
	"github.com/point-of-sale-system/order-service/src/validators"
)

type CheckoutHandler struct {
	db                 *sql.DB
	redisClient        *redis.Client
	cartService        *services.CartService
	inventoryService   *services.InventoryService
	paymentService     *services.PaymentService
	geocodingService   *services.GeocodingService
	deliveryFeeService *services.DeliveryFeeService
	addressRepo        *repository.AddressRepository
	settingsRepo       *repository.OrderSettingsRepository
	guestOrderRepo     *repository.GuestOrderRepository
	kafkaProducer      interface { // Interface for Kafka producer
		Publish(ctx context.Context, key string, value interface{}) error
	}
	consentProducer interface {
		Publish(ctx context.Context, key string, value interface{}) error
	}
}

func NewCheckoutHandler(
	db *sql.DB,
	redisClient *redis.Client,
	cartService *services.CartService,
	inventoryService *services.InventoryService,
	paymentService *services.PaymentService,
	geocodingService *services.GeocodingService,
	deliveryFeeService *services.DeliveryFeeService,
	addressRepo *repository.AddressRepository,
	settingsRepo *repository.OrderSettingsRepository,
	guestOrderRepo *repository.GuestOrderRepository,
	kafkaProducer interface {
		Publish(ctx context.Context, key string, value interface{}) error
	},
	consentProducer interface {
		Publish(ctx context.Context, key string, value interface{}) error
	},
) *CheckoutHandler {
	return &CheckoutHandler{
		db:                 db,
		redisClient:        redisClient,
		cartService:        cartService,
		inventoryService:   inventoryService,
		paymentService:     paymentService,
		geocodingService:   geocodingService,
		deliveryFeeService: deliveryFeeService,
		addressRepo:        addressRepo,
		settingsRepo:       settingsRepo,
		guestOrderRepo:     guestOrderRepo,
		kafkaProducer:      kafkaProducer,
		consentProducer:    consentProducer,
	}
}

type CheckoutRequest struct {
	DeliveryType    string   `json:"delivery_type"`
	CustomerName    string   `json:"customer_name"`
	CustomerPhone   string   `json:"customer_phone"`
	CustomerEmail   *string  `json:"customer_email,omitempty"`
	DeliveryAddress *string  `json:"delivery_address,omitempty"`
	TableNumber     *string  `json:"table_number,omitempty"`
	Notes           *string  `json:"notes,omitempty"`
	Consents        []string `json:"consents"` // Optional consents granted (required consents implicit)
}

type CheckoutResponse struct {
	OrderReference string    `json:"order_reference"`
	OrderID        string    `json:"order_id"`
	Status         string    `json:"status"`
	Total          int64     `json:"total"`
	DeliveryType   string    `json:"delivery_type"`
	PaymentURL     *string   `json:"payment_url,omitempty"`
	PaymentToken   *string   `json:"payment_token,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateOrder handles POST /public/checkout/:tenant_id
func (h *CheckoutHandler) CreateOrder(c echo.Context) error {
	ctx := context.Background()
	tenantID := c.Param("tenantId")
	sessionID := c.Request().Header.Get("X-Session-Id")

	if tenantID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "tenant_id is required",
		})
	}

	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "X-Session-Id header is required",
		})
	}

	// Parse request
	var req CheckoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	// Validate delivery type
	validDeliveryTypes := map[string]bool{
		"pickup":   true,
		"delivery": true,
		"dine_in":  true,
	}
	if !validDeliveryTypes[req.DeliveryType] {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid_delivery_type",
			"message": "Invalid delivery type. Must be: pickup, delivery, or dine_in",
		})
	}

	// Validate optional consent codes (required consents are implicit)
	if err := validators.ValidateGuestConsents(req.Consents); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "invalid_consent",
			"message": fmt.Sprintf("Invalid consent codes: %v", err),
		})
	}

	// Validate against tenant config
	isEnabled, err := h.validateDeliveryTypeWithTenant(ctx, tenantID, req.DeliveryType)
	if err != nil {
		log.Error().Err(err).
			Str("tenant_id", tenantID).
			Str("delivery_type", req.DeliveryType).
			Msg("Failed to validate delivery type")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "validation_failed",
			"message": "Failed to validate delivery type",
		})
	}

	if !isEnabled {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "delivery_type_disabled",
			"message": fmt.Sprintf("Delivery type '%s' is not enabled for this merchant", req.DeliveryType),
		})
	}

	// Validate contact information
	if err := h.validateContactInfo(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "validation_failed",
			"message": err.Error(),
		})
	}

	// Validate conditional fields based on delivery type
	if err := h.validateConditionalFields(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "validation_failed",
			"message": err.Error(),
		})
	}

	// Get cart from Redis
	cart, err := h.getCartFromRedis(ctx, tenantID, sessionID)
	if err != nil {
		log.Error().Err(err).
			Str("tenant_id", tenantID).
			Str("session_id", sessionID).
			Msg("Failed to retrieve cart")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "cart_not_found",
			"message": "Cart not found or expired",
		})
	}

	if len(cart.Items) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "empty_cart",
			"message": "Cart is empty",
		})
	}

	// Begin transaction
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to begin transaction")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create order",
		})
	}
	defer tx.Rollback()

	// Check inventory availability with row-level locks to prevent race conditions
	if err := h.inventoryService.CheckAvailabilityWithLock(ctx, tx, tenantID, cart.Items); err != nil {
		log.Error().Err(err).
			Str("tenant_id", tenantID).
			Str("session_id", sessionID).
			Msg("Inventory check failed")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":   "insufficient stock",
			"message": err.Error(),
		})
	}

	// Generate order reference
	orderReference, err := utils.GenerateOrderReference()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate order reference")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create order",
		})
	}

	// Get order settings for delivery fee
	settings, err := h.settingsRepo.GetOrCreate(ctx, tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to get order settings")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create order",
		})
	}

	// Calculate delivery fee based on delivery type and settings
	// Only charge delivery fee if enabled in settings and delivery type is delivery
	deliveryFee := 0
	if settings.ChargeDeliveryFee && strings.ToLower(req.DeliveryType) == "delivery" {
		deliveryFee = settings.DefaultDeliveryFee
		log.Info().
			Str("tenant_id", tenantID).
			Int("delivery_fee", deliveryFee).
			Msg("Applying delivery fee from settings")
	} else if !settings.ChargeDeliveryFee && strings.ToLower(req.DeliveryType) == "delivery" {
		log.Info().
			Str("tenant_id", tenantID).
			Msg("Delivery fee collection disabled - tenant handles fees externally")
	}

	// Create order
	order := &models.GuestOrder{
		TenantID:       tenantID,
		SessionID:      sessionID,
		OrderReference: orderReference,
		Status:         models.OrderStatusPending,
		DeliveryType:   models.DeliveryType(req.DeliveryType),
		CustomerName:   req.CustomerName,
		CustomerPhone:  req.CustomerPhone,
		CustomerEmail:  req.CustomerEmail,
		TableNumber:    req.TableNumber,
		Notes:          req.Notes,
		SubtotalAmount: cart.GetTotal(),
		DeliveryFee:    deliveryFee,
		TotalAmount:    cart.GetTotal() + deliveryFee,
	}

	// Insert order
	orderID, err := h.insertOrder(ctx, tx, order)
	if err != nil {
		log.Error().Err(err).
			Str("order_reference", orderReference).
			Msg("Failed to insert order")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create order",
		})
	}

	// Insert order items
	for _, item := range cart.Items {
		orderItem := &models.OrderItem{
			OrderID:     orderID,
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			UnitPrice:   item.UnitPrice,
			Quantity:    item.Quantity,
			TotalPrice:  item.TotalPrice,
		}

		if err := h.insertOrderItem(ctx, tx, orderItem); err != nil {
			log.Error().Err(err).
				Str("order_id", orderID).
				Str("product_id", item.ProductID).
				Msg("Failed to insert order item")
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create order",
			})
		}
	}

	// Create inventory reservations with 15min TTL
	if err := h.inventoryService.CreateReservations(ctx, tx, orderID, cart.Items); err != nil {
		log.Error().Err(err).
			Str("order_id", orderID).
			Str("order_reference", orderReference).
			Msg("Failed to create inventory reservations")
		// Order created but reservations failed - this is logged but not returned as error
		// Cleanup job will eventually free any partial reservations
	}

	// Clear cart from Redis
	if err := h.clearCart(ctx, tenantID, sessionID); err != nil {
		log.Warn().Err(err).
			Str("tenant_id", tenantID).
			Str("session_id", sessionID).
			Msg("Failed to clear cart after order creation")
	}

	// Create QRIS charge and get QR code URL (T066)
	var paymentURL *string

	// Update order with ID for payment service
	order.ID = orderID
	order.CreatedAt = time.Now()
	// TotalAmount already set correctly with delivery fee

	qrisResp, err := h.paymentService.CreateQRISCharge(ctx, order, cart.Items)
	if err != nil {
		log.Error().Err(err).
			Str("order_id", orderID).
			Str("order_reference", orderReference).
			Msg("Failed to create QRIS charge")
		// Return error - payment is required to proceed
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create payment",
		})
	}

	// Save QRIS payment info to database
	if err := h.paymentService.SaveQRISPaymentInfo(ctx, tx, orderID, order.TotalAmount, qrisResp); err != nil {
		log.Error().Err(err).
			Str("order_id", orderID).
			Str("transaction_id", qrisResp.TransactionID).
			Msg("Failed to save QRIS payment info")
		// Continue - payment was created, info will be saved via webhook
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("Failed to commit transaction")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create order",
		})
	}

	// Get QR code URL from actions array
	if len(qrisResp.Actions) > 0 {
		paymentURL = &qrisResp.Actions[0].URL
	}

	log.Info().
		Str("order_reference", orderReference).
		Str("tenant_id", tenantID).
		Str("delivery_type", req.DeliveryType).
		Int64("total", int64(order.TotalAmount)).
		Int("delivery_fee", deliveryFee).
		Str("transaction_id", qrisResp.TransactionID).
		Str("qr_code_url", *paymentURL).
		Msg("Order created successfully with QRIS payment")

	// Publish invoice notification event if customer provided email
	if req.CustomerEmail != nil && *req.CustomerEmail != "" {
		h.publishInvoiceEvent(ctx, orderID, orderReference, tenantID, order, cart.Items, req.CustomerEmail)
	}

	// Publish ConsentGrantedEvent to Kafka (async, after transaction committed)
	// This ensures we have the real order_id and prevents consent recording failures from blocking checkout
	// Uses dedicated consent-events topic for audit-service consumption
	if h.consentProducer != nil {
		go func() {
			consentEvent := events.ConsentGrantedEvent{
				EventID:          uuid.New().String(),
				EventType:        "consent.granted",
				TenantID:         tenantID,
				SubjectType:      "guest",
				SubjectID:        orderID, // Real order_id from database
				ConsentMethod:    "checkout",
				PolicyVersion:    "1.0.0", // TODO: Get from database
				Consents:         req.Consents, // Only optional consents provided by user
				RequiredConsents: validators.GetRequiredGuestConsents(), // Required consents (implicit)
				Metadata: events.ConsentMetadata{
					IPAddress: c.RealIP(),
					UserAgent: c.Request().UserAgent(),
					SessionID: &sessionID,
					RequestID: c.Response().Header().Get("X-Request-ID"),
				},
				Timestamp: time.Now(),
			}

			if err := h.consentProducer.Publish(context.Background(), tenantID, consentEvent); err != nil {
				log.Error().Err(err).
					Str("order_id", orderID).
					Str("tenant_id", tenantID).
					Msg("Failed to publish consent event")
				// TODO: Add to retry queue or alert for manual intervention
			}
		}()
	}

	return c.JSON(http.StatusCreated, CheckoutResponse{
		OrderReference: orderReference,
		OrderID:        orderID,
		Status:         "PENDING",
		Total:          int64(order.TotalAmount),
		DeliveryType:   req.DeliveryType,
		PaymentURL:     paymentURL,
		PaymentToken:   nil, // Not used for QRIS
		CreatedAt:      order.CreatedAt,
	})
}

func (h *CheckoutHandler) validateContactInfo(req *CheckoutRequest) error {
	// Validate name
	name := strings.TrimSpace(req.CustomerName)
	if name == "" {
		return fmt.Errorf("customer name is required")
	}
	if len(name) < 2 {
		return fmt.Errorf("customer name must be at least 2 characters")
	}
	if len(name) > 100 {
		return fmt.Errorf("customer name is too long (max 100 characters)")
	}

	// Validate phone number (Indonesian format)
	phone := strings.TrimSpace(req.CustomerPhone)
	if phone == "" {
		return fmt.Errorf("customer phone is required")
	}

	// Remove spaces and dashes
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	// Indonesian phone format: +62, 62, or 0 followed by 9-12 digits
	phoneRegex := regexp.MustCompile(`^(\+62|62|0)[0-9]{9,12}$`)
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid phone number format. Expected format: 081234567890")
	}

	return nil
}

func (h *CheckoutHandler) validateConditionalFields(req *CheckoutRequest) error {
	switch req.DeliveryType {
	case "delivery":
		// Delivery address is required for delivery orders
		if req.DeliveryAddress == nil || strings.TrimSpace(*req.DeliveryAddress) == "" {
			return fmt.Errorf("delivery address is required for delivery orders")
		}
		if len(strings.TrimSpace(*req.DeliveryAddress)) < 10 {
			return fmt.Errorf("please provide a complete delivery address")
		}

	case "dine_in":
	// Table number is optional for dine-in
	// No validation needed

	case "pickup":
		// No additional fields required for pickup
		// No validation needed
	}

	return nil
}

func (h *CheckoutHandler) validateDeliveryTypeWithTenant(ctx context.Context, tenantID, deliveryType string) (bool, error) {
	// Call tenant-service to get tenant config
	// For now, return true (will be implemented when integrating with tenant-service)
	// TODO: Make HTTP call to tenant-service /public/tenants/:tenant_id/config
	return true, nil
}

func (h *CheckoutHandler) getCartFromRedis(ctx context.Context, tenantID, sessionID string) (*models.Cart, error) {
	key := fmt.Sprintf("cart:%s:%s", tenantID, sessionID)
	data, err := h.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var cart models.Cart
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		return nil, err
	}

	// Validate and adjust cart items based on current stock availability
	if err := h.cartService.ValidateAndAdjustCart(ctx, &cart); err != nil {
		return nil, fmt.Errorf("failed to validate cart: %w", err)
	}

	return &cart, nil
}

func (h *CheckoutHandler) insertOrder(ctx context.Context, tx *sql.Tx, order *models.GuestOrder) (string, error) {
	return h.guestOrderRepo.Create(ctx, tx, order)
}

func (h *CheckoutHandler) insertOrderItem(ctx context.Context, tx *sql.Tx, item *models.OrderItem) error {
	query := `
		INSERT INTO order_items (
			order_id, product_id, product_name, quantity, unit_price, total_price
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		item.OrderID,
		item.ProductID,
		item.ProductName,
		item.Quantity,
		item.UnitPrice,
		item.TotalPrice,
	)

	return err
}

func (h *CheckoutHandler) clearCart(ctx context.Context, tenantID, sessionID string) error {
	key := fmt.Sprintf("cart:%s:%s", tenantID, sessionID)
	return h.redisClient.Del(ctx, key).Err()
}

// processDeliveryAddressAndFee handles geocoding and delivery fee calculation for delivery orders
// Implements T080-T083: Geocode address, validate service area, calculate delivery fee
func (h *CheckoutHandler) processDeliveryAddressAndFee(
	ctx context.Context,
	tx *sql.Tx,
	orderID string,
	tenantID string,
	deliveryAddress string,
	serviceArea *models.ServiceArea,
	deliveryFeeConfig *services.DeliveryFeeConfig,
) (int, error) {
	// T080: Geocode the delivery address
	geocodingResult, err := h.geocodingService.GeocodeAddress(ctx, deliveryAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to geocode address: %w", err)
	}

	// T081: Validate service area
	isWithinArea, distance, err := h.geocodingService.ValidateServiceArea(
		ctx,
		geocodingResult.Latitude,
		geocodingResult.Longitude,
		serviceArea,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to validate service area: %w", err)
	}

	if !isWithinArea {
		return 0, fmt.Errorf("delivery address is outside service area")
	}

	// T082: Calculate delivery fee
	var deliveryFee int
	var zoneID *string

	// Determine zone ID if using zone-based pricing
	if serviceArea.Type == "polygon" {
		// For polygon areas, we could map coordinates to zone IDs
		// For now, we'll use nil and let the fee service use distance to centroid
		zoneID = nil
	}

	if deliveryFeeConfig != nil {
		deliveryFee, err = h.deliveryFeeService.CalculateFee(ctx, distance, zoneID, deliveryFeeConfig)
		if err != nil {
			return 0, fmt.Errorf("failed to calculate delivery fee: %w", err)
		}
	} else {
		// No delivery fee config means free delivery or manual fee setting
		deliveryFee = 0
	}

	// T083: Create delivery_address record with geocoded coordinates and calculated fee
	deliveryAddressRecord := &models.DeliveryAddress{
		ID:                   generateUUID(), // Helper function needed
		OrderID:              orderID,
		TenantID:             tenantID,
		FullAddress:          geocodingResult.FormattedAddress,
		Latitude:             geocodingResult.Latitude,
		Longitude:            geocodingResult.Longitude,
		GeocodingResult:      &geocodingResult.FormattedAddress,
		ServiceAreaValidated: isWithinArea,
		CalculatedFee:        deliveryFee,
		DistanceKm:           &distance,
		ZoneID:               zoneID,
	}

	// Save delivery address to database
	if err := h.addressRepo.Create(ctx, deliveryAddressRecord); err != nil {
		return 0, fmt.Errorf("failed to create delivery address record: %w", err)
	}

	return deliveryFee, nil
}

// GetPublicOrder handles GET /public/orders/:orderReference
// Public endpoint for guests to check their order status
func (h *CheckoutHandler) GetPublicOrder(c echo.Context) error {
	ctx := c.Request().Context()
	orderReference := c.Param("orderReference")

	if orderReference == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "order_reference is required",
		})
	}

	// Get order from database
	orderRepo, err := repository.NewOrderRepositoryWithVault(h.db)
	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize OrderRepository")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "internal server error",
		})
	}
	order, err := orderRepo.GetOrderByReference(ctx, orderReference)
	if err != nil {
		log.Error().Err(err).Str("order_reference", orderReference).Msg("Failed to fetch order")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch order",
		})
	}

	if order == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "order not found",
		})
	}

	// Get payment transaction info for QR code display
	paymentRepo := repository.NewPaymentRepository(h.db)
	payment, err := paymentRepo.GetPaymentByOrderID(ctx, order.ID)
	if err != nil {
		log.Error().Err(err).Str("order_id", order.ID).Msg("Failed to fetch payment info")
		// Continue without payment info - not critical
	}

	// Get order items
	items, itemsErr := orderRepo.GetOrderItemsByOrderID(ctx, order.ID)
	if itemsErr != nil {
		log.Warn().Err(itemsErr).Str("order_id", order.ID).Msg("Failed to fetch order items")
		items = []models.OrderItem{} // Empty array on error
	}

	// Get latest order note only
	notes, notesErr := orderRepo.GetOrderNotesByOrderID(ctx, order.ID)
	if notesErr != nil {
		log.Warn().Err(notesErr).Str("order_id", order.ID).Msg("Failed to fetch order notes")
		notes = []*models.OrderNote{} // Empty array on error
	} else if len(notes) > 0 {
		// Only keep the latest note
		notes = notes[:1]
	}

	// Build response with order and payment info
	response := map[string]interface{}{
		"order": order,
		"items": items,
		"notes": notes,
	}

	if payment != nil {
		now := time.Now()
		log.Debug().Str("server_time", now.Format(time.RFC3339)).Msg("Current server time for payment expiry calculation")

		var expiryTimeStr *string
		remainingTime := int64(0)
		if payment.ExpiryTime != nil {
			expiryTime := payment.ExpiryTime
			log.Debug().Str("expiry_time", expiryTime.Format(time.RFC3339)).Msg("the payment expiry time for payment expiry calculation")

			remainingTime = max(int64(expiryTime.Add(-7*time.Hour).Sub(now).Seconds()), 0)
			log.Debug().Str("remaining_time", fmt.Sprintf("%d", remainingTime)).Msg("the remaining time for payment expiry")

			// Format as RFC3339 to ensure timezone is preserved
			formatted := payment.ExpiryTime.Format(time.RFC3339)
			expiryTimeStr = &formatted
		}

		response["payment"] = map[string]interface{}{
			"transaction_id":     payment.MidtransTransactionID,
			"transaction_status": payment.TransactionStatus,
			"qr_code_url":        payment.QRCodeURL,
			"expiry_time":        expiryTimeStr,
			"server_time":        now.Format(time.RFC3339),
			"remaining_time":     remainingTime,
			"payment_type":       payment.PaymentType,
		}
	}

	return c.JSON(http.StatusOK, response)
}

// getTenantDeliveryConfig fetches service area and delivery fee configuration from tenant service
// This is a placeholder that should be replaced with actual tenant-service API call
func (h *CheckoutHandler) getTenantDeliveryConfig(ctx context.Context, tenantID string) (*models.ServiceArea, *services.DeliveryFeeConfig, error) {
	// TODO: Implement actual HTTP call to tenant-service
	// GET /api/v1/tenants/:tenant_id/delivery-config
	// For now, return nil to indicate no automatic delivery fee calculation
	return nil, nil, nil
}

// generateUUID generates a UUID for delivery address
// This should use a proper UUID library like github.com/google/uuid
func generateUUID() string {
	// Placeholder - should use proper UUID generation
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// publishInvoiceEvent publishes an invoice notification event to Kafka
func (h *CheckoutHandler) publishInvoiceEvent(
	ctx context.Context,
	orderID string,
	orderReference string,
	tenantID string,
	order *models.GuestOrder,
	items []models.CartItem,
	customerEmail *string,
) {
	if h.kafkaProducer == nil {
		log.Warn().Msg("Kafka producer not initialized, skipping invoice notification")
		return
	}

	// Prepare order items for email
	orderItems := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		orderItems = append(orderItems, map[string]interface{}{
			"product_name": item.ProductName,
			"quantity":     item.Quantity,
			"unit_price":   item.UnitPrice,
			"total_price":  item.TotalPrice,
		})
	}

	// Create event payload
	event := map[string]interface{}{
		"event_type": "order.invoice",
		"tenant_id":  tenantID,
		"user_id":    "", // Empty for guest orders
		"data": map[string]interface{}{
			"order_id":        orderID,
			"order_reference": orderReference,
			"customer_name":   order.CustomerName,
			"customer_email":  *customerEmail,
			"delivery_type":   order.DeliveryType,
			"subtotal_amount": order.SubtotalAmount,
			"delivery_fee":    order.DeliveryFee,
			"total_amount":    order.TotalAmount,
			"items":           orderItems,
			"created_at":      order.CreatedAt.Format(time.RFC3339),
		},
	}

	// Publish to Kafka
	if err := h.kafkaProducer.Publish(ctx, orderReference, event); err != nil {
		log.Error().Err(err).
			Str("order_reference", orderReference).
			Str("customer_email", *customerEmail).
			Msg("Failed to publish invoice notification event")
	} else {
		log.Info().
			Str("order_reference", orderReference).
			Str("customer_email", *customerEmail).
			Msg("Invoice notification event published successfully")
	}
}
