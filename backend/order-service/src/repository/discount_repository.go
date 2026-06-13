package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/point-of-sale-system/order-service/src/models"
)

type DiscountRepository struct {
	db *sql.DB
}

func NewDiscountRepository(db *sql.DB) *DiscountRepository {
	return &DiscountRepository{db: db}
}

func (r *DiscountRepository) Create(ctx context.Context, rule *models.DiscountRule) (*models.DiscountRule, error) {
	query := `
		INSERT INTO discount_rules (
			tenant_id, name, description, target_type, target_id, discount_type,
			discount_value, starts_at, ends_at, is_active, exclusive, priority
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, tenant_id, name, description, target_type, target_id,
			discount_type, discount_value, starts_at, ends_at, is_active,
			exclusive, priority, created_at, updated_at
	`

	var created models.DiscountRule
	if err := r.db.QueryRowContext(ctx, query,
		rule.TenantID,
		rule.Name,
		rule.Description,
		rule.TargetType,
		rule.TargetID,
		rule.DiscountType,
		rule.DiscountValue,
		rule.StartsAt,
		rule.EndsAt,
		rule.IsActive,
		rule.Exclusive,
		rule.Priority,
	).Scan(discountRuleScanDest(&created)...); err != nil {
		return nil, err
	}

	return &created, nil
}

func (r *DiscountRepository) GetByID(ctx context.Context, tenantID, id string) (*models.DiscountRule, error) {
	query := `
		SELECT id, tenant_id, name, description, target_type, target_id,
			discount_type, discount_value, starts_at, ends_at, is_active,
			exclusive, priority, created_at, updated_at
		FROM discount_rules
		WHERE tenant_id = $1 AND id = $2
	`

	var rule models.DiscountRule
	if err := r.db.QueryRowContext(ctx, query, tenantID, id).Scan(discountRuleScanDest(&rule)...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &rule, nil
}

func (r *DiscountRepository) List(ctx context.Context, filter models.ListDiscountRulesFilter) ([]models.DiscountRule, error) {
	var args []interface{}
	args = append(args, filter.TenantID)

	query := strings.Builder{}
	query.WriteString(`
		SELECT id, tenant_id, name, description, target_type, target_id,
			discount_type, discount_value, starts_at, ends_at, is_active,
			exclusive, priority, created_at, updated_at
		FROM discount_rules
		WHERE tenant_id = $1
	`)

	if filter.TargetType != nil {
		args = append(args, *filter.TargetType)
		query.WriteString(fmt.Sprintf(" AND target_type = $%d", len(args)))
	}
	if filter.TargetID != nil {
		args = append(args, *filter.TargetID)
		query.WriteString(fmt.Sprintf(" AND target_id = $%d", len(args)))
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", len(args)))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	args = append(args, limit, offset)
	query.WriteString(fmt.Sprintf(" ORDER BY priority ASC, created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args)))

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.DiscountRule
	for rows.Next() {
		var rule models.DiscountRule
		if err := rows.Scan(discountRuleScanDest(&rule)...); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

func (r *DiscountRepository) Update(ctx context.Context, rule *models.DiscountRule) (*models.DiscountRule, error) {
	query := `
		UPDATE discount_rules
		SET name = $3,
			description = $4,
			target_type = $5,
			target_id = $6,
			discount_type = $7,
			discount_value = $8,
			starts_at = $9,
			ends_at = $10,
			is_active = $11,
			exclusive = $12,
			priority = $13,
			updated_at = NOW()
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, name, description, target_type, target_id,
			discount_type, discount_value, starts_at, ends_at, is_active,
			exclusive, priority, created_at, updated_at
	`

	var updated models.DiscountRule
	if err := r.db.QueryRowContext(ctx, query,
		rule.TenantID,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.TargetType,
		rule.TargetID,
		rule.DiscountType,
		rule.DiscountValue,
		rule.StartsAt,
		rule.EndsAt,
		rule.IsActive,
		rule.Exclusive,
		rule.Priority,
	).Scan(discountRuleScanDest(&updated)...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &updated, nil
}

func (r *DiscountRepository) Delete(ctx context.Context, tenantID, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM discount_rules WHERE tenant_id = $1 AND id = $2", tenantID, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *DiscountRepository) FindApplicable(ctx context.Context, tenantID string, targetType models.DiscountTargetType, targetID string, at time.Time) ([]models.DiscountRule, error) {
	query := `
		SELECT id, tenant_id, name, description, target_type, target_id,
			discount_type, discount_value, starts_at, ends_at, is_active,
			exclusive, priority, created_at, updated_at
		FROM discount_rules
		WHERE tenant_id = $1
			AND target_type = $2
			AND target_id = $3
			AND is_active = TRUE
			AND (starts_at IS NULL OR starts_at <= $4)
			AND (ends_at IS NULL OR ends_at >= $4)
		ORDER BY exclusive DESC, priority ASC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID, targetType, targetID, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.DiscountRule
	for rows.Next() {
		var rule models.DiscountRule
		if err := rows.Scan(discountRuleScanDest(&rule)...); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

func discountRuleScanDest(rule *models.DiscountRule) []interface{} {
	return []interface{}{
		&rule.ID,
		&rule.TenantID,
		&rule.Name,
		&rule.Description,
		&rule.TargetType,
		&rule.TargetID,
		&rule.DiscountType,
		&rule.DiscountValue,
		&rule.StartsAt,
		&rule.EndsAt,
		&rule.IsActive,
		&rule.Exclusive,
		&rule.Priority,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	}
}
