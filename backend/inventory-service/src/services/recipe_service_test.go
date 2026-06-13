package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
	"github.com/shopspring/decimal"
)

func TestCalculateRecipeCost(t *testing.T) {
	aggregate := &models.RecipeAggregate{
		Recipe: models.Recipe{
			ID:            uuid.New(),
			TenantID:      uuid.New(),
			ProductID:     uuid.New(),
			Version:       1,
			YieldQuantity: decimal.RequireFromString("2"),
			IsActive:      true,
			EffectiveFrom: time.Now(),
		},
		Product: models.ProductSnapshot{
			ID:           uuid.New(),
			Name:         "Coffee",
			SKU:          "COF-001",
			SellingPrice: decimal.RequireFromString("50"),
		},
		Items: []models.RecipeItem{
			{
				ID:                     uuid.New(),
				IngredientID:           uuid.New(),
				Quantity:               decimal.RequireFromString("100"),
				NormalizedQuantityBase: decimal.RequireFromString("100"),
				WastePercentage:        decimal.RequireFromString("10"),
				AverageCostPerBaseUnit: decimal.RequireFromString("0.20"),
			},
			{
				ID:                     uuid.New(),
				IngredientID:           uuid.New(),
				Quantity:               decimal.RequireFromString("1"),
				NormalizedQuantityBase: decimal.RequireFromString("1"),
				WastePercentage:        decimal.RequireFromString("0"),
				AverageCostPerBaseUnit: decimal.RequireFromString("4"),
			},
		},
		Overheads: []models.RecipeOverhead{
			{Name: "Packaging", CalculationType: models.OverheadFixed, Amount: decimal.RequireFromString("3")},
			{Name: "Utilities", CalculationType: models.OverheadPercentage, Amount: decimal.RequireFromString("10")},
		},
	}

	cost := calculateRecipeCost(aggregate)

	assertDecimal(t, cost.MaterialCost.Decimal, "13.00")
	assertDecimal(t, cost.FixedOverheadCost.Decimal, "3.00")
	assertDecimal(t, cost.PercentageOverheadCost.Decimal, "1.30")
	assertDecimal(t, cost.TotalCOGS.Decimal, "17.30")
	assertDecimal(t, cost.GrossProfit.Decimal, "32.70")
	assertDecimal(t, cost.GrossMarginPercentage.Decimal, "65.40")
}

func TestNewRecipeResponseIncludesItemCosts(t *testing.T) {
	aggregate := &models.RecipeAggregate{
		Recipe: models.Recipe{
			ID:            uuid.New(),
			TenantID:      uuid.New(),
			ProductID:     uuid.New(),
			Version:       1,
			YieldQuantity: decimal.NewFromInt(1),
			IsActive:      true,
			EffectiveFrom: time.Now(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		Product: models.ProductSnapshot{
			ID:           uuid.New(),
			Name:         "Tea",
			SKU:          "TEA",
			SellingPrice: decimal.NewFromInt(20),
		},
		Items: []models.RecipeItem{
			{
				ID:                     uuid.New(),
				IngredientID:           uuid.New(),
				IngredientName:         "Leaves",
				Quantity:               decimal.NewFromInt(5),
				NormalizedQuantityBase: decimal.NewFromInt(5),
				WastePercentage:        decimal.NewFromInt(20),
				AverageCostPerBaseUnit: decimal.NewFromInt(2),
			},
		},
	}

	response := newRecipeResponse(aggregate)

	if len(response.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(response.Items))
	}
	assertDecimal(t, response.Items[0].IngredientCost.Decimal, "10")
	assertDecimal(t, response.Items[0].CostWithWaste.Decimal, "12")
	assertDecimal(t, response.Cost.TotalCOGS.Decimal, "12")
}

func TestRecipeServiceGetCostUsesTenantTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.New()
	productID := uuid.New()
	recipeID := uuid.New()
	uomID := uuid.New()
	itemOneID := uuid.New()
	itemTwoID := uuid.New()
	ingredientOneID := uuid.New()
	ingredientTwoID := uuid.New()
	overheadFixedID := uuid.New()
	overheadPercentID := uuid.New()
	now := time.Now()

	mock.ExpectBegin()
	expectTenantContext(mock, tenantID)
	mock.ExpectQuery(`(?s)FROM recipes.*WHERE tenant_id = \$1 AND product_id = \$2 AND is_active = TRUE`).
		WithArgs(tenantID, productID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "version", "yield_quantity", "is_active", "effective_from", "created_at", "updated_at",
		}).AddRow(recipeID.String(), tenantID.String(), productID.String(), 1, "2", true, now, now, now))
	mock.ExpectQuery(`(?s)FROM products.*WHERE id = \$1 AND tenant_id = \$2 AND archived_at IS NULL`).
		WithArgs(productID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name", "sku", "selling_price"}).
			AddRow(productID.String(), tenantID.String(), "Coffee", "COF-001", "50"))
	mock.ExpectQuery(`(?s)FROM recipe_items ri.*WHERE ri\.tenant_id = \$1 AND ri\.recipe_id = \$2`).
		WithArgs(tenantID, recipeID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "recipe_id", "ingredient_id", "name", "quantity", "uom_id", "code",
			"normalized_quantity_base", "waste_percentage", "average_cost_per_base_unit", "created_at", "updated_at",
		}).
			AddRow(itemOneID.String(), tenantID.String(), recipeID.String(), ingredientOneID.String(), "Beans", "100", uomID.String(), "g", "100", "10", "0.20", now, now).
			AddRow(itemTwoID.String(), tenantID.String(), recipeID.String(), ingredientTwoID.String(), "Milk", "1", uomID.String(), "l", "1", "0", "4", now, now))
	mock.ExpectQuery(`(?s)FROM recipe_overheads.*WHERE tenant_id = \$1 AND recipe_id = \$2`).
		WithArgs(tenantID, recipeID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "recipe_id", "name", "calculation_type", "amount", "created_at", "updated_at",
		}).
			AddRow(overheadFixedID.String(), tenantID.String(), recipeID.String(), "Packaging", models.OverheadFixed, "3", now, now).
			AddRow(overheadPercentID.String(), tenantID.String(), recipeID.String(), "Utilities", models.OverheadPercentage, "10", now, now))
	mock.ExpectCommit()

	service := NewRecipeService(repository.NewRecipeRepository(db), nil, nil)
	cost, err := service.GetCost(context.Background(), tenantID, productID)
	if err != nil {
		t.Fatalf("GetCost error: %v", err)
	}

	assertDecimal(t, cost.MaterialCost.Decimal, "13.00")
	assertDecimal(t, cost.TotalCOGS.Decimal, "17.30")
	assertDecimal(t, cost.GrossMarginPercentage.Decimal, "65.40")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func assertDecimal(t *testing.T, got decimal.Decimal, want string) {
	t.Helper()
	expected := decimal.RequireFromString(want)
	if !got.Equal(expected) {
		t.Fatalf("decimal = %s, want %s", got, expected)
	}
}
