'use client';

import React, { useState, useEffect, useRef } from 'react';
import { useRouter, useParams } from 'next/navigation';
import PublicMenu from '../../../src/components/guest/PublicMenu';
import GuestCart from '../../../src/components/guest/GuestCart';
import PublicLayout from '../../../src/components/layout/PublicLayout';
import { cart } from '../../../src/services/cart';
import { tenant } from '../../../src/services/tenant';
import { Cart, Product } from '../../../src/types/cart';
import { useTranslation } from 'react-i18next';

export default function PublicMenuPage() {
  const { t } = useTranslation(['common']);
  const router = useRouter();
  const params = useParams();
  const tenantId = params?.tenantId as string;

  const [cartData, setCartData] = useState<Cart | null>(null);
  const [cartLoading, setCartLoading] = useState<boolean>(true);
  const [tenantInfo, setTenantInfo] = useState<any>(null);
  const [tenantConfig, setTenantConfig] = useState<any>(null); // T107
  const [tenantError, setTenantError] = useState<string | null>(null); // T105
  const [tenantLoading, setTenantLoading] = useState<boolean>(true); // T105
  const [cartMessage, setCartMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [totalItem, setTotalItem] = useState<number>(0);
  const cartRef = useRef<HTMLDivElement>(null);

  const scrollToCart = () => {
    if (cartRef.current) {
      cartRef.current.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  };

  useEffect(() => {
    if (tenantId) {
      fetchTenantData(); // T107: Fetch tenant config
      loadCart();
    }

    // Add visibility change listener to reload cart when user returns to page
    const handleVisibilityChange = () => {
      if (!document.hidden && tenantId) {
        loadCart(true); // Show adjustment notice when reloading
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [tenantId]);

  // T107: Fetch tenant configuration
  const fetchTenantData = async () => {
    if (!tenantId) return;

    try {
      setTenantLoading(true);
      setTenantError(null);

      // Fetch tenant config (includes delivery types, branding, etc.)
      const configData = await tenant.getTenantConfig(tenantId);
      setTenantConfig(configData);

      // Extract tenant branding info for T104
      setTenantInfo({
        name: configData.tenant_name || 'Restaurant',
        logo: configData.logo_url,
        description: configData.description,
      });
    } catch (error: any) {
      console.error('Failed to fetch tenant data:', error);
      // T105: Handle invalid tenant error
      if (error.response?.status === 404) {
        setTenantError('Tenant not found. Please check the URL.');
      } else if (error.response?.status === 403) {
        setTenantError('This tenant is currently inactive.');
      } else {
        setTenantError('Failed to load restaurant information.');
      }
    } finally {
      setTenantLoading(false);
    }
  };

  const loadCart = async (showAdjustmentNotice: boolean = false) => {
    if (!tenantId) return;

    try {
      setCartLoading(true);

      // Store current cart for comparison if needed
      const previousCart = cartData;

      const cartResponse = await cart.getCart(tenantId);
      setCartData(cartResponse);

      // Check if cart was adjusted and notify user
      if (showAdjustmentNotice && previousCart && cartResponse) {
        const adjustments = detectCartAdjustments(previousCart, cartResponse);
        if (adjustments.length > 0) {
          showAdjustmentNotification(adjustments);
        }
      }
    } catch (error) {
      console.error('Failed to load cart:', error);
      // Initialize empty cart
      setCartData(null);
    } finally {
      setCartLoading(false);
    }
  };

  const detectCartAdjustments = (oldCart: Cart, newCart: Cart): Array<{ product: string, change: string }> => {
    const adjustments: Array<{ product: string, change: string }> = [];

    // Check for removed items
    oldCart.items.forEach(oldItem => {
      const newItem = newCart.items.find(i => i.product_id === oldItem.product_id);
      if (!newItem) {
        adjustments.push({
          product: oldItem.product_name,
          change: 'removed (out of stock)'
        });
      } else if (newItem.quantity < oldItem.quantity) {
        adjustments.push({
          product: oldItem.product_name,
          change: `reduced from ${oldItem.quantity} to ${newItem.quantity}`
        });
      }
    });

    return adjustments;
  };

  const showAdjustmentNotification = (adjustments: Array<{ product: string, change: string }>) => {
    const message = t('common.cart.adjusted', 'Your cart has been updated due to stock changes:') +
      '\n\n' +
      adjustments.map(adj => `• ${adj.product}: ${adj.change}`).join('\n');
    alert(message);
  };

  const handleAddToCart = async (product: Product) => {
    if (!tenantId) return;

    try {
      const updatedCart = await cart.addItem(
        tenantId,
        product.id,
        product.name,
        1,
        product.price
      );
      setCartData(updatedCart);
      setCartMessage({ type: 'success', text: t('common.cart.added', 'Item added to cart!') });
      setTotalItem(updatedCart.items.reduce((sum, item) => sum + item.quantity, 0));
    } catch (error: any) {
      console.error('Failed to add item to cart:', error);
      // Extract actual error message from backend
      const errorMessage = error.response?.data?.message || error.message || 'Failed to add item to cart';
      setCartMessage({ type: 'error', text: errorMessage });
    }
    // Hide message after 2.5 seconds
    setTimeout(() => setCartMessage(null), 2500);
  };

  const handleUpdateQuantity = async (productId: string, quantity: number) => {
    if (!tenantId) return;

    try {
      const updatedCart = await cart.updateItem(tenantId, productId, quantity);
      setCartData(updatedCart);
      setTotalItem(updatedCart.items.reduce((sum, item) => sum + item.quantity, 0));
    } catch (error: any) {
      console.error('Failed to update quantity:', error);
      // Extract actual error message from backend
      const errorMessage = error.response?.data?.message || error.message || 'Failed to update item quantity';
      alert(errorMessage);
      // Reload cart to sync state
      await loadCart();
    }
  };

  const handleRemoveItem = async (productId: string) => {
    if (!tenantId) return;

    try {
      const updatedCart = await cart.removeItem(tenantId, productId);
      setCartData(updatedCart);
      setTotalItem(updatedCart.items.reduce((sum, item) => sum + item.quantity, 0));
    } catch (error: any) {
      console.error('Failed to remove item:', error);
      // Extract actual error message from backend
      const errorMessage = error.response?.data?.message || error.message || 'Failed to remove item';
      alert(errorMessage);
      // Reload cart to sync state
      await loadCart();
    }
  };

  const handleClearCart = async () => {
    if (!tenantId) return;

    if (!confirm(t('common.cart.confirmClear'))) {
      return;
    }

    try {
      await cart.clearCart(tenantId);
      setCartData(null);
    } catch (error: any) {
      console.error('Failed to clear cart:', error);
      // Extract actual error message from backend
      const errorMessage = error.response?.data?.message || error.message || 'Failed to clear cart';
      alert(errorMessage);
    }
  };

  const validateCartStock = async (): Promise<boolean> => {
    if (!tenantId || !cartData) return false;

    try {
      // Reload cart to get latest data and validate each item
      const freshCart = await cart.getCart(tenantId);
      setCartData(freshCart);

      if (!freshCart || freshCart.items.length === 0) {
        alert(t('common.cart.empty', 'Your cart is empty'));
        return false;
      }

      // Check each item in cart - this will trigger validation on backend
      let hasErrors = false;
      for (const item of freshCart.items) {
        try {
          // Validate by attempting to update with current quantity
          await cart.updateItem(tenantId, item.product_id, item.quantity);
        } catch (error: any) {
          hasErrors = true;
          const errorMessage = error.response?.data?.message || error.message || 'Stock validation failed';
          alert(`${item.product_name}: ${errorMessage}`);
        }
      }

      if (hasErrors) {
        // Reload cart again to sync adjusted quantities
        await loadCart();
        return false;
      }

      return true;
    } catch (error: any) {
      console.error('Failed to validate cart:', error);
      const errorMessage = error.response?.data?.message || error.message || 'Failed to validate cart';
      alert(errorMessage);
      return false;
    }
  };

  const handleCheckout = async () => {
    if (!tenantId) return;

    // Validate cart stock before navigating to checkout
    const isValid = await validateCartStock();
    if (!isValid) {
      return;
    }

    // Navigate to checkout page
    router.push(`/checkout/${tenantId}`);
  };

  if (!tenantId) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <p className="text-gray-600">Loading...</p>
        </div>
      </div>
    );
  }

  // T105: Invalid tenant error page
  if (tenantError) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-8">
          <div className="text-center">
            {/* Error Icon */}
            <div className="inline-flex items-center justify-center w-16 h-16 bg-red-100 rounded-full mb-4">
              <svg
                className="w-8 h-8 text-red-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>

            {/* Error Title */}
            <h2 className="text-2xl font-bold text-gray-900 mb-2">
              {tenantError.includes('not found')
                ? 'Restaurant Not Found'
                : tenantError.includes('inactive')
                  ? 'Restaurant Unavailable'
                  : 'Error Loading Restaurant'}
            </h2>

            {/* Error Message */}
            <p className="text-gray-600 mb-6">{tenantError}</p>

            {/* Actions */}
            <div className="space-y-3">
              <button
                onClick={() => fetchTenantData()}
                className="w-full px-6 py-3 bg-blue-600 text-white rounded-lg font-medium hover:bg-blue-700 transition-colors"
              >
                Try Again
              </button>
              <button
                onClick={() => router.push('/')}
                className="w-full px-6 py-3 bg-gray-100 text-gray-700 rounded-lg font-medium hover:bg-gray-200 transition-colors"
              >
                Go to Home
              </button>
            </div>

            {/* Help Text */}
            <p className="mt-6 text-sm text-gray-500">
              If you believe this is an error, please contact support or verify
              the restaurant URL.
            </p>
          </div>
        </div>
      </div>
    );
  }

  // Loading state
  if (tenantLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mb-4"></div>
          <p className="text-gray-600">Loading restaurant...</p>
        </div>
      </div>
    );
  }

  return (
    <PublicLayout>
      <div className="min-h-screen bg-gray-50">
        {/* Main Content */}
        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          {/* Menu Header with Tenant Info and Cart Counter */}
          <div className="mb-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                {tenantInfo?.logo && (
                  <img
                    src={tenantInfo.logo}
                    alt={`${tenantInfo.name} logo`}
                    className="w-12 h-12 rounded-lg object-cover"
                  />
                )}
                <div>
                  <h1 className="text-3xl font-bold text-gray-900">
                    {tenantInfo?.name || t('common.menu.title')}
                  </h1>
                  {tenantInfo?.description && (
                    <p className="text-gray-600 mt-1">{tenantInfo.description}</p>
                  )}
                </div>
              </div>
              {/* Floating Cart Button for Mobile */}
              <div>
                <button
                  onClick={scrollToCart}
                  className="lg:hidden fixed bottom-6 right-6 z-50 bg-blue-600 hover:bg-blue-700 text-white rounded-full shadow-lg w-16 h-16 flex items-center justify-center transition-colors"
                  style={{ boxShadow: '0 4px 24px rgba(0,0,0,0.15)' }}
                  aria-label={t('common.menu.itemsInCart')}
                >
                  {/* Cart Icon */}
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2 9m13-9l2 9m-5-9V6a2 2 0 10-4 0v3" />
                  </svg>
                  {/* Badge */}
                  {cartData && cartData.items.length > 0 ? (
                    <span className="absolute top-2 right-2 bg-red-500 text-white text-xs font-bold rounded-full px-2 py-0.5 min-w-[1.5rem] text-center border-2 border-white">
                      {totalItem}
                    </span>
                  ) : null}
                </button>
              </div>
              <div className="hidden lg:block bg-blue-600 text-white px-4 py-2 rounded-lg shadow-sm">
                <span className="text-sm font-medium">
                  {t('common.menu.itemsInCart')}: {totalItem}
                </span>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
            {/* Menu Section - 2 columns */}
            <div className="lg:col-span-2">
              <PublicMenu
                tenantId={tenantId}
                onAddToCart={handleAddToCart}
                tenantInfo={tenantInfo}
              />
            </div>

            {/* Cart Section - 1 column */}
            <div className="lg:col-span-1" ref={cartRef}>
              <GuestCart
                cart={cartData}
                loading={cartLoading}
                tenantId={tenantId}
                onUpdateQuantity={handleUpdateQuantity}
                onRemoveItem={handleRemoveItem}
                onClearCart={handleClearCart}
                onCheckout={handleCheckout}
              />
            </div>

            {/* show timed message for items success or error added to cart */}
            {cartMessage && (
              <div
                className={`fixed bottom-4 left-1/2 transform -translate-x-1/2 z-50 px-6 py-3 rounded-lg shadow-lg text-white transition-all duration-300
                  ${cartMessage.type === 'success' ? 'bg-green-600' : 'bg-red-600'}`}
                role="alert"
              >
                {cartMessage.text}
              </div>
            )}


          </div>
        </main>
      </div>
    </PublicLayout>
  );
}
