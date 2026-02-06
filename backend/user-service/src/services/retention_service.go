package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pos/user-service/src/models"
	"github.com/rs/zerolog/log"
)

// RetentionPolicyService handles retention policy management and evaluation
type RetentionPolicyService struct {
	db *sql.DB
}

// NewRetentionPolicyService creates a new retention policy service
func NewRetentionPolicyService(db *sql.DB) *RetentionPolicyService {
	return &RetentionPolicyService{db: db}
}

// GetActivePolicies retrieves all active retention policies
func (s *RetentionPolicyService) GetActivePolicies(ctx context.Context) ([]*models.RetentionPolicy, error) {
	query := `
		SELECT id, table_name, record_type, retention_period_days, retention_field,
		       grace_period_days, legal_minimum_days, cleanup_method,
		       notification_days_before, is_active, created_at, updated_at
		FROM retention_policies
		WHERE is_active = TRUE
		ORDER BY table_name, record_type
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query retention policies: %w", err)
	}
	defer rows.Close()

	var policies []*models.RetentionPolicy
	for rows.Next() {
		var policy models.RetentionPolicy
		err := rows.Scan(
			&policy.ID,
			&policy.TableName,
			&policy.RecordType,
			&policy.RetentionPeriodDays,
			&policy.RetentionField,
			&policy.GracePeriodDays,
			&policy.LegalMinimumDays,
			&policy.CleanupMethod,
			&policy.NotificationDaysBefore,
			&policy.IsActive,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan retention policy: %w", err)
		}
		policies = append(policies, &policy)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating retention policies: %w", err)
	}

	return policies, nil
}

// GetPolicyByTable retrieves retention policy for a specific table and optional record type
func (s *RetentionPolicyService) GetPolicyByTable(ctx context.Context, tableName string, recordType *string) (*models.RetentionPolicy, error) {
	query := `
		SELECT id, table_name, record_type, retention_period_days, retention_field,
		       grace_period_days, legal_minimum_days, cleanup_method,
		       notification_days_before, is_active, created_at, updated_at
		FROM retention_policies
		WHERE table_name = $1 
		  AND is_active = TRUE
		  AND (record_type = $2 OR (record_type IS NULL AND $2 IS NULL))
		LIMIT 1
	`

	var policy models.RetentionPolicy
	var recordTypeParam sql.NullString
	if recordType != nil {
		recordTypeParam = sql.NullString{String: *recordType, Valid: true}
	}

	err := s.db.QueryRowContext(ctx, query, tableName, recordTypeParam).Scan(
		&policy.ID,
		&policy.TableName,
		&policy.RecordType,
		&policy.RetentionPeriodDays,
		&policy.RetentionField,
		&policy.GracePeriodDays,
		&policy.LegalMinimumDays,
		&policy.CleanupMethod,
		&policy.NotificationDaysBefore,
		&policy.IsActive,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No policy found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get retention policy for table %s: %w", tableName, err)
	}

	return &policy, nil
}

// EvaluatePolicy checks if a record should be cleaned up based on its policy
func (s *RetentionPolicyService) EvaluatePolicy(policy *models.RetentionPolicy, recordTimestamp time.Time) (shouldCleanup bool, shouldNotify bool) {
	if policy == nil || !policy.IsActive {
		return false, false
	}

	// Check if record has expired
	shouldCleanup = policy.IsExpired(recordTimestamp)

	// Check if notification should be sent (before expiry)
	shouldNotify = policy.ShouldNotify(recordTimestamp)

	return shouldCleanup, shouldNotify
}

// ValidateRetentionPeriod validates that retention period meets legal minimum
func (s *RetentionPolicyService) ValidateRetentionPeriod(retentionDays int, legalMinimumDays *int) error {
	if legalMinimumDays != nil && retentionDays < *legalMinimumDays {
		return fmt.Errorf(
			"retention period (%d days) is below legal minimum (%d days)",
			retentionDays,
			*legalMinimumDays,
		)
	}
	return nil
}

// UpdatePolicy updates an existing retention policy
func (s *RetentionPolicyService) UpdatePolicy(ctx context.Context, policy *models.RetentionPolicy) error {
	// Validate retention period
	if err := s.ValidateRetentionPeriod(policy.RetentionPeriodDays, policy.LegalMinimumDays); err != nil {
		return err
	}

	query := `
		UPDATE retention_policies
		SET retention_period_days = $1,
		    grace_period_days = $2,
		    notification_days_before = $3,
		    is_active = $4,
		    updated_at = NOW()
		WHERE id = $5
	`

	result, err := s.db.ExecContext(ctx, query,
		policy.RetentionPeriodDays,
		policy.GracePeriodDays,
		policy.NotificationDaysBefore,
		policy.IsActive,
		policy.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update retention policy: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("retention policy not found: %s", policy.ID)
	}

	log.Info().
		Str("policy_id", policy.ID).
		Str("table_name", policy.TableName).
		Int("retention_days", policy.RetentionPeriodDays).
		Msg("Retention policy updated")

	return nil
}

// GetExpiredRecordCount returns the count of records that would be cleaned up for a policy
func (s *RetentionPolicyService) GetExpiredRecordCount(ctx context.Context, policy *models.RetentionPolicy) (int, error) {
	// Build dynamic query based on policy configuration
	var recordTypeFilter string
	if policy.RecordType != nil {
		// This would require additional logic based on the actual table schema
		// For now, we'll just return a simple count
		recordTypeFilter = ""
	}

	// Calculate expiry date
	expiryDate := time.Now().AddDate(0, 0, -policy.RetentionPeriodDays)

	// Generic query (specific implementations should override this)
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s 
		WHERE %s < $1
		%s
	`, policy.TableName, policy.RetentionField, recordTypeFilter)

	var count int
	err := s.db.QueryRowContext(ctx, query, expiryDate).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count expired records for table %s: %w", policy.TableName, err)
	}

	return count, nil
}
