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

func (e *TenantStatusEnforcer) enforce(c echo.Context, tenantID string, allowed map[string]bool, next echo.HandlerFunc) error {
	status := e.fetchTenantStatus(c, tenantID)
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

func (e *TenantStatusEnforcer) fetchTenantStatus(c echo.Context, tenantID string) string {
	if e == nil || e.tenantURL == "" {
		return "active"
	}
	resp, err := e.httpClient.Get(fmt.Sprintf("%s/internal/tenants/%s/status", e.tenantURL, tenantID)) //nolint:noctx
	if err != nil {
		c.Logger().Warnf("tenant-service unreachable for tenant %s, failing open: %v", tenantID, err)
		return "active"
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "deleted"
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Logger().Warnf("tenant-service status read error for tenant %s, failing open: %v", tenantID, err)
		return "active"
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		c.Logger().Warnf("tenant-service status parse error for tenant %s, failing open: %v", tenantID, err)
		return "active"
	}
	if result.Status == "" {
		return "active"
	}
	return result.Status
}
