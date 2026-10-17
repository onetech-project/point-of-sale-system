package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// analyticsCacheKeyPattern matches every cached analytics response of a tenant.
// Must stay in sync with GenerateKey in analytics-service (analytics:tenant:<id>:...).
const analyticsCacheKeyPattern = "analytics:tenant:%s:*"

// AnalyticsCacheInvalidator drops a tenant's cached dashboard data so order changes show up
// on the next dashboard load instead of after the cache TTL.
type AnalyticsCacheInvalidator struct {
	client *redis.Client
}

// NewAnalyticsCacheInvalidator creates an invalidator; a nil client disables it
func NewAnalyticsCacheInvalidator(client *redis.Client) *AnalyticsCacheInvalidator {
	return &AnalyticsCacheInvalidator{client: client}
}

// InvalidateTenant removes all cached analytics for the tenant. Failures are logged and
// otherwise ignored: the cache expires on its own, so a stale dashboard is not worth failing
// the order change for.
func (i *AnalyticsCacheInvalidator) InvalidateTenant(ctx context.Context, tenantID string) {
	if i == nil || i.client == nil || tenantID == "" {
		return
	}

	// Detach from the request so a cancelled request doesn't leave the cache stale
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	var keys []string
	iter := i.client.Scan(ctx, 0, fmt.Sprintf(analyticsCacheKeyPattern, tenantID), 100).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Warn().Err(err).Str("tenant_id", tenantID).Msg("Failed to scan analytics cache keys")
		return
	}
	if len(keys) == 0 {
		return
	}

	if err := i.client.Del(ctx, keys...).Err(); err != nil {
		log.Warn().Err(err).Str("tenant_id", tenantID).Msg("Failed to invalidate analytics cache")
		return
	}

	log.Debug().Str("tenant_id", tenantID).Int("keys", len(keys)).Msg("Analytics cache invalidated")
}
