'use client';

import React, { useState } from 'react';
import { LineChart } from '@/components/charts/LineChart';
import { BarChart } from '@/components/charts/BarChart';
import { TimeSeriesDataPoint } from '@/types/analytics';
import { formatCurrency } from '@/utils/format';

export type ChartType = 'line' | 'bar';
type Metric = 'revenue' | 'profit' | 'orders';

const METRIC_OPTIONS: { value: Metric; label: string; axisLabel: string; color: string }[] = [
  { value: 'revenue', label: 'Revenue', axisLabel: 'Revenue (IDR)', color: '#3b82f6' },
  { value: 'profit', label: 'Profit', axisLabel: 'Gross Profit (IDR)', color: '#8b5cf6' },
  { value: 'orders', label: 'Orders', axisLabel: 'Number of Orders', color: '#10b981' },
];

export interface SalesChartProps {
  revenueData: TimeSeriesDataPoint[];
  ordersData: TimeSeriesDataPoint[];
  profitData?: TimeSeriesDataPoint[];
  loading?: boolean;
  height?: number;
}

export const SalesChart: React.FC<SalesChartProps> = ({
  revenueData,
  ordersData,
  profitData = [],
  loading = false,
  height = 400,
}) => {
  const [chartType, setChartType] = useState<ChartType>('line');
  const [metric, setMetric] = useState<Metric>('revenue');

  const currentData = { revenue: revenueData, profit: profitData, orders: ordersData }[metric];
  const isCurrency = metric !== 'orders';
  const selectedOption = METRIC_OPTIONS.find(option => option.value === metric)!;
  const ChartComponent = chartType === 'line' ? LineChart : BarChart;

  return (
    <div className="bg-white rounded-lg shadow p-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <h3 className="text-lg font-semibold text-gray-900">Sales Trend</h3>

        <div className="flex items-center gap-3">
          {/* Metric Selector */}
          <div className="flex bg-gray-100 rounded-lg p-1">
            {METRIC_OPTIONS.map(option => (
              <button
                key={option.value}
                onClick={() => setMetric(option.value)}
                className={`px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
                  metric === option.value
                    ? 'bg-white text-gray-900 shadow-sm'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                {option.label}
              </button>
            ))}
          </div>

          {/* Chart Type Selector */}
          <div className="flex bg-gray-100 rounded-lg p-1">
            <button
              onClick={() => setChartType('line')}
              className={`p-1.5 rounded-md transition-colors ${
                chartType === 'line'
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
              title="Line Chart"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z"
                />
              </svg>
            </button>
            <button
              onClick={() => setChartType('bar')}
              className={`p-1.5 rounded-md transition-colors ${
                chartType === 'bar'
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
              title="Bar Chart"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                />
              </svg>
            </button>
          </div>
        </div>
      </div>

      {/* Chart */}
      <ChartComponent
        data={currentData}
        dataKey="value"
        xAxisKey="label"
        yAxisLabel={selectedOption.axisLabel}
        color={selectedOption.color}
        height={height}
        formatter={isCurrency ? formatCurrency : undefined}
        loading={loading}
      />

      {/* Summary Stats */}
      {!loading && currentData.length > 0 && (
        <div className="mt-6 grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="bg-gray-50 rounded-lg p-4">
            <p className="text-xs text-gray-500 mb-1">Total</p>
            <p className="text-lg font-semibold text-gray-900">
              {isCurrency
                ? formatCurrency(currentData.reduce((sum, d) => sum + d.value, 0))
                : currentData.reduce((sum, d) => sum + d.value, 0).toLocaleString()}
            </p>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <p className="text-xs text-gray-500 mb-1">Average</p>
            <p className="text-lg font-semibold text-gray-900">
              {isCurrency
                ? formatCurrency(currentData.reduce((sum, d) => sum + d.value, 0) / currentData.length)
                : (currentData.reduce((sum, d) => sum + d.value, 0) / currentData.length).toFixed(1)}
            </p>
          </div>
          <div className="bg-gray-50 rounded-lg p-4">
            <p className="text-xs text-gray-500 mb-1">Peak</p>
            <p className="text-lg font-semibold text-gray-900">
              {isCurrency
                ? formatCurrency(Math.max(...currentData.map((d) => d.value)))
                : Math.max(...currentData.map((d) => d.value)).toLocaleString()}
            </p>
          </div>
        </div>
      )}
    </div>
  );
};
