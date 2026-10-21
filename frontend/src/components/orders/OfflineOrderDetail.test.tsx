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

const mockUseAuth = jest.fn();
jest.mock('../../store/auth', () => ({
  useAuth: () => mockUseAuth(),
}));

jest.mock('../../services/offlineOrders', () => ({
  __esModule: true,
  default: {
    batchDownloadDocuments: jest.fn(),
    completeOfflineOrder: jest.fn(),
    deleteCompletedOfflineOrder: jest.fn(),
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

const completedOrder: OfflineOrder = {
  ...paidOrder,
  status: 'COMPLETE',
  completed_at: '2026-05-20T03:00:00Z',
};

const asRole = (role: string) => mockUseAuth.mockReturnValue({ user: { id: 'user-1', email: 'u@example.com', role } });

beforeEach(() => {
  jest.clearAllMocks();
  asRole('owner');
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

describe('OfflineOrderDetail completed order corrections', () => {
  it('shows edit and delete for completed orders only to the owner', () => {
    const { rerender } = render(<OfflineOrderDetail order={completedOrder} />);

    expect(screen.getByRole('button', { name: 'Edit Completed Order' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Delete Completed Order' })).toBeInTheDocument();

    for (const role of ['manager', 'cashier']) {
      asRole(role);
      rerender(<OfflineOrderDetail order={{ ...completedOrder }} />);
      expect(screen.queryByRole('button', { name: 'Edit Completed Order' })).not.toBeInTheDocument();
      expect(screen.queryByRole('button', { name: 'Delete Completed Order' })).not.toBeInTheDocument();
    }
  });

  it('hides the pending order delete button from cashiers', () => {
    asRole('cashier');
    render(<OfflineOrderDetail order={pendingOrder} />);

    expect(screen.getByRole('button', { name: 'Edit Order' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument();
  });

  it('requires a reason and acknowledgement before deleting a completed order', async () => {
    mockOfflineOrderService.deleteCompletedOfflineOrder.mockResolvedValue({
      message: 'deleted',
      order_id: 'order-1',
    });
    const onDeleted = jest.fn();

    render(<OfflineOrderDetail order={completedOrder} onDeleted={onDeleted} />);

    fireEvent.click(screen.getByRole('button', { name: 'Delete Completed Order' }));
    expect(screen.getByText('Perubahan ini akan memengaruhi pencatatan')).toBeInTheDocument();

    const confirm = screen.getByRole('button', { name: 'Ya, Hapus Order' });
    expect(confirm).toBeDisabled();

    fireEvent.change(screen.getByLabelText(/Alasan penghapusan/), {
      target: { value: '  Pesanan dobel  ' },
    });
    expect(confirm).toBeDisabled();

    fireEvent.click(screen.getByRole('checkbox'));
    expect(confirm).toBeEnabled();

    fireEvent.click(confirm);

    await waitFor(() => {
      expect(mockOfflineOrderService.deleteCompletedOfflineOrder).toHaveBeenCalledWith('order-1', {
        reason: 'Pesanan dobel',
        acknowledge_accounting_impact: true,
      });
    });
    await waitFor(() => expect(onDeleted).toHaveBeenCalled());
  });

  it('keeps the dialog open and shows the API error when deletion fails', async () => {
    mockOfflineOrderService.deleteCompletedOfflineOrder.mockRejectedValue({
      response: { data: { error: 'order is not complete' } },
    });

    render(<OfflineOrderDetail order={completedOrder} />);

    fireEvent.click(screen.getByRole('button', { name: 'Delete Completed Order' }));
    fireEvent.change(screen.getByLabelText(/Alasan penghapusan/), { target: { value: 'Pesanan dobel' } });
    fireEvent.click(screen.getByRole('checkbox'));
    fireEvent.click(screen.getByRole('button', { name: 'Ya, Hapus Order' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('order is not complete');
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });
});
