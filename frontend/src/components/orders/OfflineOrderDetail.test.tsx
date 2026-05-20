import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { OfflineOrderDetail } from './OfflineOrderDetail';
import { OfflineOrder } from '../../types/offlineOrder';

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
  }),
}));

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
