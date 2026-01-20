package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CleanupCompletedEvent represents a data cleanup completion event
type CleanupCompletedEvent struct {
	EventID           string    `json:"event_id"`
	EventType         string    `json:"event_type"` // "CLEANUP_COMPLETED"
	TableName         string    `json:"table_name"`
	RecordType        *string   `json:"record_type,omitempty"`
	RecordsProcessed  int       `json:"records_processed"`
	CleanupMethod     string    `json:"cleanup_method"` // "soft_delete", "hard_delete", "anonymize"
	RetentionDays     int       `json:"retention_days"`
	DurationMs        int64     `json:"duration_ms"`
	ExecutedAt        time.Time `json:"executed_at"`
	Status            string    `json:"status"` // "SUCCESS", "PARTIAL", "FAILED"
	ErrorMessage      *string   `json:"error_message,omitempty"`
	ComplianceTag     string    `json:"compliance_tag"` // "UU_PDP_Article_5"
	Timestamp         time.Time `json:"timestamp"`
}

// NewCleanupCompletedEvent creates a new cleanup completed event
func NewCleanupCompletedEvent(
	tableName string,
	recordType *string,
	recordsProcessed int,
	cleanupMethod string,
	retentionDays int,
	durationMs int64,
	status string,
	errorMessage *string,
) *CleanupCompletedEvent {
	return &CleanupCompletedEvent{
		EventID:          uuid.New().String(),
		EventType:        "CLEANUP_COMPLETED",
		TableName:        tableName,
		RecordType:       recordType,
		RecordsProcessed: recordsProcessed,
		CleanupMethod:    cleanupMethod,
		RetentionDays:    retentionDays,
		DurationMs:       durationMs,
		ExecutedAt:       time.Now().UTC(),
		Status:           status,
		ErrorMessage:     errorMessage,
		ComplianceTag:    "UU_PDP_Article_5",
		Timestamp:        time.Now().UTC(),
	}
}

// ToJSON converts the event to JSON bytes
func (e *CleanupCompletedEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}
