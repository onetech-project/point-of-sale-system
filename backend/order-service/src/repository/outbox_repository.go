package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
)

// OutboxRepository handles persistence of events in the transactional outbox
// Implements the Transactional Outbox Pattern for reliable Kafka event publishing
type OutboxRepository struct {
	db *sql.DB
}

// NewOutboxRepository creates a new outbox repository
func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

// Create inserts a new event into the outbox within a transaction
// This ensures atomic writes: business operation + event creation
func (r *OutboxRepository) Create(ctx context.Context, tx *sql.Tx, event *models.EventOutbox) error {
	query := `
		INSERT INTO event_outbox (
			event_type, event_key, event_payload, topic, created_at, retry_count
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := tx.QueryRowContext(
		ctx,
		query,
		event.EventType,
		event.EventKey,
		event.EventPayload,
		event.Topic,
		time.Now(),
		0,
	).Scan(&event.ID)

	if err != nil {
		return fmt.Errorf("failed to insert event into outbox: %w", err)
	}

	return nil
}

// GetPendingEvents retrieves unpublished events from the outbox
// Used by the background worker to poll for events to publish
func (r *OutboxRepository) GetPendingEvents(ctx context.Context, limit int) ([]models.EventOutbox, error) {
	query := `
		SELECT id, event_type, event_key, event_payload, topic, 
		       created_at, published_at, retry_count, last_error
		FROM event_outbox
		WHERE published_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending events: %w", err)
	}
	defer rows.Close()

	var events []models.EventOutbox
	for rows.Next() {
		var event models.EventOutbox
		err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.EventKey,
			&event.EventPayload,
			&event.Topic,
			&event.CreatedAt,
			&event.PublishedAt,
			&event.RetryCount,
			&event.LastError,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event rows: %w", err)
	}

	return events, nil
}

// MarkAsPublished updates an event's published_at timestamp
// Called after successful Kafka publish to prevent reprocessing
func (r *OutboxRepository) MarkAsPublished(ctx context.Context, eventID string) error {
	query := `
		UPDATE event_outbox
		SET published_at = $1
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event as published: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found: %s", eventID)
	}

	return nil
}

// RecordError updates an event's retry count and error message
// Called after failed Kafka publish attempt
func (r *OutboxRepository) RecordError(ctx context.Context, eventID string, errorMsg string) error {
	query := `
		UPDATE event_outbox
		SET retry_count = retry_count + 1,
		    last_error = $1
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, errorMsg, eventID)
	if err != nil {
		return fmt.Errorf("failed to record error: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found: %s", eventID)
	}

	return nil
}

// DeletePublishedEvents removes successfully published events older than the retention period
// Used for periodic cleanup to prevent unbounded table growth
func (r *OutboxRepository) DeletePublishedEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM event_outbox
		WHERE published_at IS NOT NULL
		  AND published_at < $1
	`

	cutoffTime := time.Now().Add(-olderThan)
	result, err := r.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete published events: %w", err)
	}

	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows deleted: %w", err)
	}

	return rowsDeleted, nil
}

// GetFailedEvents retrieves events that have exceeded max retry attempts
// Used for monitoring and manual intervention
func (r *OutboxRepository) GetFailedEvents(ctx context.Context, maxRetries int) ([]models.EventOutbox, error) {
	query := `
		SELECT id, event_type, event_key, event_payload, topic, 
		       created_at, published_at, retry_count, last_error
		FROM event_outbox
		WHERE published_at IS NULL
		  AND retry_count >= $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, maxRetries)
	if err != nil {
		return nil, fmt.Errorf("failed to query failed events: %w", err)
	}
	defer rows.Close()

	var events []models.EventOutbox
	for rows.Next() {
		var event models.EventOutbox
		err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.EventKey,
			&event.EventPayload,
			&event.Topic,
			&event.CreatedAt,
			&event.PublishedAt,
			&event.RetryCount,
			&event.LastError,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}
		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event rows: %w", err)
	}

	return events, nil
}
