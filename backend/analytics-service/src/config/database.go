package config

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/pos/analytics-service/src/utils"
	"github.com/rs/zerolog/log"
)

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

var DB *sql.DB

// InitDatabase initializes the PostgreSQL database connection
func InitDatabase() error {
	cfg := loadDatabaseConfig()

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", connStr)
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
		Str("host", cfg.Host).
		Str("dbname", cfg.DBName).
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

func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            utils.GetEnv("DB_HOST"),
		Port:            utils.GetEnv("DB_PORT"),
		User:            utils.GetEnv("DB_USER"),
		Password:        utils.GetEnv("DB_PASSWORD"),
		DBName:          utils.GetEnv("DB_NAME"),
		SSLMode:         utils.GetEnv("DB_SSLMODE"),
		MaxOpenConns:    utils.GetEnvInt("DB_MAX_OPEN_CONNS"),
		MaxIdleConns:    utils.GetEnvInt("DB_MAX_IDLE_CONNS"),
		ConnMaxLifetime: utils.GetEnvAsDuration("DB_CONN_MAX_LIFETIME"),
	}
}
