import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { OrderConfirmation } from './OrderConfirmation';

jest.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (_key: string, fallback?: string) => fallback ?? _key }),
}));

const baseProps = {
  orderReference: 'GO-0001',
  deliveryType: 'pickup',
  customerName: 'Customer',
  customerPhone: '081234567890',
  subtotal: 30000,
  deliveryFee: 0,
  total: 30000,
  orderStatus: 'PENDING',
  createdAt: '2026-05-21T01:00:00Z',
};

it('keeps showing the hosted QR image for Midtrans payments', () => {
  render(
    <OrderConfirmation
      {...baseProps}
      paymentInfo={{
        transaction_id: 'trx-1',
        transaction_status: 'pending',
        payment_type: 'qris',
        payment_gateway: 'midtrans',
        qr_code_url: 'https://api.sandbox.midtrans.com/qris/qr.png',
        qr_string: '000201...',
      }}
    />
  );

  const qr = screen.getByTestId('payment-qris-qr-code');
  expect(qr.tagName).toBe('IMG');
  expect(qr).toHaveAttribute('src', 'https://api.sandbox.midtrans.com/qris/qr.png');
});

it('renders the QR from the raw QRIS string for Manjo payments', () => {
  render(
    <OrderConfirmation
      {...baseProps}
      paymentInfo={{
        transaction_id: 'A0000001702',
        transaction_status: 'pending',
        payment_type: 'qris',
        payment_gateway: 'manjo',
        qr_code_url: '-',
        qr_string: '00020101021226620015ID.CO.MANJO.WWW0118936008580175125703',
      }}
    />
  );

  const qr = screen.getByTestId('payment-qris-qr-code');
  expect(qr.tagName).toBe('CANVAS');
  expect(screen.getByText('Download QR Code')).toBeInTheDocument();
});

it('shows no QR when the payment has neither an image nor a QRIS string', () => {
  render(
    <OrderConfirmation
      {...baseProps}
      paymentInfo={{
        transaction_id: 'A0000001702',
        transaction_status: 'pending',
        payment_type: 'qris',
        payment_gateway: 'manjo',
        qr_string: null,
      }}
    />
  );

  expect(screen.queryByTestId('payment-qris-qr-code')).not.toBeInTheDocument();
});
