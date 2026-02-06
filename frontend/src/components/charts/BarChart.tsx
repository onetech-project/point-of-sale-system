'use client';

import React from 'react';
import { BarChart as RechartsBarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import { TimeSeriesDataPoint } from '@/types/analytics';

export interface BarChartProps {
  data: TimeSeriesDataPoint[];
  dataKey: string;
  xAxisKey?: string;
  yAxisLabel?: string;
  color?: string;
  height?: number;
  formatter?: (value: number, compact?: boolean, decimals?: number) => string;
  loading?: boolean;
}

export const BarChart: React.FC<BarChartProps> = ({
  data,
  dataKey,
  xAxisKey = 'label',
  yAxisLabel,
  color = '#3b82f6',
  height = 300,
  formatter,
  loading = false,
}) => {
  if (loading) {
    return (
      <div style={{ height }} className="flex items-center justify-center bg-gray-50 rounded-lg animate-pulse">
        <div className="text-gray-400">Loading chart...</div>
      </div>
    );
  }

  if (!data || data.length === 0) {
    return (
      <div style={{ height }} className="flex items-center justify-center bg-gray-50 rounded-lg">
        <div className="text-center">
          <svg
            className="mx-auto w-12 h-12 text-gray-400 mb-2"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
            />
          </svg>
          <p className="text-sm text-gray-500">No data available for this period</p>
        </div>
      </div>
    );
  }

  const tooltipFormatter = (value: number | undefined) => {
    if (value === undefined) {
      return '';
    }
    if (formatter) {
      return formatter(value, false, 0);
    }
    return value.toLocaleString();
  };

  const tickFormatter = (value: number) => {
    if (value === undefined) {
      return '';
    }
    if (formatter) {
      return formatter(value, true, 0);
    }
    return value.toLocaleString();
  }

  return (
    <ResponsiveContainer width="100%" height={height}>
      <RechartsBarChart data={data} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
        <XAxis
          dataKey={xAxisKey}
          stroke="#6b7280"
          style={{ fontSize: '12px' }}
        />
        <YAxis
          stroke="#6b7280"
          style={{ fontSize: '12px' }}
          label={yAxisLabel ? { value: yAxisLabel, angle: -90, position: 'insideLeft' } : undefined}
          tickFormatter={tickFormatter}
        />
        <Tooltip
          formatter={tooltipFormatter}
          contentStyle={{
            backgroundColor: 'white',
            border: '1px solid #e5e7eb',
            borderRadius: '8px',
            padding: '8px 12px',
          }}
        />
        <Bar dataKey={dataKey} fill={color} radius={[4, 4, 0, 0]} />
      </RechartsBarChart>
    </ResponsiveContainer>
  );
};
