import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { OrderManagement } from './OrderManagement';
import { order } from '../../services/order';

jest.mock('next/navigation', () => ({
  useRouter: () => ({
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

const mockOrderService = order as jest.Mocked<typeof order>;

const selectedOrder = {
  order: {
    id: 'order-123',
    tenant_id: 'tenant-1',
    order_reference: 'GO-0001',
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
});

it('opens the detail modal for an initial order id fetched directly', async () => {
  render(<OrderManagement initialOrderId="order-123" />);

  await waitFor(() => {
    expect(mockOrderService.getOrderById).toHaveBeenCalledWith('order-123');
  });

  expect(await screen.findByText('Order GO-0001')).toBeInTheDocument();
  expect(screen.getByText('Product 1')).toBeInTheDocument();
});
