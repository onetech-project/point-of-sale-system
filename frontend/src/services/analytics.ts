import apiClient from './api';
import {
  SalesOverviewResponse,
  TopProductsResponse,
  TopCustomersResponse,
  OperationalTasksResponse,
  SalesTrendResponse,
  TimeRange,
} from '../types/analytics';

const ANALYTICS_BASE = '/api/v1/analytics';

class AnalyticsService {
  /**
   * Get sales overview with metrics, daily sales chart, and category breakdown
   * @param timeRange - Time range for analytics (today, this_month, custom, etc.)
   * @param startDate - Start date for custom range (YYYY-MM-DD format)
   * @param endDate - End date for custom range (YYYY-MM-DD format)
   * @returns Sales overview data
   */
  async getSalesOverview(
    timeRange: TimeRange = 'this_month',
    startDate?: string,
    endDate?: string
  ): Promise<SalesOverviewResponse> {
    const params = new URLSearchParams();
    params.append('time_range', timeRange);

    if (timeRange === 'custom' && startDate && endDate) {
      params.append('start_date', startDate);
      params.append('end_date', endDate);
    }

    const url = `${ANALYTICS_BASE}/overview?${params.toString()}`;
    return apiClient.get<SalesOverviewResponse>(url);
  }

  /**
   * Get top and bottom products by revenue and quantity
   * @param timeRange - Time range for analytics
   * @param limit - Number of products to return per category (default: 5, max: 20)
   * @param startDate - Start date for custom range
   * @param endDate - End date for custom range
   * @returns Top and bottom products
   */
  async getTopProducts(
    timeRange: TimeRange = 'this_month',
    limit: number = 5,
    startDate?: string,
    endDate?: string
  ): Promise<TopProductsResponse> {
    const params = new URLSearchParams();
    params.append('time_range', timeRange);
    params.append('limit', limit.toString());

    if (timeRange === 'custom' && startDate && endDate) {
      params.append('start_date', startDate);
      params.append('end_date', endDate);
    }

    const url = `${ANALYTICS_BASE}/top-products?${params.toString()}`;
    return apiClient.get<TopProductsResponse>(url);
  }

  /**
   * Get top customers by spending and order count
   * Note: Customer PII (name, phone, email) is masked for privacy
   * @param timeRange - Time range for analytics
   * @param limit - Number of customers to return per category (default: 5, max: 20)
   * @param startDate - Start date for custom range
   * @param endDate - End date for custom range
   * @returns Top customers with masked PII
   */
  async getTopCustomers(
    timeRange: TimeRange = 'this_month',
    limit: number = 5,
    startDate?: string,
    endDate?: string
  ): Promise<TopCustomersResponse> {
    const params = new URLSearchParams();
    params.append('time_range', timeRange);
    params.append('limit', limit.toString());

    if (timeRange === 'custom' && startDate && endDate) {
      params.append('start_date', startDate);
      params.append('end_date', endDate);
    }

    const url = `${ANALYTICS_BASE}/top-customers?${params.toString()}`;
    return apiClient.get<TopCustomersResponse>(url);
  }

  /**
   * Get operational tasks: delayed orders (>15 min) and low stock alerts
   * @returns Delayed orders and restock alerts with counts
   */
  async getOperationalTasks(): Promise<OperationalTasksResponse> {
    const url = `${ANALYTICS_BASE}/tasks`;
    return apiClient.get<OperationalTasksResponse>(url);
  }

  /**
   * Get sales trend time series data
   * @param granularity - Time series granularity (daily, weekly, monthly, quarterly, yearly)
   * @param startDate - Start date (YYYY-MM-DD format)
   * @param endDate - End date (YYYY-MM-DD format)
   * @returns Revenue and orders time series data
   */
  async getSalesTrend(
    granularity: 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'yearly',
    startDate: string,
    endDate: string
  ): Promise<SalesTrendResponse> {
    const params = new URLSearchParams();
    params.append('granularity', granularity);
    params.append('start_date', startDate);
    params.append('end_date', endDate);

    const url = `${ANALYTICS_BASE}/sales-trend?${params.toString()}`;
    return apiClient.get<SalesTrendResponse>(url);
  }
}

const analyticsService = new AnalyticsService();

export default analyticsService;
