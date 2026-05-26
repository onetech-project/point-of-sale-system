import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { OrderManagement } from './OrderManagement';
import { order } from '../../services/order';
import offlineOrderService from '../../services/offlineOrders';

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
  }),
}));

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

jest.mock('../../services/order', () => ({
  order: {
    listOrders: jest.fn(),
    getOrderById: jest.fn(),
    updateOrderStatus: jest.fn(),
    addOrderNote: jest.fn(),
    downloadOrderDocument: jest.fn(),
    resendOrderDocument: jest.fn(),
    batchDownloadOrderDocuments: jest.fn(),
  },
}));

jest.mock('../../services/offlineOrders', () => ({
  __esModule: true,
  default: {
    getOfflineOrderWithDetails: jest.fn(),
    downloadDocument: jest.fn(),
    resendDocument: jest.fn(),
  },
}));

jest.mock('../orders/OfflineOrderDetail', () => ({
  OfflineOrderDetail: ({ order }: { order: { order_reference: string } }) => (
    <div data-testid="offline-order-detail">Offline {order.order_reference}</div>
  ),
}));

const mockOrderService = order as jest.Mocked<typeof order>;
const mockOfflineOrderService = offlineOrderService as jest.Mocked<typeof offlineOrderService>;

const selectedOrder = {
  order: {
    id: 'order-123',
    tenant_id: 'tenant-1',
    order_reference: 'GO-0001',
    order_type: 'online',
    customer_name: 'Customer',
    customer_phone: '+628123456789',
    customer_email: 'customer@example.com',
    delivery_type: 'TAKEAWAY',
    subtotal_amount: 30000,
    delivery_fee: 0,
    total_amount: 30000,
    status: 'PENDING',
    items: [],
    created_at: '2026-05-21T01:00:00Z',
    updated_at: '2026-05-21T01:00:00Z',
    tenant_slug: 'tenant-one',
  },
  items: [
    {
      id: 'item-1',
      product_id: 'product-1',
      product_name: 'Product 1',
      quantity: 2,
      unit_price: 15000,
      total_price: 30000,
    },
  ],
};

beforeEach(() => {
  jest.clearAllMocks();
  mockOrderService.listOrders.mockResolvedValue({
    orders: [],
    pagination: { limit: 20, offset: 0, count: 0 },
  });
  mockOrderService.getOrderById.mockResolvedValue(selectedOrder as any);
  mockOfflineOrderService.getOfflineOrderWithDetails.mockResolvedValue({
    order: {
      ...selectedOrder.order,
      id: 'offline-123',
      order_reference: 'OFF-0001',
      order_type: 'offline',
      data_consent_given: true,
    },
    items: selectedOrder.items,
  } as any);
});

it('requests both online and offline orders for the unified list', async () => {
  render(<OrderManagement />);

  await waitFor(() => {
    expect(mockOrderService.listOrders).toHaveBeenCalledWith({
      page: 1,
      limit: 20,
      order_type: 'all',
    });
  });
});

it('opens the detail modal for an initial order id fetched directly', async () => {
  render(<OrderManagement initialOrderId="order-123" />);

  await waitFor(() => {
    expect(mockOrderService.getOrderById).toHaveBeenCalledWith('order-123');
  });

  expect(await screen.findByText('Order GO-0001')).toBeInTheDocument();
  expect(screen.getByText('Product 1')).toBeInTheDocument();
});

it('opens the offline detail view for an initial offline order id', async () => {
  render(<OrderManagement initialOrderId="offline-123" initialOrderType="offline" />);

  await waitFor(() => {
    expect(mockOfflineOrderService.getOfflineOrderWithDetails).toHaveBeenCalledWith('offline-123');
  });

  expect(await screen.findByTestId('offline-order-detail')).toHaveTextContent('Offline OFF-0001');
  expect(mockOrderService.getOrderById).not.toHaveBeenCalled();
});
