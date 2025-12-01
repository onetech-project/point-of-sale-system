package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
)

type StockRepository struct {
	db *sql.DB
}

func NewStockRepository(db *sql.DB) *StockRepository {
	return &StockRepository{db: db}
}

// CreateAdjustment records a stock adjustment in the audit log
func (r *StockRepository) CreateAdjustment(ctx context.Context, adjustment *models.StockAdjustment) error {
	query := `
		INSERT INTO stock_adjustments 
		(tenant_id, product_id, user_id, previous_quantity, new_quantity, reason, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, quantity_delta
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		adjustment.TenantID,
		adjustment.ProductID,
		adjustment.UserID,
		adjustment.PreviousQuantity,
		adjustment.NewQuantity,
		adjustment.Reason,
		adjustment.Notes,
		time.Now(),
	).Scan(&adjustment.ID, &adjustment.QuantityDelta)

	if err != nil {
		return fmt.Errorf("failed to create stock adjustment: %w", err)
	}

	return nil
}

// GetAdjustmentHistory retrieves stock adjustment history for a product
func (r *StockRepository) GetAdjustmentHistory(ctx context.Context, productID uuid.UUID, limit, offset int) ([]*models.StockAdjustment, int, error) {
	// Get total count
	countQuery := `
		SELECT COUNT(*) 
		FROM stock_adjustments 
		WHERE product_id = $1
	`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, productID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count adjustments: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, tenant_id, product_id, user_id, previous_quantity, new_quantity, 
		       quantity_delta, reason, notes, created_at
		FROM stock_adjustments
		WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, productID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query adjustments: %w", err)
	}
	defer rows.Close()

	adjustments := make([]*models.StockAdjustment, 0)
	for rows.Next() {
		adj := &models.StockAdjustment{}
		err := rows.Scan(
			&adj.ID,
			&adj.TenantID,
			&adj.ProductID,
			&adj.UserID,
			&adj.PreviousQuantity,
			&adj.NewQuantity,
			&adj.QuantityDelta,
			&adj.Reason,
			&adj.Notes,
			&adj.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan adjustment: %w", err)
		}
		adjustments = append(adjustments, adj)
	}

	return adjustments, total, nil
}

// GetAdjustmentsByTenant retrieves all stock adjustments for a tenant with filters
func (r *StockRepository) GetAdjustmentsByTenant(ctx context.Context, tenantID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*models.StockAdjustment, int, error) {
	// Build WHERE clause dynamically based on filters
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argIndex := 2

	if reason, ok := filters["reason"].(string); ok && reason != "" {
		whereClause += fmt.Sprintf(" AND reason = $%d", argIndex)
		args = append(args, reason)
		argIndex++
	}

	if startDate, ok := filters["start_date"].(time.Time); ok {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if endDate, ok := filters["end_date"].(time.Time); ok {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, endDate)
		argIndex++
	}

	if productID, ok := filters["product_id"].(uuid.UUID); ok {
		whereClause += fmt.Sprintf(" AND product_id = $%d", argIndex)
		args = append(args, productID)
		argIndex++
	}

	if userID, ok := filters["user_id"].(uuid.UUID); ok {
		whereClause += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM stock_adjustments %s", whereClause)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count adjustments: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, tenant_id, product_id, user_id, previous_quantity, new_quantity,
		       quantity_delta, reason, notes, created_at
		FROM stock_adjustments
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query adjustments: %w", err)
	}
	defer rows.Close()

	adjustments := make([]*models.StockAdjustment, 0)
	for rows.Next() {
		adj := &models.StockAdjustment{}
		err := rows.Scan(
			&adj.ID,
			&adj.TenantID,
			&adj.ProductID,
			&adj.UserID,
			&adj.PreviousQuantity,
			&adj.NewQuantity,
			&adj.QuantityDelta,
			&adj.Reason,
			&adj.Notes,
			&adj.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan adjustment: %w", err)
		}
		adjustments = append(adjustments, adj)
	}

	return adjustments, total, nil
}
