package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/utils"
	"github.com/redis/go-redis/v9"
)

type SubscriptionEnforcer struct {
	redis      *redis.Client
	billingURL string
	httpClient *http.Client
}

func NewSubscriptionEnforcer() *SubscriptionEnforcer {
	redisHost := utils.GetEnv("REDIS_HOST")
	redisPass := utils.GetEnv("REDIS_PASSWORD")
	client := redis.NewClient(&redis.Options{
		Addr:         redisHost,
		Password:     redisPass,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	return &SubscriptionEnforcer{
		redis:      client,
		billingURL: utils.GetEnv("BILLING_SERVICE_URL"),
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

func (se *SubscriptionEnforcer) EnforceSubscription() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID, _ := c.Get("tenant_id").(string)
			if tenantID == "" {
				return next(c)
			}

			status := se.getSubscriptionStatus(c, tenantID)

			switch status {
			case "expired":
				return c.JSON(http.StatusPaymentRequired, map[string]string{
					"error":         "Subscription expired",
					"status":        status,
					"subscribe_url": "/subscription",
				})
			case "cancelled":
				return c.JSON(http.StatusPaymentRequired, map[string]string{
					"error":         "Subscription cancelled",
					"status":        status,
					"subscribe_url": "/subscription",
				})
			case "grace_period":
				c.Response().Header().Set("X-Subscription-Warning", "grace_period")
			}

			return next(c)
		}
	}
}

func (se *SubscriptionEnforcer) getSubscriptionStatus(c echo.Context, tenantID string) string {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("sub:%s", tenantID)

	cached, err := se.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		return cached
	}

	status := se.fetchFromBillingService(c, tenantID)

	if status != "" {
		if setErr := se.redis.Set(ctx, cacheKey, status, 5*time.Minute).Err(); setErr != nil {
			c.Logger().Warnf("subscription cache write error for tenant %s: %v", tenantID, setErr)
		}
	}

	return status
}

func (se *SubscriptionEnforcer) fetchFromBillingService(c echo.Context, tenantID string) string {
	url := fmt.Sprintf("%s/internal/subscription/%s", se.billingURL, tenantID)
	resp, err := se.httpClient.Get(url) //nolint:noctx
	if err != nil {
		c.Logger().Warnf("billing-service unreachable for tenant %s, failing open: %v", tenantID, err)
		return "trial"
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.Logger().Warnf("billing-service response read error for tenant %s, failing open: %v", tenantID, err)
		return "trial"
	}

	var result struct {
		SubscriptionStatus string `json:"subscription_status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		c.Logger().Warnf("billing-service response parse error for tenant %s, failing open: %v", tenantID, err)
		return "trial"
	}

	if result.SubscriptionStatus == "" {
		return "trial"
	}
	return result.SubscriptionStatus
}
