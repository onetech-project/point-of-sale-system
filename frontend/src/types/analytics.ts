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
  order_id: number;
  order_number: string;
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
  urgent_count: number;   // > 30 minutes
  warning_count: number;  // 15-30 minutes
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
  status: 'critical' | 'low';  // critical = 0 stock, low = below threshold
  selling_price: number;
  cost_price: number;
  image_url?: string;
}

export interface RestockAlertsResponse {
  count: number;
  critical_count: number;  // 0 stock
  low_stock_count: number; // below threshold
  restock_alerts: RestockAlert[];
}

export interface OperationalTasksResponse {
  delayed_orders: DelayedOrdersResponse;
  restock_alerts: RestockAlertsResponse;
}

// Time Series Types
export interface SalesTrendResponse {
  period: string;       // e.g., "daily", "monthly"
  start_date: string;   // ISO 8601 date
  end_date: string;     // ISO 8601 date
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

// Error Types
export interface AnalyticsError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}
