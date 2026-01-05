package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/audit-service/src/models"
)

// ConsentRepository handles database operations for consent-related tables
type ConsentRepository struct {
	db *sql.DB
}

// NewConsentRepository creates a new consent repository
func NewConsentRepository(db *sql.DB) *ConsentRepository {
	return &ConsentRepository{db: db}
}

// ConsentQueryFilter defines search criteria for consent records
type ConsentQueryFilter struct {
	TenantID    string
	SubjectType *string
	SubjectID   *string
	PurposeCode *string
	Granted     *bool
	Limit       int
	Offset      int
}

// ListConsentRecords retrieves consent records matching filter criteria
func (r *ConsentRepository) ListConsentRecords(ctx context.Context, filter ConsentQueryFilter) ([]*models.ConsentRecord, error) {
	query := `
		SELECT record_id, tenant_id, subject_type, subject_id, purpose_code,
		       granted, policy_version, consent_method, ip_address, user_agent,
		       revoked_at, created_at, updated_at
		FROM consent_records
		WHERE tenant_id = $1
	`
	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.SubjectType != nil {
		query += fmt.Sprintf(" AND subject_type = $%d", argIdx)
		args = append(args, *filter.SubjectType)
		argIdx++
	}
	if filter.SubjectID != nil {
		query += fmt.Sprintf(" AND subject_id = $%d", argIdx)
		args = append(args, *filter.SubjectID)
		argIdx++
	}
	if filter.PurposeCode != nil {
		query += fmt.Sprintf(" AND purpose_code = $%d", argIdx)
		args = append(args, *filter.PurposeCode)
		argIdx++
	}
	if filter.Granted != nil {
		query += fmt.Sprintf(" AND granted = $%d", argIdx)
		args = append(args, *filter.Granted)
		argIdx++
	}

	query += " ORDER BY created_at DESC"
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
		return nil, fmt.Errorf("failed to query consent records: %w", err)
	}
	defer rows.Close()

	var records []*models.ConsentRecord
	for rows.Next() {
		var record models.ConsentRecord
		err := rows.Scan(
			&record.RecordID,
			&record.TenantID,
			&record.SubjectType,
			&record.SubjectID,
			&record.PurposeCode,
			&record.Granted,
			&record.PolicyVersion,
			&record.ConsentMethod,
			&record.IPAddress,
			&record.UserAgent,
			&record.RevokedAt,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent record: %w", err)
		}
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return records, nil
}

// GetConsentRecord retrieves a single consent record by record_id
func (r *ConsentRepository) GetConsentRecord(ctx context.Context, recordID uuid.UUID) (*models.ConsentRecord, error) {
	query := `
		SELECT record_id, tenant_id, subject_type, subject_id, purpose_code,
		       granted, policy_version, consent_method, ip_address, user_agent,
		       revoked_at, created_at, updated_at
		FROM consent_records
		WHERE record_id = $1
	`

	var record models.ConsentRecord
	err := r.db.QueryRowContext(ctx, query, recordID).Scan(
		&record.RecordID,
		&record.TenantID,
		&record.SubjectType,
		&record.SubjectID,
		&record.PurposeCode,
		&record.Granted,
		&record.PolicyVersion,
		&record.ConsentMethod,
		&record.IPAddress,
		&record.UserAgent,
		&record.RevokedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("consent record not found: %s", recordID.String())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query consent record: %w", err)
	}

	return &record, nil
}

// ListConsentPurposes retrieves all available consent purposes
func (r *ConsentRepository) ListConsentPurposes(ctx context.Context) ([]*models.ConsentPurpose, error) {
	query := `
		SELECT purpose_code, display_name_id, description_id, is_required,
		       display_order, created_at, updated_at
		FROM consent_purposes
		ORDER BY display_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query consent purposes: %w", err)
	}
	defer rows.Close()

	var purposes []*models.ConsentPurpose
	for rows.Next() {
		var purpose models.ConsentPurpose
		err := rows.Scan(
			&purpose.PurposeCode,
			&purpose.DisplayNameID,
			&purpose.DescriptionID,
			&purpose.IsRequired,
			&purpose.DisplayOrder,
			&purpose.CreatedAt,
			&purpose.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent purpose: %w", err)
		}
		purposes = append(purposes, &purpose)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return purposes, nil
}

// GetCurrentPrivacyPolicy retrieves the current active privacy policy
func (r *ConsentRepository) GetCurrentPrivacyPolicy(ctx context.Context) (*models.PrivacyPolicy, error) {
	query := `
		SELECT version, policy_text_id, effective_date, is_current,
		       created_at, updated_at
		FROM privacy_policies
		WHERE is_current = true
		LIMIT 1
	`

	var policy models.PrivacyPolicy
	err := r.db.QueryRowContext(ctx, query).Scan(
		&policy.Version,
		&policy.PolicyTextID,
		&policy.EffectiveDate,
		&policy.IsCurrent,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no current privacy policy found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query privacy policy: %w", err)
	}

	return &policy, nil
}
