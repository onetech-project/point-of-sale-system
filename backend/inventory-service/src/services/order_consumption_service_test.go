package services

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
)

func TestOrderConsumptionSkipsAlreadySnapshottedItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.New()
	orderID := uuid.New()
	orderItemID := uuid.New()
	productID := uuid.New()

	mock.ExpectBegin()
	expectTenantContext(mock, tenantID)
	mock.ExpectQuery(`(?s)SELECT EXISTS.*FROM order_item_cost_snapshots.*WHERE tenant_id = \$1 AND order_item_id = \$2`).
		WithArgs(tenantID, orderItemID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectCommit()

	service := NewOrderConsumptionService(
		db,
		repository.NewRecipeRepository(db),
		repository.NewBundleRepository(db),
		repository.NewStockRepository(db),
		repository.NewSnapshotRepository(db),
	)

	err = service.ProcessFulfilledOrder(context.Background(), models.OrderLifecycleEvent{
		EventID:   "order.fulfilled:" + orderID.String(),
		EventType: models.OrderFulfilledEventType,
		TenantID:  tenantID,
		OrderID:   orderID,
		Items: []models.OrderLifecycleItemEvent{{
			OrderItemID: orderItemID,
			ItemType:    models.OrderItemTypeProduct,
			ProductID:   &productID,
			Quantity:    1,
			UnitPrice:   10000,
			TotalPrice:  10000,
		}},
	})
	if err != nil {
		t.Fatalf("ProcessFulfilledOrder error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestOrderConsumptionCreatesSaleMovementAndSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.New()
	orderID := uuid.New()
	orderItemID := uuid.New()
	productID := uuid.New()
	recipeID := uuid.New()
	recipeItemID := uuid.New()
	ingredientID := uuid.New()
	uomID := uuid.New()
	now := time.Now()

	mock.ExpectBegin()
	expectTenantContext(mock, tenantID)
	mock.ExpectQuery(`(?s)SELECT EXISTS.*FROM order_item_cost_snapshots.*WHERE tenant_id = \$1 AND order_item_id = \$2`).
		WithArgs(tenantID, orderItemID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(`(?s)FROM recipes.*WHERE tenant_id = \$1 AND product_id = \$2 AND is_active = TRUE`).
		WithArgs(tenantID, productID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "version", "yield_quantity", "is_active", "effective_from", "created_at", "updated_at",
		}).AddRow(recipeID.String(), tenantID.String(), productID.String(), 1, "1", true, now, now, now))
	mock.ExpectQuery(`(?s)FROM products.*WHERE id = \$1 AND tenant_id = \$2 AND archived_at IS NULL`).
		WithArgs(productID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name", "sku", "selling_price"}).
			AddRow(productID.String(), tenantID.String(), "Coffee", "COF-001", "10000"))
	mock.ExpectQuery(`(?s)FROM recipe_items ri.*WHERE ri\.tenant_id = \$1 AND ri\.recipe_id = \$2`).
		WithArgs(tenantID, recipeID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "recipe_id", "ingredient_id", "name", "quantity", "uom_id", "code",
			"normalized_quantity_base", "waste_percentage", "average_cost_per_base_unit", "created_at", "updated_at",
		}).AddRow(recipeItemID.String(), tenantID.String(), recipeID.String(), ingredientID.String(), "Beans", "2", uomID.String(), "g", "2", "0", "100", now, now))
	mock.ExpectQuery(`(?s)FROM recipe_overheads.*WHERE tenant_id = \$1 AND recipe_id = \$2`).
		WithArgs(tenantID, recipeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "recipe_id", "name", "calculation_type", "amount", "created_at", "updated_at"}))
	mock.ExpectQuery(`(?s)SELECT i\.id, i\.tenant_id.*FROM ingredients i.*WHERE i\.id = \$1 AND i\.tenant_id = \$2.*FOR UPDATE`).
		WithArgs(ingredientID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "outlet_id", "sku", "name", "base_uom_id", "code",
			"current_stock_base", "average_cost_per_base_unit", "minimum_stock_base",
			"track_stock", "is_active", "created_at", "updated_at",
		}).AddRow(ingredientID.String(), tenantID.String(), nil, "BEANS", "Beans", uomID.String(), "g", "20", "100", "1", true, true, now, now))
	mock.ExpectQuery(`(?s)INSERT INTO stock_movements`).
		WithArgs(
			tenantID, nil, ingredientID, models.MovementSaleConsumption,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			models.OrderFulfilledEventType, orderID.String(), sqlmock.AnyArg(),
			"order fulfillment consumption", now, nil,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uuid.New().String(), now))
	mock.ExpectExec(`(?s)UPDATE ingredients.*SET current_stock_base = \$1, average_cost_per_base_unit = \$2.*WHERE id = \$3 AND tenant_id = \$4`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), ingredientID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)INSERT INTO order_item_cost_snapshots`).
		WithArgs(
			tenantID, orderID, orderItemID, models.OrderItemTypeProduct,
			sqlmock.AnyArg(), nil, sqlmock.AnyArg(), 1,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uuid.New().String(), now))
	mock.ExpectCommit()

	service := NewOrderConsumptionService(
		db,
		repository.NewRecipeRepository(db),
		repository.NewBundleRepository(db),
		repository.NewStockRepository(db),
		repository.NewSnapshotRepository(db),
	)

	err = service.ProcessFulfilledOrder(context.Background(), models.OrderLifecycleEvent{
		EventID:        "order.fulfilled:" + orderID.String(),
		EventType:      models.OrderFulfilledEventType,
		TenantID:       tenantID,
		OrderID:        orderID,
		OrderReference: "ORD-1",
		OrderType:      "offline",
		OccurredAt:     now,
		Items: []models.OrderLifecycleItemEvent{{
			OrderItemID: orderItemID,
			ItemType:    models.OrderItemTypeProduct,
			ProductID:   &productID,
			ProductName: "Coffee",
			Quantity:    3,
			UnitPrice:   10000,
			TotalPrice:  30000,
		}},
	})
	if err != nil {
		t.Fatalf("ProcessFulfilledOrder error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestOrderConsumptionExpandsBundleIntoProductRecipeConsumption(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	tenantID := uuid.New()
	orderID := uuid.New()
	orderItemID := uuid.New()
	bundleID := uuid.New()
	bundleItemID := uuid.New()
	productID := uuid.New()
	recipeID := uuid.New()
	recipeItemID := uuid.New()
	ingredientID := uuid.New()
	uomID := uuid.New()
	now := time.Now()

	mock.ExpectBegin()
	expectTenantContext(mock, tenantID)
	mock.ExpectQuery(`(?s)SELECT EXISTS.*FROM order_item_cost_snapshots.*WHERE tenant_id = \$1 AND order_item_id = \$2`).
		WithArgs(tenantID, orderItemID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(`(?s)FROM bundles.*WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs(bundleID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "sku", "name", "description", "selling_price", "is_active", "created_at", "updated_at",
		}).AddRow(bundleID.String(), tenantID.String(), "BNDL", "Breakfast", nil, "25000", true, now, now))
	mock.ExpectQuery(`(?s)FROM bundle_items bi.*WHERE bi\.tenant_id = \$1 AND bi\.bundle_id = \$2`).
		WithArgs(tenantID, bundleID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "bundle_id", "product_id", "name", "sku", "quantity", "created_at", "updated_at",
		}).AddRow(bundleItemID.String(), tenantID.String(), bundleID.String(), productID.String(), "Coffee", "COF", "2", now, now))
	mock.ExpectQuery(`(?s)FROM recipes.*WHERE tenant_id = \$1 AND product_id = \$2 AND is_active = TRUE`).
		WithArgs(tenantID, productID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "product_id", "version", "yield_quantity", "is_active", "effective_from", "created_at", "updated_at",
		}).AddRow(recipeID.String(), tenantID.String(), productID.String(), 1, "1", true, now, now, now))
	mock.ExpectQuery(`(?s)FROM products.*WHERE id = \$1 AND tenant_id = \$2 AND archived_at IS NULL`).
		WithArgs(productID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "name", "sku", "selling_price"}).
			AddRow(productID.String(), tenantID.String(), "Coffee", "COF", "10000"))
	mock.ExpectQuery(`(?s)FROM recipe_items ri.*WHERE ri\.tenant_id = \$1 AND ri\.recipe_id = \$2`).
		WithArgs(tenantID, recipeID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "recipe_id", "ingredient_id", "name", "quantity", "uom_id", "code",
			"normalized_quantity_base", "waste_percentage", "average_cost_per_base_unit", "created_at", "updated_at",
		}).AddRow(recipeItemID.String(), tenantID.String(), recipeID.String(), ingredientID.String(), "Beans", "2", uomID.String(), "g", "2", "0", "100", now, now))
	mock.ExpectQuery(`(?s)FROM recipe_overheads.*WHERE tenant_id = \$1 AND recipe_id = \$2`).
		WithArgs(tenantID, recipeID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "recipe_id", "name", "calculation_type", "amount", "created_at", "updated_at"}))
	mock.ExpectQuery(`(?s)SELECT i\.id, i\.tenant_id.*FROM ingredients i.*WHERE i\.id = \$1 AND i\.tenant_id = \$2.*FOR UPDATE`).
		WithArgs(ingredientID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "tenant_id", "outlet_id", "sku", "name", "base_uom_id", "code",
			"current_stock_base", "average_cost_per_base_unit", "minimum_stock_base",
			"track_stock", "is_active", "created_at", "updated_at",
		}).AddRow(ingredientID.String(), tenantID.String(), nil, "BEANS", "Beans", uomID.String(), "g", "20", "100", "1", true, true, now, now))
	mock.ExpectQuery(`(?s)INSERT INTO stock_movements`).
		WithArgs(
			tenantID, nil, ingredientID, models.MovementSaleConsumption,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			models.OrderFulfilledEventType, orderID.String(), sqlmock.AnyArg(),
			"order fulfillment consumption", now, nil,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uuid.New().String(), now))
	mock.ExpectExec(`(?s)UPDATE ingredients.*SET current_stock_base = \$1, average_cost_per_base_unit = \$2.*WHERE id = \$3 AND tenant_id = \$4`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), ingredientID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`(?s)INSERT INTO order_item_cost_snapshots`).
		WithArgs(
			tenantID, orderID, orderItemID, models.OrderItemTypeBundle,
			nil, sqlmock.AnyArg(), nil, 0,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uuid.New().String(), now))
	mock.ExpectCommit()

	service := NewOrderConsumptionService(
		db,
		repository.NewRecipeRepository(db),
		repository.NewBundleRepository(db),
		repository.NewStockRepository(db),
		repository.NewSnapshotRepository(db),
	)

	err = service.ProcessFulfilledOrder(context.Background(), models.OrderLifecycleEvent{
		EventID:        "order.fulfilled:" + orderID.String(),
		EventType:      models.OrderFulfilledEventType,
		TenantID:       tenantID,
		OrderID:        orderID,
		OrderReference: "ORD-2",
		OrderType:      "offline",
		OccurredAt:     now,
		Items: []models.OrderLifecycleItemEvent{{
			OrderItemID: orderItemID,
			ItemType:    models.OrderItemTypeBundle,
			BundleID:    &bundleID,
			ProductName: "Breakfast",
			Quantity:    3,
			UnitPrice:   25000,
			TotalPrice:  75000,
		}},
	})
	if err != nil {
		t.Fatalf("ProcessFulfilledOrder error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
