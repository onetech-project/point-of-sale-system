package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos/audit-service/src/models"
	"github.com/pos/audit-service/src/utils"
)

// ConsentRepository handles database operations for consent-related tables
type ConsentRepository struct {
	db        *sql.DB
	encryptor utils.Encryptor
}

// NewConsentRepository creates a new consent repository
func NewConsentRepository(db *sql.DB, encryptor utils.Encryptor) *ConsentRepository {
	return &ConsentRepository{
		db:        db,
		encryptor: encryptor,
	}
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
		SELECT cr.id, cr.tenant_id, cr.subject_type, 
		       COALESCE(cr.subject_id::text, cr.guest_order_id::text) as subject_id,
		       cp.purpose_code,
		       cr.granted, cr.policy_version, cr.consent_method, cr.ip_address, cr.user_agent,
		       cr.revoked_at, cr.created_at, cr.created_at as updated_at
		FROM consent_records cr
		JOIN consent_purposes cp ON cr.purpose_id = cp.id
		WHERE cr.tenant_id = $1
	`
	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.SubjectType != nil {
		query += fmt.Sprintf(" AND cr.subject_type = $%d", argIdx)
		args = append(args, *filter.SubjectType)
		argIdx++
	}
	if filter.SubjectID != nil {
		query += fmt.Sprintf(" AND (cr.subject_id::text = $%d OR cr.guest_order_id::text = $%d)", argIdx, argIdx)
		args = append(args, *filter.SubjectID)
		argIdx++
	}
	if filter.PurposeCode != nil {
		query += fmt.Sprintf(" AND cp.purpose_code = $%d", argIdx)
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
		var encryptedIP string
		err := rows.Scan(
			&record.RecordID,
			&record.TenantID,
			&record.SubjectType,
			&record.SubjectID,
			&record.PurposeCode,
			&record.Granted,
			&record.PolicyVersion,
			&record.ConsentMethod,
			&encryptedIP,
			&record.UserAgent,
			&record.RevokedAt,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent record: %w", err)
		}

		// Decrypt IP address with context
		if encryptedIP != "" {
			decrypted, err := r.encryptor.DecryptWithContext(ctx, encryptedIP, "consent_record:ip_address")
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt IP address: %w", err)
			}
			record.IPAddress = &decrypted
		}

		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return records, nil
}

// GetConsentRecord retrieves a single consent record by id
func (r *ConsentRepository) GetConsentRecord(ctx context.Context, recordID uuid.UUID) (*models.ConsentRecord, error) {
	query := `
		SELECT cr.id, cr.tenant_id, cr.subject_type,
		       COALESCE(cr.subject_id::text, cr.guest_order_id::text) as subject_id,
		       cp.purpose_code,
		       cr.granted, cr.policy_version, cr.consent_method, cr.ip_address, cr.user_agent,
		       cr.revoked_at, cr.created_at, cr.created_at as updated_at
		FROM consent_records cr
		JOIN consent_purposes cp ON cr.purpose_id = cp.id
		WHERE cr.id = $1
	`

	var record models.ConsentRecord
	var encryptedIP string
	err := r.db.QueryRowContext(ctx, query, recordID).Scan(
		&record.RecordID,
		&record.TenantID,
		&record.SubjectType,
		&record.SubjectID,
		&record.PurposeCode,
		&record.Granted,
		&record.PolicyVersion,
		&record.ConsentMethod,
		&encryptedIP,
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

	// Decrypt IP address with context
	if encryptedIP != "" {
		decrypted, err := r.encryptor.DecryptWithContext(ctx, encryptedIP, "consent_record:ip_address")
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt IP address: %w", err)
		}
		record.IPAddress = &decrypted
	}

	return &record, nil
}

// ListConsentPurposes retrieves all available consent purposes, optionally filtered by context
func (r *ConsentRepository) ListConsentPurposes(ctx context.Context, acceptLanguage string, contextFilter string) ([]*models.ConsentPurpose, error) {
	baseQuery := `
		SELECT purpose_code, purpose_name_%s, description_%s, is_required,
		       context, display_order, created_at
		FROM consent_purposes
	`

	var query string
	var args []interface{}

	if contextFilter != "" {
		query = fmt.Sprintf(baseQuery+` WHERE context = $1 ORDER BY display_order ASC`, acceptLanguage, acceptLanguage)
		args = append(args, contextFilter)
	} else {
		query = fmt.Sprintf(baseQuery+` ORDER BY display_order ASC`, acceptLanguage, acceptLanguage)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)

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
			&purpose.Context,
			&purpose.DisplayOrder,
			&purpose.CreatedAt,
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

// GetConsentPurposeByCode retrieves a specific consent purpose by its code
func (r *ConsentRepository) GetConsentPurposeByCode(ctx context.Context, purposeCode string) (*models.ConsentPurpose, error) {
	query := `
		SELECT purpose_code, purpose_name_id, description_id, is_required,
		       display_order, created_at
		FROM consent_purposes
		WHERE purpose_code = $1
	`

	var purpose models.ConsentPurpose
	err := r.db.QueryRowContext(ctx, query, purposeCode).Scan(
		&purpose.PurposeCode,
		&purpose.DisplayNameID,
		&purpose.DescriptionID,
		&purpose.IsRequired,
		&purpose.DisplayOrder,
		&purpose.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("consent purpose not found: %s", purposeCode)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query consent purpose: %w", err)
	}

	return &purpose, nil
}

// GetCurrentPrivacyPolicy retrieves the current active privacy policy
func (r *ConsentRepository) GetCurrentPrivacyPolicy(ctx context.Context, acceptLanguage string) (*models.PrivacyPolicy, error) {
	query := fmt.Sprintf(`
		SELECT version, policy_text_%s, effective_date, is_current,
		       created_at
		FROM privacy_policies
		WHERE is_current = true
		LIMIT 1
	`, acceptLanguage)

	var policy models.PrivacyPolicy
	err := r.db.QueryRowContext(ctx, query).Scan(
		&policy.Version,
		&policy.PolicyTextID,
		&policy.EffectiveDate,
		&policy.IsCurrent,
		&policy.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no current privacy policy found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query privacy policy: %w", err)
	}

	return &policy, nil
}

// GetPrivacyPolicyByVersion retrieves a specific privacy policy by version
func (r *ConsentRepository) GetPrivacyPolicyByVersion(ctx context.Context, version string) (*models.PrivacyPolicy, error) {
	query := `
		SELECT version, policy_text_id, effective_date, is_current,
		       created_at, updated_at
		FROM privacy_policies
		WHERE version = $1
		LIMIT 1
	`

	var policy models.PrivacyPolicy
	err := r.db.QueryRowContext(ctx, query, version).Scan(
		&policy.Version,
		&policy.PolicyTextID,
		&policy.EffectiveDate,
		&policy.IsCurrent,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("privacy policy version not found: %s", version)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query privacy policy: %w", err)
	}

	return &policy, nil
}

// CreateConsentRecord creates a new consent record
func (r *ConsentRepository) CreateConsentRecord(ctx context.Context, record *models.ConsentRecord) error {
	// First, get the purpose_id from purpose_code
	var purposeID uuid.UUID
	err := r.db.QueryRowContext(ctx, "SELECT id FROM consent_purposes WHERE purpose_code = $1", record.PurposeCode).Scan(&purposeID)
	if err != nil {
		return fmt.Errorf("failed to get purpose_id for code %s: %w", record.PurposeCode, err)
	}

	// Determine subject_id and guest_order_id based on subject_type
	var subjectID, guestOrderID *uuid.UUID
	if record.SubjectID == nil || *record.SubjectID == "" {
		return fmt.Errorf("subject_id is required")
	}
	
	parsed, err := uuid.Parse(*record.SubjectID)
	if err != nil {
		return fmt.Errorf("invalid subject_id UUID format: %w", err)
	}
	
	if record.SubjectType == "tenant" {
		// For tenant, store in subject_id column (can be user_id or tenant_id)
		subjectID = &parsed
	} else if record.SubjectType == "guest" {
		// For guest, store in guest_order_id column
		guestOrderID = &parsed
	}

	query := `
		INSERT INTO consent_records (
			id, tenant_id, subject_type, subject_id, guest_order_id, purpose_id,
			granted, policy_version, consent_method, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	if record.RecordID == uuid.Nil {
		record.RecordID = uuid.New()
	}

	// Parse tenant_id to UUID
	tenantUUID, err := uuid.Parse(record.TenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant_id: %w", err)
	}

	// Encrypt IP address with context
	var encryptedIP string
	if record.IPAddress != nil && *record.IPAddress != "" {
		var err error
		encryptedIP, err = r.encryptor.EncryptWithContext(ctx, *record.IPAddress, "consent_record:ip_address")
		if err != nil {
			return fmt.Errorf("failed to encrypt IP address: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		record.RecordID,
		tenantUUID,
		record.SubjectType,
		subjectID,
		guestOrderID,
		purposeID,
		record.Granted,
		record.PolicyVersion,
		record.ConsentMethod,
		encryptedIP,
		record.UserAgent,
	)

	if err != nil {
		return fmt.Errorf("failed to create consent record: %w", err)
	}

	return nil
}

// GetActiveConsents retrieves all active (non-revoked) consents for a subject
func (r *ConsentRepository) GetActiveConsents(ctx context.Context, tenantID, subjectType, subjectID string) ([]*models.ConsentRecord, error) {
	query := `
		SELECT cr.id, cr.tenant_id, cr.subject_type,
		       COALESCE(cr.subject_id::text, cr.guest_order_id::text) as subject_id,
		       cp.purpose_code,
		       cr.granted, cr.policy_version, cr.consent_method, cr.ip_address, cr.user_agent,
		       cr.revoked_at, cr.created_at, cr.created_at as updated_at
		FROM consent_records cr
		JOIN consent_purposes cp ON cr.purpose_id = cp.id
		WHERE cr.tenant_id = $1
		  AND cr.subject_type = $2
		  AND (cr.subject_id::text = $3 OR cr.guest_order_id::text = $3)
		  AND cr.granted = true
		  AND cr.revoked_at IS NULL
		ORDER BY cr.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, subjectType, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active consents: %w", err)
	}
	defer rows.Close()

	var records []*models.ConsentRecord
	for rows.Next() {
		var record models.ConsentRecord
		var encryptedIP string
		err := rows.Scan(
			&record.RecordID,
			&record.TenantID,
			&record.SubjectType,
			&record.SubjectID,
			&record.PurposeCode,
			&record.Granted,
			&record.PolicyVersion,
			&record.ConsentMethod,
			&encryptedIP,
			&record.UserAgent,
			&record.RevokedAt,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent record: %w", err)
		}

		// Decrypt IP address with context
		if encryptedIP != "" {
			decrypted, err := r.encryptor.DecryptWithContext(ctx, encryptedIP, "consent_record:ip_address")
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt IP address: %w", err)
			}
			record.IPAddress = &decrypted
		}

		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return records, nil
}

// RevokeConsent marks a consent record as revoked
func (r *ConsentRepository) RevokeConsent(ctx context.Context, recordID uuid.UUID) error {
	query := `
		UPDATE consent_records
		SET revoked_at = NOW()
		WHERE id = $1
		  AND revoked_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, recordID)
	if err != nil {
		return fmt.Errorf("failed to revoke consent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent record not found or already revoked: %s", recordID.String())
	}

	return nil
}

// GetConsentHistory retrieves all consent records (including revoked) for a subject
func (r *ConsentRepository) GetConsentHistory(ctx context.Context, tenantID, subjectType, subjectID string) ([]*models.ConsentRecord, error) {
	query := `
		SELECT cr.id, cr.tenant_id, cr.subject_type,
		       COALESCE(cr.subject_id::text, cr.guest_order_id::text) as subject_id,
		       cp.purpose_code,
		       cr.granted, cr.policy_version, cr.consent_method, cr.ip_address, cr.user_agent,
		       cr.revoked_at, cr.created_at, cr.created_at as updated_at
		FROM consent_records cr
		JOIN consent_purposes cp ON cr.purpose_id = cp.id
		WHERE cr.tenant_id = $1
		  AND cr.subject_type = $2
		  AND (cr.subject_id::text = $3 OR cr.guest_order_id::text = $3)
		ORDER BY cr.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, subjectType, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query consent history: %w", err)
	}
	defer rows.Close()

	var records []*models.ConsentRecord
	for rows.Next() {
		var record models.ConsentRecord
		var encryptedIP string
		err := rows.Scan(
			&record.RecordID,
			&record.TenantID,
			&record.SubjectType,
			&record.SubjectID,
			&record.PurposeCode,
			&record.Granted,
			&record.PolicyVersion,
			&record.ConsentMethod,
			&encryptedIP,
			&record.UserAgent,
			&record.RevokedAt,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan consent record: %w", err)
		}

		// Decrypt IP address with context
		if encryptedIP != "" {
			decrypted, err := r.encryptor.DecryptWithContext(ctx, encryptedIP, "consent_record:ip_address")
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt IP address: %w", err)
			}
			record.IPAddress = &decrypted
		}

		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return records, nil
}
