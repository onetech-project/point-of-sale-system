package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type TenantStatusEnforcer struct {
	tenantURL  string
	httpClient *http.Client
}

type TenantAvailability struct {
	TenantID           string `json:"tenant_id"`
	Status             string `json:"status"`
	SubscriptionStatus string `json:"subscription_status"`
}

func NewTenantStatusEnforcer(tenantURL string) *TenantStatusEnforcer {
	return &TenantStatusEnforcer{
		tenantURL:  tenantURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

func (e *TenantStatusEnforcer) EnforceTenantAccount() echo.MiddlewareFunc {
	return e.AllowTenantStatuses("active")
}

func (e *TenantStatusEnforcer) AllowTenantStatuses(statuses ...string) echo.MiddlewareFunc {
	allowed := map[string]bool{}
	for _, status := range statuses {
		allowed[status] = true
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID, _ := c.Get("tenant_id").(string)
			if tenantID == "" {
				return next(c)
			}
			return e.enforce(c, tenantID, allowed, next)
		}
	}
}

func (e *TenantStatusEnforcer) RequireActiveTenantParam(paramName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Param(paramName)
			if tenantID == "" {
				return next(c)
			}
			return e.enforce(c, tenantID, map[string]bool{"active": true}, next)
		}
	}
}

func (e *TenantStatusEnforcer) RequirePublicTenantAvailableParam(paramName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Param(paramName)
			if tenantID == "" {
				return next(c)
			}
			availability := e.fetchTenantAvailability(c, tenantID)
			if availability.Status == "" {
				availability.Status = "active"
			}
			if availability.SubscriptionStatus == "" {
				availability.SubscriptionStatus = "active"
			}
			if availability.Status == "active" && (availability.SubscriptionStatus == "trial" || availability.SubscriptionStatus == "active") {
				return next(c)
			}
			return c.JSON(http.StatusForbidden, map[string]string{
				"error":               "Tenant currently unavailable",
				"message":             "This tenant is currently not available at this moment.",
				"status":              availability.Status,
				"subscription_status": availability.SubscriptionStatus,
			})
		}
	}
}

func (e *TenantStatusEnforcer) enforce(c echo.Context, tenantID string, allowed map[string]bool, next echo.HandlerFunc) error {
	status := e.fetchTenantAvailability(c, tenantID).Status
	if status == "" {
		status = "active"
	}
	if allowed[status] {
		return next(c)
	}
	switch status {
	case "suspended":
		return c.JSON(http.StatusForbidden, map[string]string{
			"error":  "Tenant account suspended",
			"status": status,
		})
	case "inactive":
		return c.JSON(http.StatusForbidden, map[string]string{
			"error":  "Tenant account deactivated",
			"status": status,
		})
	default:
		return c.JSON(http.StatusForbidden, map[string]string{
			"error":  "Tenant account unavailable",
			"status": status,
		})
	}
}

func (e *TenantStatusEnforcer) fetchTenantAvailability(c echo.Context, tenantID string) TenantAvailability {
	fallback := TenantAvailability{
		TenantID:           tenantID,
		Status:             "active",
		SubscriptionStatus: "active",
	}
	if e == nil || e.tenantURL == "" {
		return fallback
	}
	resp, err := e.httpClient.Get(fmt.Sprintf("%s/internal/tenants/%s/status", e.tenantURL, tenantID)) //nolint:noctx
	if err != nil {
		c.Logger().Warnf("tenant-service unreachable for tenant %s, failing open: %v", tenantID, err)
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return TenantAvailability{TenantID: tenantID, Status: "deleted", SubscriptionStatus: "active"}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Logger().Warnf("tenant-service status read error for tenant %s, failing open: %v", tenantID, err)
		return fallback
	}
	var result TenantAvailability
	if err := json.Unmarshal(body, &result); err != nil {
		c.Logger().Warnf("tenant-service status parse error for tenant %s, failing open: %v", tenantID, err)
		return fallback
	}
	result.TenantID = tenantID
	if result.Status == "" {
		result.Status = "active"
	}
	if result.SubscriptionStatus == "" {
		result.SubscriptionStatus = "active"
	}
	return result
}
