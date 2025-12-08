package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
	"github.com/redis/go-redis/v9"
)

type CartRepository struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewCartRepository(redisClient *redis.Client, ttl time.Duration) *CartRepository {
	return &CartRepository{
		redis: redisClient,
		ttl:   ttl,
	}
}

func (r *CartRepository) GetCartKey(tenantID, sessionID string) string {
	return fmt.Sprintf("cart:%s:%s", tenantID, sessionID)
}

func (r *CartRepository) Get(ctx context.Context, tenantID, sessionID string) (*models.Cart, error) {
	key := r.GetCartKey(tenantID, sessionID)
	data, err := r.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return &models.Cart{
			TenantID:  tenantID,
			SessionID: sessionID,
			Items:     []models.CartItem{},
			UpdatedAt: time.Now().Format(time.RFC3339),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cart from redis: %w", err)
	}
	var cart models.Cart
	if err := json.Unmarshal([]byte(data), &cart); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cart: %w", err)
	}
	return &cart, nil
}

func (r *CartRepository) Save(ctx context.Context, cart *models.Cart) error {
	key := r.GetCartKey(cart.TenantID, cart.SessionID)
	cart.UpdatedAt = time.Now().Format(time.RFC3339)
	data, err := json.Marshal(cart)
	if err != nil {
		return fmt.Errorf("failed to marshal cart: %w", err)
	}
	if err := r.redis.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to save cart to redis: %w", err)
	}
	return nil
}

func (r *CartRepository) Delete(ctx context.Context, tenantID, sessionID string) error {
	key := r.GetCartKey(tenantID, sessionID)
	if err := r.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cart from redis: %w", err)
	}
	return nil
}

func (r *CartRepository) Extend(ctx context.Context, tenantID, sessionID string) error {
	key := r.GetCartKey(tenantID, sessionID)
	if err := r.redis.Expire(ctx, key, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to extend cart TTL: %w", err)
	}
	return nil
}
