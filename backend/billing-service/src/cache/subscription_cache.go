package cache

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// SubscriptionCache invalidates the API Gateway subscription status cache.
type SubscriptionCache struct {
	client *redis.Client
}

// NewSubscriptionCacheFromEnv creates a cache client when Redis is configured.
func NewSubscriptionCacheFromEnv() *SubscriptionCache {
	addr := normalizeRedisAddr(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))
	if addr == "" {
		return nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	return &SubscriptionCache{client: client}
}

// Close closes the underlying Redis client.
func (c *SubscriptionCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

// InvalidateSubscriptionStatus removes a tenant's cached subscription status.
func (c *SubscriptionCache) InvalidateSubscriptionStatus(ctx context.Context, tenantID string) error {
	if c == nil || c.client == nil || tenantID == "" {
		return nil
	}
	key := fmt.Sprintf("sub:%s", tenantID)
	return c.client.Del(ctx, key).Err()
}

func normalizeRedisAddr(host, port string) string {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" {
		return ""
	}
	if strings.Contains(host, ":") {
		return host
	}
	if port == "" {
		port = "6379"
	}
	return host + ":" + port
}
