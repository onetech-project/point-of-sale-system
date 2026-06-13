// Analytics Dashboard Types

export type TimeRange =
  | 'today'
  | 'yesterday'
  | 'this_week'
  | 'last_week'
  | 'this_month'
  | 'last_month'
  | 'this_year'
  | 'last_30_days'
  | 'last_90_days'
  | 'custom';

export interface TimeSeriesDataPoint {
  date: string;
  label: string;
  value: number;
}

// Sales Overview Types
export interface SalesMetrics {
  total_revenue: number;
  total_orders: number;
  average_order_value: number;
  inventory_value: number; // Sum of (product.cost * product.quantity)
  revenue_change: number; // Percentage
  orders_change: number; // Percentage
  aov_change: number; // Percentage

  // US5: Offline Order Metrics (T107)
  offline_order_count: number;
  offline_revenue: number;
  offline_percentage: number;
  online_order_count: number;
  online_revenue: number;
  installment_count: number;
  installment_revenue: number;
  pending_installments: number;
}

export interface DailySalesData {
  date: string;
  revenue: number;
  orders: number;
}

export interface TopProduct {
  productId: number;
  name: string;
  quantitySold: number;
  revenue: number;
  imageUrl?: string;
}

export interface CategorySales {
  category: string;
  revenue: number;
  percentage: number;
}

export interface SalesOverviewResponse {
  metrics: SalesMetrics;
  salesChart: TimeSeriesDataPoint[];
  topProducts: TopProduct[];
  categoryBreakdown: CategorySales[];
}

// Customer Insights Types
export interface CustomerMetrics {
  totalCustomers: number;
  newCustomers: number;
  returningCustomers: number;
  customerGrowth: number; // Percentage
}

export interface CustomerSegment {
  segment: string; // 'new', 'returning', 'loyal'
  count: number;
  percentage: number;
  averageOrderValue: number;
}

export interface TopCustomer {
  customerId: number;
  name: string; // Masked for privacy
  orderCount: number;
  totalSpent: number;
  email?: string; // Masked
  phone?: string; // Masked
}

export interface CustomerInsightsResponse {
  metrics: CustomerMetrics;
  segmentBreakdown: CustomerSegment[];
  topCustomers: TopCustomer[];
  newCustomersChart: TimeSeriesDataPoint[];
}

// Product Ranking (from backend API)
export interface ProductRanking {
  product_id: number;
  name: string;
  category_name: string;
  quantity_sold: number;
  revenue: number;
  sku: string;
  image_url?: string;
}

export interface TopProductsResponse {
  top_by_revenue: ProductRanking[];
  top_by_quantity: ProductRanking[];
  bottom_by_revenue: ProductRanking[];
  bottom_by_quantity: ProductRanking[];
}

// Customer Ranking (from backend API with masked PII)
export interface CustomerRanking {
  name: string;
  phone: string;
  email: string;
  order_count: number;
  total_spent: number;
  avg_order_value: number;
}

export interface TopCustomersResponse {
  top_by_spending: CustomerRanking[];
  top_by_orders: CustomerRanking[];
}

// Operational Tasks Types
export interface DelayedOrder {
  order_id: string;
  order_number: string;
  order_type?: 'online' | 'offline';
  customer_id: number;
  masked_phone: string;
  masked_name: string;
  masked_email: string;
  total_amount: number;
  item_count: number;
  status: string;
  created_at: string;
  elapsed_minutes: number;
}

export interface DelayedOrdersResponse {
  count: number;
  urgent_count: number; // > 30 minutes
  warning_count: number; // 15-30 minutes
  delayed_orders: DelayedOrder[];
}

export interface RestockAlert {
  product_id: number;
  product_name: string;
  category_name: string;
  sku: string;
  current_stock: number;
  low_stock_threshold: number;
  recommended_reorder: number;
  status: 'critical' | 'low'; // critical = 0 stock, low = below threshold
  selling_price: number;
  cost_price: number;
  image_url?: string;
}

export interface RestockAlertsResponse {
  count: number;
  critical_count: number; // 0 stock
  low_stock_count: number; // below threshold
  restock_alerts: RestockAlert[];
}

export interface OperationalTasksResponse {
  delayed_orders: DelayedOrdersResponse;
  restock_alerts: RestockAlertsResponse;
}

// Time Series Types
export interface SalesTrendResponse {
  period: string; // e.g., "daily", "monthly"
  start_date: string; // ISO 8601 date
  end_date: string; // ISO 8601 date
  revenue_data: TimeSeriesDataPoint[];
  orders_data: TimeSeriesDataPoint[];
}

// Inventory & Product Tasks Types
export interface InventoryAlert {
  productId: number;
  name: string;
  currentStock: number;
  lowStockThreshold: number;
  status: 'critical' | 'warning' | 'ok';
  daysUntilStockout?: number;
}

export interface ProductTask {
  taskId: string;
  productId: number;
  productName: string;
  taskType: 'restock' | 'review_price' | 'update_info';
  priority: 'high' | 'medium' | 'low';
  description: string;
  createdAt: string;
  dueDate?: string;
}

export interface InventoryMetrics {
  totalProducts: number;
  lowStockProducts: number;
  outOfStockProducts: number;
  totalInventoryValue: number;
}

export interface InventoryTasksResponse {
  metrics: InventoryMetrics;
  alerts: InventoryAlert[];
  tasks: ProductTask[];
}

// Chart Configuration Types
export interface ChartConfig {
  dataKey: string;
  name: string;
  color: string;
  formatter?: (value: number) => string;
}

// API Request Types
export interface AnalyticsRequest {
  timeRange: TimeRange;
  startDate?: string; // ISO 8601 format (YYYY-MM-DD)
  endDate?: string; // ISO 8601 format (YYYY-MM-DD)
}

export interface TaskFilterRequest {
  priority?: 'high' | 'medium' | 'low';
  taskType?: 'restock' | 'review_price' | 'update_info';
}

// Inventory Analytics Types (Phase 5)

export interface ProductProfitability {
  product_id: string;
  product_name: string;
  total_revenue: number;
  total_cogs: number;
  gross_profit: number;
  gross_margin_pct: number;
  total_quantity_sold: number;
  avg_selling_price: number;
  avg_cogs_per_unit: number;
  recipe_version: number;
}

export interface ProductProfitabilityResponse {
  products: ProductProfitability[];
  start_date: string;
  end_date: string;
  total_items: number;
}

export interface IngredientUsage {
  ingredient_id: string;
  ingredient_name: string;
  total_consumed: number;
  base_unit: string;
  total_cost: number;
  avg_cost_per_unit: number;
  current_stock: number;
  stock_after_forecast: number;
  days_until_stockout: number | null;
}

export interface DailyBurnRate {
  date: string;
  ingredient_id: string;
  quantity_used: number;
}

export interface IngredientForecastResponse {
  ingredients: IngredientUsage[];
  start_date: string;
  end_date: string;
  daily_burn_rate: DailyBurnRate[];
}

export interface RevenueTarget {
  target_revenue: number;
  period_days: number;
  daily_target: number;
  weekly_target: number;
}

export interface RecommendedProductMix {
  product_id: string;
  product_name: string;
  recommended_quantity: number;
  estimated_revenue: number;
  estimated_cogs: number;
  gross_margin_pct: number;
  score_reason: string;
}

export interface RevenueSimulatorResponse {
  target: RevenueTarget;
  recommended_mix: RecommendedProductMix[];
  estimated_ingredient_usage: IngredientUsage[];
  estimated_budget: number;
  projected_gross_profit: number;
  projected_gross_margin: number;
  feasibility_score: number;
  feasibility_notes: string[];
}

export interface IngredientBudgetLine {
  ingredient_id: string;
  ingredient_name: string;
  allocated_budget: number;
  quantity_to_buy: number;
  unit: string;
  cost_per_unit: number;
}

export interface BudgetSimulatorResponse {
  budget: number;
  start_date: string;
  end_date: string;
  recommended_mix: RecommendedProductMix[];
  ingredient_breakdown: IngredientBudgetLine[];
  projected_revenue: number;
  projected_gross_profit: number;
  projected_gross_margin: number;
  budget_utilization: number;
}

export interface ProductRecommendation {
  product_id: string;
  product_name: string;
  total_score: number;
  margin_score: number;
  velocity_score: number;
  stock_readiness_score: number;
  trend_score: number;
  bundle_opportunity: boolean;
  recommendation_note: string;
}

export interface ScoringWeights {
  margin: number;
  velocity: number;
  stock_readiness: number;
  trend: number;
}

export interface ProductRecommendationResponse {
  recommendations: ProductRecommendation[];
  generated_at: string;
  scoring_weights: ScoringWeights;
}

// Error Types
export interface AnalyticsError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}
