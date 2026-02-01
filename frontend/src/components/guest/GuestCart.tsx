import React, { useState } from 'react';
import { Cart, CartItem } from '../../types/cart';
import { product as productService } from '../../services/product';
import { useTranslation } from 'react-i18next';
import { formatPrice } from '../../utils/format';

interface GuestCartProps {
  cart: Cart | null;
  loading: boolean;
  tenantId: string;
  onUpdateQuantity: (productId: string, quantity: number) => Promise<void>;
  onRemoveItem: (productId: string) => Promise<void>;
  onClearCart: () => Promise<void>;
  onCheckout: () => void;
}

export const GuestCart: React.FC<GuestCartProps> = ({
  cart,
  loading,
  tenantId,
  onUpdateQuantity,
  onRemoveItem,
  onClearCart,
  onCheckout,
}) => {
  const { t } = useTranslation(['common']);
  const [updatingItems, setUpdatingItems] = useState<Set<string>>(new Set());

  const handleQuantityChange = async (productId: string, newQuantity: number) => {
    if (newQuantity < 1) return;

    setUpdatingItems(prev => new Set(prev).add(productId));
    try {
      await onUpdateQuantity(productId, newQuantity);
    } finally {
      setUpdatingItems(prev => {
        const next = new Set(prev);
        next.delete(productId);
        return next;
      });
    }
  };

  const handleRemoveItem = async (productId: string) => {
    setUpdatingItems(prev => new Set(prev).add(productId));
    try {
      await onRemoveItem(productId);
    } finally {
      setUpdatingItems(prev => {
        const next = new Set(prev);
        next.delete(productId);
        return next;
      });
    }
  };

  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <div className="flex justify-center items-center h-32">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      </div>
    );
  }

  if (!cart || cart.items.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-xl font-semibold mb-4">{t('common.cart.title')}</h2>
        <div className="text-center py-8">
          <p className="text-gray-600">{t('common.cart.empty')}</p>
          <p className="text-sm text-gray-500 mt-2">{t('common.cart.emptyDescription')}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-md p-6 sticky top-4">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold">{t('common.cart.title')}</h2>
        <button
          onClick={onClearCart}
          className="text-sm text-red-600 hover:text-red-700 font-medium"
        >
          {t('common.cart.clearCart')}
        </button>
      </div>

      <div className="space-y-4 mb-6">
        {cart.items.map((item) => {
          const isUpdating = updatingItems.has(item.product_id);

          return (
            <div
              key={item.product_id}
              className={`flex gap-3 pb-4 border-gray-200 ${isUpdating ? 'opacity-50' : ''
                }`}
            >
              {/* Item Image */}
              {item.image_url && (
                <div className="w-16 h-16 bg-gray-200 rounded overflow-hidden">
                  <img
                    src={productService.getPhotoUrl(item.product_id, tenantId, item.image_url)}
                    alt={item.product_name}
                    className="w-full h-full object-cover"
                    onError={(e) => {
                      const target = e.target as HTMLImageElement;
                      target.style.display = 'none';
                      const parent = target.parentElement;
                      if (parent) {
                        parent.innerHTML = '<span class="text-xs text-gray-400 flex items-center justify-center h-full">No Image</span>';
                      }
                    }}
                  />
                </div>
              )}
              {/* Item Details */}
              <div className="flex-1">
                <h3 className="font-medium text-gray-800">{item.product_name}</h3>
                <p className="text-sm text-gray-600">{formatPrice(item.unit_price)}</p>

                {/* Quantity Controls */}
                <div className="flex items-center gap-2 mt-2">
                  <button
                    onClick={() => handleQuantityChange(item.product_id, item.quantity - 1)}
                    disabled={isUpdating || item.quantity <= 1}
                    className="w-8 h-8 flex items-center justify-center bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    -
                  </button>
                  <span className="w-8 text-center font-medium">{item.quantity}</span>
                  <button
                    onClick={() => handleQuantityChange(item.product_id, item.quantity + 1)}
                    disabled={isUpdating}
                    className="w-8 h-8 flex items-center justify-center bg-gray-100 rounded hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    +
                  </button>
                  <button
                    onClick={() => handleRemoveItem(item.product_id)}
                    disabled={isUpdating}
                    title={t('common.cart.remove')}
                  >
                    {/* html emoji of trash 🗑️ */}
                    🗑️
                  </button>
                </div>
              </div>

              {/* Item Total */}
              <div className="text-right">
                <p className="font-medium text-gray-800">
                  {formatPrice(item.total_price)}
                </p>
              </div>
            </div>
          );
        })}
      </div>

      {/* Cart Summary */}
      <div className="space-y-3 mb-6">
        <div className="flex justify-between text-lg font-semibold border-t pt-4">
          <span data-testid="cart-total">{t('common.cart.total')}:</span>
          <span className="text-blue-600">
            {formatPrice(cart.items.reduce((sum, item) => sum + item.total_price, 0))}
          </span>
        </div>
      </div>

      {/* Checkout Button */}
      <button
        data-testid="proceed-to-checkout-button"
        onClick={onCheckout}
        className="w-full py-3 bg-blue-600 text-white rounded-lg font-semibold hover:bg-blue-700 transition-colors"
      >
        {t('common.cart.proceedToCheckout')}
      </button>

      {/* Cart Info */}
      {cart.expires_at && (
        <p className="text-xs text-gray-500 text-center mt-4">
          {t('common.cart.expiresAt')} {new Date(cart.expires_at).toLocaleString('id-ID', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
          })}
        </p>
      )}
    </div>
  );
};

export default GuestCart;
