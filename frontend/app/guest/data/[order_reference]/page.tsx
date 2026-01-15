'use client';

import React, { useState, useEffect } from 'react';
import { useParams, useSearchParams, useRouter } from 'next/navigation';
import { useTranslation } from '@/i18n/provider';
import PublicLayout from '@/components/layout/PublicLayout';
import GuestDataSection from '@/components/guest/GuestDataSection';
import DeleteGuestDataButton from '@/components/guest/DeleteGuestDataButton';

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

export default function GuestDataPage() {
  const { t } = useTranslation(['guest_data', 'common']);
  const params = useParams();
  const searchParams = useSearchParams();
  const router = useRouter();
  
  const orderReference = params.order_reference as string;
  const email = searchParams.get('email');
  const phone = searchParams.get('phone');

  const [data, setData] = useState<GuestDataResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [isDeleted, setIsDeleted] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      if (!orderReference || (!email && !phone)) {
        setError(t('guest_data:errors.missing_credentials'));
        setIsLoading(false);
        return;
      }

      try {
        const params = new URLSearchParams();
        if (email) params.set('email', email);
        if (phone) params.set('phone', phone);

        const response = await fetch(
          `/api/guest/order/${orderReference}/data?${params.toString()}`,
          {
            method: 'GET',
            headers: {
              'Content-Type': 'application/json',
            },
          }
        );

        if (!response.ok) {
          if (response.status === 403) {
            throw new Error(t('guest_data:errors.verification_failed'));
          } else if (response.status === 404) {
            throw new Error(t('guest_data:errors.order_not_found'));
          } else {
            throw new Error(t('guest_data:errors.fetch_failed'));
          }
        }

        const result = await response.json();
        setData(result);
      } catch (err: any) {
        setError(err.message);
      } finally {
        setIsLoading(false);
      }
    };

    fetchData();
  }, [orderReference, email, phone, t]);

  const handleDeletionSuccess = () => {
    setIsDeleted(true);
  };

  const handleBackToLookup = () => {
    router.push('/guest/order-lookup');
  };

  if (isLoading) {
    return (
      <PublicLayout>
        <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl shadow-xl p-8">
            <div className="flex items-center space-x-4">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
              <p className="text-gray-600">{t('common:loading')}</p>
            </div>
          </div>
        </div>
      </PublicLayout>
    );
  }

  if (error) {
    return (
      <PublicLayout>
        <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
          <div className="max-w-md w-full">
            <div className="bg-white rounded-2xl shadow-xl p-8">
              <div className="text-center">
                <div className="inline-flex items-center justify-center w-16 h-16 bg-red-100 rounded-full mb-4">
                  <svg className="w-8 h-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                </div>
                <h2 className="text-2xl font-bold text-gray-900 mb-2">
                  {t('guest_data:errors.title')}
                </h2>
                <p className="text-gray-600 mb-6">{error}</p>
                <button
                  onClick={handleBackToLookup}
                  className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg font-medium hover:bg-blue-700 transition-colors"
                >
                  {t('guest_data:actions.back_to_lookup')}
                </button>
              </div>
            </div>
          </div>
        </div>
      </PublicLayout>
    );
  }

  if (isDeleted) {
    return (
      <PublicLayout>
        <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4">
          <div className="max-w-md w-full">
            <div className="bg-white rounded-2xl shadow-xl p-8">
              <div className="text-center">
                <div className="inline-flex items-center justify-center w-16 h-16 bg-green-100 rounded-full mb-4">
                  <svg className="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
                <h2 className="text-2xl font-bold text-gray-900 mb-2">
                  {t('guest_data:deletion.success_title')}
                </h2>
                <p className="text-gray-600 mb-6">
                  {t('guest_data:deletion.success_message')}
                </p>
                <button
                  onClick={handleBackToLookup}
                  className="w-full bg-blue-600 text-white py-3 px-4 rounded-lg font-medium hover:bg-blue-700 transition-colors"
                >
                  {t('guest_data:actions.back_to_lookup')}
                </button>
              </div>
            </div>
          </div>
        </div>
      </PublicLayout>
    );
  }

  if (!data) {
    return null;
  }

  return (
    <PublicLayout>
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 py-8 px-4">
        <div className="max-w-4xl mx-auto">
          {/* Header */}
          <div className="bg-white rounded-2xl shadow-xl p-8 mb-6">
            <div className="flex items-start justify-between">
              <div>
                <h1 className="text-3xl font-bold text-gray-900 mb-2">
                  {t('guest_data:data_page.title')}
                </h1>
                <p className="text-gray-600">
                  {t('guest_data:data_page.subtitle', { order_reference: orderReference })}
                </p>
              </div>
              <button
                onClick={handleBackToLookup}
                className="text-gray-500 hover:text-gray-700 transition-colors"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>

          {/* Data Sections */}
          <GuestDataSection data={data} />

          {/* Actions */}
          <div className="bg-white rounded-2xl shadow-xl p-8 mt-6">
            <h2 className="text-xl font-bold text-gray-900 mb-4">
              {t('guest_data:actions.title')}
            </h2>
            <p className="text-gray-600 mb-6">
              {t('guest_data:actions.deletion_info')}
            </p>
            <DeleteGuestDataButton
              orderReference={orderReference}
              email={email || undefined}
              phone={phone || undefined}
              onSuccess={handleDeletionSuccess}
            />
          </div>
        </div>
      </div>
    </PublicLayout>
  );
}
