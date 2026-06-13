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

type IngredientService struct {
	repo    *repository.IngredientRepository
	uomRepo *repository.UOMRepository
}

func NewIngredientService(repo *repository.IngredientRepository, uomRepo *repository.UOMRepository) *IngredientService {
	return &IngredientService{repo: repo, uomRepo: uomRepo}
}

func (s *IngredientService) List(ctx context.Context, tenantID uuid.UUID, search string, activeOnly bool, limit, offset int) ([]models.IngredientResponse, int, error) {
	ingredients, total, err := s.repo.List(ctx, tenantID, search, activeOnly, normalizeLimit(limit), normalizeOffset(offset))
	if err != nil {
		return nil, 0, err
	}
	responses := make([]models.IngredientResponse, 0, len(ingredients))
	for i := range ingredients {
		responses = append(responses, models.NewIngredientResponse(&ingredients[i]))
	}
	return responses, total, nil
}

func (s *IngredientService) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.IngredientResponse, error) {
	ingredient, err := s.repo.FindByID(ctx, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	response := models.NewIngredientResponse(ingredient)
	return &response, nil
}

func (s *IngredientService) Create(ctx context.Context, tenantID uuid.UUID, req models.CreateIngredientRequest) (*models.IngredientResponse, error) {
	sku := strings.TrimSpace(req.SKU)
	name := strings.TrimSpace(req.Name)
	if sku == "" || name == "" {
		return nil, fmt.Errorf("%w: sku and name are required", ErrInvalidInput)
	}
	if req.MinimumStockBase.IsNegative() || req.AverageCostPerBaseUnit.IsNegative() {
		return nil, fmt.Errorf("%w: costs and thresholds cannot be negative", ErrInvalidInput)
	}
	uom, err := s.uomRepo.FindByID(ctx, tenantID, req.BaseUOMID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: base uom not found", ErrInvalidInput)
	}
	if err != nil {
		return nil, err
	}
	if !uom.IsActive {
		return nil, fmt.Errorf("%w: base uom is inactive", ErrInvalidInput)
	}

	trackStock := true
	if req.TrackStock != nil {
		trackStock = *req.TrackStock
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	ingredient := &models.Ingredient{
		TenantID:               tenantID,
		OutletID:               req.OutletID,
		SKU:                    sku,
		Name:                   name,
		BaseUOMID:              req.BaseUOMID,
		AverageCostPerBaseUnit: req.AverageCostPerBaseUnit.Decimal,
		MinimumStockBase:       req.MinimumStockBase.Decimal,
		TrackStock:             trackStock,
		IsActive:               isActive,
	}
	if err := s.repo.Create(ctx, ingredient); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, fmt.Errorf("%w: duplicate ingredient sku", ErrConflict)
		}
		return nil, err
	}
	created, err := s.repo.FindByID(ctx, tenantID, ingredient.ID)
	if err != nil {
		return nil, err
	}
	response := models.NewIngredientResponse(created)
	return &response, nil
}

func (s *IngredientService) Update(ctx context.Context, tenantID, id uuid.UUID, req models.UpdateIngredientRequest) (*models.IngredientResponse, error) {
	ingredient, err := s.repo.FindByID(ctx, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if req.SKU != nil {
		sku := strings.TrimSpace(*req.SKU)
		if sku == "" {
			return nil, fmt.Errorf("%w: sku is required", ErrInvalidInput)
		}
		ingredient.SKU = sku
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
		}
		ingredient.Name = name
	}
	if req.BaseUOMID != nil {
		uom, err := s.uomRepo.FindByID(ctx, tenantID, *req.BaseUOMID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: base uom not found", ErrInvalidInput)
		}
		if err != nil {
			return nil, err
		}
		if !uom.IsActive {
			return nil, fmt.Errorf("%w: base uom is inactive", ErrInvalidInput)
		}
		ingredient.BaseUOMID = *req.BaseUOMID
	}
	if req.OutletID != nil {
		ingredient.OutletID = req.OutletID
	}
	if req.MinimumStockBase != nil {
		if req.MinimumStockBase.IsNegative() {
			return nil, fmt.Errorf("%w: minimum stock cannot be negative", ErrInvalidInput)
		}
		ingredient.MinimumStockBase = req.MinimumStockBase.Decimal
	}
	if req.AverageCostPerBaseUnit != nil {
		if req.AverageCostPerBaseUnit.IsNegative() {
			return nil, fmt.Errorf("%w: average cost cannot be negative", ErrInvalidInput)
		}
		ingredient.AverageCostPerBaseUnit = req.AverageCostPerBaseUnit.Decimal
	}
	if req.TrackStock != nil {
		ingredient.TrackStock = *req.TrackStock
	}
	if req.IsActive != nil {
		ingredient.IsActive = *req.IsActive
	}
	if err := s.repo.Update(ctx, ingredient); err != nil {
		return nil, err
	}
	updated, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	response := models.NewIngredientResponse(updated)
	return &response, nil
}

func (s *IngredientService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, tenantID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func weightedAverageCost(currentStock, currentAvg, addedQty, addedUnitCost decimal.Decimal) decimal.Decimal {
	if addedQty.LessThanOrEqual(decimal.Zero) {
		return currentAvg
	}
	totalQty := currentStock.Add(addedQty)
	if totalQty.LessThanOrEqual(decimal.Zero) {
		return addedUnitCost
	}
	currentValue := currentStock.Mul(currentAvg)
	addedValue := addedQty.Mul(addedUnitCost)
	return currentValue.Add(addedValue).Div(totalQty)
}
