import React from 'react';
import { product as productService } from '../../services/product';
import { useTranslation } from 'react-i18next';
import { Product } from '../../types/cart';
import { formatPrice } from '../../utils/format';

interface ProductCardProps {
  product: Product;
  onAddToCart: (product: Product) => void;
  tenantId: string;
}

export const ProductCard: React.FC<ProductCardProps> = ({ product, onAddToCart, tenantId }) => {
  const { t } = useTranslation(['common']);

  return (
    <div data-testid={`product-card`} className="bg-white rounded-lg shadow-md overflow-hidden hover:shadow-lg transition-shadow flex flex-col h-full">
      {product.image_url ? (
        <img
          src={productService.getPhotoUrl(product.id, tenantId, product.image_url)}
          alt={product.name}
          className="w-full h-48 object-cover"
        />
      ) : (
        <div className="w-full h-48 bg-gray-200 flex items-center justify-center">
          <span className="text-gray-400">No Image</span>
        </div>
      )}

      <div className="p-4 flex flex-col flex-grow">
        <h3 data-testid={`product-name`} className="text-lg font-semibold text-gray-800 mb-1 line-clamp-1">{product.name}</h3>

        {product.description && (
          <p className="text-sm text-gray-600 mb-2 line-clamp-2 h-10">{product.description}</p>
        )}
        {!product.description && (
          <div className="h-10 mb-2"></div>
        )}

        <div className="mt-auto space-y-3">
          <div className="flex flex-col gap-2">
            <span className={`text-xs px-2 py-1 rounded self-start whitespace-nowrap ${product.is_available && product.available_stock > 0
              ? 'bg-green-100 text-green-800'
              : 'bg-red-100 text-red-800'
              }`}>
              {product.is_available && product.available_stock > 0
                ? `${t('common.product.stock')}: ${product.available_stock}`
                : t('common.product.outOfStock')}
            </span>

            <span className="text-xl font-bold text-blue-600">
              {formatPrice(product.price)}
            </span>
          </div>

          <button
            data-testid={`add-to-cart-button`}
            onClick={() => onAddToCart(product)}
            disabled={!product.is_available || product.available_stock <= 0}
            className={`w-full py-2 px-4 rounded-lg font-medium transition-colors ${product.is_available && product.available_stock > 0
              ? 'bg-blue-600 text-white hover:bg-blue-700'
              : 'bg-gray-300 text-gray-500 cursor-not-allowed'
              }`}
          >
            {product.is_available && product.available_stock > 0 ? t('common.product.addToCart') : t('common.product.unavailable')}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ProductCard;
