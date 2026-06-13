package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ProductProfitability struct {
	ProductID            uuid.UUID       `json:"product_id"`
	ProductName          string          `json:"product_name"`
	TotalRevenue         decimal.Decimal `json:"total_revenue"`
	TotalCOGS            decimal.Decimal `json:"total_cogs"`
	GrossProfit          decimal.Decimal `json:"gross_profit"`
	GrossMarginPct       decimal.Decimal `json:"gross_margin_pct"`
	TotalQuantitySold    int64           `json:"total_quantity_sold"`
	AvgSellingPrice      decimal.Decimal `json:"avg_selling_price"`
	AvgCOGSPerUnit       decimal.Decimal `json:"avg_cogs_per_unit"`
	RecipeVersion        int             `json:"recipe_version"`
}

type ProductProfitabilityResponse struct {
	Products   []ProductProfitability `json:"products"`
	StartDate  time.Time              `json:"start_date"`
	EndDate    time.Time              `json:"end_date"`
	TotalItems int                    `json:"total_items"`
}

type IngredientUsage struct {
	IngredientID       uuid.UUID       `json:"ingredient_id"`
	IngredientName     string          `json:"ingredient_name"`
	TotalConsumed      decimal.Decimal `json:"total_consumed"`
	BaseUnit           string          `json:"base_unit"`
	TotalCost          decimal.Decimal `json:"total_cost"`
	AvgCostPerUnit     decimal.Decimal `json:"avg_cost_per_unit"`
	CurrentStock       decimal.Decimal `json:"current_stock"`
	StockAfterForecast decimal.Decimal `json:"stock_after_forecast"`
	DaysUntilStockout  *int            `json:"days_until_stockout"`
	AllocatedBudget    decimal.Decimal `json:"allocated_budget,omitempty"`
	QuantityToBuy      decimal.Decimal `json:"quantity_to_buy,omitempty"`
}

type IngredientForecastResponse struct {
	Ingredients   []IngredientUsage `json:"ingredients"`
	StartDate     time.Time         `json:"start_date"`
	EndDate       time.Time         `json:"end_date"`
	DailyBurnRate []DailyBurnRate   `json:"daily_burn_rate"`
}

type DailyBurnRate struct {
	Date          time.Time       `json:"date"`
	IngredientID  uuid.UUID       `json:"ingredient_id"`
	QuantityUsed  decimal.Decimal `json:"quantity_used"`
}

type RevenueTarget struct {
	TargetRevenue    decimal.Decimal `json:"target_revenue"`
	PeriodDays       int             `json:"period_days"`
	DailyTarget      decimal.Decimal `json:"daily_target"`
	WeeklyTarget     decimal.Decimal `json:"weekly_target"`
}

type RevenueTargetRequest struct {
	TargetRevenue string `json:"target_revenue" validate:"required"`
	StartDate     string `json:"start_date" validate:"required"`
	EndDate       string `json:"end_date" validate:"required"`
}

type RevenueSimulatorResponse struct {
	Target                RevenueTarget                   `json:"target"`
	RecommendedMix        []RecommendedProductMix         `json:"recommended_mix"`
	EstimatedIngredientUsage []IngredientUsage            `json:"estimated_ingredient_usage"`
	EstimatedBudget       decimal.Decimal                 `json:"estimated_budget"`
	ProjectedGrossProfit  decimal.Decimal                 `json:"projected_gross_profit"`
	ProjectedGrossMargin  decimal.Decimal                 `json:"projected_gross_margin"`
	FeasibilityScore      decimal.Decimal                 `json:"feasibility_score"`
	FeasibilityNotes      []string                        `json:"feasibility_notes"`
}

type RecommendedProductMix struct {
	ProductID           uuid.UUID       `json:"product_id"`
	ProductName         string          `json:"product_name"`
	RecommendedQuantity int             `json:"recommended_quantity"`
	EstimatedRevenue    decimal.Decimal `json:"estimated_revenue"`
	EstimatedCOGS       decimal.Decimal `json:"estimated_cogs"`
	GrossMarginPct      decimal.Decimal `json:"gross_margin_pct"`
	ScoreReason         string          `json:"score_reason"`
}

type BudgetSimulatorRequest struct {
	BudgetAmount  string `json:"budget_amount" validate:"required"`
	StartDate     string `json:"start_date" validate:"required"`
	EndDate       string `json:"end_date" validate:"required"`
}

type BudgetSimulatorResponse struct {
	Budget              decimal.Decimal             `json:"budget"`
	StartDate           time.Time                   `json:"start_date"`
	EndDate             time.Time                   `json:"end_date"`
	RecommendedMix      []RecommendedProductMix     `json:"recommended_mix"`
	IngredientBreakdown []IngredientBudgetLine      `json:"ingredient_breakdown"`
	ProjectedRevenue    decimal.Decimal             `json:"projected_revenue"`
	ProjectedGrossProfit decimal.Decimal            `json:"projected_gross_profit"`
	ProjectedGrossMargin decimal.Decimal            `json:"projected_gross_margin"`
	BudgetUtilization   decimal.Decimal             `json:"budget_utilization"`
}

type IngredientBudgetLine struct {
	IngredientID    uuid.UUID       `json:"ingredient_id"`
	IngredientName  string          `json:"ingredient_name"`
	AllocatedBudget decimal.Decimal `json:"allocated_budget"`
	QuantityToBuy   decimal.Decimal `json:"quantity_to_buy"`
	Unit            string          `json:"unit"`
	CostPerUnit     decimal.Decimal `json:"cost_per_unit"`
}

type ProductRecommendation struct {
	ProductID          uuid.UUID       `json:"product_id"`
	ProductName        string          `json:"product_name"`
	TotalScore         decimal.Decimal `json:"total_score"`
	MarginScore        decimal.Decimal `json:"margin_score"`
	VelocityScore      decimal.Decimal `json:"velocity_score"`
	StockReadinessScore decimal.Decimal `json:"stock_readiness_score"`
	TrendScore         decimal.Decimal `json:"trend_score"`
	BundleOpportunity  bool            `json:"bundle_opportunity"`
	RecommendationNote string          `json:"recommendation_note"`
}

type ProductRecommendationResponse struct {
	Recommendations []ProductRecommendation `json:"recommendations"`
	GeneratedAt     time.Time               `json:"generated_at"`
	ScoringWeights  ScoringWeights          `json:"scoring_weights"`
}

type ScoringWeights struct {
	Margin       decimal.Decimal `json:"margin"`
	Velocity     decimal.Decimal `json:"velocity"`
	StockReadiness decimal.Decimal `json:"stock_readiness"`
	Trend        decimal.Decimal `json:"trend"`
}

type SalesMixRecommendation struct {
	ProductID        uuid.UUID       `json:"product_id"`
	ProductName      string          `json:"product_name"`
	TargetQuantity   int             `json:"target_quantity"`
	EstimatedRevenue decimal.Decimal `json:"estimated_revenue"`
	EstimatedCOGS    decimal.Decimal `json:"estimated_cogs"`
	GrossProfit      decimal.Decimal `json:"gross_profit"`
	GrossMarginPct   decimal.Decimal `json:"gross_margin_pct"`
	Priority         int             `json:"priority"`
}

type ForecastSummary struct {
	TotalTargetRevenue   decimal.Decimal `json:"total_target_revenue"`
	TotalEstimatedCOGS   decimal.Decimal `json:"total_estimated_cogs"`
	TotalGrossProfit     decimal.Decimal `json:"total_gross_profit"`
	TotalGrossMargin     decimal.Decimal `json:"total_gross_margin"`
	DailyRevenueTarget   decimal.Decimal `json:"daily_revenue_target"`
	WeeklyRevenueTarget  decimal.Decimal `json:"weekly_revenue_target"`
	OperationalDays      int             `json:"operational_days"`
}
