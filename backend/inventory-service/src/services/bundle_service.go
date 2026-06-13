package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
	"github.com/shopspring/decimal"
)

type BundleService struct {
	repo       *repository.BundleRepository
	recipeRepo *repository.RecipeRepository
}

type bundleRecipeCost struct {
	item     models.BundleItem
	recipe   *models.RecipeAggregate
	unitCost models.RecipeCostResponse
}

func NewBundleService(repo *repository.BundleRepository, recipeRepo *repository.RecipeRepository) *BundleService {
	return &BundleService{repo: repo, recipeRepo: recipeRepo}
}

func (s *BundleService) List(ctx context.Context, tenantID uuid.UUID, search string, activeOnly bool, limit, offset int) ([]models.BundleResponse, int, error) {
	bundles, total, err := s.repo.List(ctx, tenantID, search, activeOnly, normalizeLimit(limit), normalizeOffset(offset))
	if err != nil {
		return nil, 0, err
	}
	responses := make([]models.BundleResponse, 0, len(bundles))
	for i := range bundles {
		responses = append(responses, models.NewBundleResponse(&bundles[i]))
	}
	return responses, total, nil
}

func (s *BundleService) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.BundleResponse, error) {
	aggregate, err := s.repo.FindByID(ctx, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	response := models.NewBundleResponse(aggregate)
	return &response, nil
}

func (s *BundleService) Create(ctx context.Context, tenantID uuid.UUID, req models.CreateBundleRequest) (*models.BundleResponse, error) {
	sku := strings.TrimSpace(req.SKU)
	name := strings.TrimSpace(req.Name)
	if sku == "" || name == "" {
		return nil, fmt.Errorf("%w: sku and name are required", ErrInvalidInput)
	}
	if req.SellingPrice.IsNegative() {
		return nil, fmt.Errorf("%w: selling_price cannot be negative", ErrInvalidInput)
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: at least one bundle item is required", ErrInvalidInput)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	bundle := &models.Bundle{
		TenantID:     tenantID,
		SKU:          sku,
		Name:         name,
		Description:  normalizeOptionalText(req.Description),
		SellingPrice: req.SellingPrice.Decimal.Round(2),
		IsActive:     isActive,
	}

	tx, err := s.repo.DB().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin bundle transaction: %w", err)
	}
	defer tx.Rollback()

	if err := repository.SetTenantContextTx(ctx, tx, tenantID); err != nil {
		return nil, err
	}
	if err := s.repo.CreateTx(ctx, tx, bundle); err != nil {
		return nil, translateBundleDBError(err)
	}
	items, err := s.buildBundleItemsTx(ctx, tx, tenantID, bundle.ID, req.Items)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if err := s.repo.CreateItemTx(ctx, tx, &items[i]); err != nil {
			return nil, translateBundleDBError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit bundle: %w", err)
	}
	return s.Get(ctx, tenantID, bundle.ID)
}

func (s *BundleService) Update(ctx context.Context, tenantID, id uuid.UUID, req models.UpdateBundleRequest) (*models.BundleResponse, error) {
	tx, err := s.repo.DB().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin bundle transaction: %w", err)
	}
	defer tx.Rollback()

	if err := repository.SetTenantContextTx(ctx, tx, tenantID); err != nil {
		return nil, err
	}
	aggregate, err := s.repo.FindByIDTx(ctx, tx, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	bundle := aggregate.Bundle
	if req.SKU != nil {
		sku := strings.TrimSpace(*req.SKU)
		if sku == "" {
			return nil, fmt.Errorf("%w: sku is required", ErrInvalidInput)
		}
		bundle.SKU = sku
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
		}
		bundle.Name = name
	}
	if req.Description != nil {
		bundle.Description = normalizeOptionalText(req.Description)
	}
	if req.SellingPrice != nil {
		if req.SellingPrice.IsNegative() {
			return nil, fmt.Errorf("%w: selling_price cannot be negative", ErrInvalidInput)
		}
		bundle.SellingPrice = req.SellingPrice.Decimal.Round(2)
	}
	if req.IsActive != nil {
		bundle.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateTx(ctx, tx, &bundle); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, translateBundleDBError(err)
	}
	if req.Items != nil {
		if len(*req.Items) == 0 {
			return nil, fmt.Errorf("%w: at least one bundle item is required", ErrInvalidInput)
		}
		items, err := s.buildBundleItemsTx(ctx, tx, tenantID, id, *req.Items)
		if err != nil {
			return nil, err
		}
		if err := s.repo.ReplaceItemsTx(ctx, tx, tenantID, id, items); err != nil {
			return nil, translateBundleDBError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit bundle: %w", err)
	}
	return s.Get(ctx, tenantID, id)
}

func (s *BundleService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *BundleService) Cost(ctx context.Context, tenantID, id uuid.UUID) (*models.BundleCostResponse, error) {
	var aggregate *models.BundleAggregate
	var costs []bundleRecipeCost
	err := repository.WithTenantTx(ctx, s.repo.DB(), tenantID, nil, func(tx *sql.Tx) error {
		var err error
		aggregate, err = s.repo.FindByIDTx(ctx, tx, tenantID, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		costs = make([]bundleRecipeCost, 0, len(aggregate.Items))
		for i := range aggregate.Items {
			item := aggregate.Items[i]
			recipe, err := s.recipeRepo.GetActiveTx(ctx, tx, tenantID, item.ProductID)
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: active recipe not found for product %s", ErrNotFound, item.ProductID)
			}
			if err != nil {
				return err
			}
			costs = append(costs, bundleRecipeCost{
				item:     item,
				recipe:   recipe,
				unitCost: calculateRecipeCost(recipe),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	response := newBundleCostResponse(aggregate, costs)
	return &response, nil
}

func (s *BundleService) buildBundleItemsTx(ctx context.Context, tx *sql.Tx, tenantID, bundleID uuid.UUID, reqItems []models.CreateBundleItem) ([]models.BundleItem, error) {
	seen := make(map[uuid.UUID]struct{}, len(reqItems))
	for _, req := range reqItems {
		if req.ProductID == uuid.Nil {
			return nil, fmt.Errorf("%w: product_id is required", ErrInvalidInput)
		}
		if req.Quantity.LessThanOrEqual(decimal.Zero) {
			return nil, fmt.Errorf("%w: item quantity must be greater than zero", ErrInvalidInput)
		}
		if _, exists := seen[req.ProductID]; exists {
			return nil, fmt.Errorf("%w: duplicate product in bundle", ErrInvalidInput)
		}
		seen[req.ProductID] = struct{}{}
	}

	items := make([]models.BundleItem, 0, len(reqItems))
	for _, req := range reqItems {
		if _, err := s.recipeRepo.FindProductTx(ctx, tx, tenantID, req.ProductID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrProductNotFound
			}
			return nil, err
		}
		items = append(items, models.BundleItem{
			TenantID:  tenantID,
			BundleID:  bundleID,
			ProductID: req.ProductID,
			Quantity:  req.Quantity.Decimal,
		})
	}
	return items, nil
}

func newBundleCostResponse(aggregate *models.BundleAggregate, costs []bundleRecipeCost) models.BundleCostResponse {
	materialCost := decimal.Zero
	fixedOverheadCost := decimal.Zero
	percentageOverheadCost := decimal.Zero
	totalCOGS := decimal.Zero
	items := make([]models.BundleItemCostResponse, 0, len(costs))

	for i := range costs {
		line := costs[i]
		quantity := line.item.Quantity
		lineMaterialCost := line.unitCost.MaterialCost.Decimal.Mul(quantity).Round(2)
		lineFixedOverheadCost := line.unitCost.FixedOverheadCost.Decimal.Mul(quantity).Round(2)
		linePercentageOverheadCost := line.unitCost.PercentageOverheadCost.Decimal.Mul(quantity).Round(2)
		lineTotalCOGS := line.unitCost.TotalCOGS.Decimal.Mul(quantity).Round(2)

		materialCost = materialCost.Add(lineMaterialCost)
		fixedOverheadCost = fixedOverheadCost.Add(lineFixedOverheadCost)
		percentageOverheadCost = percentageOverheadCost.Add(linePercentageOverheadCost)
		totalCOGS = totalCOGS.Add(lineTotalCOGS)

		items = append(items, models.BundleItemCostResponse{
			BundleItemID:               line.item.ID,
			ProductID:                  line.item.ProductID,
			ProductName:                line.item.ProductName,
			ProductSKU:                 line.item.ProductSKU,
			Quantity:                   models.NewNumber(quantity),
			RecipeID:                   line.recipe.Recipe.ID,
			RecipeVersion:              line.recipe.Recipe.Version,
			UnitMaterialCost:           line.unitCost.MaterialCost,
			UnitFixedOverheadCost:      line.unitCost.FixedOverheadCost,
			UnitPercentageOverheadCost: line.unitCost.PercentageOverheadCost,
			UnitTotalCOGS:              line.unitCost.TotalCOGS,
			LineMaterialCost:           models.NewNumber(lineMaterialCost),
			LineFixedOverheadCost:      models.NewNumber(lineFixedOverheadCost),
			LinePercentageOverheadCost: models.NewNumber(linePercentageOverheadCost),
			LineTotalCOGS:              models.NewNumber(lineTotalCOGS),
		})
	}

	totalCOGS = totalCOGS.Round(2)
	sellingPrice := aggregate.Bundle.SellingPrice.Round(2)
	grossProfit := sellingPrice.Sub(totalCOGS).Round(2)
	grossMargin := decimal.Zero
	if sellingPrice.GreaterThan(decimal.Zero) {
		grossMargin = grossProfit.Div(sellingPrice).Mul(decimal.NewFromInt(100)).Round(2)
	}

	return models.BundleCostResponse{
		BundleID:               aggregate.Bundle.ID,
		BundleName:             aggregate.Bundle.Name,
		BundleSKU:              aggregate.Bundle.SKU,
		Items:                  items,
		MaterialCost:           models.NewNumber(materialCost.Round(2)),
		FixedOverheadCost:      models.NewNumber(fixedOverheadCost.Round(2)),
		PercentageOverheadCost: models.NewNumber(percentageOverheadCost.Round(2)),
		TotalCOGS:              models.NewNumber(totalCOGS),
		SellingPrice:           models.NewNumber(sellingPrice),
		GrossProfit:            models.NewNumber(grossProfit),
		GrossMarginPercentage:  models.NewNumber(grossMargin),
	}
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func translateBundleDBError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return fmt.Errorf("%w: duplicate bundle data", ErrConflict)
	}
	return err
}
