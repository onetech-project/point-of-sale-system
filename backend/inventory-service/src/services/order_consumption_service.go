package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
	"github.com/shopspring/decimal"
)

type OrderConsumptionService struct {
	db           *sql.DB
	recipeRepo   *repository.RecipeRepository
	bundleRepo   *repository.BundleRepository
	stockRepo    *repository.StockRepository
	snapshotRepo *repository.SnapshotRepository
}

type consumedRecipeCost struct {
	ProductID                  uuid.UUID       `json:"product_id"`
	ProductName                string          `json:"product_name"`
	ProductSKU                 string          `json:"product_sku"`
	BundleItemID               *uuid.UUID      `json:"bundle_item_id,omitempty"`
	Quantity                   decimal.Decimal `json:"-"`
	RecipeID                   uuid.UUID       `json:"recipe_id"`
	RecipeVersion              int             `json:"recipe_version"`
	MaterialCost               decimal.Decimal `json:"-"`
	FixedOverheadCost          decimal.Decimal `json:"-"`
	PercentageOverheadCost     decimal.Decimal `json:"-"`
	TotalCOGS                  decimal.Decimal `json:"-"`
	GrossMarginPercentageBasis decimal.Decimal `json:"-"`
}

type consumedRecipeCostJSON struct {
	ProductID              uuid.UUID     `json:"product_id"`
	ProductName            string        `json:"product_name"`
	ProductSKU             string        `json:"product_sku"`
	BundleItemID           *uuid.UUID    `json:"bundle_item_id,omitempty"`
	Quantity               models.Number `json:"quantity"`
	RecipeID               uuid.UUID     `json:"recipe_id"`
	RecipeVersion          int           `json:"recipe_version"`
	MaterialCost           models.Number `json:"material_cost"`
	FixedOverheadCost      models.Number `json:"fixed_overhead_cost"`
	PercentageOverheadCost models.Number `json:"percentage_overhead_cost"`
	TotalCOGS              models.Number `json:"total_cogs"`
}

func NewOrderConsumptionService(
	db *sql.DB,
	recipeRepo *repository.RecipeRepository,
	bundleRepo *repository.BundleRepository,
	stockRepo *repository.StockRepository,
	snapshotRepo *repository.SnapshotRepository,
) *OrderConsumptionService {
	return &OrderConsumptionService{
		db:           db,
		recipeRepo:   recipeRepo,
		bundleRepo:   bundleRepo,
		stockRepo:    stockRepo,
		snapshotRepo: snapshotRepo,
	}
}

func (s *OrderConsumptionService) ProcessFulfilledOrder(ctx context.Context, event models.OrderLifecycleEvent) error {
	if event.EventType != "" && event.EventType != models.OrderFulfilledEventType {
		return nil
	}
	if event.EventID == "" {
		return fmt.Errorf("%w: event_id is required", ErrInvalidInput)
	}
	if event.TenantID == uuid.Nil {
		return fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if event.OrderID == uuid.Nil {
		return fmt.Errorf("%w: order_id is required", ErrInvalidInput)
	}
	if len(event.Items) == 0 {
		return fmt.Errorf("%w: fulfilled order event must include items", ErrInvalidInput)
	}

	occurredAt := event.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin order consumption transaction: %w", err)
	}
	defer tx.Rollback()

	if err := repository.SetTenantContextTx(ctx, tx, event.TenantID); err != nil {
		return err
	}

	for i := range event.Items {
		item := event.Items[i]
		if err := s.processFulfilledItemTx(ctx, tx, event, item, occurredAt); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit order consumption transaction: %w", err)
	}
	return nil
}

func (s *OrderConsumptionService) processFulfilledItemTx(
	ctx context.Context,
	tx *sql.Tx,
	event models.OrderLifecycleEvent,
	item models.OrderLifecycleItemEvent,
	occurredAt time.Time,
) error {
	if err := validateFulfilledItem(item); err != nil {
		return err
	}

	exists, err := s.snapshotRepo.ExistsByOrderItemTx(ctx, tx, event.TenantID, item.OrderItemID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	switch itemType(item) {
	case models.OrderItemTypeProduct:
		return s.processProductItemTx(ctx, tx, event, item, occurredAt)
	case models.OrderItemTypeBundle:
		return s.processBundleItemTx(ctx, tx, event, item, occurredAt)
	default:
		return fmt.Errorf("%w: item_type must be product or bundle", ErrInvalidInput)
	}
}

func (s *OrderConsumptionService) processProductItemTx(
	ctx context.Context,
	tx *sql.Tx,
	event models.OrderLifecycleEvent,
	item models.OrderLifecycleItemEvent,
	occurredAt time.Time,
) error {
	if item.ProductID == nil || *item.ProductID == uuid.Nil {
		return fmt.Errorf("%w: product_id is required for product order item", ErrInvalidInput)
	}
	aggregate, err := s.recipeRepo.GetActiveTx(ctx, tx, event.TenantID, *item.ProductID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: active recipe not found for product %s", ErrNotFound, *item.ProductID)
	}
	if err != nil {
		return err
	}

	quantity := decimal.NewFromInt(int64(item.Quantity))
	component, err := s.consumeRecipeTx(ctx, tx, event, item, aggregate, quantity, fmt.Sprintf("%s:%s:%s", event.EventID, item.OrderItemID, aggregate.Recipe.ID), nil, occurredAt)
	if err != nil {
		return err
	}
	componentCosts, err := marshalComponentCosts([]consumedRecipeCost{*component})
	if err != nil {
		return err
	}

	recipeID := aggregate.Recipe.ID
	return s.createCostSnapshotTx(ctx, tx, event, item, models.OrderItemTypeProduct, item.ProductID, nil, &recipeID, aggregate.Recipe.Version, quantity, []consumedRecipeCost{*component}, componentCosts)
}

func (s *OrderConsumptionService) processBundleItemTx(
	ctx context.Context,
	tx *sql.Tx,
	event models.OrderLifecycleEvent,
	item models.OrderLifecycleItemEvent,
	occurredAt time.Time,
) error {
	if item.BundleID == nil || *item.BundleID == uuid.Nil {
		return fmt.Errorf("%w: bundle_id is required for bundle order item", ErrInvalidInput)
	}
	if s.bundleRepo == nil {
		return fmt.Errorf("%w: bundle repository is unavailable", ErrInvalidInput)
	}

	bundle, err := s.bundleRepo.FindByIDTx(ctx, tx, event.TenantID, *item.BundleID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: bundle not found %s", ErrNotFound, *item.BundleID)
	}
	if err != nil {
		return err
	}
	if !bundle.Bundle.IsActive {
		return fmt.Errorf("%w: bundle is inactive", ErrInvalidInput)
	}
	if len(bundle.Items) == 0 {
		return fmt.Errorf("%w: bundle has no items", ErrInvalidInput)
	}

	orderQuantity := decimal.NewFromInt(int64(item.Quantity))
	components := make([]consumedRecipeCost, 0, len(bundle.Items))
	for i := range bundle.Items {
		bundleItem := bundle.Items[i]
		recipe, err := s.recipeRepo.GetActiveTx(ctx, tx, event.TenantID, bundleItem.ProductID)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: active recipe not found for bundle product %s", ErrNotFound, bundleItem.ProductID)
		}
		if err != nil {
			return err
		}
		componentQuantity := orderQuantity.Mul(bundleItem.Quantity)
		prefix := fmt.Sprintf("%s:%s:%s:%s", event.EventID, item.OrderItemID, bundleItem.ID, recipe.Recipe.ID)
		component, err := s.consumeRecipeTx(ctx, tx, event, item, recipe, componentQuantity, prefix, &bundleItem.ID, occurredAt)
		if err != nil {
			return err
		}
		components = append(components, *component)
	}

	componentCosts, err := marshalComponentCosts(components)
	if err != nil {
		return err
	}
	return s.createCostSnapshotTx(ctx, tx, event, item, models.OrderItemTypeBundle, nil, item.BundleID, nil, 0, orderQuantity, components, componentCosts)
}

func (s *OrderConsumptionService) consumeRecipeTx(
	ctx context.Context,
	tx *sql.Tx,
	event models.OrderLifecycleEvent,
	item models.OrderLifecycleItemEvent,
	aggregate *models.RecipeAggregate,
	quantity decimal.Decimal,
	idempotencyPrefix string,
	bundleItemID *uuid.UUID,
	occurredAt time.Time,
) (*consumedRecipeCost, error) {
	recipeForCost := *aggregate
	recipeForCost.Items = make([]models.RecipeItem, len(aggregate.Items))
	copy(recipeForCost.Items, aggregate.Items)

	for i := range recipeForCost.Items {
		recipeItem := &recipeForCost.Items[i]
		ingredient, err := s.stockRepo.LockIngredientTx(ctx, tx, event.TenantID, recipeItem.IngredientID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: ingredient not found for recipe item %s", ErrInvalidInput, recipeItem.ID)
		}
		if err != nil {
			return nil, err
		}
		if !ingredient.IsActive {
			return nil, ErrInactiveIngredient
		}

		recipeItem.AverageCostPerBaseUnit = ingredient.AverageCostPerBaseUnit
		consumptionQuantity := recipeItem.NormalizedQuantityBase.Mul(quantity)
		newStock := ingredient.CurrentStockBase.Sub(consumptionQuantity)
		if newStock.IsNegative() {
			return nil, fmt.Errorf("%w: ingredient %s does not have enough stock", ErrInsufficientStock, ingredient.ID)
		}

		referenceType := models.OrderFulfilledEventType
		referenceID := event.OrderID.String()
		idempotencyKey := fmt.Sprintf("%s:%s", idempotencyPrefix, recipeItem.ID)
		reason := "order fulfillment consumption"
		unitCost := ingredient.AverageCostPerBaseUnit
		totalCost := consumptionQuantity.Mul(unitCost).Round(2)
		movement := &models.StockMovement{
			TenantID:       event.TenantID,
			IngredientID:   recipeItem.IngredientID,
			MovementType:   models.MovementSaleConsumption,
			QuantityBase:   consumptionQuantity.Neg(),
			UnitCost:       unitCost,
			TotalCost:      totalCost,
			ReferenceType:  &referenceType,
			ReferenceID:    &referenceID,
			IdempotencyKey: &idempotencyKey,
			Reason:         &reason,
			OccurredAt:     occurredAt,
		}

		if err := s.stockRepo.CreateMovementTx(ctx, tx, movement); err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				return nil, fmt.Errorf("%w: duplicate sale consumption movement", ErrConflict)
			}
			return nil, err
		}
		if err := s.stockRepo.UpdateIngredientStockTx(ctx, tx, event.TenantID, recipeItem.IngredientID, newStock, unitCost); err != nil {
			return nil, err
		}
	}

	cost := calculateRecipeCost(&recipeForCost)
	return &consumedRecipeCost{
		ProductID:              aggregate.Product.ID,
		ProductName:            aggregate.Product.Name,
		ProductSKU:             aggregate.Product.SKU,
		BundleItemID:           bundleItemID,
		Quantity:               quantity,
		RecipeID:               aggregate.Recipe.ID,
		RecipeVersion:          aggregate.Recipe.Version,
		MaterialCost:           cost.MaterialCost.Decimal.Mul(quantity).Round(2),
		FixedOverheadCost:      cost.FixedOverheadCost.Decimal.Mul(quantity).Round(2),
		PercentageOverheadCost: cost.PercentageOverheadCost.Decimal.Mul(quantity).Round(2),
		TotalCOGS:              cost.TotalCOGS.Decimal.Mul(quantity).Round(2),
	}, nil
}

func (s *OrderConsumptionService) createCostSnapshotTx(
	ctx context.Context,
	tx *sql.Tx,
	event models.OrderLifecycleEvent,
	item models.OrderLifecycleItemEvent,
	itemType string,
	productID *uuid.UUID,
	bundleID *uuid.UUID,
	recipeID *uuid.UUID,
	recipeVersion int,
	quantity decimal.Decimal,
	components []consumedRecipeCost,
	componentCosts json.RawMessage,
) error {
	materialCost := decimal.Zero
	fixedOverheadCost := decimal.Zero
	percentageOverheadCost := decimal.Zero
	totalCOGS := decimal.Zero
	for i := range components {
		materialCost = materialCost.Add(components[i].MaterialCost)
		fixedOverheadCost = fixedOverheadCost.Add(components[i].FixedOverheadCost)
		percentageOverheadCost = percentageOverheadCost.Add(components[i].PercentageOverheadCost)
		totalCOGS = totalCOGS.Add(components[i].TotalCOGS)
	}
	materialCost = materialCost.Round(2)
	fixedOverheadCost = fixedOverheadCost.Round(2)
	percentageOverheadCost = percentageOverheadCost.Round(2)
	totalCOGS = totalCOGS.Round(2)
	sellingPrice := decimal.NewFromInt(int64(item.TotalPrice)).Round(2)
	grossProfit := sellingPrice.Sub(totalCOGS).Round(2)
	grossMargin := decimal.Zero
	if sellingPrice.GreaterThan(decimal.Zero) {
		grossMargin = grossProfit.Div(sellingPrice).Mul(decimal.NewFromInt(100)).Round(2)
	}

	pricingSnapshot, err := orderItemPricingSnapshot(event, item, itemType)
	if err != nil {
		return err
	}

	snapshot := &models.OrderItemCostSnapshot{
		TenantID:               event.TenantID,
		OrderID:                event.OrderID,
		OrderItemID:            item.OrderItemID,
		ItemType:               itemType,
		ProductID:              productID,
		BundleID:               bundleID,
		RecipeID:               recipeID,
		RecipeVersion:          recipeVersion,
		Quantity:               quantity,
		MaterialCost:           materialCost,
		FixedOverheadCost:      fixedOverheadCost,
		PercentageOverheadCost: percentageOverheadCost,
		TotalCOGS:              totalCOGS,
		SellingPrice:           sellingPrice,
		GrossProfit:            grossProfit,
		GrossMarginPercentage:  grossMargin,
		DiscountAmount:         decimal.NewFromInt(int64(item.DiscountAmount)).Round(2),
		PricingSnapshot:        pricingSnapshot,
		ComponentCosts:         componentCosts,
	}
	if err := s.snapshotRepo.CreateTx(ctx, tx, snapshot); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return fmt.Errorf("%w: duplicate order item cost snapshot", ErrConflict)
		}
		return err
	}
	return nil
}

func validateFulfilledItem(item models.OrderLifecycleItemEvent) error {
	if item.OrderItemID == uuid.Nil {
		return fmt.Errorf("%w: order_item_id is required", ErrInvalidInput)
	}
	if item.Quantity <= 0 {
		return fmt.Errorf("%w: item quantity must be greater than zero", ErrInvalidInput)
	}
	if item.UnitPrice < 0 || item.TotalPrice < 0 || item.ListUnitPrice < 0 {
		return fmt.Errorf("%w: item price cannot be negative", ErrInvalidInput)
	}
	return nil
}

func itemType(item models.OrderLifecycleItemEvent) string {
	if item.ItemType == "" {
		return models.OrderItemTypeProduct
	}
	return item.ItemType
}

func marshalComponentCosts(components []consumedRecipeCost) (json.RawMessage, error) {
	payload := make([]consumedRecipeCostJSON, 0, len(components))
	for i := range components {
		component := components[i]
		payload = append(payload, consumedRecipeCostJSON{
			ProductID:              component.ProductID,
			ProductName:            component.ProductName,
			ProductSKU:             component.ProductSKU,
			BundleItemID:           component.BundleItemID,
			Quantity:               models.NewNumber(component.Quantity),
			RecipeID:               component.RecipeID,
			RecipeVersion:          component.RecipeVersion,
			MaterialCost:           models.NewNumber(component.MaterialCost),
			FixedOverheadCost:      models.NewNumber(component.FixedOverheadCost),
			PercentageOverheadCost: models.NewNumber(component.PercentageOverheadCost),
			TotalCOGS:              models.NewNumber(component.TotalCOGS),
		})
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal component costs: %w", err)
	}
	return json.RawMessage(data), nil
}

func orderItemPricingSnapshot(event models.OrderLifecycleEvent, item models.OrderLifecycleItemEvent, itemType string) (json.RawMessage, error) {
	payload := map[string]interface{}{
		"event_id":            event.EventID,
		"order_reference":     event.OrderReference,
		"order_type":          event.OrderType,
		"item_type":           itemType,
		"product_id":          item.ProductID,
		"bundle_id":           item.BundleID,
		"product_name":        item.ProductName,
		"quantity":            item.Quantity,
		"list_unit_price":     item.ListUnitPrice,
		"unit_price":          item.UnitPrice,
		"total_price":         item.TotalPrice,
		"discount_rule_id":    item.DiscountRuleID,
		"discount_type":       item.DiscountType,
		"discount_value":      item.DiscountValue,
		"discount_amount":     item.DiscountAmount,
		"order_item_snapshot": item.PricingSnapshot,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pricing snapshot: %w", err)
	}
	return json.RawMessage(data), nil
}
