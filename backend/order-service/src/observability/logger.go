package observability

import (
	"os"

	"github.com/point-of-sale-system/order-service/src/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", config.GetEnvAsString("SERVICE_NAME")).
		Str("env", config.GetEnvAsString("ENVIRONMENT")).
		Logger()
}
