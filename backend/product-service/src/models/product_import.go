package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ProductImportStatus string

const (
	ProductImportStatusQueued     ProductImportStatus = "queued"
	ProductImportStatusProcessing ProductImportStatus = "processing"
	ProductImportStatusCompleted  ProductImportStatus = "completed"
	ProductImportStatusFailed     ProductImportStatus = "failed"
)

const (
	ProductImportFailureValidation = "validation_failed"
	ProductImportFailureParse      = "parse_failed"
	ProductImportFailureMutation   = "mutation_failed"
	ProductImportFailureInternal   = "internal_error"
)

type ProductImportIssue struct {
	Row     int    `json:"row,omitempty"`
	SKU     string `json:"sku,omitempty"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

type ProductImportJob struct {
	ID             uuid.UUID           `json:"import_id"`
	TenantID       uuid.UUID           `json:"tenant_id,omitempty"`
	UserID         uuid.UUID           `json:"user_id,omitempty"`
	Status         ProductImportStatus `json:"status"`
	FailureCode    *string             `json:"failure_code,omitempty"`
	SourceFileName string              `json:"source_file_name,omitempty"`
	SourceFormat   string              `json:"source_format,omitempty"`
	SourceFileSize int64               `json:"source_file_size,omitempty"`
	RowCount       int                 `json:"row_count"`
	CreatedCount   int                 `json:"created_count"`
	ErrorCount     int                 `json:"error_count"`
	WarningCount   int                 `json:"warning_count"`
	Errors         json.RawMessage     `json:"errors"`
	Warnings       json.RawMessage     `json:"warnings"`
	StartedAt      *time.Time          `json:"started_at,omitempty"`
	CompletedAt    *time.Time          `json:"completed_at,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}
