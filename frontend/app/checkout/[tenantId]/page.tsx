'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import CheckoutForm from '../../../src/components/guest/CheckoutForm';
import { CheckoutData } from '../../../src/types/checkout';
import { cart as cartService } from '../../../src/services/cart';
import { order } from '../../../src/services/order';
import { tenant } from '../../../src/services/tenant';
import consentService from '../../../src/services/consent';
import { Cart } from '../../../src/types/cart';
import PublicLayout from '../../../src/components/layout/PublicLayout';
import { useTranslation } from 'react-i18next';
import { formatPrice } from '../../../src/utils/format';

export default function CheckoutPage() {
  const { t } = useTranslation(['common']);
  const router = useRouter();
  const params = useParams();
  const tenantId = params?.tenantId as string;

  const [cart, setCart] = useState<Cart | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [deliveryFee, setDeliveryFee] = useState<number>(0);
  const [cartAdjusted, setCartAdjusted] = useState(false);

  useEffect(() => {
    if (tenantId) {
      loadCheckoutData();
    }
  }, [tenantId]);

  const loadCheckoutData = async () => {
    if (!tenantId) return;

    try {
      setLoading(true);
      setError(null);

      // Store previous cart for comparison
      const previousCart = cart;

      // Load both cart and tenant config in parallel
      const [cartData, tenantConfig] = await Promise.all([
        cartService.getCart(tenantId),
        tenant.getTenantConfig(tenantId)
      ]);

      if (!cartData || cartData.items.length === 0) {
        setError(t('common.checkout.emptyCart'));
        return;
      }

      // Check if cart was adjusted by backend
      if (previousCart && previousCart.items.length > 0) {
        const wasAdjusted = detectCartAdjustments(previousCart, cartData);
        if (wasAdjusted) {
          setCartAdjusted(true);
          setError(t('common.checkout.cartAdjusted', 'Your cart has been updated due to stock changes. Please review your order.'));
        }
      }

      setCart(cartData);

      // Set delivery fee from tenant config if charge_delivery_fee is enabled
      if (tenantConfig.charge_delivery_fee && tenantConfig.default_delivery_fee) {
        setDeliveryFee(tenantConfig.default_delivery_fee);
      }
    } catch (err: any) {
      console.error('Failed to load checkout data:', err);
      // Extract actual error message from backend response
      const errorMessage = err.response?.data?.message || err.response?.data?.error || err.message || t('common.checkout.loadError');
      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const detectCartAdjustments = (oldCart: Cart, newCart: Cart): boolean => {
    // Check if any items were removed
    if (oldCart.items.length !== newCart.items.length) {
      return true;
    }

    // Check if any quantities changed
    for (const oldItem of oldCart.items) {
      const newItem = newCart.items.find(i => i.product_id === oldItem.product_id);
      if (!newItem || newItem.quantity !== oldItem.quantity) {
        return true;
      }
    }

    return false;
  };

  const handleCheckout = async (data: CheckoutData) => {
    if (!tenantId || !cart) return;

    try {
      setSubmitting(true);
      setError(null);

      // Validate cart one final time before checkout
      const freshCart = await cartService.getCart(tenantId);

      // Check if cart was adjusted
      if (detectCartAdjustments(cart, freshCart)) {
        setCart(freshCart);
        setError(t('common.checkout.cartAdjustedBeforeSubmit', 'Your cart was updated. Please review before proceeding.'));
        setSubmitting(false);
        return;
      }

      // Create order via checkout API using guestOrderService
      const sessionId = cart.session_id;
      const orderResponse = await order.createOrder(
        tenantId,
        sessionId,
        data
      );

      // Grant consents for guest order (using order_id as guest identifier)
      if (data.consents && Object.keys(data.consents).length > 0) {
        const grantedConsents = Object.entries(data.consents)
          .filter(([, granted]) => granted)
          .map(([purpose_code]) => ({ purpose_code, granted: true }));

        if (grantedConsents.length > 0) {
          try {
            await consentService.grantConsents({
              tenant_id: tenantId,
              subject_type: 'guest',
              subject_id: orderResponse.order_id,
              consents: grantedConsents,
              metadata: consentService.getConsentMetadata(),
            });
          } catch (consentErr) {
            // Log but don't block order creation - consents are ancillary
            console.error('Failed to grant consents:', consentErr);
          }
        }
      }

      // T068: Redirect to order confirmation page
      // The order confirmation page will display the QR code for payment
      // Customer can scan and pay directly on the order detail page
      router.push(`/orders/${orderResponse.order_reference}`);

    } catch (err: any) {
      console.error('Checkout failed:', err);
      // Extract actual error message from backend response
      const errorMessage = err.response?.data?.message || err.response?.data?.error || err.message || t('common.checkout.submitError');
      setError(errorMessage);
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (error && !cart) {
    return (
      <PublicLayout>
        <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg shadow-md p-8 max-w-md w-full text-center">
            <div className="text-red-500 text-5xl mb-4">⚠️</div>
            <h1 className="text-2xl font-bold text-gray-900 mb-2">{t('common.checkout.error')}</h1>
            <p className="text-gray-600 mb-6">{error}</p>
            <button
              onClick={() => router.push(`/menu/${tenantId}`)}
              className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              {t('common.checkout.returnToMenu')}
            </button>
          </div>
        </div>
      </PublicLayout>
    );
  }

  return (
    <PublicLayout>
      <div className="min-h-screen bg-gray-50">
        {/* Page Header - Menu Style */}
        <div className="py-4">
          <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex items-center justify-between">
              <h1 className="text-3xl font-bold text-gray-900">{t('common.checkout.title')}</h1>
              <button
                onClick={() => router.push(`/menu/${tenantId}`)}
                className="text-gray-600 hover:text-gray-900 font-medium flex items-center gap-2"
              >
                <span>←</span>
                <span>{t('common.checkout.backToMenu')}</span>
              </button>
            </div>
          </div>
        </div>

        {/* Main Content */}
        <main className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 pb-8">
          {error && (
            <div className="mb-6 bg-red-50 border border-red-200 rounded-lg p-4">
              <p className="text-red-800">{error}</p>
            </div>
          )}

          {cart && (
            <>
              {/* Cart Summary */}
              <div className="bg-white rounded-lg shadow-md p-6 mb-6">
                <h2 className="text-lg font-semibold mb-4">{t('common.checkout.orderItems', { count: cart.items.length })}</h2>
                <div className="space-y-3">
                  {cart.items.map((item) => (
                    <div key={item.product_id} className="flex justify-between text-sm">
                      <span className="text-gray-700">
                        {item.quantity}x {item.product_name}
                      </span>
                      <span className="font-medium text-gray-900">
                        {formatPrice(item.total_price)}
                      </span>
                    </div>
                  ))}
                  <div className="flex justify-between text-base font-semibold border-t pt-3 mt-3">
                    <span className="text-gray-900">{t('common.cart.total')}:</span>
                    <span className="text-blue-600">
                      {formatPrice(cart.items.reduce((sum, item) => sum + item.total_price, 0))}
                    </span>
                  </div>
                </div>
              </div>

              {/* Checkout Form */}
              <CheckoutForm
                tenantId={tenantId}
                cartTotal={cart.items.reduce((sum, item) => sum + item.total_price, 0)}
                estimatedDeliveryFee={deliveryFee}
                onSubmit={handleCheckout}
                loading={submitting}
              />
            </>
          )}
        </main>
      </div>
    </PublicLayout>
  );
}
