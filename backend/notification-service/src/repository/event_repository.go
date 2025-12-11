package repository

import (
	"context"
	"database/sql"
	"encoding/json"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// Exists returns true if the given event_id is already recorded
func (r *EventRepository) Exists(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM event_records WHERE event_id = $1)`, eventID).Scan(&exists)
	return exists, err
}

// Insert records a processed event into event_records
func (r *EventRepository) Insert(ctx context.Context, eventID, eventType, tenantID string, payload map[string]interface{}) error {
	payloadJSON, _ := json.Marshal(payload)
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO event_records (event_id, event_type, tenant_id, payload, processed_at, created_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
        ON CONFLICT (event_id) DO NOTHING
    `, eventID, eventType, tenantID, payloadJSON)
	return err
}
