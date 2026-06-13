import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import InventoryDashboard from './InventoryDashboard';
import { product } from '@/services/product';

jest.mock('@/i18n/provider', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

jest.mock('@/services/product', () => ({
  product: {
    getInventorySummary: jest.fn(),
  },
}));

const mockProduct = product as jest.Mocked<typeof product>;

it('uses responsive and wrapping classes for product KPI values', async () => {
  mockProduct.getInventorySummary.mockResolvedValue({
    total_products: 1200,
    total_value: 1234567890,
    low_stock_count: 8,
    out_of_stock_count: 2,
    categories_count: 15,
  });

  render(<InventoryDashboard />);

  await waitFor(() => {
    expect(screen.getByTestId('inventory-kpi-value-1')).toBeInTheDocument();
  });

  const value = screen.getByTestId('inventory-kpi-value-1');
  expect(value).toHaveClass('text-xl');
  expect(value).toHaveClass('lg:text-2xl');
  expect(value).toHaveClass('xl:text-3xl');
  expect(value).toHaveClass('leading-tight');
  expect(value).toHaveClass('break-words');
  expect(value).toHaveClass('[overflow-wrap:anywhere]');
  expect(value.querySelector('.xl\\:hidden')).toBeInTheDocument();
});
