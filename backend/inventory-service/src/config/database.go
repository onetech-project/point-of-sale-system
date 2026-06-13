package config

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/pos/backend/inventory-service/src/utils"
)

var DB *sql.DB

func InitDatabase() error {
	dbURL := utils.GetEnv("DATABASE_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	return nil
}

func CloseDatabase() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
