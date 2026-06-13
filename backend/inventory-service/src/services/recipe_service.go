package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
	"github.com/shopspring/decimal"
)

type RecipeService struct {
	repo              *repository.RecipeRepository
	ingredientRepo    *repository.IngredientRepository
	conversionService *ConversionService
}

func NewRecipeService(repo *repository.RecipeRepository, ingredientRepo *repository.IngredientRepository, conversionService *ConversionService) *RecipeService {
	return &RecipeService{
		repo:              repo,
		ingredientRepo:    ingredientRepo,
		conversionService: conversionService,
	}
}

func (s *RecipeService) Create(ctx context.Context, tenantID, productID uuid.UUID, req models.CreateRecipeRequest) (*models.RecipeResponse, error) {
	if req.YieldQuantity.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("%w: yield_quantity must be greater than zero", ErrInvalidInput)
	}
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: at least one recipe item is required", ErrInvalidInput)
	}

	tx, err := s.repo.DB().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := repository.SetTenantContextTx(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	if _, err := s.repo.FindProductTx(ctx, tx, tenantID, productID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}

	version, err := s.repo.NextVersionTx(ctx, tx, tenantID, productID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.DeactivateActiveTx(ctx, tx, tenantID, productID); err != nil {
		return nil, err
	}

	effectiveFrom := time.Now()
	if req.EffectiveFrom != nil {
		effectiveFrom = *req.EffectiveFrom
	}
	recipe := &models.Recipe{
		TenantID:      tenantID,
		ProductID:     productID,
		Version:       version,
		YieldQuantity: req.YieldQuantity.Decimal,
		IsActive:      true,
		EffectiveFrom: effectiveFrom,
	}
	if err := s.repo.CreateRecipeTx(ctx, tx, recipe); err != nil {
		return nil, translateRecipeDBError(err)
	}

	for _, itemReq := range req.Items {
		item, err := s.buildRecipeItemTx(ctx, tx, tenantID, recipe.ID, itemReq)
		if err != nil {
			return nil, err
		}
		if err := s.repo.CreateItemTx(ctx, tx, item); err != nil {
			return nil, translateRecipeDBError(err)
		}
	}

	for _, overheadReq := range req.Overheads {
		overhead, err := s.buildOverhead(tenantID, recipe.ID, overheadReq)
		if err != nil {
			return nil, err
		}
		if err := s.repo.CreateOverheadTx(ctx, tx, overhead); err != nil {
			return nil, translateRecipeDBError(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit recipe: %w", err)
	}

	return s.GetVersion(ctx, tenantID, productID, version)
}

func (s *RecipeService) GetActive(ctx context.Context, tenantID, productID uuid.UUID) (*models.RecipeResponse, error) {
	aggregate, err := s.repo.GetActive(ctx, tenantID, productID)
	return s.aggregateResponse(aggregate, err)
}

func (s *RecipeService) GetCost(ctx context.Context, tenantID, productID uuid.UUID) (*models.RecipeCostResponse, error) {
	aggregate, err := s.repo.GetActive(ctx, tenantID, productID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	cost := calculateRecipeCost(aggregate)
	return &cost, nil
}

func (s *RecipeService) GetVersion(ctx context.Context, tenantID, productID uuid.UUID, version int) (*models.RecipeResponse, error) {
	if version <= 0 {
		return nil, fmt.Errorf("%w: version must be greater than zero", ErrInvalidInput)
	}
	aggregate, err := s.repo.GetVersion(ctx, tenantID, productID, version)
	return s.aggregateResponse(aggregate, err)
}

func (s *RecipeService) ListVersions(ctx context.Context, tenantID, productID uuid.UUID) ([]models.RecipeVersionSummary, error) {
	var versions []models.RecipeVersionSummary
	err := repository.WithTenantTx(ctx, s.repo.DB(), tenantID, nil, func(tx *sql.Tx) error {
		if _, err := s.repo.FindProductTx(ctx, tx, tenantID, productID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrProductNotFound
			}
			return err
		}
		var err error
		versions, err = s.repo.ListVersionsTx(ctx, tx, tenantID, productID)
		return err
	})
	return versions, err
}

func (s *RecipeService) buildRecipeItemTx(ctx context.Context, tx *sql.Tx, tenantID, recipeID uuid.UUID, req models.CreateRecipeItem) (*models.RecipeItem, error) {
	if req.Quantity.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("%w: item quantity must be greater than zero", ErrInvalidInput)
	}
	if req.WastePercentage.IsNegative() || req.WastePercentage.GreaterThan(decimal.NewFromInt(100)) {
		return nil, fmt.Errorf("%w: waste percentage must be between 0 and 100", ErrInvalidInput)
	}
	ingredient, err := s.ingredientRepo.FindByIDTx(ctx, tx, tenantID, req.IngredientID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: ingredient not found", ErrInvalidInput)
	}
	if err != nil {
		return nil, err
	}
	if !ingredient.IsActive {
		return nil, ErrInactiveIngredient
	}
	normalized, err := s.conversionService.NormalizeQuantityTx(ctx, tx, tenantID, ingredient, req.Quantity.Decimal, req.UOMID)
	if err != nil {
		return nil, err
	}
	return &models.RecipeItem{
		TenantID:               tenantID,
		RecipeID:               recipeID,
		IngredientID:           req.IngredientID,
		Quantity:               req.Quantity.Decimal,
		UOMID:                  req.UOMID,
		NormalizedQuantityBase: normalized,
		WastePercentage:        req.WastePercentage.Decimal,
	}, nil
}

func (s *RecipeService) buildOverhead(tenantID, recipeID uuid.UUID, req models.CreateRecipeOverhead) (*models.RecipeOverhead, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: overhead name is required", ErrInvalidInput)
	}
	calculationType := strings.ToUpper(strings.TrimSpace(req.CalculationType))
	if calculationType != models.OverheadFixed && calculationType != models.OverheadPercentage {
		return nil, fmt.Errorf("%w: invalid overhead calculation type", ErrInvalidInput)
	}
	if req.Amount.IsNegative() {
		return nil, fmt.Errorf("%w: overhead amount cannot be negative", ErrInvalidInput)
	}
	return &models.RecipeOverhead{
		TenantID:        tenantID,
		RecipeID:        recipeID,
		Name:            name,
		CalculationType: calculationType,
		Amount:          req.Amount.Decimal,
	}, nil
}

func (s *RecipeService) aggregateResponse(aggregate *models.RecipeAggregate, err error) (*models.RecipeResponse, error) {
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	response := newRecipeResponse(aggregate)
	return &response, nil
}

func newRecipeResponse(aggregate *models.RecipeAggregate) models.RecipeResponse {
	cost := calculateRecipeCost(aggregate)
	items := make([]models.RecipeItemResponse, 0, len(aggregate.Items))
	for i := range aggregate.Items {
		item := aggregate.Items[i]
		ingredientCost := item.NormalizedQuantityBase.Mul(item.AverageCostPerBaseUnit)
		costWithWaste := ingredientCost.Mul(decimal.NewFromInt(1).Add(item.WastePercentage.Div(decimal.NewFromInt(100))))
		items = append(items, models.RecipeItemResponse{
			ID:                     item.ID,
			IngredientID:           item.IngredientID,
			IngredientName:         item.IngredientName,
			Quantity:               models.NewNumber(item.Quantity),
			UOMID:                  item.UOMID,
			UOMCode:                item.UOMCode,
			NormalizedQuantityBase: models.NewNumber(item.NormalizedQuantityBase),
			WastePercentage:        models.NewNumber(item.WastePercentage),
			IngredientCost:         models.NewNumber(ingredientCost.Round(2)),
			CostWithWaste:          models.NewNumber(costWithWaste.Round(2)),
		})
	}
	overheads := make([]models.RecipeOverheadResponse, 0, len(aggregate.Overheads))
	for i := range aggregate.Overheads {
		overhead := aggregate.Overheads[i]
		overheads = append(overheads, models.RecipeOverheadResponse{
			ID:              overhead.ID,
			Name:            overhead.Name,
			CalculationType: overhead.CalculationType,
			Amount:          models.NewNumber(overhead.Amount),
		})
	}
	return models.RecipeResponse{
		ID:            aggregate.Recipe.ID,
		TenantID:      aggregate.Recipe.TenantID,
		ProductID:     aggregate.Recipe.ProductID,
		ProductName:   aggregate.Product.Name,
		ProductSKU:    aggregate.Product.SKU,
		Version:       aggregate.Recipe.Version,
		YieldQuantity: models.NewNumber(aggregate.Recipe.YieldQuantity),
		IsActive:      aggregate.Recipe.IsActive,
		EffectiveFrom: aggregate.Recipe.EffectiveFrom,
		Items:         items,
		Overheads:     overheads,
		Cost:          cost,
		CreatedAt:     aggregate.Recipe.CreatedAt,
		UpdatedAt:     aggregate.Recipe.UpdatedAt,
	}
}

func calculateRecipeCost(aggregate *models.RecipeAggregate) models.RecipeCostResponse {
	materialTotal := decimal.Zero
	for i := range aggregate.Items {
		item := aggregate.Items[i]
		ingredientCost := item.NormalizedQuantityBase.Mul(item.AverageCostPerBaseUnit)
		costWithWaste := ingredientCost.Mul(decimal.NewFromInt(1).Add(item.WastePercentage.Div(decimal.NewFromInt(100))))
		materialTotal = materialTotal.Add(costWithWaste)
	}
	materialCost := materialTotal.Div(aggregate.Recipe.YieldQuantity)

	fixedOverhead := decimal.Zero
	percentageRate := decimal.Zero
	for i := range aggregate.Overheads {
		overhead := aggregate.Overheads[i]
		switch overhead.CalculationType {
		case models.OverheadFixed:
			fixedOverhead = fixedOverhead.Add(overhead.Amount)
		case models.OverheadPercentage:
			percentageRate = percentageRate.Add(overhead.Amount)
		}
	}
	percentageOverhead := materialCost.Mul(percentageRate).Div(decimal.NewFromInt(100))
	totalCOGS := materialCost.Add(fixedOverhead).Add(percentageOverhead)
	grossProfit := aggregate.Product.SellingPrice.Sub(totalCOGS)
	grossMargin := decimal.Zero
	if aggregate.Product.SellingPrice.GreaterThan(decimal.Zero) {
		grossMargin = grossProfit.Div(aggregate.Product.SellingPrice).Mul(decimal.NewFromInt(100))
	}

	return models.RecipeCostResponse{
		MaterialCost:           models.NewNumber(materialCost.Round(2)),
		FixedOverheadCost:      models.NewNumber(fixedOverhead.Round(2)),
		PercentageOverheadCost: models.NewNumber(percentageOverhead.Round(2)),
		TotalCOGS:              models.NewNumber(totalCOGS.Round(2)),
		SellingPrice:           models.NewNumber(aggregate.Product.SellingPrice.Round(2)),
		GrossProfit:            models.NewNumber(grossProfit.Round(2)),
		GrossMarginPercentage:  models.NewNumber(grossMargin.Round(2)),
	}
}

func translateRecipeDBError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return fmt.Errorf("%w: duplicate recipe version", ErrConflict)
	}
	return err
}
