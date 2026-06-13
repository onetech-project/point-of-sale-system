package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pos/analytics-service/src/models"
	"github.com/shopspring/decimal"
)

func TestCalculateFeasibilityScore(t *testing.T) {
	s := &InventoryAnalyticsService{}

	tests := []struct {
		name           string
		projected      decimal.Decimal
		target         decimal.Decimal
		margin         decimal.Decimal
		expectedScore  decimal.Decimal
	}{
		{
			name:          "exact match with good margin",
			projected:     decimal.NewFromInt(1000000),
			target:        decimal.NewFromInt(1000000),
			margin:        decimal.NewFromInt(35),
			expectedScore: decimal.NewFromInt(100),
		},
		{
			name:          "below target with low margin",
			projected:     decimal.NewFromInt(500000),
			target:        decimal.NewFromInt(1000000),
			margin:        decimal.NewFromInt(15),
			expectedScore: decimal.NewFromInt(45),
		},
		{
			name:          "above target with good margin",
			projected:     decimal.NewFromInt(1500000),
			target:        decimal.NewFromInt(1000000),
			margin:        decimal.NewFromInt(40),
			expectedScore: decimal.NewFromInt(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := s.calculateFeasibilityScore(tt.projected, tt.target, tt.margin)
			if !score.Equal(tt.expectedScore) {
				t.Errorf("calculateFeasibilityScore() = %v, want %v", score, tt.expectedScore)
			}
		})
	}
}

func TestScoreProduct(t *testing.T) {
	s := &InventoryAnalyticsService{}

	product := models.ProductProfitability{
		ProductID:           uuid.New(),
		ProductName:         "Test Product",
		GrossMarginPct:      decimal.NewFromInt(35),
		TotalQuantitySold:   100,
		AvgSellingPrice:     decimal.NewFromInt(50000),
		AvgCOGSPerUnit:      decimal.NewFromInt(32500),
	}

	velocity := decimal.NewFromInt(5)
	inBundle := false
	weights := models.ScoringWeights{
		Margin:         decimal.NewFromFloat(0.35),
		Velocity:       decimal.NewFromFloat(0.25),
		StockReadiness: decimal.NewFromFloat(0.25),
		Trend:          decimal.NewFromFloat(0.15),
	}

	rec := s.scoreProduct(product, velocity, inBundle, weights)

	if rec.ProductID != product.ProductID {
		t.Errorf("scoreProduct() ProductID = %v, want %v", rec.ProductID, product.ProductID)
	}

	if rec.ProductName != product.ProductName {
		t.Errorf("scoreProduct() ProductName = %v, want %v", rec.ProductName, product.ProductName)
	}

	if rec.TotalScore.IsNegative() {
		t.Errorf("scoreProduct() TotalScore should be positive, got %v", rec.TotalScore)
	}

	if rec.BundleOpportunity != true {
		t.Errorf("scoreProduct() BundleOpportunity should be true when not in bundle")
	}
}

func TestCalculateRecommendedMix(t *testing.T) {
	s := &InventoryAnalyticsService{}

	products := []models.ProductProfitability{
		{
			ProductID:        uuid.New(),
			ProductName:      "High Margin Product",
			AvgSellingPrice:  decimal.NewFromInt(100000),
			AvgCOGSPerUnit:   decimal.NewFromInt(60000),
			GrossMarginPct:   decimal.NewFromInt(40),
		},
		{
			ProductID:        uuid.New(),
			ProductName:      "Low Margin Product",
			AvgSellingPrice:  decimal.NewFromInt(50000),
			AvgCOGSPerUnit:   decimal.NewFromInt(40000),
			GrossMarginPct:   decimal.NewFromInt(20),
		},
	}

	targetRevenue := decimal.NewFromInt(500000)
	periodDays := 30

	mix := s.calculateRecommendedMix(products, targetRevenue, periodDays)

	if len(mix) == 0 {
		t.Error("calculateRecommendedMix() returned empty mix")
	}

	totalRevenue := decimal.Zero
	for _, m := range mix {
		totalRevenue = totalRevenue.Add(m.EstimatedRevenue)
	}

	if totalRevenue.GreaterThan(targetRevenue) {
		t.Errorf("calculateRecommendedMix() total revenue %v exceeds target %v", totalRevenue, targetRevenue)
	}
}

func TestAllocateBudgetToIngredients(t *testing.T) {
	s := &InventoryAnalyticsService{}

	budget := decimal.NewFromInt(1000000)
	usage := []models.IngredientUsage{
		{
			IngredientID:   uuid.New(),
			IngredientName: "Flour",
			TotalCost:      decimal.NewFromInt(300000),
			AvgCostPerUnit: decimal.NewFromInt(10000),
		},
		{
			IngredientID:   uuid.New(),
			IngredientName: "Sugar",
			TotalCost:      decimal.NewFromInt(200000),
			AvgCostPerUnit: decimal.NewFromInt(8000),
		},
	}

	result := s.allocateBudgetToIngredients(budget, usage)

	if len(result) != 2 {
		t.Errorf("allocateBudgetToIngredients() returned %d ingredients, want 2", len(result))
	}

	totalAllocated := decimal.Zero
	for _, r := range result {
		totalAllocated = totalAllocated.Add(r.AllocatedBudget)
		if r.QuantityToBuy.IsNegative() {
			t.Errorf("allocateBudgetToIngredients() QuantityToBuy should be positive for %s", r.IngredientName)
		}
	}

	if !totalAllocated.Equal(budget) {
		t.Errorf("allocateBudgetToIngredients() total allocated %v != budget %v", totalAllocated, budget)
	}
}
