package services

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/pos/analytics-service/src/models"
	"github.com/pos/analytics-service/src/repository"
	"github.com/shopspring/decimal"
)

type InventoryAnalyticsService struct {
	inventoryRepo *repository.InventoryAnalyticsRepository
}

func NewInventoryAnalyticsService(inventoryRepo *repository.InventoryAnalyticsRepository) *InventoryAnalyticsService {
	return &InventoryAnalyticsService{
		inventoryRepo: inventoryRepo,
	}
}

func (s *InventoryAnalyticsService) GetProductProfitability(ctx context.Context, tenantID string, start, end time.Time, limit int) (*models.ProductProfitabilityResponse, error) {
	products, err := s.inventoryRepo.GetProductProfitability(ctx, tenantID, start, end, limit)
	if err != nil {
		return nil, err
	}

	return &models.ProductProfitabilityResponse{
		Products:   products,
		StartDate:  start,
		EndDate:    end,
		TotalItems: len(products),
	}, nil
}

func (s *InventoryAnalyticsService) GetIngredientForecast(ctx context.Context, tenantID string, start, end time.Time) (*models.IngredientForecastResponse, error) {
	usage, err := s.inventoryRepo.GetIngredientUsage(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	dailyBurn, err := s.inventoryRepo.GetIngredientDailyBurnRate(ctx, tenantID, start, end)
	if err != nil {
		return nil, err
	}

	days := int(end.Sub(start).Hours() / 24)
	if days == 0 {
		days = 1
	}

	for i := range usage {
		dailyRate := usage[i].TotalConsumed.Div(decimal.NewFromInt(int64(days)))
		stockAfterForecast := usage[i].CurrentStock.Sub(usage[i].TotalConsumed)
		if stockAfterForecast.IsNegative() {
			stockAfterForecast = decimal.Zero
		}
		usage[i].StockAfterForecast = stockAfterForecast

		if dailyRate.IsPositive() {
			daysUntilStockout := int(stockAfterForecast.Div(dailyRate).IntPart())
			usage[i].DaysUntilStockout = &daysUntilStockout
		}
	}

	return &models.IngredientForecastResponse{
		Ingredients:   usage,
		StartDate:     start,
		EndDate:       end,
		DailyBurnRate: dailyBurn,
	}, nil
}

func (s *InventoryAnalyticsService) SimulateRevenueTarget(ctx context.Context, tenantID string, req *models.RevenueTargetRequest) (*models.RevenueSimulatorResponse, error) {
	targetRevenue, err := decimal.NewFromString(req.TargetRevenue)
	if err != nil {
		return nil, err
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, err
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, err
	}

	periodDays := int(endDate.Sub(startDate).Hours() / 24)
	if periodDays == 0 {
		periodDays = 1
	}

	dailyTarget := targetRevenue.Div(decimal.NewFromInt(int64(periodDays)))
	weeklyTarget := dailyTarget.Mul(decimal.NewFromInt(7))

	products, err := s.inventoryRepo.GetProductProfitability(ctx, tenantID, startDate.AddDate(0, -1, 0), endDate, 50)
	if err != nil {
		return nil, err
	}

	recommendedMix := s.calculateRecommendedMix(products, targetRevenue, periodDays)

	ingredientUsage, err := s.inventoryRepo.GetIngredientUsage(ctx, tenantID, startDate.AddDate(0, -1, 0), endDate)
	if err != nil {
		return nil, err
	}

	estimatedUsage := s.estimateIngredientUsage(recommendedMix, ingredientUsage)
	estimatedBudget := s.calculateEstimatedBudget(estimatedUsage)
	projectedCOGS := s.calculateProjectedCOGS(recommendedMix)
	projectedRevenue := s.calculateProjectedRevenue(recommendedMix)
	projectedGrossProfit := projectedRevenue.Sub(projectedCOGS)
	projectedGrossMargin := decimal.Zero
	if projectedRevenue.IsPositive() {
		projectedGrossMargin = projectedGrossProfit.Div(projectedRevenue).Mul(decimal.NewFromInt(100))
	}

	feasibilityScore := s.calculateFeasibilityScore(projectedRevenue, targetRevenue, projectedGrossMargin)
	feasibilityNotes := s.generateFeasibilityNotes(projectedRevenue, targetRevenue, projectedGrossMargin, estimatedBudget)

	return &models.RevenueSimulatorResponse{
		Target: models.RevenueTarget{
			TargetRevenue: targetRevenue,
			PeriodDays:    periodDays,
			DailyTarget:   dailyTarget,
			WeeklyTarget:  weeklyTarget,
		},
		RecommendedMix:          recommendedMix,
		EstimatedIngredientUsage: estimatedUsage,
		EstimatedBudget:         estimatedBudget,
		ProjectedGrossProfit:    projectedGrossProfit,
		ProjectedGrossMargin:    projectedGrossMargin,
		FeasibilityScore:        feasibilityScore,
		FeasibilityNotes:        feasibilityNotes,
	}, nil
}

func (s *InventoryAnalyticsService) SimulateBudget(ctx context.Context, tenantID string, req *models.BudgetSimulatorRequest) (*models.BudgetSimulatorResponse, error) {
	budget, err := decimal.NewFromString(req.BudgetAmount)
	if err != nil {
		return nil, err
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, err
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, err
	}

	ingredientUsage, err := s.inventoryRepo.GetIngredientUsage(ctx, tenantID, startDate.AddDate(0, -1, 0), endDate)
	if err != nil {
		return nil, err
	}

	ingredientBudget := s.allocateBudgetToIngredients(budget, ingredientUsage)
	recommendedMix := s.findMixWithinBudget(ingredientBudget, ingredientUsage)

	projectedRevenue := s.calculateProjectedRevenue(recommendedMix)
	projectedCOGS := s.calculateProjectedCOGS(recommendedMix)
	projectedGrossProfit := projectedRevenue.Sub(projectedCOGS)
	projectedGrossMargin := decimal.Zero
	if projectedRevenue.IsPositive() {
		projectedGrossMargin = projectedGrossProfit.Div(projectedRevenue).Mul(decimal.NewFromInt(100))
	}

	budgetUtilization := decimal.Zero
	if budget.IsPositive() {
		usedBudget := s.calculateEstimatedBudget(ingredientUsage)
		budgetUtilization = usedBudget.Div(budget).Mul(decimal.NewFromInt(100))
	}

	ingredientBreakdown := make([]models.IngredientBudgetLine, len(ingredientBudget))
	for i, ib := range ingredientBudget {
		ingredientBreakdown[i] = models.IngredientBudgetLine{
			IngredientID:    ib.IngredientID,
			IngredientName:  ib.IngredientName,
			AllocatedBudget: ib.AllocatedBudget,
			QuantityToBuy:   ib.QuantityToBuy,
			Unit:            ib.BaseUnit,
			CostPerUnit:     ib.AvgCostPerUnit,
		}
	}

	return &models.BudgetSimulatorResponse{
		Budget:              budget,
		StartDate:           startDate,
		EndDate:             endDate,
		RecommendedMix:      recommendedMix,
		IngredientBreakdown: ingredientBreakdown,
		ProjectedRevenue:    projectedRevenue,
		ProjectedGrossProfit: projectedGrossProfit,
		ProjectedGrossMargin: projectedGrossMargin,
		BudgetUtilization:   budgetUtilization,
	}, nil
}

func (s *InventoryAnalyticsService) GetProductRecommendations(ctx context.Context, tenantID string) (*models.ProductRecommendationResponse, error) {
	products, err := s.inventoryRepo.GetProductProfitability(ctx, tenantID, time.Now().AddDate(0, -3, 0), time.Now(), 100)
	if err != nil {
		return nil, err
	}

	velocity, err := s.inventoryRepo.GetProductSalesVelocity(ctx, tenantID, 30)
	if err != nil {
		return nil, err
	}

	bundleItems, err := s.inventoryRepo.GetBundleItems(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	productInBundles := make(map[uuid.UUID]bool)
	for _, items := range bundleItems {
		for _, item := range items {
			productInBundles[item] = true
		}
	}

	weights := models.ScoringWeights{
		Margin:         decimal.NewFromFloat(0.35),
		Velocity:       decimal.NewFromFloat(0.25),
		StockReadiness: decimal.NewFromFloat(0.25),
		Trend:          decimal.NewFromFloat(0.15),
	}

	var recommendations []models.ProductRecommendation
	for _, p := range products {
		rec := s.scoreProduct(p, velocity[p.ProductID], productInBundles[p.ProductID], weights)
		recommendations = append(recommendations, rec)
	}

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].TotalScore.GreaterThan(recommendations[j].TotalScore)
	})

	if len(recommendations) > 20 {
		recommendations = recommendations[:20]
	}

	return &models.ProductRecommendationResponse{
		Recommendations: recommendations,
		GeneratedAt:     time.Now(),
		ScoringWeights:  weights,
	}, nil
}

func (s *InventoryAnalyticsService) calculateRecommendedMix(products []models.ProductProfitability, targetRevenue decimal.Decimal, periodDays int) []models.RecommendedProductMix {
	var mix []models.RecommendedProductMix
	remaining := targetRevenue

	for _, p := range products {
		if !remaining.IsPositive() {
			break
		}

		if !p.AvgSellingPrice.IsPositive() {
			continue
		}

		qty := int(remaining.Div(p.AvgSellingPrice).IntPart())
		if qty > periodDays*10 {
			qty = periodDays * 10
		}
		if qty < 1 {
			qty = 1
		}

		estRevenue := p.AvgSellingPrice.Mul(decimal.NewFromInt(int64(qty)))
		estCOGS := p.AvgCOGSPerUnit.Mul(decimal.NewFromInt(int64(qty)))

		if estRevenue.GreaterThan(remaining) {
			estRevenue = remaining
			qty = int(remaining.Div(p.AvgSellingPrice).IntPart())
			if qty < 1 {
				qty = 1
			}
			estRevenue = p.AvgSellingPrice.Mul(decimal.NewFromInt(int64(qty)))
			estCOGS = p.AvgCOGSPerUnit.Mul(decimal.NewFromInt(int64(qty)))
		}

		marginPct := decimal.Zero
		if estRevenue.IsPositive() {
			marginPct = estRevenue.Sub(estCOGS).Div(estRevenue).Mul(decimal.NewFromInt(100))
		}

		reason := "high margin"
		if p.GrossMarginPct.LessThan(decimal.NewFromInt(30)) {
			reason = "balanced margin & demand"
		}

		mix = append(mix, models.RecommendedProductMix{
			ProductID:           p.ProductID,
			ProductName:         p.ProductName,
			RecommendedQuantity: qty,
			EstimatedRevenue:    estRevenue,
			EstimatedCOGS:       estCOGS,
			GrossMarginPct:      marginPct,
			ScoreReason:         reason,
		})

		remaining = remaining.Sub(estRevenue)
	}

	return mix
}

func (s *InventoryAnalyticsService) estimateIngredientUsage(mix []models.RecommendedProductMix, usage []models.IngredientUsage) []models.IngredientUsage {
	totalRevenue := decimal.Zero
	for _, m := range mix {
		totalRevenue = totalRevenue.Add(m.EstimatedRevenue)
	}

	estimated := make([]models.IngredientUsage, len(usage))
	for i, u := range usage {
		estimated[i] = u
		if totalRevenue.IsPositive() && u.TotalCost.IsPositive() {
			estimated[i].TotalCost = u.TotalCost.Mul(totalRevenue).Div(u.TotalCost)
		}
	}

	return estimated
}

func (s *InventoryAnalyticsService) calculateEstimatedBudget(usage []models.IngredientUsage) decimal.Decimal {
	total := decimal.Zero
	for _, u := range usage {
		total = total.Add(u.TotalCost)
	}
	return total
}

func (s *InventoryAnalyticsService) calculateProjectedCOGS(mix []models.RecommendedProductMix) decimal.Decimal {
	total := decimal.Zero
	for _, m := range mix {
		total = total.Add(m.EstimatedCOGS)
	}
	return total
}

func (s *InventoryAnalyticsService) calculateProjectedRevenue(mix []models.RecommendedProductMix) decimal.Decimal {
	total := decimal.Zero
	for _, m := range mix {
		total = total.Add(m.EstimatedRevenue)
	}
	return total
}

func (s *InventoryAnalyticsService) calculateFeasibilityScore(projected, target, margin decimal.Decimal) decimal.Decimal {
	if !target.IsPositive() {
		return decimal.Zero
	}

	revenueRatio := projected.Div(target)
	score := revenueRatio.Mul(decimal.NewFromInt(70))

	if margin.GreaterThanOrEqual(decimal.NewFromInt(30)) {
		score = score.Add(decimal.NewFromInt(30))
	} else if margin.GreaterThanOrEqual(decimal.NewFromInt(20)) {
		score = score.Add(decimal.NewFromInt(20))
	} else if margin.GreaterThanOrEqual(decimal.NewFromInt(10)) {
		score = score.Add(decimal.NewFromInt(10))
	}

	if score.GreaterThan(decimal.NewFromInt(100)) {
		score = decimal.NewFromInt(100)
	}

	return score
}

func (s *InventoryAnalyticsService) generateFeasibilityNotes(projected, target, margin, budget decimal.Decimal) []string {
	var notes []string

	ratio := projected.Div(target)
	if ratio.LessThan(decimal.NewFromFloat(0.8)) {
		notes = append(notes, "Projected revenue is below 80% of target. Consider adding more products or adjusting prices.")
	} else if ratio.GreaterThan(decimal.NewFromFloat(1.2)) {
		notes = append(notes, "Projected revenue exceeds target by 20%+. Consider scaling back or increasing target.")
	}

	if margin.LessThan(decimal.NewFromInt(20)) {
		notes = append(notes, "Gross margin is below 20%. Review ingredient costs and pricing strategy.")
	}

	if budget.IsPositive() {
		estimatedBudget := s.calculateEstimatedBudget([]models.IngredientUsage{{TotalCost: budget}})
		if estimatedBudget.GreaterThan(budget) {
			notes = append(notes, "Estimated ingredient budget exceeds available budget. Consider reducing quantities or finding cheaper suppliers.")
		}
	}

	if len(notes) == 0 {
		notes = append(notes, "Revenue target appears feasible with current product mix and margin structure.")
	}

	return notes
}

func (s *InventoryAnalyticsService) scoreProduct(p models.ProductProfitability, velocity decimal.Decimal, inBundle bool, weights models.ScoringWeights) models.ProductRecommendation {
	marginScore := p.GrossMarginPct.Div(decimal.NewFromInt(100)).Mul(weights.Margin)

	velocityScore := decimal.Zero
	if velocity.IsPositive() {
		normalizedVelocity := velocity.Div(decimal.NewFromInt(100))
		if normalizedVelocity.GreaterThan(decimal.NewFromInt(1)) {
			normalizedVelocity = decimal.NewFromInt(1)
		}
		velocityScore = normalizedVelocity.Mul(weights.Velocity)
	}

	stockScore := decimal.Zero
	if p.TotalQuantitySold > 0 {
		stockScore = weights.StockReadiness.Mul(decimal.NewFromFloat(0.7))
	}

	trendScore := decimal.Zero
	if p.GrossMarginPct.GreaterThan(decimal.NewFromInt(30)) {
		trendScore = weights.Trend.Mul(decimal.NewFromFloat(0.8))
	}

	totalScore := marginScore.Add(velocityScore).Add(stockScore).Add(trendScore)

	note := "standard product"
	if inBundle {
		note = "bundle component"
	}
	if p.GrossMarginPct.GreaterThan(decimal.NewFromInt(40)) {
		note = "high margin champion"
	}

	return models.ProductRecommendation{
		ProductID:           p.ProductID,
		ProductName:         p.ProductName,
		TotalScore:          totalScore,
		MarginScore:         marginScore,
		VelocityScore:       velocityScore,
		StockReadinessScore: stockScore,
		TrendScore:          trendScore,
		BundleOpportunity:   !inBundle,
		RecommendationNote:  note,
	}
}

func (s *InventoryAnalyticsService) allocateBudgetToIngredients(budget decimal.Decimal, usage []models.IngredientUsage) []models.IngredientUsage {
	totalCost := decimal.Zero
	for _, u := range usage {
		totalCost = totalCost.Add(u.TotalCost)
	}

	result := make([]models.IngredientUsage, len(usage))
	for i, u := range usage {
		result[i] = u
		if totalCost.IsPositive() {
			allocRatio := u.TotalCost.Div(totalCost)
			allocatedBudget := budget.Mul(allocRatio)
			result[i].AllocatedBudget = allocatedBudget

			if u.AvgCostPerUnit.IsPositive() {
				result[i].QuantityToBuy = allocatedBudget.Div(u.AvgCostPerUnit)
			}
		}
	}

	return result
}

func (s *InventoryAnalyticsService) findMixWithinBudget(ingredientBudget []models.IngredientUsage, historicalUsage []models.IngredientUsage) []models.RecommendedProductMix {
	var mix []models.RecommendedProductMix

	_ = math.MaxFloat64

	for _, ib := range ingredientBudget {
		if ib.QuantityToBuy.IsPositive() && ib.AvgCostPerUnit.IsPositive() {
			revenue := ib.QuantityToBuy.Mul(ib.AvgCostPerUnit).Mul(decimal.NewFromFloat(1.5))

			mix = append(mix, models.RecommendedProductMix{
				ProductID:           ib.IngredientID,
				ProductName:         ib.IngredientName + " (budget allocation)",
				RecommendedQuantity: int(ib.QuantityToBuy.IntPart()),
				EstimatedRevenue:    revenue,
				EstimatedCOGS:       ib.QuantityToBuy.Mul(ib.AvgCostPerUnit),
				GrossMarginPct:      decimal.NewFromFloat(33.3),
				ScoreReason:         "budget-based allocation",
			})
		}
	}

	return mix
}
