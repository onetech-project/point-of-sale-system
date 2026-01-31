'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/store/auth';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import DashboardLayout from '../../src/components/layout/DashboardLayout';
import { DashboardLayout as AnalyticsLayout } from '@/components/dashboard/DashboardLayout';
import { MetricCard } from '@/components/dashboard/MetricCard';
import { ProductRankingTable } from '@/components/dashboard/ProductRankingTable';
import { CustomerRankingTable } from '@/components/dashboard/CustomerRankingTable';
import { TaskAlerts } from '@/components/dashboard/TaskAlerts';
import { TimeSeriesFilter, Granularity } from '@/components/dashboard/TimeSeriesFilter';
import { SalesChart } from '@/components/dashboard/SalesChart';
import { QuickActions } from '@/components/dashboard/QuickActions';
import { DashboardErrorBoundary } from '@/components/common/ErrorBoundary';
import analytics from '@/services/analytics';
import { formatCurrency, formatNumber } from '@/utils/format';
import type { 
  SalesOverviewResponse, 
  TopProductsResponse, 
  TopCustomersResponse,
  OperationalTasksResponse,
  SalesTrendResponse,
  TimeRange 
} from '@/types/analytics';

export default function AnalyticsDashboardPage() {
  const router = useRouter();
  
  // Helper function to format date in user's local timezone
  const formatLocalDate = (date: Date): string => {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  };

  // State for overview section
  const [timeRange, setTimeRange] = useState<TimeRange>('this_month');
  const [salesData, setSalesData] = useState<SalesOverviewResponse | null>(null);
  const [topProducts, setTopProducts] = useState<TopProductsResponse | null>(null);
  const [topCustomers, setTopCustomers] = useState<TopCustomersResponse | null>(null);
  const [tasks, setTasks] = useState<OperationalTasksResponse | null>(null);
  
  // State for time series section
  const [granularity, setGranularity] = useState<Granularity>('daily');
  const [startDate, setStartDate] = useState(() => {
    const now = new Date();
    const firstOfMonth = new Date(now.getFullYear(), now.getMonth(), 1);
    return formatLocalDate(firstOfMonth);
  });
  const [endDate, setEndDate] = useState(() => formatLocalDate(new Date()));
  const [trendData, setTrendData] = useState<SalesTrendResponse | null>(null);
  
  const [loading, setLoading] = useState(true);
  const [trendLoading, setTrendLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Sync time series date range with timeRange selector
  useEffect(() => {
    const now = new Date();
    let start: Date;
    let end: Date;

    switch (timeRange) {
      case 'today':
        start = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        end = new Date(now);
        break;
      case 'yesterday':
        start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1);
        end = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1);
        break;
      case 'this_week':
        const dayOfWeek = now.getDay();
        const mondayOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;
        start = new Date(now.getFullYear(), now.getMonth(), now.getDate() + mondayOffset);
        end = new Date(now);
        break;
      case 'last_week':
        // Monday-based week calculation
        const currentDayOfWeek = now.getDay();
        const daysFromMonday = currentDayOfWeek === 0 ? 6 : currentDayOfWeek - 1; // Convert Sunday (0) to 6, others to Mon=0, Tue=1, etc.
        const lastMonday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - daysFromMonday - 7);
        const lastSunday = new Date(lastMonday.getFullYear(), lastMonday.getMonth(), lastMonday.getDate() + 6);
        start = lastMonday;
        end = lastSunday;
        break;
      case 'this_month':
        start = new Date(now.getFullYear(), now.getMonth(), 1);
        end = new Date(now);
        break;
      case 'last_month':
        start = new Date(now.getFullYear(), now.getMonth() - 1, 1);
        end = new Date(now.getFullYear(), now.getMonth(), 0);
        break;
      case 'last_30_days':
        start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 30);
        end = new Date(now);
        break;
      case 'last_90_days':
        start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 90);
        end = new Date(now);
        break;
      case 'this_year':
        start = new Date(now.getFullYear(), 0, 1);
        end = new Date(now);
        break;
      default:
        start = new Date(now.getFullYear(), now.getMonth(), 1);
        end = new Date(now);
    }

    setStartDate(formatLocalDate(start));
    setEndDate(formatLocalDate(end));
  }, [timeRange]);

  // Fetch overview dashboard data
  useEffect(() => {
    fetchDashboardData();
  }, [timeRange]);

  // Fetch trend data when filters change
  useEffect(() => {
    fetchTrendData();
  }, [granularity, startDate, endDate]);

  const fetchDashboardData = async () => {
    try {
      setLoading(true);
      setError(null);

      // Fetch all data in parallel
      const [sales, products, customers, operationalTasks] = await Promise.all([
        analytics.getSalesOverview(timeRange),
        analytics.getTopProducts(timeRange, 5),
        analytics.getTopCustomers(timeRange, 5),
        analytics.getOperationalTasks(),
      ]);

      setSalesData(sales);
      setTopProducts(products);
      setTopCustomers(customers);
      setTasks(operationalTasks);
    } catch (err: any) {
      console.error('Failed to fetch dashboard data:', err);
      setError(
        err.response?.data?.message || 
        'Failed to load dashboard data. Please try again later.'
      );
    } finally {
      setLoading(false);
    }
  };

  const fetchTrendData = async () => {
    try {
      setTrendLoading(true);
      const trend = await analytics.getSalesTrend(granularity, startDate, endDate);
      setTrendData(trend);
    } catch (err: any) {
      console.error('Failed to fetch trend data:', err);
      // Don't set global error for trend failures, just log
    } finally {
      setTrendLoading(false);
    }
  };

  const handleDateRangeChange = (newStartDate: string, newEndDate: string) => {
    setStartDate(newStartDate);
    setEndDate(newEndDate);
  };

  // Time range selector options
  const timeRangeOptions: { value: TimeRange; label: string }[] = [
    { value: 'today', label: 'Today' },
    { value: 'yesterday', label: 'Yesterday' },
    { value: 'this_week', label: 'This Week' },
    { value: 'last_week', label: 'Last Week' },
    { value: 'this_month', label: 'This Month' },
    { value: 'last_month', label: 'Last Month' },
    { value: 'last_30_days', label: 'Last 30 Days' },
    { value: 'last_90_days', label: 'Last 90 Days' },
  ];

  // Render error state
  if (error && !loading) {
    return (
      <ProtectedRoute>
        <DashboardLayout>
          <AnalyticsLayout title="Business Insights">
            <div className="bg-red-50 border border-red-200 rounded-lg p-6">
              <div className="flex items-start gap-3">
                <svg
                  className="w-6 h-6 text-red-600 flex-shrink-0 mt-0.5"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                <div>
                  <h3 className="text-sm font-medium text-red-800">Error Loading Dashboard</h3>
                  <p className="mt-1 text-sm text-red-700">{error}</p>
                  <button
                    onClick={fetchDashboardData}
                    className="mt-3 text-sm text-red-600 underline hover:text-red-800"
                  >
                    Try Again
                  </button>
                </div>
              </div>
            </div>
          </AnalyticsLayout>
        </DashboardLayout>
      </ProtectedRoute>
    );
  }

  return (
    <ProtectedRoute>
      <DashboardLayout>
        <AnalyticsLayout
          title="Business Insights"
          actions={
            <div className="flex items-center gap-3">
              <select
                value={timeRange}
                onChange={(e) => setTimeRange(e.target.value as TimeRange)}
                className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              >
                {timeRangeOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
              <button
                onClick={fetchDashboardData}
                disabled={loading}
                className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                <svg
                  className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                  />
                </svg>
                Refresh
              </button>
            </div>
          }
        >
          {/* Metrics Cards */}
          <DashboardErrorBoundary sectionName="Sales Metrics">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <MetricCard
              title="Total Revenue"
              value={salesData ? formatCurrency(salesData.metrics.total_revenue, true) : '—'}
              change={salesData?.metrics.revenue_change}
              changeLabel="vs previous period"
              loading={loading}
              icon={
                <svg
                  className="w-6 h-6 text-primary-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              }
            />
            <MetricCard
              title="Total Orders"
              value={salesData ? formatNumber(salesData.metrics.total_orders, 0) : '—'}
              change={salesData?.metrics.orders_change}
              changeLabel="vs previous period"
              loading={loading}
              icon={
                <svg
                  className="w-6 h-6 text-primary-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z"
                  />
                </svg>
              }
            />
            <MetricCard
              title="Average Order Value"
              value={salesData ? formatCurrency(salesData.metrics.average_order_value, true) : '—'}
              change={salesData?.metrics.aov_change}
              changeLabel="vs previous period"
              loading={loading}
              icon={
                <svg
                  className="w-6 h-6 text-primary-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 7h6m0 10v-3m-3 3h.01M9 17h.01M9 14h.01M12 14h.01M15 11h.01M12 11h.01M9 11h.01M7 21h10a2 2 0 002-2V5a2 2 0 00-2-2H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                  />
                </svg>
              }
            />
            <MetricCard
              title="Inventory Value"
              value={salesData ? formatCurrency(salesData.metrics.inventory_value, true) : '—'}
              changeLabel="total stock value"
              loading={loading}
              icon={
                <svg
                  className="w-6 h-6 text-primary-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
                  />
                </svg>
              }
            />
          </div>
          </DashboardErrorBoundary>

          {/* Time Series Chart Section */}
          <DashboardErrorBoundary sectionName="Sales Trends">
            <div className="space-y-1">
              <TimeSeriesFilter
                granularity={granularity}
                startDate={startDate}
                endDate={endDate}
                onGranularityChange={setGranularity}
                onDateRangeChange={handleDateRangeChange}
                loading={trendLoading}
              />
              <SalesChart
                revenueData={trendData?.revenue_data || []}
                ordersData={trendData?.orders_data || []}
                loading={trendLoading}
                height={400}
              />
            </div>
          </DashboardErrorBoundary>

          {/* Quick Actions */}
          <DashboardErrorBoundary sectionName="Quick Actions">
            <QuickActions />
          </DashboardErrorBoundary>

          {/* Operational Tasks (Delayed Orders & Low Stock) */}
          <DashboardErrorBoundary sectionName="Operational Tasks">
            <TaskAlerts
              delayedOrders={tasks?.delayed_orders.delayed_orders || []}
              restockAlerts={tasks?.restock_alerts.restock_alerts || []}
              loading={loading}
              onNavigateToOrder={(orderId) => router.push(`/orders/${orderId}`)}
              onNavigateToProduct={(productId) => router.push(`/products/${productId}`)}
            />
          </DashboardErrorBoundary>

          {/* Product Rankings */}
          <DashboardErrorBoundary sectionName="Product Rankings">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <ProductRankingTable
                products={topProducts?.top_by_revenue || []}
                loading={loading}
                type="top"
                metric="revenue"
              />
              <ProductRankingTable
                products={topProducts?.top_by_quantity || []}
                loading={loading}
                type="top"
                metric="quantity"
              />
            </div>

            {/* Bottom Performers */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <ProductRankingTable
                products={topProducts?.bottom_by_revenue || []}
                loading={loading}
                type="bottom"
                metric="revenue"
              />
              <ProductRankingTable
                products={topProducts?.bottom_by_quantity || []}
                loading={loading}
                type="bottom"
                metric="quantity"
              />
            </div>
          </DashboardErrorBoundary>

          {/* Customer Rankings */}
          <DashboardErrorBoundary sectionName="Customer Rankings">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <CustomerRankingTable
              customers={topCustomers?.top_by_spending || []}
              loading={loading}
            />
            <CustomerRankingTable
              customers={topCustomers?.top_by_orders || []}
              loading={loading}
            />
          </div>
          </DashboardErrorBoundary>

          {/* Footer Note */}
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <div className="flex items-start gap-3">
              <svg
                className="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <div className="text-sm text-blue-700">
                <p className="font-medium">Data Privacy</p>
                <p className="mt-1">
                  Customer information is encrypted and masked to protect privacy. 
                  All analytics data is cached for performance and refreshed periodically.
                </p>
              </div>
            </div>
          </div>
        </AnalyticsLayout>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
