package config

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type RedisConfig struct {
	URL        string
	Password   string
	DB         int
	MaxRetries int
	PoolSize   int
}

var RedisClient *redis.Client

// InitRedis initializes the Redis client
func InitRedis() error {
	cfg := loadRedisConfig()

	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// Override with additional config
	if cfg.Password != "" {
		opt.Password = cfg.Password
	}
	opt.DB = cfg.DB
	opt.MaxRetries = cfg.MaxRetries
	opt.PoolSize = cfg.PoolSize

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	RedisClient = client
	log.Info().
		Int("db", cfg.DB).
		Int("max_retries", cfg.MaxRetries).
		Int("pool_size", cfg.PoolSize).
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
		URL:        GetEnvAsString("REDIS_URL"),
		Password:   GetEnvAsString("REDIS_PASSWORD"),
		DB:         GetEnvAsInt("REDIS_DB"),
		MaxRetries: GetEnvAsInt("REDIS_MAX_RETRIES"),
		PoolSize:   GetEnvAsInt("REDIS_POOL_SIZE"),
	}
}
