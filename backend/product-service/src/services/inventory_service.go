package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pos/backend/product-service/src/models"
	"github.com/pos/backend/product-service/src/repository"
)

type InventoryService struct {
	productRepo repository.ProductRepository
	stockRepo   *repository.StockRepository
	db          *sql.DB
}

func NewInventoryService(productRepo repository.ProductRepository, stockRepo *repository.StockRepository, db *sql.DB) *InventoryService {
	return &InventoryService{
		productRepo: productRepo,
		stockRepo:   stockRepo,
		db:          db,
	}
}

// AdjustStock updates product stock quantity and creates an audit log entry
// This operation is performed in a transaction to ensure consistency
func (s *InventoryService) AdjustStock(ctx context.Context, productID, tenantID, userID uuid.UUID, newQuantity int, reason, notes string) (*models.Product, error) {
	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current product to capture previous quantity
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	if product == nil {
		return nil, errors.New("product not found")
	}

	// Verify tenant ownership
	if product.TenantID != tenantID {
		return nil, errors.New("product not found")
	}

	previousQuantity := product.StockQuantity

	// Update product stock quantity
	_, err = tx.ExecContext(ctx, `
		UPDATE products 
		SET stock_quantity = $1, updated_at = $2
		WHERE id = $3 AND tenant_id = $4
	`, newQuantity, time.Now(), productID, tenantID)

	if err != nil {
		return nil, fmt.Errorf("failed to update stock: %w", err)
	}

	// Create stock adjustment record
	notesPtr := &notes
	if notes == "" {
		notesPtr = nil
	}

	adjustment := &models.StockAdjustment{
		TenantID:         tenantID,
		ProductID:        productID,
		UserID:           userID,
		PreviousQuantity: previousQuantity,
		NewQuantity:      newQuantity,
		Reason:           reason,
		Notes:            notesPtr,
		CreatedAt:        time.Now(),
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO stock_adjustments 
		(tenant_id, product_id, user_id, previous_quantity, new_quantity, reason, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, adjustment.TenantID, adjustment.ProductID, adjustment.UserID,
		adjustment.PreviousQuantity, adjustment.NewQuantity,
		adjustment.Reason, adjustment.Notes, adjustment.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create adjustment record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated product
	product.StockQuantity = newQuantity
	return product, nil
}

// GetAdjustmentHistory retrieves stock adjustment history for a product
func (s *InventoryService) GetAdjustmentHistory(ctx context.Context, productID uuid.UUID, limit, offset int) ([]*models.StockAdjustment, int, error) {
	return s.stockRepo.GetAdjustmentHistory(ctx, productID, limit, offset)
}

// GetAdjustmentsByFilters retrieves stock adjustments for a tenant with optional filters
func (s *InventoryService) GetAdjustmentsByFilters(ctx context.Context, tenantID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*models.StockAdjustment, int, error) {
	return s.stockRepo.GetAdjustmentsByTenant(ctx, tenantID, filters, limit, offset)
}
