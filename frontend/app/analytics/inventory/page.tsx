'use client';

import { useState, useEffect } from 'react';
import analyticsService from '@/services/analytics';
import type {
  ProductProfitabilityResponse,
  IngredientForecastResponse,
  RevenueSimulatorResponse,
  BudgetSimulatorResponse,
  ProductRecommendationResponse,
} from '@/types/analytics';

type Tab = 'profitability' | 'forecast' | 'revenue-sim' | 'budget-sim' | 'recommendations';

export default function InventoryAnalyticsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('profitability');
  const [startDate, setStartDate] = useState(() => {
    const d = new Date();
    d.setMonth(d.getMonth() - 1);
    return d.toISOString().split('T')[0];
  });
  const [endDate, setEndDate] = useState(() => new Date().toISOString().split('T')[0]);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Inventory Analytics</h2>
          <p className="text-sm text-gray-500">
            Profitability analysis, ingredient forecasting, and strategic planning.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <input
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="rounded-md border border-gray-300 px-3 py-2 text-sm"
          />
          <span className="text-gray-500">to</span>
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="rounded-md border border-gray-300 px-3 py-2 text-sm"
          />
        </div>
      </div>

      <div className="flex space-x-1 rounded-lg bg-gray-100 p-1">
        {(
          [
            { id: 'profitability', label: 'Profitability' },
            { id: 'forecast', label: 'Ingredient Forecast' },
            { id: 'revenue-sim', label: 'Revenue Simulator' },
            { id: 'budget-sim', label: 'Budget Simulator' },
            { id: 'recommendations', label: 'Recommendations' },
          ] as const
        ).map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? 'bg-white text-gray-900 shadow-sm'
                : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === 'profitability' && (
        <ProfitabilityTab startDate={startDate} endDate={endDate} />
      )}
      {activeTab === 'forecast' && (
        <ForecastTab startDate={startDate} endDate={endDate} />
      )}
      {activeTab === 'revenue-sim' && (
        <RevenueSimulatorTab startDate={startDate} endDate={endDate} />
      )}
      {activeTab === 'budget-sim' && (
        <BudgetSimulatorTab startDate={startDate} endDate={endDate} />
      )}
      {activeTab === 'recommendations' && <RecommendationsTab />}
    </div>
  );
}

function ProfitabilityTab({ startDate, endDate }: { startDate: string; endDate: string }) {
  const [data, setData] = useState<ProductProfitabilityResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [startDate, endDate]);

  const loadData = async () => {
    try {
      setLoading(true);
      const result = await analyticsService.getProductProfitability(startDate, endDate, 20);
      setData(result);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <LoadingSpinner />;
  if (!data) return <ErrorMessage message="Failed to load profitability data" />;

  const products = data.products || [];
  
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <StatCard
          title="Total Revenue"
          value={`Rp ${products.reduce((sum, p) => sum + p.total_revenue, 0).toLocaleString()}`}
        />
        <StatCard
          title="Total COGS"
          value={`Rp ${products.reduce((sum, p) => sum + p.total_cogs, 0).toLocaleString()}`}
        />
        <StatCard
          title="Total Gross Profit"
          value={`Rp ${products.reduce((sum, p) => sum + p.gross_profit, 0).toLocaleString()}`}
          highlight
        />
      </div>

      <div className="rounded-lg border bg-white shadow-sm overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Product
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Qty Sold
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Revenue
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                COGS
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Gross Profit
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Margin %
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {products.map((product) => (
              <tr key={product.product_id} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="font-medium text-gray-900">{product.product_name}</div>
                  <div className="text-sm text-gray-500">v{product.recipe_version}</div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-900">
                  {product.total_quantity_sold}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-900">
                  Rp {product.total_revenue.toLocaleString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-900">
                  Rp {product.total_cogs.toLocaleString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-green-600">
                  Rp {product.gross_profit.toLocaleString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      product.gross_margin_pct >= 30
                        ? 'bg-green-100 text-green-800'
                        : product.gross_margin_pct >= 20
                        ? 'bg-yellow-100 text-yellow-800'
                        : 'bg-red-100 text-red-800'
                    }`}
                  >
                    {Number(product.gross_margin_pct).toFixed(1)}%
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function ForecastTab({ startDate, endDate }: { startDate: string; endDate: string }) {
  const [data, setData] = useState<IngredientForecastResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, [startDate, endDate]);

  const loadData = async () => {
    try {
      setLoading(true);
      const result = await analyticsService.getIngredientForecast(startDate, endDate);
      setData(result);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <LoadingSpinner />;
  if (!data) return <ErrorMessage message="Failed to load forecast data" />;

  const ingredients = data.ingredients || [];

  return (
    <div className="space-y-6">
      <div className="rounded-lg border bg-white shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">Ingredient Usage Forecast</h3>
          <p className="text-sm text-gray-500">Based on historical consumption patterns</p>
        </div>
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Ingredient
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Consumed
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Current Stock
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                After Forecast
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Days Until Stockout
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Total Cost
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {ingredients.map((ingredient) => (
              <tr key={ingredient.ingredient_id} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="font-medium text-gray-900">{ingredient.ingredient_name}</div>
                  <div className="text-sm text-gray-500">{ingredient.base_unit}</div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-900">
                  {Number(ingredient.total_consumed).toFixed(2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-900">
                  {Number(ingredient.current_stock).toFixed(2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-900">
                  {Number(ingredient.stock_after_forecast).toFixed(2)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                  {ingredient.days_until_stockout !== null ? (
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        ingredient.days_until_stockout <= 7
                          ? 'bg-red-100 text-red-800'
                          : ingredient.days_until_stockout <= 14
                          ? 'bg-yellow-100 text-yellow-800'
                          : 'bg-green-100 text-green-800'
                      }`}
                    >
                      {ingredient.days_until_stockout} days
                    </span>
                  ) : (
                    <span className="text-gray-400">N/A</span>
                  )}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm text-gray-900">
                  Rp {ingredient.total_cost.toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function RevenueSimulatorTab({ startDate, endDate }: { startDate: string; endDate: string }) {
  const [targetRevenue, setTargetRevenue] = useState('10000000');
  const [data, setData] = useState<RevenueSimulatorResponse | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSimulate = async () => {
    try {
      setLoading(true);
      const result = await analyticsService.simulateRevenueTarget(targetRevenue, startDate, endDate);
      setData(result);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="rounded-lg border bg-white p-6 shadow-sm">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Revenue Target Simulator</h3>
        <div className="flex items-end gap-4">
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">Target Revenue (Rp)</label>
            <input
              type="number"
              value={targetRevenue}
              onChange={(e) => setTargetRevenue(e.target.value)}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
              placeholder="e.g., 10000000"
            />
          </div>
          <button
            onClick={handleSimulate}
            disabled={loading}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? 'Simulating...' : 'Simulate'}
          </button>
        </div>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
            <StatCard title="Daily Target" value={`Rp ${data.target.daily_target.toLocaleString()}`} />
            <StatCard title="Weekly Target" value={`Rp ${data.target.weekly_target.toLocaleString()}`} />
            <StatCard
              title="Projected Gross Profit"
              value={`Rp ${data.projected_gross_profit.toLocaleString()}`}
              highlight
            />
            <StatCard
              title="Feasibility Score"
              value={`${Number(data.feasibility_score).toFixed(0)}%`}
              highlight={Number(data.feasibility_score) >= 70}
            />
          </div>

          <div className="rounded-lg border bg-white p-6 shadow-sm">
            <h4 className="font-medium text-gray-900 mb-3">Recommended Product Mix</h4>
            <div className="space-y-3">
              {(data.recommended_mix || []).map((item) => (
                <div
                  key={item.product_id}
                  className="flex items-center justify-between rounded-lg bg-gray-50 p-3"
                >
                  <div>
                    <div className="font-medium text-gray-900">{item.product_name}</div>
                    <div className="text-sm text-gray-500">
                      {item.recommended_quantity} units • {item.score_reason}
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="font-medium text-gray-900">
                      Rp {item.estimated_revenue.toLocaleString()}
                    </div>
                    <div className="text-sm text-green-600">
                        {Number(item.gross_margin_pct).toFixed(1)}% margin
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {data.feasibility_notes.length > 0 && (
            <div className="rounded-lg border bg-yellow-50 p-4">
              <h4 className="font-medium text-yellow-800 mb-2">Feasibility Notes</h4>
              <ul className="list-disc list-inside space-y-1 text-sm text-yellow-700">
                {data.feasibility_notes.map((note, i) => (
                  <li key={i}>{note}</li>
                ))}
              </ul>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function BudgetSimulatorTab({ startDate, endDate }: { startDate: string; endDate: string }) {
  const [budgetAmount, setBudgetAmount] = useState('5000000');
  const [data, setData] = useState<BudgetSimulatorResponse | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSimulate = async () => {
    try {
      setLoading(true);
      const result = await analyticsService.simulateBudget(budgetAmount, startDate, endDate);
      setData(result);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="rounded-lg border bg-white p-6 shadow-sm">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Budget-Based Simulator</h3>
        <div className="flex items-end gap-4">
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">Budget Amount (Rp)</label>
            <input
              type="number"
              value={budgetAmount}
              onChange={(e) => setBudgetAmount(e.target.value)}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
              placeholder="e.g., 5000000"
            />
          </div>
          <button
            onClick={handleSimulate}
            disabled={loading}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? 'Simulating...' : 'Simulate'}
          </button>
        </div>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
            <StatCard title="Budget" value={`Rp ${data.budget.toLocaleString()}`} />
            <StatCard
              title="Projected Revenue"
              value={`Rp ${data.projected_revenue.toLocaleString()}`}
            />
            <StatCard
              title="Projected Gross Profit"
              value={`Rp ${data.projected_gross_profit.toLocaleString()}`}
              highlight
            />
            <StatCard
              title="Budget Utilization"
              value={`${Number(data.budget_utilization).toFixed(1)}%`}
            />
          </div>

          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <div className="rounded-lg border bg-white p-6 shadow-sm">
              <h4 className="font-medium text-gray-900 mb-3">Ingredient Budget Allocation</h4>
              <div className="space-y-3">
                {(data.ingredient_breakdown || []).map((item) => (
                  <div
                    key={item.ingredient_id}
                    className="flex items-center justify-between rounded-lg bg-gray-50 p-3"
                  >
                    <div>
                      <div className="font-medium text-gray-900">{item.ingredient_name}</div>
                      <div className="text-sm text-gray-500">
                        {Number(item.quantity_to_buy).toFixed(2)} {item.unit}
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="font-medium text-gray-900">
                        Rp {item.allocated_budget.toLocaleString()}
                      </div>
                      <div className="text-sm text-gray-500">
                        @ Rp {item.cost_per_unit.toLocaleString()}/{item.unit}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-lg border bg-white p-6 shadow-sm">
              <h4 className="font-medium text-gray-900 mb-3">Recommended Product Mix</h4>
              <div className="space-y-3">
                {(data.recommended_mix || []).map((item) => (
                  <div
                    key={item.product_id}
                    className="flex items-center justify-between rounded-lg bg-gray-50 p-3"
                  >
                    <div>
                      <div className="font-medium text-gray-900">{item.product_name}</div>
                      <div className="text-sm text-gray-500">
                        {item.recommended_quantity} units
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="font-medium text-gray-900">
                        Rp {item.estimated_revenue.toLocaleString()}
                      </div>
                      <div className="text-sm text-green-600">
                      {Number(item.gross_margin_pct).toFixed(1)}% margin
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

function RecommendationsTab() {
  const [data, setData] = useState<ProductRecommendationResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const result = await analyticsService.getProductRecommendations();
      setData(result);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <LoadingSpinner />;
  if (!data) return <ErrorMessage message="Failed to load recommendations" />;

  return (
    <div className="space-y-6">
      <div className="rounded-lg border bg-white p-6 shadow-sm">
        <h3 className="text-lg font-medium text-gray-900 mb-2">Product Recommendations</h3>
        <p className="text-sm text-gray-500 mb-4">
          Scored by margin ({(Number(data.scoring_weights.margin) * 100).toFixed(0)}%), velocity ({(Number(data.scoring_weights.velocity) * 100).toFixed(0)}%), stock readiness ({(Number(data.scoring_weights.stock_readiness) * 100).toFixed(0)}%), and trend ({(Number(data.scoring_weights.trend) * 100).toFixed(0)}%)
        </p>

        <div className="space-y-3">
          {(data.recommendations || []).map((rec, index) => (
            <div
              key={rec.product_id}
              className="flex items-center justify-between rounded-lg border p-4 hover:bg-gray-50"
            >
              <div className="flex items-center gap-4">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-100 text-blue-600 font-bold">
                  {index + 1}
                </div>
                <div>
                  <div className="font-medium text-gray-900">{rec.product_name}</div>
                  <div className="text-sm text-gray-500">{rec.recommendation_note}</div>
                </div>
              </div>
              <div className="flex items-center gap-6">
                <div className="text-right">
                  <div className="text-sm text-gray-500">Score</div>
                  <div className="font-bold text-gray-900">{Number(rec.total_score).toFixed(2)}</div>
                </div>
                <div className="text-right">
                  <div className="text-sm text-gray-500">Margin</div>
                  <div className="text-gray-900">{(Number(rec.margin_score) * 100).toFixed(0)}%</div>
                </div>
                <div className="text-right">
                  <div className="text-sm text-gray-500">Velocity</div>
                  <div className="text-gray-900">{(Number(rec.velocity_score) * 100).toFixed(0)}%</div>
                </div>
                {rec.bundle_opportunity && (
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
                    Bundle Opportunity
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function StatCard({
  title,
  value,
  highlight,
}: {
  title: string;
  value: string;
  highlight?: boolean;
}) {
  return (
    <div className={`rounded-lg border p-4 ${highlight ? 'border-green-200 bg-green-50' : 'bg-white'}`}>
      <div className="text-sm font-medium text-gray-500">{title}</div>
      <div className={`mt-1 text-xl font-bold ${highlight ? 'text-green-700' : 'text-gray-900'}`}>
        {value}
      </div>
    </div>
  );
}

function LoadingSpinner() {
  return (
    <div className="flex items-center justify-center h-64">
      <div className="text-gray-500">Loading...</div>
    </div>
  );
}

function ErrorMessage({ message }: { message: string }) {
  return (
    <div className="rounded-lg bg-red-50 p-4 text-red-700">
      {message}
    </div>
  );
}
