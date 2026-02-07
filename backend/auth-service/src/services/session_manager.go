package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/pos/auth-service/src/models"
)

type SessionManager struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewSessionManager(redisClient *redis.Client, ttlMinutes int) *SessionManager {
	return &SessionManager{
		redis: redisClient,
		ttl:   time.Duration(ttlMinutes) * time.Minute,
	}
}

// Create creates a new session in Redis
func (sm *SessionManager) Create(ctx context.Context, user *models.User) (string, error) {
	sessionID := uuid.New().String()

	sessionData := models.SessionData{
		UserID:    user.ID,
		TenantID:  user.TenantID,
		Email:     user.Email,
		Role:      user.Role,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: time.Now().Unix(),
	}

	data, err := json.Marshal(sessionData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	key := fmt.Sprintf("session:%s", sessionID)
	err = sm.redis.Set(ctx, key, data, sm.ttl).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store session in Redis: %w", err)
	}

	return sessionID, nil
}

// Get retrieves a session from Redis
func (sm *SessionManager) Get(ctx context.Context, sessionID string) (*models.SessionData, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	data, err := sm.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Session not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	var sessionData models.SessionData
	err = json.Unmarshal([]byte(data), &sessionData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	return &sessionData, nil
}

// Exists checks if a session exists in Redis
func (sm *SessionManager) Exists(ctx context.Context, sessionID string) (bool, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	result, err := sm.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return result > 0, nil
}

// Renew extends the TTL of a session (sliding window expiration)
func (sm *SessionManager) Renew(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)

	err := sm.redis.Expire(ctx, key, sm.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to renew session TTL: %w", err)
	}

	return nil
}

// Delete removes a session from Redis
func (sm *SessionManager) Delete(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)

	err := sm.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session from Redis: %w", err)
	}

	return nil
}

// DeleteByUserID deletes all sessions for a specific user
func (sm *SessionManager) DeleteByUserID(ctx context.Context, userID string) error {
	// Scan for all session keys
	pattern := "session:*"
	iter := sm.redis.Scan(ctx, 0, pattern, 0).Iterator()

	var keysToDelete []string

	for iter.Next(ctx) {
		key := iter.Val()

		// Get session data to check user ID
		data, err := sm.redis.Get(ctx, key).Result()
		if err != nil {
			continue // Skip on error
		}

		var sessionData models.SessionData
		if err := json.Unmarshal([]byte(data), &sessionData); err != nil {
			continue // Skip on error
		}

		if sessionData.UserID == userID {
			keysToDelete = append(keysToDelete, key)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan Redis keys: %w", err)
	}

	// Delete all matching keys
	if len(keysToDelete) > 0 {
		err := sm.redis.Del(ctx, keysToDelete...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete user sessions: %w", err)
		}
	}

	return nil
}

// GetTTL returns the remaining TTL for a session
func (sm *SessionManager) GetTTL(ctx context.Context, sessionID string) (time.Duration, error) {
	key := fmt.Sprintf("session:%s", sessionID)

	ttl, err := sm.redis.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get session TTL: %w", err)
	}

	return ttl, nil
}
