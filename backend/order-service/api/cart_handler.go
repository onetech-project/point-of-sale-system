package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/point-of-sale-system/order-service/src/config"
	"github.com/point-of-sale-system/order-service/src/repository"
	"github.com/point-of-sale-system/order-service/src/services"
)

type CartHandler struct {
	cartService *services.CartService
}

func NewCartHandler() *CartHandler {
	ttl := time.Duration(config.GetEnvAsInt("CART_SESSION_TTL", 86400)) * time.Second
	cartRepo := repository.NewCartRepository(config.GetRedis(), ttl)
	reservationRepo := repository.NewReservationRepository(config.GetDB())
	cartService := services.NewCartService(cartRepo, reservationRepo, config.GetDB())

	return &CartHandler{
		cartService: cartService,
	}
}

// NewCartHandlerWithService creates a new CartHandler with an existing CartService
func NewCartHandlerWithService(cartService *services.CartService) *CartHandler {
	return &CartHandler{
		cartService: cartService,
	}
}

func (h *CartHandler) GetCart(c echo.Context) error {
	tenantID := c.Param("tenantId")
	sessionID := c.Request().Header.Get("X-Session-Id")

	if sessionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Session-Id header required")
	}

	// GetCart now automatically validates and adjusts cart items
	cart, err := h.cartService.GetCart(c.Request().Context(), tenantID, sessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get cart")
	}

	return c.JSON(http.StatusOK, cart)
}

type AddItemRequest struct {
	ProductID   string `json:"product_id" validate:"required"`
	ProductName string `json:"product_name" validate:"required"`
	Quantity    int    `json:"quantity" validate:"required,min=1"`
	UnitPrice   int    `json:"unit_price" validate:"required,min=0"`
}

func (h *CartHandler) AddItem(c echo.Context) error {
	tenantID := c.Param("tenantId")
	sessionID := c.Request().Header.Get("X-Session-Id")

	if sessionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Session-Id header required")
	}

	var req AddItemRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	cart, err := h.cartService.AddItem(
		c.Request().Context(),
		tenantID,
		sessionID,
		req.ProductID,
		req.ProductName,
		req.Quantity,
		req.UnitPrice,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{
			"error":   "failed to add item to cart",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, cart)
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity" validate:"required,min=0"`
}

func (h *CartHandler) UpdateItem(c echo.Context) error {
	tenantID := c.Param("tenantId")
	productID := c.Param("productId")
	sessionID := c.Request().Header.Get("X-Session-Id")

	if sessionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Session-Id header required")
	}

	var req UpdateItemRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	cart, err := h.cartService.UpdateItem(
		c.Request().Context(),
		tenantID,
		sessionID,
		productID,
		req.Quantity,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, map[string]string{
			"error":   "failed to update cart item",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, cart)
}

func (h *CartHandler) RemoveItem(c echo.Context) error {
	tenantID := c.Param("tenantId")
	productID := c.Param("productId")
	sessionID := c.Request().Header.Get("X-Session-Id")

	if sessionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Session-Id header required")
	}

	cart, err := h.cartService.RemoveItem(
		c.Request().Context(),
		tenantID,
		sessionID,
		productID,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to remove cart item")
	}

	return c.JSON(http.StatusOK, cart)
}

func (h *CartHandler) ClearCart(c echo.Context) error {
	tenantID := c.Param("tenantId")
	sessionID := c.Request().Header.Get("X-Session-Id")

	if sessionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Session-Id header required")
	}

	if err := h.cartService.ClearCart(c.Request().Context(), tenantID, sessionID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to clear cart")
	}

	return c.NoContent(http.StatusNoContent)
}
