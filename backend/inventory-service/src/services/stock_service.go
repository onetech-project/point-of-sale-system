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

type StockService struct {
	repo              *repository.StockRepository
	conversionService *ConversionService
}

func NewStockService(repo *repository.StockRepository, conversionService *ConversionService) *StockService {
	return &StockService{repo: repo, conversionService: conversionService}
}

func (s *StockService) InitialStock(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req models.StockMutationRequest) (*models.StockMovementResponse, error) {
	return s.stockMutation(ctx, tenantID, userID, req, models.MovementInitialStock, true)
}

func (s *StockService) PurchaseIn(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req models.StockMutationRequest) (*models.StockMovementResponse, error) {
	return s.stockMutation(ctx, tenantID, userID, req, models.MovementPurchaseIn, true)
}

func (s *StockService) Waste(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req models.StockMutationRequest) (*models.StockMovementResponse, error) {
	if req.Reason == nil || strings.TrimSpace(*req.Reason) == "" {
		return nil, fmt.Errorf("%w: waste reason is required", ErrInvalidInput)
	}
	return s.stockMutation(ctx, tenantID, userID, req, models.MovementWaste, false)
}

func (s *StockService) ManualAdjustment(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req models.ManualAdjustmentRequest) (*models.StockMovementResponse, error) {
	if strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf("%w: adjustment reason is required", ErrInvalidInput)
	}
	mutation := models.StockMutationRequest{
		IngredientID:   req.IngredientID,
		OutletID:       req.OutletID,
		Quantity:       req.QuantityDelta,
		UOMID:          req.UOMID,
		UnitCost:       req.UnitCost,
		IdempotencyKey: req.IdempotencyKey,
		Reason:         &req.Reason,
		OccurredAt:     req.OccurredAt,
	}
	return s.stockMutation(ctx, tenantID, userID, mutation, models.MovementManualAdjustment, req.QuantityDelta.GreaterThan(decimal.Zero))
}

func (s *StockService) stockMutation(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req models.StockMutationRequest, movementType string, inbound bool) (*models.StockMovementResponse, error) {
	if req.Quantity.IsZero() {
		return nil, fmt.Errorf("%w: quantity must be greater than zero", ErrInvalidInput)
	}
	if req.Quantity.IsNegative() && movementType != models.MovementManualAdjustment {
		return nil, fmt.Errorf("%w: quantity cannot be negative", ErrInvalidInput)
	}
	if req.UnitCost.IsNegative() {
		return nil, fmt.Errorf("%w: unit cost cannot be negative", ErrInvalidInput)
	}

	tx, err := s.repo.DB().BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := repository.SetTenantContextTx(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	ingredient, err := s.repo.LockIngredientTx(ctx, tx, tenantID, req.IngredientID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !ingredient.IsActive {
		return nil, ErrInactiveIngredient
	}

	quantityBase, err := s.conversionService.NormalizeQuantityTx(ctx, tx, tenantID, ingredient, req.Quantity.Decimal.Abs(), req.UOMID)
	if err != nil {
		return nil, err
	}
	signedQuantity := quantityBase
	if !inbound {
		signedQuantity = signedQuantity.Neg()
	}
	if movementType == models.MovementManualAdjustment && req.Quantity.IsNegative() {
		signedQuantity = quantityBase.Neg()
	}

	newStock := ingredient.CurrentStockBase.Add(signedQuantity)
	if newStock.IsNegative() {
		return nil, ErrInsufficientStock
	}

	averageCost := ingredient.AverageCostPerBaseUnit
	if signedQuantity.GreaterThan(decimal.Zero) {
		averageCost = weightedAverageCost(ingredient.CurrentStockBase, ingredient.AverageCostPerBaseUnit, signedQuantity, req.UnitCost.Decimal)
	}

	occurredAt := time.Now()
	if req.OccurredAt != nil {
		occurredAt = *req.OccurredAt
	}
	totalCost := quantityBase.Mul(req.UnitCost.Decimal).Round(2)
	movement := &models.StockMovement{
		TenantID:       tenantID,
		OutletID:       req.OutletID,
		IngredientID:   req.IngredientID,
		MovementType:   movementType,
		QuantityBase:   signedQuantity,
		UnitCost:       req.UnitCost.Decimal,
		TotalCost:      totalCost,
		ReferenceType:  req.ReferenceType,
		ReferenceID:    req.ReferenceID,
		IdempotencyKey: req.IdempotencyKey,
		Reason:         req.Reason,
		OccurredAt:     occurredAt,
		CreatedBy:      userID,
	}

	if err := s.repo.CreateMovementTx(ctx, tx, movement); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, fmt.Errorf("%w: duplicate idempotency key", ErrConflict)
		}
		return nil, err
	}
	if err := s.repo.UpdateIngredientStockTx(ctx, tx, tenantID, req.IngredientID, newStock, averageCost); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit stock movement: %w", err)
	}

	response := models.NewStockMovementResponse(movement)
	return &response, nil
}

func (s *StockService) Movements(ctx context.Context, tenantID, ingredientID uuid.UUID, limit, offset int) ([]models.StockMovementResponse, int, error) {
	movements, total, err := s.repo.ListMovements(ctx, tenantID, ingredientID, normalizeLimit(limit), normalizeOffset(offset))
	if err != nil {
		return nil, 0, err
	}
	responses := make([]models.StockMovementResponse, 0, len(movements))
	for i := range movements {
		responses = append(responses, models.NewStockMovementResponse(&movements[i]))
	}
	return responses, total, nil
}

func (s *StockService) LowStock(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]models.IngredientResponse, int, error) {
	ingredients, total, err := s.repo.ListLowStock(ctx, tenantID, normalizeLimit(limit), normalizeOffset(offset))
	if err != nil {
		return nil, 0, err
	}
	responses := make([]models.IngredientResponse, 0, len(ingredients))
	for i := range ingredients {
		responses = append(responses, models.NewIngredientResponse(&ingredients[i]))
	}
	return responses, total, nil
}

func (s *StockService) Valuation(ctx context.Context, tenantID uuid.UUID) (models.Number, error) {
	total, err := s.repo.Valuation(ctx, tenantID)
	if err != nil {
		return models.ZeroNumber(), err
	}
	return models.NewNumber(total.Round(2)), nil
}
