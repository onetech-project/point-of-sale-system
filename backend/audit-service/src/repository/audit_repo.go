package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/audit-service/src/models"
)

// AuditRepository handles database operations for audit_events table
type AuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create inserts a new audit event into the appropriate monthly partition
func (r *AuditRepository) Create(ctx context.Context, event *models.AuditEvent) error {
	// Generate UUID if not provided
	if event.EventID == uuid.Nil {
		event.EventID = uuid.New()
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Insert into partitioned table (PostgreSQL routing handles partition selection)
	query := `
		INSERT INTO audit_events (
			event_id, tenant_id, timestamp, actor_type, actor_id, actor_email,
			session_id, action, resource_type, resource_id, ip_address,
			user_agent, request_id, purpose, before_value, after_value,
			metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		event.EventID,
		event.TenantID,
		event.Timestamp,
		event.ActorType,
		event.ActorID,
		event.ActorEmail,
		event.SessionID,
		event.Action,
		event.ResourceType,
		event.ResourceID,
		event.IPAddress,
		event.UserAgent,
		event.RequestID,
		event.Purpose,
		event.BeforeValue,
		event.AfterValue,
		event.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to insert audit event: %w", err)
	}

	return nil
}

// GetByID retrieves a single audit event by event_id
func (r *AuditRepository) GetByID(ctx context.Context, eventID uuid.UUID) (*models.AuditEvent, error) {
	query := `
		SELECT event_id, tenant_id, timestamp, actor_type, actor_id, actor_email,
		       session_id, action, resource_type, resource_id, ip_address,
		       user_agent, request_id, purpose, before_value, after_value,
		       metadata
		FROM audit_events
		WHERE event_id = $1
	`

	var event models.AuditEvent
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(
		&event.EventID,
		&event.TenantID,
		&event.Timestamp,
		&event.ActorType,
		&event.ActorID,
		&event.ActorEmail,
		&event.SessionID,
		&event.Action,
		&event.ResourceType,
		&event.ResourceID,
		&event.IPAddress,
		&event.UserAgent,
		&event.RequestID,
		&event.Purpose,
		&event.BeforeValue,
		&event.AfterValue,
		&event.Metadata,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("audit event not found: %s", eventID.String())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query audit event: %w", err)
	}

	return &event, nil
}

// AuditQueryFilter defines search criteria for audit events
type AuditQueryFilter struct {
	TenantID     string
	ActorType    *string
	ActorID      *string
	Action       *string
	ResourceType *string
	ResourceID   *string
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// List retrieves audit events matching filter criteria (tenant isolation enforced)
func (r *AuditRepository) List(ctx context.Context, filter AuditQueryFilter) ([]*models.AuditEvent, error) {
	query := `
		SELECT event_id, tenant_id, timestamp, actor_type, actor_id, actor_email,
		       session_id, action, resource_type, resource_id, ip_address,
		       user_agent, request_id, purpose, before_value, after_value,
		       metadata
		FROM audit_events
		WHERE tenant_id = $1
	`
	args := []interface{}{filter.TenantID}
	argIdx := 2

	// Optional filters
	if filter.ActorType != nil {
		query += fmt.Sprintf(" AND actor_type = $%d", argIdx)
		args = append(args, *filter.ActorType)
		argIdx++
	}
	if filter.ActorID != nil {
		query += fmt.Sprintf(" AND actor_id = $%d", argIdx)
		args = append(args, *filter.ActorID)
		argIdx++
	}
	if filter.Action != nil {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, *filter.Action)
		argIdx++
	}
	if filter.ResourceType != nil {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, *filter.ResourceType)
		argIdx++
	}
	if filter.ResourceID != nil {
		query += fmt.Sprintf(" AND resource_id = $%d", argIdx)
		args = append(args, *filter.ResourceID)
		argIdx++
	}
	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIdx)
		args = append(args, *filter.StartTime)
		argIdx++
	}
	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIdx)
		args = append(args, *filter.EndTime)
		argIdx++
	}

	// Order and pagination
	query += " ORDER BY timestamp DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit events: %w", err)
	}
	defer rows.Close()

	var events []*models.AuditEvent
	for rows.Next() {
		var event models.AuditEvent
		err := rows.Scan(
			&event.EventID,
			&event.TenantID,
			&event.Timestamp,
			&event.ActorType,
			&event.ActorID,
			&event.ActorEmail,
			&event.SessionID,
			&event.Action,
			&event.ResourceType,
			&event.ResourceID,
			&event.IPAddress,
			&event.UserAgent,
			&event.RequestID,
			&event.Purpose,
			&event.BeforeValue,
			&event.AfterValue,
			&event.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit event: %w", err)
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return events, nil
}

// Count returns the total number of audit events matching filter (for pagination)
func (r *AuditRepository) Count(ctx context.Context, filter AuditQueryFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM audit_events WHERE tenant_id = $1`
	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.ActorType != nil {
		query += fmt.Sprintf(" AND actor_type = $%d", argIdx)
		args = append(args, *filter.ActorType)
		argIdx++
	}
	if filter.ActorID != nil {
		query += fmt.Sprintf(" AND actor_id = $%d", argIdx)
		args = append(args, *filter.ActorID)
		argIdx++
	}
	if filter.Action != nil {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, *filter.Action)
		argIdx++
	}
	if filter.ResourceType != nil {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, *filter.ResourceType)
		argIdx++
	}
	if filter.ResourceID != nil {
		query += fmt.Sprintf(" AND resource_id = $%d", argIdx)
		args = append(args, *filter.ResourceID)
		argIdx++
	}
	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIdx)
		args = append(args, *filter.StartTime)
		argIdx++
	}
	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIdx)
		args = append(args, *filter.EndTime)
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	return count, nil
}
