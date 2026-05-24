import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import OrdersPage from './page';

const mockSearchParams = jest.fn();

jest.mock('next/navigation', () => ({
  useSearchParams: () => mockSearchParams(),
}));

jest.mock('@/components/auth/ProtectedRoute', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/components/layout/DashboardLayout', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/components/admin/OrderManagement', () => ({
  OrderManagement: ({ initialOrderId }: { initialOrderId?: string }) => (
    <div data-testid="order-management">{initialOrderId || 'no-initial-order'}</div>
  ),
}));

beforeEach(() => {
  jest.clearAllMocks();
});

it('passes order_id from the URL into OrderManagement', async () => {
  mockSearchParams.mockReturnValue(new URLSearchParams('order_id=order-123'));

  render(<OrdersPage />);

  expect(await screen.findByTestId('order-management')).toHaveTextContent('order-123');
});

it('renders OrderManagement without an initial order when order_id is absent', async () => {
  mockSearchParams.mockReturnValue(new URLSearchParams());

  render(<OrdersPage />);

  expect(await screen.findByTestId('order-management')).toHaveTextContent('no-initial-order');
});
