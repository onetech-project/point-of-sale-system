'use client';

import { useState, useEffect } from 'react';
import analyticsService from '@/services/analytics';
import type { SalesOverviewResponse, TopProductsResponse } from '@/types/analytics';

export default function AnalyticsPage() {
  const [overview, setOverview] = useState<SalesOverviewResponse | null>(null);
  const [topProducts, setTopProducts] = useState<TopProductsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [overviewData, topProductsData] = await Promise.all([
        analyticsService.getSalesOverview('this_month'),
        analyticsService.getTopProducts('this_month', 5),
      ]);
      setOverview(overviewData);
      setTopProducts(topProductsData);
    } catch (err) {
      setError('Failed to load analytics data');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">Loading analytics...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg bg-red-50 p-4 text-red-700">
        {error}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Revenue"
          value={`Rp ${(overview?.metrics.total_revenue || 0).toLocaleString()}`}
          change={overview?.metrics.revenue_change}
        />
        <MetricCard
          title="Total Orders"
          value={(overview?.metrics.total_orders || 0).toLocaleString()}
          change={overview?.metrics.orders_change}
        />
        <MetricCard
          title="Average Order Value"
          value={`Rp ${(overview?.metrics.average_order_value || 0).toLocaleString()}`}
          change={overview?.metrics.aov_change}
        />
        <MetricCard
          title="Inventory Value"
          value={`Rp ${(overview?.metrics.inventory_value || 0).toLocaleString()}`}
        />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div className="rounded-lg border bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Top Products by Revenue</h3>
          <div className="space-y-3">
            {topProducts?.top_by_revenue?.slice(0, 5).map((product) => (
              <div key={product.product_id} className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="h-10 w-10 rounded bg-gray-100 flex items-center justify-center text-gray-500 text-sm">
                    {product.name.charAt(0)}
                  </div>
                  <div>
                    <div className="font-medium text-gray-900">{product.name}</div>
                    <div className="text-sm text-gray-500">{product.quantity_sold} sold</div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="font-medium text-gray-900">Rp {product.revenue.toLocaleString()}</div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-lg border bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Quick Actions</h3>
          <div className="space-y-3">
            <a
              href="/analytics/inventory"
              className="block rounded-lg border p-4 hover:bg-gray-50 transition-colors"
            >
              <div className="font-medium text-gray-900">Inventory Analytics</div>
              <div className="text-sm text-gray-500">
                Product profitability, ingredient forecasting, and revenue simulation
              </div>
            </a>
            <a
              href="/inventory"
              className="block rounded-lg border p-4 hover:bg-gray-50 transition-colors"
            >
              <div className="font-medium text-gray-900">Manage Inventory</div>
              <div className="text-sm text-gray-500">
                Ingredients, stock movements, and recipes
              </div>
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}

function MetricCard({
  title,
  value,
  change,
}: {
  title: string;
  value: string;
  change?: number;
}) {
  return (
    <div className="rounded-lg border bg-white p-6 shadow-sm">
      <div className="text-sm font-medium text-gray-500">{title}</div>
      <div className="mt-2 text-2xl font-bold text-gray-900">{value}</div>
      {change !== undefined && (
        <div
          className={`mt-1 text-sm ${
            change >= 0 ? 'text-green-600' : 'text-red-600'
          }`}
        >
          {change >= 0 ? '+' : ''}{change.toFixed(1)}% vs last period
        </div>
      )}
    </div>
  );
}
