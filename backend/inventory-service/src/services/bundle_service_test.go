package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
	"github.com/shopspring/decimal"
)

func TestNewBundleCostResponseSumsProductRecipeCosts(t *testing.T) {
	bundleID := uuid.New()
	itemOneID := uuid.New()
	itemTwoID := uuid.New()
	productOneID := uuid.New()
	productTwoID := uuid.New()
	recipeOneID := uuid.New()
	recipeTwoID := uuid.New()

	response := newBundleCostResponse(&models.BundleAggregate{
		Bundle: models.Bundle{
			ID:           bundleID,
			SKU:          "BNDL-001",
			Name:         "Breakfast Set",
			SellingPrice: decimal.NewFromInt(60),
		},
	}, []bundleRecipeCost{
		{
			item: models.BundleItem{
				ID:          itemOneID,
				ProductID:   productOneID,
				ProductName: "Coffee",
				ProductSKU:  "COF",
				Quantity:    decimal.NewFromInt(2),
			},
			recipe: &models.RecipeAggregate{Recipe: models.Recipe{ID: recipeOneID, Version: 3}},
			unitCost: models.RecipeCostResponse{
				MaterialCost:           models.NewNumber(decimal.NewFromInt(10)),
				FixedOverheadCost:      models.NewNumber(decimal.NewFromInt(2)),
				PercentageOverheadCost: models.NewNumber(decimal.NewFromInt(1)),
				TotalCOGS:              models.NewNumber(decimal.NewFromInt(13)),
			},
		},
		{
			item: models.BundleItem{
				ID:          itemTwoID,
				ProductID:   productTwoID,
				ProductName: "Toast",
				ProductSKU:  "TST",
				Quantity:    decimal.NewFromInt(3),
			},
			recipe: &models.RecipeAggregate{Recipe: models.Recipe{ID: recipeTwoID, Version: 1}},
			unitCost: models.RecipeCostResponse{
				MaterialCost:           models.NewNumber(decimal.NewFromInt(5)),
				FixedOverheadCost:      models.NewNumber(decimal.Zero),
				PercentageOverheadCost: models.NewNumber(decimal.Zero),
				TotalCOGS:              models.NewNumber(decimal.NewFromInt(5)),
			},
		},
	})

	if response.BundleID != bundleID {
		t.Fatalf("bundle_id = %s, want %s", response.BundleID, bundleID)
	}
	if len(response.Items) != 2 {
		t.Fatalf("items len = %d, want 2", len(response.Items))
	}
	assertDecimal(t, response.Items[0].LineTotalCOGS.Decimal, "26")
	assertDecimal(t, response.Items[1].LineTotalCOGS.Decimal, "15")
	assertDecimal(t, response.MaterialCost.Decimal, "35")
	assertDecimal(t, response.FixedOverheadCost.Decimal, "4")
	assertDecimal(t, response.PercentageOverheadCost.Decimal, "2")
	assertDecimal(t, response.TotalCOGS.Decimal, "41")
	assertDecimal(t, response.GrossProfit.Decimal, "19")
	assertDecimal(t, response.GrossMarginPercentage.Decimal, "31.67")
}

func TestBundleServiceCostUsesTenantTransactionAndActiveRecipes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.New()
	bundleID := uuid.New()
	itemOneID := uuid.New()
	itemTwoID := uuid.New()
	productOneID := uuid.New()
	productTwoID := uuid.New()
	recipeOneID := uuid.New()
	recipeTwoID := uuid.New()
	uomID := uuid.New()
	ingredientOneID := uuid.New()
	ingredientTwoID := uuid.New()
	overheadID := uuid.New()
	now := time.Now()

	mock.ExpectBegin()
	expectTenantContext(mock, tenantID)
	mock.ExpectQuery(`(?s)FROM bundles.*WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs(bundleID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "sku", "name", "description", "selling_price", "is_active", "created_at", "updated_at",
		}).AddRow(bundleID.String(), tenantID.String(), "BNDL-001", "Breakfast Set", nil, "60", true, now, now))
	mock.ExpectQuery(`(?s)FROM bundle_items bi.*WHERE bi\.tenant_id = \$1 AND bi\.bundle_id = \$2`).
		WithArgs(tenantID, bundleID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "bundle_id", "product_id", "name", "sku", "quantity", "created_at", "updated_at",
		}).
			AddRow(itemOneID.String(), tenantID.String(), bundleID.String(), productOneID.String(), "Coffee", "COF", "2", now, now).
			AddRow(itemTwoID.String(), tenantID.String(), bundleID.String(), productTwoID.String(), "Toast", "TST", "3", now, now))

	expectActiveRecipe(mock, tenantID, productOneID, recipeOneID, uomID, ingredientOneID, overheadID, now, "Coffee", "COF", "1", "10", "0", "1", "2", models.OverheadFixed)
	expectActiveRecipe(mock, tenantID, productTwoID, recipeTwoID, uomID, ingredientTwoID, uuid.Nil, now, "Toast", "TST", "1", "5", "0", "1", "0", "")

	mock.ExpectCommit()

	service := NewBundleService(repository.NewBundleRepository(db), repository.NewRecipeRepository(db))
	cost, err := service.Cost(context.Background(), tenantID, bundleID)
	if err != nil {
		t.Fatalf("Cost error: %v", err)
	}

	assertDecimal(t, cost.TotalCOGS.Decimal, "39.00")
	assertDecimal(t, cost.GrossProfit.Decimal, "21.00")
	assertDecimal(t, cost.GrossMarginPercentage.Decimal, "35.00")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestBundleServiceRejectsDuplicateBundleProducts(t *testing.T) {
	productID := uuid.New()
	service := NewBundleService(nil, nil)
	_, err := service.buildBundleItemsTx(context.Background(), nil, uuid.New(), uuid.New(), []models.CreateBundleItem{
		{ProductID: productID, Quantity: models.NewNumber(decimal.NewFromInt(1))},
		{ProductID: productID, Quantity: models.NewNumber(decimal.NewFromInt(2))},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func expectActiveRecipe(
	mock sqlmock.Sqlmock,
	tenantID uuid.UUID,
	productID uuid.UUID,
	recipeID uuid.UUID,
	uomID uuid.UUID,
	ingredientID uuid.UUID,
	overheadID uuid.UUID,
	now time.Time,
	productName string,
	productSKU string,
	yieldQuantity string,
	normalizedQuantity string,
	wastePercentage string,
	averageCost string,
	overheadAmount string,
	overheadType string,
) {
	mock.ExpectQuery(`(?s)FROM recipes.*WHERE tenant_id = \$1 AND product_id = \$2 AND is_active = TRUE`).
		WithArgs(tenantID, productID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "version", "yield_quantity", "is_active", "effective_from", "created_at", "updated_at",
		}).AddRow(recipeID.String(), tenantID.String(), productID.String(), 1, yieldQuantity, true, now, now, now))
	mock.ExpectQuery(`(?s)FROM products.*WHERE id = \$1 AND tenant_id = \$2 AND archived_at IS NULL`).
		WithArgs(productID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name", "sku", "selling_price"}).
			AddRow(productID.String(), tenantID.String(), productName, productSKU, "50"))
	mock.ExpectQuery(`(?s)FROM recipe_items ri.*WHERE ri\.tenant_id = \$1 AND ri\.recipe_id = \$2`).
		WithArgs(tenantID, recipeID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "recipe_id", "ingredient_id", "name", "quantity", "uom_id", "code",
			"normalized_quantity_base", "waste_percentage", "average_cost_per_base_unit", "created_at", "updated_at",
		}).AddRow(uuid.New().String(), tenantID.String(), recipeID.String(), ingredientID.String(), "Ingredient", normalizedQuantity, uomID.String(), "piece", normalizedQuantity, wastePercentage, averageCost, now, now))

	overheadRows := sqlmock.NewRows([]string{
		"id", "tenant_id", "recipe_id", "name", "calculation_type", "amount", "created_at", "updated_at",
	})
	if overheadType != "" {
		overheadRows.AddRow(overheadID.String(), tenantID.String(), recipeID.String(), "Overhead", overheadType, overheadAmount, now, now)
	}
	mock.ExpectQuery(`(?s)FROM recipe_overheads.*WHERE tenant_id = \$1 AND recipe_id = \$2`).
		WithArgs(tenantID, recipeID).
		WillReturnRows(overheadRows)
}
