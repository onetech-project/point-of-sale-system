'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import {
  AlertTriangle,
  Boxes,
  Calculator,
  PackageSearch,
  RefreshCw,
  Ruler,
  Utensils,
} from 'lucide-react';
import { inventory } from '@/services/inventory';
import type {
  Ingredient,
  InventoryOverview as InventoryOverviewData,
  UOM,
} from '@/types/inventory';
import { formatCurrency } from '@/utils/format';
import {
  formatQuantity,
  getApiErrorMessage,
  getStockStatus,
  getUOMCode,
  toNumber,
} from './inventoryUtils';

interface MetricCardProps {
  label: string;
  value: string;
  icon: React.ReactNode;
  tone: 'blue' | 'green' | 'yellow' | 'gray';
}

const toneClasses = {
  blue: 'bg-blue-50 text-blue-700',
  green: 'bg-green-50 text-green-700',
  yellow: 'bg-yellow-50 text-yellow-800',
  gray: 'bg-gray-100 text-gray-700',
};

const MetricCard = ({ label, value, icon, tone }: MetricCardProps) => (
  <div className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
    <div className="flex items-center justify-between gap-4">
      <div>
        <p className="text-sm font-medium text-gray-500">{label}</p>
        <p className="mt-2 text-2xl font-bold text-gray-900">{value}</p>
      </div>
      <div className={`flex h-11 w-11 items-center justify-center rounded-lg ${toneClasses[tone]}`}>
        {icon}
      </div>
    </div>
  </div>
);

export default function InventoryOverview() {
  const [summary, setSummary] = useState<InventoryOverviewData>({
    total_ingredients: 0,
    active_uoms: 0,
    low_stock_count: 0,
    valuation: 0,
  });
  const [lowStock, setLowStock] = useState<Ingredient[]>([]);
  const [uoms, setUOMs] = useState<UOM[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void fetchOverview();
  }, []);

  const fetchOverview = async () => {
    try {
      setLoading(true);
      setError(null);

      const [ingredientsResponse, lowStockResponse, valuationResponse, uomResponse] =
        await Promise.all([
          inventory.getIngredients({ page: 1, limit: 1 }),
          inventory.getLowStockIngredients({ page: 1, limit: 6 }),
          inventory.getValuation(),
          inventory.getUOMs(false),
        ]);

      setSummary({
        total_ingredients: ingredientsResponse.total,
        active_uoms: uomResponse.length,
        low_stock_count: lowStockResponse.total,
        valuation: valuationResponse.valuation,
      });
      setLowStock(lowStockResponse.data);
      setUOMs(uomResponse);
    } catch (err) {
      console.error('Failed to fetch inventory overview:', err);
      setError(getApiErrorMessage(err, 'Failed to load inventory overview.'));
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="text-sm text-gray-500">Loading inventory...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-md border border-red-200 bg-red-50 p-4">
        <p className="text-sm text-red-700">{error}</p>
        <button
          type="button"
          onClick={fetchOverview}
          className="mt-2 text-sm font-medium text-red-700 underline hover:text-red-900"
        >
          Try again
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          label="Ingredients"
          value={summary.total_ingredients.toLocaleString('en-US')}
          icon={<PackageSearch className="h-5 w-5" aria-hidden="true" />}
          tone="blue"
        />
        <MetricCard
          label="Inventory value"
          value={formatCurrency(toNumber(summary.valuation), false, 0)}
          icon={<Calculator className="h-5 w-5" aria-hidden="true" />}
          tone="green"
        />
        <MetricCard
          label="Low stock"
          value={summary.low_stock_count.toLocaleString('en-US')}
          icon={<AlertTriangle className="h-5 w-5" aria-hidden="true" />}
          tone="yellow"
        />
        <MetricCard
          label="Active units"
          value={summary.active_uoms.toLocaleString('en-US')}
          icon={<Ruler className="h-5 w-5" aria-hidden="true" />}
          tone="gray"
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-[1.4fr_0.9fr]">
        <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
          <div className="flex items-center justify-between gap-4 border-b border-gray-200 px-5 py-4">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">Low-stock ingredients</h2>
              <p className="text-sm text-gray-500">Items at or below their minimum stock.</p>
            </div>
            <button
              type="button"
              onClick={fetchOverview}
              className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              <RefreshCw className="h-4 w-4" aria-hidden="true" />
              Refresh
            </button>
          </div>

          {lowStock.length === 0 ? (
            <div className="px-5 py-10 text-center text-sm text-gray-500">
              No low-stock ingredients.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                      Ingredient
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                      Current
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                      Minimum
                    </th>
                    <th className="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 bg-white">
                  {lowStock.map(ingredient => {
                    const status = getStockStatus(
                      ingredient.current_stock_base,
                      ingredient.minimum_stock_base,
                      ingredient.is_active
                    );

                    return (
                      <tr key={ingredient.id}>
                        <td className="whitespace-nowrap px-5 py-4">
                          <div className="font-medium text-gray-900">{ingredient.name}</div>
                          <div className="text-sm text-gray-500">{ingredient.sku}</div>
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700">
                          {formatQuantity(ingredient.current_stock_base)}{' '}
                          {ingredient.base_uom_code || getUOMCode(uoms, ingredient.base_uom_id)}
                        </td>
                        <td className="whitespace-nowrap px-5 py-4 text-sm text-gray-700">
                          {formatQuantity(ingredient.minimum_stock_base)}{' '}
                          {ingredient.base_uom_code || getUOMCode(uoms, ingredient.base_uom_id)}
                        </td>
                        <td className="whitespace-nowrap px-5 py-4">
                          <span
                            className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${status.className}`}
                          >
                            {status.label}
                          </span>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <section className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-900">Inventory tasks</h2>
          <div className="mt-4 space-y-3">
            <Link
              href="/inventory/ingredients"
              className="flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3 text-sm font-medium text-gray-700 hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700"
            >
              Manage ingredients
              <Boxes className="h-4 w-4" aria-hidden="true" />
            </Link>
            <Link
              href="/inventory/uoms"
              className="flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3 text-sm font-medium text-gray-700 hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700"
            >
              Manage units
              <Ruler className="h-4 w-4" aria-hidden="true" />
            </Link>
            <Link
              href="/inventory/stock"
              className="flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3 text-sm font-medium text-gray-700 hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700"
            >
              Record stock movement
              <PackageSearch className="h-4 w-4" aria-hidden="true" />
            </Link>
            <Link
              href="/inventory/recipes"
              className="flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3 text-sm font-medium text-gray-700 hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700"
            >
              Review recipe costing
              <Calculator className="h-4 w-4" aria-hidden="true" />
            </Link>
            <Link
              href="/inventory/bundles"
              className="flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3 text-sm font-medium text-gray-700 hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700"
            >
              Manage bundles
              <Utensils className="h-4 w-4" aria-hidden="true" />
            </Link>
          </div>
        </section>
      </div>
    </div>
  );
}
