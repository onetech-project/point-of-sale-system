package observability

import (
	"os"

	"github.com/pos/api-gateway/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", utils.GetEnv("SERVICE_NAME")).
		Str("env", utils.GetEnv("ENVIRONMENT")).
		Logger()
}
