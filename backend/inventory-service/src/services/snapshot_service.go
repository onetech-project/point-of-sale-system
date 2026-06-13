package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/pos/backend/inventory-service/src/models"
	"github.com/pos/backend/inventory-service/src/repository"
	"github.com/shopspring/decimal"
)

type SnapshotService struct {
	repo *repository.SnapshotRepository
}

func NewSnapshotService(repo *repository.SnapshotRepository) *SnapshotService {
	return &SnapshotService{repo: repo}
}

func (s *SnapshotService) ListByOrder(ctx context.Context, tenantID, orderID uuid.UUID) ([]models.OrderItemCostSnapshotResponse, error) {
	snapshots, err := s.repo.ListByOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	responses := make([]models.OrderItemCostSnapshotResponse, 0, len(snapshots))
	for i := range snapshots {
		responses = append(responses, models.NewOrderItemCostSnapshotResponse(&snapshots[i]))
	}
	return responses, nil
}

func (s *SnapshotService) Profitability(ctx context.Context, tenantID, orderID uuid.UUID) (*models.OrderProfitabilityResponse, error) {
	snapshots, err := s.repo.ListByOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, ErrNotFound
	}

	totalRevenue := decimal.Zero
	totalCOGS := decimal.Zero
	grossProfit := decimal.Zero
	items := make([]models.OrderItemCostSnapshotResponse, 0, len(snapshots))
	for i := range snapshots {
		snapshot := snapshots[i]
		totalRevenue = totalRevenue.Add(snapshot.SellingPrice)
		totalCOGS = totalCOGS.Add(snapshot.TotalCOGS)
		grossProfit = grossProfit.Add(snapshot.GrossProfit)
		items = append(items, models.NewOrderItemCostSnapshotResponse(&snapshot))
	}
	grossMargin := decimal.Zero
	if totalRevenue.GreaterThan(decimal.Zero) {
		grossMargin = grossProfit.Div(totalRevenue).Mul(decimal.NewFromInt(100))
	}

	return &models.OrderProfitabilityResponse{
		OrderID:               orderID,
		TotalRevenue:          models.NewNumber(totalRevenue.Round(2)),
		TotalCOGS:             models.NewNumber(totalCOGS.Round(2)),
		GrossProfit:           models.NewNumber(grossProfit.Round(2)),
		GrossMarginPercentage: models.NewNumber(grossMargin.Round(2)),
		Items:                 items,
	}, nil
}
