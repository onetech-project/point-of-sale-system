package config

import (
	"strings"

	"github.com/pos/auth-service/src/utils"
)

var (
	DB_URL                  = utils.GetEnv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/pos_db?sslmode=disable")
	REDIS_HOST              = utils.GetEnv("REDIS_HOST", "localhost:6379")
	REDIS_PASSWORD          = utils.GetEnv("REDIS_PASSWORD", "")
	SESSION_TTL_MINUTES     = utils.GetEnvInt("SESSION_TTL_MINUTES", 15)
	JWT_SECRET              = utils.GetEnv("JWT_SECRET", "supersecretkey")
	JWT_EXPIRATION_MINUTES  = utils.GetEnvInt("JWT_EXPIRATION_MINUTES", 15)
	RATE_LIMIT_LOGIN_MAX    = utils.GetEnvInt("RATE_LIMIT_LOGIN_MAX", 5)
	RATE_LIMIT_LOGIN_WINDOW = utils.GetEnvInt("RATE_LIMIT_LOGIN_WINDOW", 15)
	KAFKA_BROKERS           = strings.Split(utils.GetEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	KAFKA_TOPICS            = strings.Split(utils.GetEnv("KAFKA_TOPICS", ""), ",")
	PORT                    = utils.GetEnv("PORT", "8080")
	REDIS_DB                = utils.GetEnvInt("REDIS_DB", 0)
)
