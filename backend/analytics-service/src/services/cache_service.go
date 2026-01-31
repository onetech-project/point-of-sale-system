package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// CacheService handles Redis caching operations
type CacheService struct {
	client *redis.Client
}

// NewCacheService creates a new cache service
func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{client: client}
}

// Get retrieves a value from cache and unmarshals it into the target
func (cs *CacheService) Get(ctx context.Context, key string, target interface{}) error {
	val, err := cs.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("cache miss")
	}
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("Failed to get from cache")
		return err
	}

	if err := json.Unmarshal([]byte(val), target); err != nil {
		log.Error().Err(err).Str("key", key).Msg("Failed to unmarshal cached value")
		return err
	}

	log.Debug().Str("key", key).Msg("Cache hit")
	return nil
}

// Set stores a value in cache with the specified TTL
func (cs *CacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("Failed to marshal value for cache")
		return err
	}

	if err := cs.client.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Error().Err(err).Str("key", key).Dur("ttl", ttl).Msg("Failed to set cache")
		return err
	}

	log.Debug().Str("key", key).Dur("ttl", ttl).Msg("Cache set")
	return nil
}

// Delete removes a value from cache
func (cs *CacheService) Delete(ctx context.Context, key string) error {
	if err := cs.client.Del(ctx, key).Err(); err != nil {
		log.Error().Err(err).Str("key", key).Msg("Failed to delete from cache")
		return err
	}

	log.Debug().Str("key", key).Msg("Cache deleted")
	return nil
}

// DeletePattern removes all keys matching the pattern
func (cs *CacheService) DeletePattern(ctx context.Context, pattern string) error {
	iter := cs.client.Scan(ctx, 0, pattern, 0).Iterator()
	keys := []string{}

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		log.Error().Err(err).Str("pattern", pattern).Msg("Failed to scan cache keys")
		return err
	}

	if len(keys) > 0 {
		if err := cs.client.Del(ctx, keys...).Err(); err != nil {
			log.Error().Err(err).Str("pattern", pattern).Int("count", len(keys)).Msg("Failed to delete cache keys")
			return err
		}
		log.Debug().Str("pattern", pattern).Int("count", len(keys)).Msg("Cache keys deleted")
	}

	return nil
}

// GenerateKey generates a cache key with tenant isolation
func GenerateKey(tenantID string, parts ...string) string {
	key := fmt.Sprintf("analytics:tenant:%s", tenantID)
	for _, part := range parts {
		key = fmt.Sprintf("%s:%s", key, part)
	}
	return key
}

// GenerateKeyWithTimeRange generates a cache key with time range
func GenerateKeyWithTimeRange(tenantID string, timeRange, metric string) string {
	return GenerateKey(tenantID, "metrics", metric, timeRange)
}
