'use client';

import React from 'react';

export type Granularity = 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'yearly';

export interface TimeSeriesFilterProps {
  granularity: Granularity;
  startDate: string; // YYYY-MM-DD format
  endDate: string;   // YYYY-MM-DD format
  onGranularityChange: (granularity: Granularity) => void;
  onDateRangeChange: (startDate: string, endDate: string) => void;
  loading?: boolean;
}

export const TimeSeriesFilter: React.FC<TimeSeriesFilterProps> = ({
  granularity,
  startDate,
  endDate,
  onGranularityChange,
  onDateRangeChange,
  loading = false,
}) => {
  const granularityOptions: { value: Granularity; label: string }[] = [
    { value: 'daily', label: 'Daily' },
    { value: 'weekly', label: 'Weekly' },
    { value: 'monthly', label: 'Monthly' },
    { value: 'quarterly', label: 'Quarterly' },
    { value: 'yearly', label: 'Yearly' },
  ];

  const handleStartDateChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onDateRangeChange(e.target.value, endDate);
  };

  const handleEndDateChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onDateRangeChange(startDate, e.target.value);
  };

  return (
    <div className="bg-white rounded-lg shadow p-4">
      <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center">
        {/* Granularity Selector */}
        <div className="flex-shrink-0">
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Time Period
          </label>
          <select
            value={granularity}
            onChange={(e) => onGranularityChange(e.target.value as Granularity)}
            disabled={loading}
            className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {granularityOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>

        {/* Date Range Pickers */}
        <div className="flex-1 flex flex-col sm:flex-row gap-4">
          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Start Date
            </label>
            <input
              type="date"
              value={startDate}
              onChange={handleStartDateChange}
              disabled={loading}
              max={endDate}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 disabled:opacity-50 disabled:cursor-not-allowed"
            />
          </div>

          <div className="flex-1">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              End Date
            </label>
            <input
              type="date"
              value={endDate}
              onChange={handleEndDateChange}
              disabled={loading}
              min={startDate}
              max={new Date().toISOString().split('T')[0]}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 disabled:opacity-50 disabled:cursor-not-allowed"
            />
          </div>
        </div>

        {/* Quick Presets */}
        <div className="flex-shrink-0">
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Quick Select
          </label>
          <div className="flex gap-2">
            <button
              onClick={() => {
                const end = new Date();
                const start = new Date();
                start.setDate(start.getDate() - 30);
                onDateRangeChange(
                  start.toISOString().split('T')[0],
                  end.toISOString().split('T')[0]
                );
              }}
              disabled={loading}
              className="px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Last 30 Days
            </button>
            <button
              onClick={() => {
                const end = new Date();
                const start = new Date();
                start.setDate(start.getDate() - 90);
                onDateRangeChange(
                  start.toISOString().split('T')[0],
                  end.toISOString().split('T')[0]
                );
              }}
              disabled={loading}
              className="px-3 py-2 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Last 90 Days
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
