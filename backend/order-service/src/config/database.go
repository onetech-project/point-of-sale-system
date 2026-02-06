package config

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

var DB *sql.DB

// InitDatabase initializes the PostgreSQL database connection
func InitDatabase() error {
	cfg := loadDatabaseConfig()

	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	log.Info().
		Int("max_open_conns", cfg.MaxOpenConns).
		Int("max_idle_conns", cfg.MaxIdleConns).
		Dur("conn_max_lifetime", cfg.ConnMaxLifetime).
		Msg("Database connection established")

	return nil
}

// CloseDatabase closes the database connection
func CloseDatabase() error {
	if DB != nil {
		log.Info().Msg("Closing database connection")
		return DB.Close()
	}
	return nil
}

// GetDB returns the database connection
func GetDB() *sql.DB {
	return DB
}

// BeginTx starts a new transaction
func BeginTx() (*sql.Tx, error) {
	return DB.Begin()
}

func loadDatabaseConfig() DatabaseConfig {
	maxOpenConns := GetEnvAsInt("DB_MAX_OPEN_CONNS")
	maxIdleConns := GetEnvAsInt("DB_MAX_IDLE_CONNS")
	connMaxLifetime := GetEnvAsDuration("DB_CONN_MAX_LIFETIME")

	return DatabaseConfig{
		URL:             GetEnvAsString("DATABASE_URL"),
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
	}
}
