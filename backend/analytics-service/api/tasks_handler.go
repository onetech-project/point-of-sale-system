package api

import (
	"net/http"

	"github.com/pos/analytics-service/src/middleware"
	"github.com/pos/analytics-service/src/models"
	"github.com/pos/analytics-service/src/repository"

	"github.com/labstack/echo/v4"
)

// TasksHandler handles operational task endpoints (delayed orders, low stock)
type TasksHandler struct {
	taskRepo *repository.TaskRepository
}

// NewTasksHandler creates a new tasks handler instance
func NewTasksHandler(taskRepo *repository.TaskRepository) *TasksHandler {
	return &TasksHandler{
		taskRepo: taskRepo,
	}
}

// GetOperationalTasks returns delayed orders and low stock alerts
// GET /api/v1/analytics/tasks
func (h *TasksHandler) GetOperationalTasks(c echo.Context) error {
	ctx := c.Request().Context()

	// Extract tenant ID from context (set by auth middleware)
	tenantID := middleware.GetTenantID(c)
	if tenantID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Fetch delayed orders and low stock alerts in parallel
	type delayedResult struct {
		orders []models.DelayedOrder
		err    error
	}

	type lowStockResult struct {
		alerts []models.RestockAlert
		err    error
	}

	delayedChan := make(chan delayedResult, 1)
	lowStockChan := make(chan lowStockResult, 1)

	// Fetch delayed orders
	go func() {
		orders, err := h.taskRepo.GetDelayedOrders(ctx, tenantID)
		delayedChan <- delayedResult{orders: orders, err: err}
	}()

	// Fetch low stock alerts
	go func() {
		alerts, err := h.taskRepo.GetLowStockProducts(ctx, tenantID)
		lowStockChan <- lowStockResult{alerts: alerts, err: err}
	}()

	// Wait for both results
	delayedRes := <-delayedChan
	lowStockRes := <-lowStockChan

	if delayedRes.err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch delayed orders: "+delayedRes.err.Error())
	}

	if lowStockRes.err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch low stock alerts: "+lowStockRes.err.Error())
	}

	// Build response with counts
	var delayedOrdersResp models.DelayedOrdersResponse
	delayedOrdersResp.DelayedOrders = delayedRes.orders
	delayedOrdersResp.Count = len(delayedRes.orders)

	for _, order := range delayedRes.orders {
		if order.IsUrgent() {
			delayedOrdersResp.UrgentCount++
		} else if order.IsWarning() {
			delayedOrdersResp.WarningCount++
		}
	}

	var restockAlertsResp models.RestockAlertsResponse
	restockAlertsResp.RestockAlerts = lowStockRes.alerts
	restockAlertsResp.Count = len(lowStockRes.alerts)

	for _, alert := range lowStockRes.alerts {
		if alert.IsCritical() {
			restockAlertsResp.CriticalCount++
		} else if alert.IsLowStock() {
			restockAlertsResp.LowStockCount++
		}
	}

	// Return combined response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"delayed_orders": delayedOrdersResp,
		"restock_alerts": restockAlertsResp,
	})
}
