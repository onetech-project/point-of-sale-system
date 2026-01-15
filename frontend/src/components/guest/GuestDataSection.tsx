import React from 'react';
import { useTranslation } from '@/i18n/provider';

interface CustomerInfo {
  name: string;
  phone?: string;
  email?: string;
}

interface OrderItem {
  id: string;
  name: string;
  quantity: number;
  price: number;
  subtotal: number;
}

interface OrderDetails {
  order_id: string;
  order_reference: string;
  total_amount: number;
  payment_method: string;
  order_type: string;
  status: string;
  created_at: string;
  items: OrderItem[];
}

interface DeliveryAddress {
  full_address: string;
  latitude?: number;
  longitude?: number;
}

interface GuestDataResponse {
  order_reference: string;
  customer_info: CustomerInfo;
  order_details: OrderDetails;
  delivery_address?: DeliveryAddress;
}

interface GuestDataSectionProps {
  data: GuestDataResponse;
}

export default function GuestDataSection({ data }: GuestDataSectionProps) {
  const { t } = useTranslation(['guest_data', 'common']);

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('id-ID', {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
    }).format(amount);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('id-ID', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getStatusBadgeColor = (status: string) => {
    const statusColors: { [key: string]: string } = {
      'pending': 'bg-yellow-100 text-yellow-800',
      'confirmed': 'bg-blue-100 text-blue-800',
      'preparing': 'bg-purple-100 text-purple-800',
      'ready': 'bg-green-100 text-green-800',
      'completed': 'bg-green-100 text-green-800',
      'cancelled': 'bg-red-100 text-red-800',
    };
    return statusColors[status.toLowerCase()] || 'bg-gray-100 text-gray-800';
  };

  return (
    <div className="space-y-6">
      {/* Customer Information */}
      <div className="bg-white rounded-2xl shadow-xl p-8">
        <div className="flex items-center mb-6">
          <div className="bg-blue-100 rounded-lg p-3 mr-4">
            <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
            </svg>
          </div>
          <h2 className="text-2xl font-bold text-gray-900">
            {t('guest_data:sections.customer_info')}
          </h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.name')}</p>
            <p className="text-lg font-semibold text-gray-900">{data.customer_info.name}</p>
          </div>

          {data.customer_info.email && (
            <div>
              <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.email')}</p>
              <p className="text-lg font-semibold text-gray-900">{data.customer_info.email}</p>
            </div>
          )}

          {data.customer_info.phone && (
            <div>
              <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.phone')}</p>
              <p className="text-lg font-semibold text-gray-900">{data.customer_info.phone}</p>
            </div>
          )}
        </div>
      </div>

      {/* Order Details */}
      <div className="bg-white rounded-2xl shadow-xl p-8">
        <div className="flex items-center mb-6">
          <div className="bg-green-100 rounded-lg p-3 mr-4">
            <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          </div>
          <h2 className="text-2xl font-bold text-gray-900">
            {t('guest_data:sections.order_details')}
          </h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          <div>
            <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.order_reference')}</p>
            <p className="text-lg font-semibold text-gray-900">{data.order_details.order_reference}</p>
          </div>

          <div>
            <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.status')}</p>
            <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${getStatusBadgeColor(data.order_details.status)}`}>
              {data.order_details.status}
            </span>
          </div>

          <div>
            <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.order_date')}</p>
            <p className="text-lg font-semibold text-gray-900">{formatDate(data.order_details.created_at)}</p>
          </div>

          <div>
            <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.order_type')}</p>
            <p className="text-lg font-semibold text-gray-900">{data.order_details.order_type}</p>
          </div>

          <div>
            <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.payment_method')}</p>
            <p className="text-lg font-semibold text-gray-900">{data.order_details.payment_method}</p>
          </div>

          <div>
            <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.total_amount')}</p>
            <p className="text-2xl font-bold text-green-600">{formatCurrency(data.order_details.total_amount)}</p>
          </div>
        </div>

        {/* Order Items */}
        {data.order_details.items && data.order_details.items.length > 0 && (
          <div className="mt-6 border-t pt-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              {t('guest_data:sections.order_items')}
            </h3>
            <div className="space-y-3">
              {data.order_details.items.map((item) => (
                <div key={item.id} className="flex justify-between items-center py-3 border-b last:border-b-0">
                  <div className="flex-1">
                    <p className="font-medium text-gray-900">{item.name}</p>
                    <p className="text-sm text-gray-500">
                      {item.quantity} x {formatCurrency(item.price)}
                    </p>
                  </div>
                  <p className="font-semibold text-gray-900">{formatCurrency(item.subtotal)}</p>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Delivery Address */}
      {data.delivery_address && (
        <div className="bg-white rounded-2xl shadow-xl p-8">
          <div className="flex items-center mb-6">
            <div className="bg-purple-100 rounded-lg p-3 mr-4">
              <svg className="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
            </div>
            <h2 className="text-2xl font-bold text-gray-900">
              {t('guest_data:sections.delivery_address')}
            </h2>
          </div>

          <div>
            <p className="text-sm text-gray-500 mb-1">{t('guest_data:fields.address')}</p>
            <p className="text-lg text-gray-900">{data.delivery_address.full_address}</p>
          </div>

          {data.delivery_address.latitude && data.delivery_address.longitude && (
            <div className="mt-4 text-sm text-gray-500">
              <p>
                {t('guest_data:fields.coordinates')}: {data.delivery_address.latitude.toFixed(6)}, {data.delivery_address.longitude.toFixed(6)}
              </p>
            </div>
          )}
        </div>
      )}

      {/* Privacy Notice */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
        <div className="flex">
          <div className="flex-shrink-0">
            <svg className="h-5 w-5 text-blue-400" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
            </svg>
          </div>
          <div className="ml-3">
            <h3 className="text-sm font-medium text-blue-800 mb-1">
              {t('guest_data:privacy.title')}
            </h3>
            <p className="text-sm text-blue-700">
              {t('guest_data:privacy.message')}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
