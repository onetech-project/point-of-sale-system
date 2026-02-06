package config

import (
	"context"
	"fmt"
	"log"

	"github.com/pos/backend/product-service/src/utils"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() error {
	redisHost := utils.GetEnv("REDIS_HOST")

	redisPassword := utils.GetEnv("REDIS_PASSWORD")

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPassword,
		DB:       0,
	})

	ctx := context.Background()
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connection established")
	return nil
}

func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}
