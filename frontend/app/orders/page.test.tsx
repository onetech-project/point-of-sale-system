import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import OrdersPage from './page';

const mockSearchParams = jest.fn();
const mockPush = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
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
  OrderManagement: ({
    initialOrderId,
    initialOrderType,
  }: {
    initialOrderId?: string;
    initialOrderType?: string;
  }) => (
    <div data-testid="order-management">
      {initialOrderId || 'no-initial-order'}:{initialOrderType}
    </div>
  ),
}));

jest.mock('@/components/orders/OfflineOrderForm', () => ({
  OfflineOrderForm: () => <div data-testid="offline-order-form">offline form</div>,
}));

beforeEach(() => {
  jest.clearAllMocks();
});

it('passes online order query state into OrderManagement', async () => {
  mockSearchParams.mockReturnValue(new URLSearchParams('order_id=order-123&order_type=online'));

  render(<OrdersPage />);

  expect(await screen.findByTestId('order-management')).toHaveTextContent('order-123:online');
});

it('passes offline order query state into OrderManagement', async () => {
  mockSearchParams.mockReturnValue(new URLSearchParams('order_id=order-456&order_type=offline'));

  render(<OrdersPage />);

  expect(await screen.findByTestId('order-management')).toHaveTextContent('order-456:offline');
});

it('renders OrderManagement without an initial order when order_id is absent', async () => {
  mockSearchParams.mockReturnValue(new URLSearchParams());

  render(<OrdersPage />);

  expect(await screen.findByTestId('order-management')).toHaveTextContent('no-initial-order:online');
});

it('renders the offline order form for new-offline mode', async () => {
  mockSearchParams.mockReturnValue(new URLSearchParams('mode=new-offline'));

  render(<OrdersPage />);

  expect(await screen.findByTestId('offline-order-form')).toBeInTheDocument();
});
