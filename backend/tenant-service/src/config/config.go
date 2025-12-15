package config

import (
	"strings"

	"github.com/pos/tenant-service/src/utils"
)

var (
	DB_URL        = utils.GetEnv("DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/pos_db?sslmode=disable")
	PORT          = utils.GetEnv("PORT", "8080")
	KAFKA_BROKERS = strings.Split(utils.GetEnv("KAFKA_BROKERS", "localhost:9092"), ",")
)
