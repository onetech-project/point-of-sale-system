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

// StreamMessage is a simplified representation of a Redis stream message.
type StreamMessage struct {
	ID     string
	Values map[string]interface{}
}

// ReadStream reads messages from a Redis stream starting after lastID.
// If lastID is "$" then only new messages will be returned when they arrive.
// block is a time.Duration for how long to block waiting for messages.
func (r *RedisProvider) ReadStream(ctx context.Context, stream string, lastID string, block time.Duration) ([]StreamMessage, error) {
	args := &redis.XReadArgs{
		Streams: []string{stream, lastID},
		Count:   10,
		Block:   block,
	}
	streams, err := r.client.XRead(ctx, args).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var out []StreamMessage
	for _, s := range streams {
		for _, m := range s.Messages {
			// copy values into a map[string]interface{}
			vals := make(map[string]interface{}, len(m.Values))
			for k, v := range m.Values {
				vals[k] = v
			}
			out = append(out, StreamMessage{ID: m.ID, Values: vals})
		}
	}
	return out, nil
}

// IncrWithTTL atomically increments a key and sets TTL when it was newly created (i.e., value == 1)
func (r *RedisProvider) IncrWithTTL(ctx context.Context, key string, ttlSeconds int64) (int64, error) {
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if val == 1 && ttlSeconds > 0 {
		_ = r.client.Expire(ctx, key, time.Duration(ttlSeconds)*time.Second)
	}
	return val, nil
}

// GetInt fetches an integer value for a key (returns 0 if not found)
func (r *RedisProvider) GetInt(ctx context.Context, key string) (int64, error) {
	v, err := r.client.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return v, nil
}

// DelKey deletes a key
func (r *RedisProvider) DelKey(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
