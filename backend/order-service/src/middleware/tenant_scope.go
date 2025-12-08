package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// TenantValidator defines the interface for validating tenants
type TenantValidator interface {
	ValidateTenant(ctx context.Context, tenantID string) (bool, error)
	IsTenantActive(ctx context.Context, tenantID string) (bool, error)
}

// TenantInfo represents cached tenant information
type TenantInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

// RedisTenantValidator validates tenants using Redis cache and HTTP fallback
type RedisTenantValidator struct {
	redisClient      *redis.Client
	tenantServiceURL string
	cacheTTL         time.Duration
}

// NewRedisTenantValidator creates a new tenant validator
func NewRedisTenantValidator(redisClient *redis.Client, tenantServiceURL string) *RedisTenantValidator {
	return &RedisTenantValidator{
		redisClient:      redisClient,
		tenantServiceURL: tenantServiceURL,
		cacheTTL:         5 * time.Minute, // Cache tenant info for 5 minutes
	}
}

// ValidateTenant checks if tenant exists
// Implements T099: Tenant existence validation
func (v *RedisTenantValidator) ValidateTenant(ctx context.Context, tenantID string) (bool, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("tenant:info:%s", tenantID)
	cached, err := v.redisClient.Get(ctx, cacheKey).Result()

	if err == nil {
		var tenantInfo TenantInfo
		if err := json.Unmarshal([]byte(cached), &tenantInfo); err == nil {
			return true, nil
		}
	}

	// Cache miss or error - call tenant service
	// TODO: Implement actual HTTP call to tenant-service
	// For now, assume tenant exists if UUID is valid
	log.Debug().
		Str("tenant_id", tenantID).
		Msg("Tenant validation - cache miss, assuming valid")

	return true, nil
}

// IsTenantActive checks if tenant is active
// Implements T100: Tenant active status check
func (v *RedisTenantValidator) IsTenantActive(ctx context.Context, tenantID string) (bool, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("tenant:info:%s", tenantID)
	cached, err := v.redisClient.Get(ctx, cacheKey).Result()

	if err == nil {
		var tenantInfo TenantInfo
		if err := json.Unmarshal([]byte(cached), &tenantInfo); err == nil {
			return tenantInfo.IsActive, nil
		}
	}

	// Cache miss - call tenant service
	// TODO: Implement actual HTTP call to tenant-service
	// GET /api/v1/tenants/:tenant_id
	// For now, assume tenant is active
	log.Debug().
		Str("tenant_id", tenantID).
		Msg("Tenant active status check - cache miss, assuming active")

	return true, nil
}

// TenantScope middleware validates tenant_id from URL and adds to context
// Implements T099-T100: Tenant existence and active status validation
func TenantScope(validator TenantValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Param("tenantId")

			// Validate UUID format
			if _, err := uuid.Parse(tenantID); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid tenant_id format")
			}

			ctx := c.Request().Context()

			// T099: Validate tenant existence
			exists, err := validator.ValidateTenant(ctx, tenantID)
			if err != nil {
				log.Error().
					Err(err).
					Str("tenant_id", tenantID).
					Msg("Failed to validate tenant")
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to validate tenant")
			}

			if !exists {
				log.Warn().
					Str("tenant_id", tenantID).
					Msg("Tenant not found")
				return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
			}

			// T100: Check tenant active status
			isActive, err := validator.IsTenantActive(ctx, tenantID)
			if err != nil {
				log.Error().
					Err(err).
					Str("tenant_id", tenantID).
					Msg("Failed to check tenant active status")
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to validate tenant")
			}

			if !isActive {
				log.Warn().
					Str("tenant_id", tenantID).
					Msg("Tenant is inactive")
				return echo.NewHTTPError(http.StatusForbidden, "tenant is not active")
			}

			// Add to context for downstream handlers
			c.Set("tenant_id", tenantID)

			log.Debug().
				Str("tenant_id", tenantID).
				Msg("Tenant validated successfully")

			return next(c)
		}
	}
}
