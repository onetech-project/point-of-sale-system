import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { OfflineOrderDetail } from './OfflineOrderDetail';
import { OfflineOrder } from '../../types/offlineOrder';
import offlineOrderService from '../../services/offlineOrders';

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
  }),
}));

jest.mock('../../services/offlineOrders', () => ({
  __esModule: true,
  default: {
    batchDownloadDocuments: jest.fn(),
    completeOfflineOrder: jest.fn(),
    deleteOfflineOrder: jest.fn(),
    downloadDocument: jest.fn(),
    resendDocument: jest.fn(),
  },
}));

const mockOfflineOrderService = offlineOrderService as jest.Mocked<typeof offlineOrderService>;

const pendingOrder: OfflineOrder = {
  id: 'order-1',
  tenant_id: 'tenant-1',
  order_reference: 'GO-0001',
  status: 'PENDING',
  order_type: 'offline',
  customer_name: 'Customer',
  customer_phone: '+628123456789',
  customer_email: 'customer@example.com',
  delivery_type: 'pickup',
  subtotal_amount: 30000,
  delivery_fee: 0,
  total_amount: 30000,
  data_consent_given: true,
  consent_method: 'verbal',
  created_at: '2026-05-20T02:00:00Z',
};

const paidOrder: OfflineOrder = {
  ...pendingOrder,
  status: 'PAID',
  paid_at: '2026-05-20T02:15:00Z',
};

beforeEach(() => {
  jest.clearAllMocks();
});

describe('OfflineOrderDetail document menu', () => {
  it('moves document actions into the three-dot menu and keeps workflow buttons visible', () => {
    render(<OfflineOrderDetail order={pendingOrder} />);

    expect(screen.getByRole('button', { name: 'Edit Order' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Record Payment' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Invoice' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Receipt' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByLabelText('Open document actions for order GO-0001'));

    expect(screen.getByRole('menuitem', { name: 'Download Invoice' })).toBeEnabled();
    expect(screen.getByRole('menuitem', { name: 'Download Receipt' })).toBeDisabled();
    expect(screen.getByRole('menuitem', { name: 'Resend Invoice' })).toBeEnabled();
    expect(screen.getByRole('menuitem', { name: 'Resend Receipt' })).toBeDisabled();
  });
});

describe('OfflineOrderDetail completion action', () => {
  it('shows the completion button only for paid orders', () => {
    const { rerender } = render(<OfflineOrderDetail order={pendingOrder} />);

    expect(screen.queryByRole('button', { name: 'Mark as Complete' })).not.toBeInTheDocument();

    rerender(<OfflineOrderDetail order={paidOrder} />);

    expect(screen.getByRole('button', { name: 'Mark as Complete' })).toBeInTheDocument();
  });

  it('completes a paid order and refreshes the detail page', async () => {
    mockOfflineOrderService.completeOfflineOrder.mockResolvedValue({
      message: 'Order status updated successfully',
      status: 'COMPLETE',
    });
    const onRefresh = jest.fn().mockResolvedValue(undefined);

    render(<OfflineOrderDetail order={paidOrder} onRefresh={onRefresh} />);

    fireEvent.click(screen.getByRole('button', { name: 'Mark as Complete' }));

    await waitFor(() => {
      expect(mockOfflineOrderService.completeOfflineOrder).toHaveBeenCalledWith('order-1');
    });
    await waitFor(() => {
      expect(onRefresh).toHaveBeenCalled();
    });
  });
});
