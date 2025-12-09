package providers

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisProvider wraps a redis client and provides helper methods for streams/fanout
type RedisProvider struct {
	client *redis.Client
	ctx    context.Context
	// default stream retention in seconds
	streamRetentionSeconds int64
	maxStreamLen           int64
}

// NewRedisProvider creates a new RedisProvider from an existing redis.Options
func NewRedisProvider(opts *redis.Options, retentionSeconds int64, maxLen int64) *RedisProvider {
	client := redis.NewClient(opts)
	return &RedisProvider{
		client:                 client,
		ctx:                    context.Background(),
		streamRetentionSeconds: retentionSeconds,
		maxStreamLen:           maxLen,
	}
}

// PublishToStream appends a JSON payload to the tenant stream and trims by max length
// streamName should be namespaced per tenant, e.g., "tenant:{tenantID}:stream"
func (r *RedisProvider) PublishToStream(streamName string, fieldValues map[string]interface{}) (string, error) {
	// Use XAdd with MAXLEN approximation to bound memory
	xaddArgs := &redis.XAddArgs{
		Stream: streamName,
		MaxLen: r.maxStreamLen,
		Approx: true,
		Values: fieldValues,
	}
	id, err := r.client.XAdd(r.ctx, xaddArgs).Result()
	if err != nil {
		return "", err
	}

	// Optionally set retention via EXPIRE on a per-stream key if using a ring buffer pattern
	_ = r.client.Expire(r.ctx, streamName, time.Duration(r.streamRetentionSeconds)*time.Second)
	return id, nil
}

// Close closes the underlying Redis client
func (r *RedisProvider) Close() error {
	return r.client.Close()
}
