package integration

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// This integration test verifies the presence of the `event_records` table.
// It's a test-first harness for T003: it should fail until the migration is applied.
func TestEventRecordsTableExists(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping migration check")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow(`
        SELECT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = 'public' AND table_name = 'event_records'
        )
    `).Scan(&exists)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if !exists {
		t.Fatalf("migration not applied: table 'event_records' does not exist")
	}
}
