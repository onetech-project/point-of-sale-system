import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { PaymentInfo, OrderItem } from '../../types/cart';
import { renderTextWithLinks, formatCurrency } from '../../utils/text';
import { download } from '../../utils/download';

interface OrderConfirmationProps {
  orderReference: string;
  deliveryType: string;
  customerName: string;
  customerPhone: string;
  deliveryAddress?: string;
  tableNumber?: string;
  subtotal: number;
  deliveryFee: number;
  total: number;
  orderStatus: string;
  createdAt: string;
  paymentQrUrl?: string;
  paymentInfo?: PaymentInfo;
  notes?: string;
  items?: OrderItem[];
  customerNotes?: string;
}

export const OrderConfirmation: React.FC<OrderConfirmationProps> = ({
  orderReference,
  deliveryType,
  customerName,
  customerPhone,
  deliveryAddress,
  tableNumber,
  subtotal,
  deliveryFee,
  total,
  orderStatus,
  createdAt,
  paymentQrUrl,
  paymentInfo,
  notes,
  items,
  customerNotes
}) => {
  const { t } = useTranslation();
  const [timeRemaining, setTimeRemaining] = useState<string>('');

  // Countdown timer for payment expiry
  useEffect(() => {
    if (!paymentInfo?.expiry_time || orderStatus !== 'PENDING') {
      setTimeRemaining('');
      return;
    }

    const { expiry_time, remaining_time } = paymentInfo;

    let secondsLeft = remaining_time || 0;
    const updateTimer = () => {
      if (!expiry_time) return;

      if (secondsLeft <= 0) {
        setTimeRemaining('Expired');
        return;
      }

      const minutes = Math.floor(secondsLeft / 60);
      const seconds = Math.floor(secondsLeft % 60);

      setTimeRemaining(
        `${minutes.toString().padStart(2, '0')}:${seconds
          .toString()
          .padStart(2, '0')}`
      );
      secondsLeft -= 1;
    };

    updateTimer();
    const interval = setInterval(updateTimer, 1000);

    return () => clearInterval(interval);
  }, [paymentInfo?.expiry_time, orderStatus]);

  const formatDate = (dateString: string): string => {
    return new Date(dateString).toLocaleString('id-ID', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'PAID':
        return 'text-green-600 bg-green-100';
      case 'PENDING':
        return 'text-yellow-600 bg-yellow-100';
      case 'COMPLETE':
        return 'text-blue-600 bg-blue-100';
      case 'CANCELLED':
        return 'text-red-600 bg-red-100';
      default:
        return 'text-gray-600 bg-gray-100';
    }
  };

  const getDeliveryTypeLabel = (type: string): string => {
    switch (type) {
      case 'delivery':
        return t('checkout.deliveryType.delivery', 'Delivery');
      case 'pickup':
        return t('checkout.deliveryType.pickup', 'Pickup');
      case 'dine_in':
        return t('checkout.deliveryType.dineIn', 'Dine In');
      default:
        return type;
    }
  };

  const getStatusMessage = (status: string): React.ReactElement => {
    let messageObj: { message: string; color?: string } = { message: '', color: '' };
    switch (status) {
      case 'PENDING':
        messageObj = {
          message: t(
            'orderConfirmation.pendingMessage',
            'Please complete payment to confirm your order.'
          ),
          color: 'text-yellow-800',
        }
        break;
      case 'PAID':
        messageObj = {
          message: t(
            'orderConfirmation.paidMessage',
            'Payment received! Your order is being prepared.'
          ),
          color: 'text-green-800',
        };
        break;
      case 'COMPLETE':
        messageObj = {
          message: t(
            'orderConfirmation.completeMessage',
            'Your order has been completed. Thank you!'
          ),
          color: 'text-blue-800',
        };
        break;
      case 'CANCELLED':
        messageObj = {
          message: t(
            'orderConfirmation.cancelledMessage',
            'This order has been cancelled.'
          ),
          color: 'text-red-800',
        };
        break;
      default:
        messageObj = { message: '', color: '' };
    }
    return <p className={`text-sm text-center ${messageObj.color}`}>
      {messageObj.message}
    </p>
  };

  return (
    <div className="max-w-2xl mx-auto p-6 bg-white rounded-lg shadow-lg">
      {/* Header */}
      <div className="text-center mb-6">
        <div className="inline-flex items-center justify-center w-16 h-16 bg-green-100 rounded-full mb-4">
          <svg
            className="w-8 h-8 text-green-600"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M5 13l4 4L19 7"
            />
          </svg>
        </div>
        <h1 className="text-2xl font-bold text-gray-900 mb-2">
          {t('orderConfirmation.title', 'Order Confirmed')}
        </h1>
        <p className="text-gray-600">
          {t('orderConfirmation.subtitle', 'Thank you for your order!')}
        </p>
      </div>

      {/* Payment QR Code - Show for PENDING orders */}
      {orderStatus === 'PENDING' && (paymentInfo?.qr_code_url || paymentQrUrl) && (
        <div className="my-6 p-6 bg-white border-2 border-blue-500 rounded-lg">
          <div className="text-center">
            <h2 className="text-xl font-bold text-gray-900 mb-2">
              {t('orderConfirmation.scanToPay', 'Scan to Pay')}
            </h2>
            <p className="text-sm text-gray-600 mb-4">
              {t('orderConfirmation.qrisInstruction', 'Scan this QR code with your mobile banking or e-wallet app')}
            </p>
            <div className="flex justify-center">
              <img
                src={paymentInfo?.qr_code_url || paymentQrUrl || ''}
                alt="QRIS Payment QR Code"
                className="w-64 h-64 border-4 border-gray-200 rounded-lg"
                onError={(e) => {
                  console.error('Failed to load QR code image');
                  e.currentTarget.style.display = 'none';
                }}
              />
            </div>
            {timeRemaining && (
              <div className="mt-4">
                <p className="text-lg font-semibold text-gray-900">
                  {t('orderConfirmation.timeRemaining', 'Time Remaining')}
                </p>
                <p className={`text-3xl font-bold ${timeRemaining === 'Expired' ? 'text-red-600' : 'text-blue-600'}`}>
                  {timeRemaining}
                </p>
              </div>
            )}
            <p className="mt-4 text-xs text-gray-500">
              {t('orderConfirmation.paymentExpiry', 'Please complete payment within 15 minutes')}
            </p>
          </div>
          {/* rounded block button for download the QR code with fetch */}
          <div className="mt-6 text-center">
            <div
              onClick={() => download(paymentInfo?.qr_code_url || paymentQrUrl || '', `QRIS_${orderReference}.png`, 'image/png')}
              className="inline-block px-4 py-2 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 cursor-pointer"
            >
              {t('orderConfirmation.downloadQr', 'Download QR Code')}
            </div>
          </div>

        </div>
      )}

      {/* Order Reference */}
      <div className="bg-gray-50 rounded-lg p-4 mb-6">
        <div className="flex justify-between items-center mb-2">
          <span className="text-sm text-gray-600">
            {t('orderConfirmation.reference', 'Order Reference')}
          </span>
          <span
            className={`px-3 py-1 rounded-full text-sm font-medium ${getStatusColor(
              orderStatus
            )}`}
          >
            {orderStatus}
          </span>
        </div>
        <p className="text-2xl font-mono font-bold text-gray-900">
          {orderReference}
        </p>
        <p className="text-sm text-gray-500 mt-1">{formatDate(createdAt)}</p>
      </div>

      {/* Customer Details */}
      <div className="mb-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-3">
          {t('orderConfirmation.customerDetails', 'Customer Details')}
        </h2>
        <div className="space-y-2">
          <div className="flex justify-between">
            <span className="text-gray-600">{t('checkout.name', 'Name')}</span>
            <span className="font-medium text-gray-900">{customerName}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">{t('checkout.phone', 'Phone')}</span>
            <span className="font-medium text-gray-900">{customerPhone}</span>
          </div>
        </div>
      </div>

      {/* Delivery Details */}
      <div className="mb-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-3">
          {t('orderConfirmation.deliveryDetails', 'Delivery Details')}
        </h2>
        <div className="space-y-2">
          <div className="flex justify-between">
            <span className="text-gray-600">
              {t('checkout.deliveryType', 'Delivery Type')}
            </span>
            <span className="font-medium text-gray-900">
              {getDeliveryTypeLabel(deliveryType)}
            </span>
          </div>
          {deliveryAddress && deliveryType === 'delivery' && (
            <div className="flex justify-between">
              <span className="text-gray-600">
                {t('checkout.address', 'Address')}
              </span>
              <span className="font-medium text-gray-900 text-right max-w-xs">
                {deliveryAddress}
              </span>
            </div>
          )}
          {tableNumber && deliveryType === 'dine_in' && (
            <div className="flex justify-between">
              <span className="text-gray-600">
                {t('checkout.tableNumber', 'Table Number')}
              </span>
              <span className="font-medium text-gray-900">{tableNumber}</span>
            </div>
          )}
        </div>
      </div>

      {/* Order Summary */}
      <div className="border-t pt-6">
        <h2 className="text-lg font-semibold text-gray-900 mb-3">
          {t('orderConfirmation.orderSummary', 'Order Summary')}
        </h2>

        {/* Order Items */}
        {items && items.length > 0 && (
          <div className="mb-4 space-y-2">
            {items.map((item) => (
              <div key={item.id} className="flex justify-between text-sm text-gray-600">
                <div className="flex-1">
                  <span className="font-medium text-gray-900">{item.product_name}</span>
                  <span className="text-gray-500"> × {item.quantity}</span>
                </div>
                <span className="font-medium text-gray-900">{formatCurrency(item.total_price)}</span>
              </div>
            ))}

            {/* Customer Note */}
            {customerNotes && (
              <div className="mb-4 p-4 bg-gray-50 rounded-lg">
                <h3 className="text-sm font-semibold text-gray-900 mb-2">
                  {t('orderConfirmation.customerNote', 'Notes')}
                </h3>
                <p className="text-sm text-gray-700 whitespace-pre-wrap">
                  {renderTextWithLinks(customerNotes)}
                </p>
              </div>
            )}
            <div className="border-t my-2"></div>
          </div>
        )}

        <div className="space-y-2">
          <div className="flex justify-between text-gray-600">
            <span>{t('checkout.subtotal', 'Subtotal')}</span>
            <span>{formatCurrency(subtotal)}</span>
          </div>
          {deliveryFee > 0 && (
            <div className="flex justify-between text-gray-600">
              <span>{t('checkout.deliveryFee', 'Delivery Fee')}</span>
              <span>{formatCurrency(deliveryFee)}</span>
            </div>
          )}
          <div className="flex justify-between text-xl font-bold text-gray-900 pt-2 border-t">
            <span>{t('checkout.total', 'Total')}</span>
            <span>{formatCurrency(total)}</span>
          </div>
        </div>
      </div>

      {/* Status Message */}
      <div className="mt-6 p-4 bg-blue-50 rounded-lg">
        {getStatusMessage(orderStatus)}
      </div>

      {/* Admin Notes / Courier Info */}
      {notes && (
        <div className="mt-6 p-4 bg-amber-50 border border-amber-200 rounded-lg">
          <div className="flex items-start gap-3">
            <svg className="w-5 h-5 text-amber-600 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div className="flex-1">
              <h3 className="text-sm font-semibold text-amber-900 mb-1">
                {t('orderConfirmation.noteFromAdmin', 'Tenant Notes')}
              </h3>
              <p className="text-sm text-amber-800 whitespace-pre-wrap">
                {renderTextWithLinks(notes)}
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Action Buttons - Commented out for now */}
      {/* 
      <div className="mt-6 flex gap-3">
        <button
          onClick={() => window.print()}
          className="flex-1 px-4 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium hover:bg-gray-200 transition-colors"
        >
          {t('orderConfirmation.print', 'Print Order')}
        </button>
        <button
          onClick={() => (window.location.href = `/orders/${orderReference}`)}
          className="flex-1 px-4 py-3 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-colors"
        >
          {t('orderConfirmation.trackOrder', 'Track Order')}
        </button>
      </div>
      */}

      {/* Help Text */}
      <p className="mt-6 text-center text-sm text-gray-500">
        {t(
          'orderConfirmation.helpText',
          'Keep your order reference for tracking. You can check order status anytime using the order reference.'
        )}
      </p>
    </div>
  );
};

export default OrderConfirmation;
