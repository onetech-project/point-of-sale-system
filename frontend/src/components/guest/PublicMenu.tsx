import React, { useState, useEffect } from 'react';
import { product } from '../../services/product';
import ProductCard from './ProductCard';
import { Product } from '../../types/cart';
import { useTranslation } from 'react-i18next';

interface Category {
  id: string;
  name: string;
}

interface PublicMenuProps {
  tenantId: string;
  onAddToCart: (product: Product) => void;
  tenantInfo?: {
    name: string;
    logo?: string;
    description?: string;
  };
}

export const PublicMenu: React.FC<PublicMenuProps> = ({
  tenantId,
  onAddToCart,
  tenantInfo,
}) => {
  const { t } = useTranslation(['common']);
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [showAvailableOnly, setShowAvailableOnly] = useState<boolean>(true);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchProducts();
  }, [tenantId, selectedCategory, showAvailableOnly]);

  const fetchProducts = async () => {
    try {
      setLoading(true);
      setError(null);

      const params: any = {};
      if (selectedCategory !== 'all') {
        params.category = selectedCategory;
      }
      if (showAvailableOnly) {
        params.available_only = true;
      }

      const response = await product.getPublicMenu(tenantId, params);

      setProducts(response.products || []);

      // Extract unique categories from products using category_name
      const uniqueCategories = new Map<string, string>();
      response.products?.forEach((prod: any) => {
        if (prod.category_id && prod.category_name) {
          uniqueCategories.set(prod.category_id, prod.category_name);
        }
      });

      const categoryList = Array.from(uniqueCategories.entries()).map(([id, name]) => ({
        id,
        name,
      }));
      setCategories(categoryList);

    } catch (err: any) {
      console.error('Failed to fetch products:', err);
      setError(err.response?.data?.message || 'Failed to load menu. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleCategoryChange = (categoryId: string) => {
    setSelectedCategory(categoryId);
  };

  const handleAvailabilityToggle = () => {
    setShowAvailableOnly(!showAvailableOnly);
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-[400px]">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-center">
        <p className="text-red-800">{error}</p>
        <button
          onClick={fetchProducts}
          className="mt-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
        >
          {t('common.tryAgain')}
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Filters */}
      <div className="bg-white rounded-lg shadow-md p-4">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          {/* Category Filter */}
          <div className="flex items-center gap-2">
            <label htmlFor="category" className="text-sm font-medium text-gray-700">
              {t('common.menu.category')}:
            </label>
            <select
              id="category"
              value={selectedCategory}
              onChange={(e) => handleCategoryChange(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="all">{t('common.menu.allCategories')}</option>
              {categories.map((category) => (
                <option key={category.id} value={category.id}>
                  {category.name}
                </option>
              ))}
            </select>
          </div>

          {/* Availability Toggle */}
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="available"
              checked={showAvailableOnly}
              onChange={handleAvailabilityToggle}
              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
            />
            <label htmlFor="available" className="text-sm text-gray-700">
              {t('common.menu.showAvailableOnly')}
            </label>
          </div>
        </div>
      </div>

      {/* Product Grid */}
      {products.length === 0 ? (
        <div className="bg-gray-50 rounded-lg p-8 text-center">
          <p className="text-gray-600">{t('common.menu.noProducts')}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {products.map((product) => (
            <ProductCard
              key={product.id}
              product={product}
              onAddToCart={onAddToCart}
              tenantId={tenantId}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default PublicMenu;
