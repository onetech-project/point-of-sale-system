import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import AnalyticsDashboardPage from './page';
import analytics from '@/services/analytics';
import { useAuth } from '@/store/auth';

const mockPush = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
}));

jest.mock('@/store/auth', () => ({
  useAuth: jest.fn(),
}));

jest.mock('@/components/auth/ProtectedRoute', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('../../src/components/layout/DashboardLayout', () => ({
  __esModule: true,
  default: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/components/dashboard/DashboardLayout', () => ({
  DashboardLayout: ({
    title,
    actions,
    children,
  }: {
    title: string;
    actions?: React.ReactNode;
    children: React.ReactNode;
  }) => (
    <section>
      <h1>{title}</h1>
      {actions}
      {children}
    </section>
  ),
}));

jest.mock('@/components/dashboard/MetricCard', () => ({
  MetricCard: ({ title }: { title: string }) => <div>{title}</div>,
}));

jest.mock('@/components/dashboard/TaskAlerts', () => ({
  TaskAlerts: ({
    delayedOrders,
    restockAlerts,
    onNavigateToOrder,
  }: {
    delayedOrders: any[];
    restockAlerts: any[];
    onNavigateToOrder?: (order: any) => void;
  }) => (
    <div>
      <span>Delayed: {delayedOrders.length}</span>
      <span>Restock: {restockAlerts.length}</span>
      {delayedOrders[0] && (
        <button type="button" onClick={() => onNavigateToOrder?.(delayedOrders[0])}>
          Open delayed order
        </button>
      )}
    </div>
  ),
}));

jest.mock('@/components/dashboard/ProductRankingTable', () => ({
  ProductRankingTable: () => <div>Product Ranking</div>,
}));

jest.mock('@/components/dashboard/CustomerRankingTable', () => ({
  CustomerRankingTable: () => <div>Customer Ranking</div>,
}));

jest.mock('@/components/dashboard/TimeSeriesFilter', () => ({
  TimeSeriesFilter: () => <div>Time Series Filter</div>,
}));

jest.mock('@/components/dashboard/SalesChart', () => ({
  SalesChart: () => <div>Sales Chart</div>,
}));

jest.mock('@/components/dashboard/QuickActions', () => ({
  QuickActions: () => <div>Quick Actions</div>,
}));

jest.mock('@/components/dashboard/OfflineOrderMetrics', () => ({
  OfflineOrderMetrics: () => <div>Offline Order Metrics</div>,
}));

jest.mock('@/components/common/ErrorBoundary', () => ({
  DashboardErrorBoundary: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/services/analytics', () => ({
  __esModule: true,
  default: {
    getSalesOverview: jest.fn(),
    getTopProducts: jest.fn(),
    getTopCustomers: jest.fn(),
    getOperationalTasks: jest.fn(),
    getSalesTrend: jest.fn(),
  },
}));

const mockAnalytics = analytics as jest.Mocked<typeof analytics>;
const mockUseAuth = useAuth as jest.MockedFunction<typeof useAuth>;

const operationalTasks = {
  delayed_orders: {
    count: 1,
    urgent_count: 0,
    warning_count: 1,
    delayed_orders: [
      {
        order_id: 'f423dcbf-1256-4fa8-a543-2d2c0ecb2bbe',
        order_number: 'GO-984892',
        order_type: 'offline',
      },
    ],
  },
  restock_alerts: {
    count: 1,
    critical_count: 0,
    low_stock_count: 1,
    restock_alerts: [{ product_id: 1 }],
  },
};

const delayedTasksWithOrder = (order: Record<string, unknown>) => ({
  ...operationalTasks,
  delayed_orders: {
    ...operationalTasks.delayed_orders,
    delayed_orders: [order],
  },
});

beforeEach(() => {
  jest.clearAllMocks();
  mockAnalytics.getOperationalTasks.mockResolvedValue(operationalTasks as any);
  mockAnalytics.getSalesOverview.mockResolvedValue({
    metrics: {
      total_revenue: 100000,
      total_orders: 3,
      average_order_value: 33333,
      inventory_value: 500000,
      revenue_change: 0,
      orders_change: 0,
      aov_change: 0,
      offline_order_count: 1,
      offline_revenue: 100000,
      offline_percentage: 33,
      online_order_count: 2,
      online_revenue: 200000,
      installment_count: 0,
      installment_revenue: 0,
      pending_installments: 0,
    },
  } as any);
  mockAnalytics.getTopProducts.mockResolvedValue({
    top_by_revenue: [],
    top_by_quantity: [],
    bottom_by_revenue: [],
    bottom_by_quantity: [],
  });
  mockAnalytics.getTopCustomers.mockResolvedValue({
    top_by_spending: [],
    top_by_orders: [],
  });
  mockAnalytics.getSalesTrend.mockResolvedValue({
    period: 'daily',
    start_date: '2026-05-01',
    end_date: '2026-05-21',
    revenue_data: [],
    orders_data: [],
  });
});

it('routes offline delayed orders to the unified order detail query', async () => {
  mockUseAuth.mockReturnValue({
    user: { role: 'cashier' },
    isLoading: false,
  } as any);

  render(<AnalyticsDashboardPage />);

  fireEvent.click(await screen.findByRole('button', { name: 'Open delayed order' }));

  expect(mockPush).toHaveBeenCalledWith(
    '/orders?order_id=f423dcbf-1256-4fa8-a543-2d2c0ecb2bbe&order_type=offline'
  );
});

it('routes online delayed orders to the order management detail query', async () => {
  mockAnalytics.getOperationalTasks.mockResolvedValue(
    delayedTasksWithOrder({
      order_id: 'online-order-123',
      order_number: 'GO-984893',
      order_type: 'online',
    }) as any
  );
  mockUseAuth.mockReturnValue({
    user: { role: 'cashier' },
    isLoading: false,
  } as any);

  render(<AnalyticsDashboardPage />);

  fireEvent.click(await screen.findByRole('button', { name: 'Open delayed order' }));

  expect(mockPush).toHaveBeenCalledWith('/orders?order_id=online-order-123&order_type=online');
});

it('routes delayed orders without an order type to the order management detail query', async () => {
  mockAnalytics.getOperationalTasks.mockResolvedValue(
    delayedTasksWithOrder({
      order_id: 'default-order-123',
      order_number: 'GO-984894',
    }) as any
  );
  mockUseAuth.mockReturnValue({
    user: { role: 'cashier' },
    isLoading: false,
  } as any);

  render(<AnalyticsDashboardPage />);

  fireEvent.click(await screen.findByRole('button', { name: 'Open delayed order' }));

  expect(mockPush).toHaveBeenCalledWith(
    '/orders?order_id=default-order-123&order_type=online'
  );
});

it('renders only operational alerts for cashiers', async () => {
  mockUseAuth.mockReturnValue({
    user: { role: 'cashier' },
    isLoading: false,
  } as any);

  render(<AnalyticsDashboardPage />);

  expect(await screen.findByText('Operational Dashboard')).toBeInTheDocument();
  expect(await screen.findByText('Delayed: 1')).toBeInTheDocument();
  expect(screen.getByText('Restock: 1')).toBeInTheDocument();
  expect(screen.queryByText('Total Revenue')).not.toBeInTheDocument();

  await waitFor(() => {
    expect(mockAnalytics.getOperationalTasks).toHaveBeenCalledTimes(1);
  });
  expect(mockAnalytics.getSalesOverview).not.toHaveBeenCalled();
  expect(mockAnalytics.getTopProducts).not.toHaveBeenCalled();
  expect(mockAnalytics.getTopCustomers).not.toHaveBeenCalled();
  expect(mockAnalytics.getSalesTrend).not.toHaveBeenCalled();
});

it('keeps full business insights for managers', async () => {
  mockUseAuth.mockReturnValue({
    user: { role: 'manager' },
    isLoading: false,
  } as any);

  render(<AnalyticsDashboardPage />);

  expect(await screen.findByText('Business Insights')).toBeInTheDocument();
  expect(await screen.findByText('Total Revenue')).toBeInTheDocument();

  await waitFor(() => {
    expect(mockAnalytics.getSalesOverview).toHaveBeenCalled();
    expect(mockAnalytics.getTopProducts).toHaveBeenCalled();
    expect(mockAnalytics.getTopCustomers).toHaveBeenCalled();
    expect(mockAnalytics.getOperationalTasks).toHaveBeenCalled();
    expect(mockAnalytics.getSalesTrend).toHaveBeenCalled();
  });
});
