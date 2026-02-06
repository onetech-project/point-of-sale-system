package config

import (
	"context"
	"fmt"

	"github.com/pos/analytics-service/src/utils"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

var RedisClient *redis.Client

// InitRedis initializes the Redis client connection
func InitRedis() error {
	cfg := loadRedisConfig()

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	RedisClient = client
	log.Info().
		Str("host", cfg.Host).
		Str("port", cfg.Port).
		Int("db", cfg.DB).
		Msg("Redis connection established")

	return nil
}

// CloseRedis closes the Redis connection
func CloseRedis() error {
	if RedisClient != nil {
		log.Info().Msg("Closing Redis connection")
		return RedisClient.Close()
	}
	return nil
}

// GetRedis returns the Redis client
func GetRedis() *redis.Client {
	return RedisClient
}

func loadRedisConfig() RedisConfig {
	return RedisConfig{
		Host:     utils.GetEnv("REDIS_HOST"),
		Port:     utils.GetEnv("REDIS_PORT"),
		Password: utils.GetEnv("REDIS_PASSWORD"),
		DB:       utils.GetEnvInt("REDIS_DB"),
	}
}
