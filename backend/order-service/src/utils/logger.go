package utils

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger initializes the structured logger
func InitLogger() {
	// Set log level from environment
	level := getLogLevel(os.Getenv("LOG_LEVEL"))
	zerolog.SetGlobalLevel(level)

	// Pretty logging for development
	if os.Getenv("ENV") == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	}

	log.Info().
		Str("level", level.String()).
		Str("env", os.Getenv("ENV")).
		Msg("Logger initialized")
}

// GetLogger returns a logger with context fields
func GetLogger(tenantID, orderRef string) zerolog.Logger {
	return log.With().
		Str("tenant_id", tenantID).
		Str("order_reference", orderRef).
		Logger()
}

func getLogLevel(level string) zerolog.Level {
	switch level {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
