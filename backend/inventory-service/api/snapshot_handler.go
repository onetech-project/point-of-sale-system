package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pos/backend/inventory-service/src/services"
	"github.com/pos/backend/inventory-service/src/utils"
)

type SnapshotHandler struct {
	service *services.SnapshotService
}

func NewSnapshotHandler(service *services.SnapshotService) *SnapshotHandler {
	return &SnapshotHandler{service: service}
}

func (h *SnapshotHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/inventory/orders/:orderId/cost-snapshots", h.ListByOrder)
	e.GET("/inventory/orders/:orderId/profitability", h.Profitability)
}

func (h *SnapshotHandler) ListByOrder(c echo.Context) error {
	tenantID, orderID, ok := snapshotContext(c)
	if !ok {
		return nil
	}
	snapshots, err := h.service.ListByOrder(c.Request().Context(), tenantID, orderID)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"snapshots": snapshots})
}

func (h *SnapshotHandler) Profitability(c echo.Context) error {
	tenantID, orderID, ok := snapshotContext(c)
	if !ok {
		return nil
	}
	profitability, err := h.service.Profitability(c.Request().Context(), tenantID, orderID)
	if err != nil {
		return respondServiceError(c, err)
	}
	return c.JSON(http.StatusOK, profitability)
}

func snapshotContext(c echo.Context) (uuid.UUID, uuid.UUID, bool) {
	tenantID, err := utils.GetTenantIDFromContext(c)
	if err != nil {
		_ = utils.RespondUnauthorized(c, "Tenant ID not found")
		return uuid.Nil, uuid.Nil, false
	}
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		_ = utils.RespondBadRequest(c, "Invalid order ID")
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, orderID, true
}
