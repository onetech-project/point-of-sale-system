package models

import (
	"time"

	"github.com/google/uuid"
)

// SalesMetrics represents aggregated sales performance metrics
type SalesMetrics struct {
	TotalRevenue      float64   `json:"total_revenue"`
	TotalOrders       int64     `json:"total_orders"`
	AverageOrderValue float64   `json:"average_order_value"`
	InventoryValue    float64   `json:"inventory_value"`  // Sum of (product.cost * product.quantity)
	RevenueChange     float64   `json:"revenue_change"`   // Percentage change vs previous period
	OrdersChange      float64   `json:"orders_change"`    // Percentage change vs previous period
	AOVChange         float64   `json:"aov_change"`       // Percentage change vs previous period
	PreviousRevenue   float64   `json:"previous_revenue"` // For comparison
	PreviousOrders    int64     `json:"previous_orders"`  // For comparison
	PreviousAOV       float64   `json:"previous_aov"`     // For comparison
	StartDate         time.Time `json:"start_date"`
	EndDate           time.Time `json:"end_date"`
}

// DailySalesData represents sales data for a single day
type DailySalesData struct {
	Date    time.Time `json:"date"`
	Revenue float64   `json:"revenue"`
	Orders  int64     `json:"orders"`
}

// CategorySales represents sales breakdown by category
type CategorySales struct {
	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Revenue      float64   `json:"revenue"`
	OrderCount   int64     `json:"order_count"`
	Percentage   float64   `json:"percentage"` // Percentage of total sales
}

// SalesOverviewResponse is the complete response for sales overview
type SalesOverviewResponse struct {
	Metrics           SalesMetrics     `json:"metrics"`
	SalesChart        []DailySalesData `json:"sales_chart"`
	CategoryBreakdown []CategorySales  `json:"category_breakdown"`
}
