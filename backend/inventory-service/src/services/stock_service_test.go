package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
	"github.com/shopspring/decimal"
)

func TestStockMutationDuplicateIdempotencyKeyUsesTenantTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.New()
	ingredientID := uuid.New()
	baseUOMID := uuid.New()
	fromUOMID := uuid.New()
	conversionID := uuid.New()
	idempotencyKey := "purchase-1"
	now := time.Now()

	mock.ExpectBegin()
	expectTenantContext(mock, tenantID)
	mock.ExpectQuery(`(?s)SELECT i\.id, i\.tenant_id.*FROM ingredients i.*WHERE i\.id = \$1 AND i\.tenant_id = \$2.*FOR UPDATE`).
		WithArgs(ingredientID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "outlet_id", "sku", "name", "base_uom_id", "code",
			"current_stock_base", "average_cost_per_base_unit", "minimum_stock_base",
			"track_stock", "is_active", "created_at", "updated_at",
		}).AddRow(
			ingredientID.String(), tenantID.String(), nil, "FLOUR", "Flour", baseUOMID.String(), "g",
			"10", "2.00", "1", true, true, now, now,
		))
	mock.ExpectQuery(`(?s)SELECT id, tenant_id, ingredient_id, from_uom_id, to_uom_id, multiplier, created_at, updated_at.*FROM ingredient_uom_conversions.*WHERE tenant_id = \$1 AND ingredient_id = \$2 AND from_uom_id = \$3 AND to_uom_id = \$4`).
		WithArgs(tenantID, ingredientID, fromUOMID, baseUOMID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "ingredient_id", "from_uom_id", "to_uom_id", "multiplier", "created_at", "updated_at",
		}).AddRow(
			conversionID.String(), tenantID.String(), ingredientID.String(), fromUOMID.String(), baseUOMID.String(), "2", now, now,
		))
	mock.ExpectQuery(`(?s)INSERT INTO stock_movements`).
		WithArgs(
			tenantID, nil, ingredientID, models.MovementPurchaseIn,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			nil, nil, sqlmock.AnyArg(), nil, sqlmock.AnyArg(), nil,
		).
		WillReturnError(&pq.Error{Code: "23505"})
	mock.ExpectRollback()

	conversionRepo := repository.NewConversionRepository(db)
	uomRepo := repository.NewUOMRepository(db)
	ingredientRepo := repository.NewIngredientRepository(db)
	conversionService := NewConversionService(conversionRepo, uomRepo, ingredientRepo)
	service := NewStockService(repository.NewStockRepository(db), conversionService)

	_, err = service.PurchaseIn(context.Background(), tenantID, nil, models.StockMutationRequest{
		IngredientID:   ingredientID,
		Quantity:       models.NewNumber(decimal.NewFromInt(5)),
		UOMID:          fromUOMID,
		UnitCost:       models.NewNumber(decimal.RequireFromString("3.50")),
		IdempotencyKey: &idempotencyKey,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("PurchaseIn error = %v, want ErrConflict", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
