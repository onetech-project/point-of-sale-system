'use client';

import React from 'react';
import { LineChart as RechartsLineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import { TimeSeriesDataPoint } from '@/types/analytics';

export interface LineChartProps {
  data: TimeSeriesDataPoint[];
  dataKey: string;
  xAxisKey?: string;
  yAxisLabel?: string;
  color?: string;
  height?: number;
  formatter?: (value: number, compact?: boolean, decimals?: number) => string;
  loading?: boolean;
}

export const LineChart: React.FC<LineChartProps> = ({
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
              d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z"
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
      <RechartsLineChart data={data} margin={{ top: 5, right: 20, left: 10, bottom: 5 }}>
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
        <Line
          type="monotone"
          dataKey={dataKey}
          stroke={color}
          strokeWidth={2}
          dot={{ fill: color, r: 4 }}
          activeDot={{ r: 6 }}
        />
      </RechartsLineChart>
    </ResponsiveContainer>
  );
};
