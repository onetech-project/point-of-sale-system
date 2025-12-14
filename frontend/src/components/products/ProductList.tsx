'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslation } from '@/i18n/provider';
import { product as productService } from '@/services/product';
import { Product, ProductListParams } from '@/types/product';
import { formatNumber } from '@/utils/format';

interface ProductListProps {
  categoryFilter?: string;
  showArchived?: boolean;
}

const ProductList: React.FC<ProductListProps> = ({ categoryFilter, showArchived = false }) => {
  const router = useRouter();
  const { t } = useTranslation(['products', 'common']);
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [lowStockOnly, setLowStockOnly] = useState(false);

  useEffect(() => {
    fetchProducts();
  }, [page, categoryFilter, showArchived, lowStockOnly]);

  const fetchProducts = async () => {
    try {
      setLoading(true);
      setError(null);

      const params: ProductListParams = {
        page,
        limit: 20,
        archived: showArchived,
      };

      if (searchQuery) {
        params.search = searchQuery;
      }

      if (categoryFilter) {
        params.category_id = categoryFilter;
      }

      if (lowStockOnly) {
        params.low_stock = true;
      }

      const response = await productService.getProducts(params);
      setProducts(response.data);
      setTotalPages(response.total_pages);
    } catch (err: any) {
      console.error('Failed to fetch products:', err);
      setError(err.response?.data?.message || t('products.messages.loadError'));
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    fetchProducts();
  };

  const handleProductClick = (productId: string) => {
    router.push(`/products/${productId}`);
  };

  const getStockStatus = (quantity: number) => {
    if (quantity <= 0) {
      return { label: t('products.list.outOfStock'), className: 'bg-red-100 text-red-800' };
    } else if (quantity <= 10) {
      return { label: t('products.list.lowStock'), className: 'bg-yellow-100 text-yellow-800' };
    }
    return { label: t('products.list.inStock'), className: 'bg-green-100 text-green-800' };
  };

  if (loading && products.length === 0) {
    return (
      <div className="flex justify-center items-center py-12">
        <div className="text-gray-500">{t('common.loading', { ns: 'common' })}...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded-md">
        <p className="text-sm text-red-600">{error}</p>
        <button
          onClick={fetchProducts}
          className="mt-2 text-sm text-red-600 underline hover:text-red-800"
        >
          {t('common.tryAgain', { ns: 'common' })}
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Search and Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <form onSubmit={handleSearch} className="flex-1 flex gap-2">
          <input
            type="text"
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            placeholder={t('products.list.search')}
            className="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          />
          <button
            type="submit"
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            {t('common.search', { ns: 'common' })}
          </button>
        </form>

        <div className="flex items-center">
          <label className="flex items-center space-x-2 text-sm">
            <input
              type="checkbox"
              checked={lowStockOnly}
              onChange={e => {
                setLowStockOnly(e.target.checked);
                setPage(1);
              }}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span>{t('products.list.lowStock')}</span>
          </label>
        </div>
      </div>

      {/* Product Grid */}
      {products && products.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-gray-500">{t('products.noProducts')}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {products &&
            products.map(product => {
              const stockStatus = getStockStatus(product.stock_quantity);
              return (
                <div
                  key={product.id}
                  onClick={() => handleProductClick(product.id)}
                  className="bg-white border border-gray-200 rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer"
                >
                  {/* Product Image */}
                  <div className="h-48 bg-gray-100 flex items-center justify-center">
                    {product.photo_path ? (
                      <img
                        src={productService.getPhotoUrl(product.id, undefined, product.photo_path)}
                        alt={product.name}
                        className="h-full w-full object-cover"
                      />
                    ) : (
                      <div className="text-gray-400">
                        <svg
                          className="w-16 h-16"
                          fill="none"
                          stroke="currentColor"
                          viewBox="0 0 24 24"
                        >
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                          />
                        </svg>
                      </div>
                    )}
                  </div>

                  {/* Product Details */}
                  <div className="p-4">
                    <div className="flex items-start justify-between mb-2">
                      <div className="flex-1">
                        <h3 className="font-semibold text-gray-900 truncate">{product.name}</h3>
                        <p className="text-sm text-gray-500">{product.sku}</p>
                      </div>
                      {product.archived_at && (
                        <span className="ml-2 px-2 py-1 text-xs font-medium bg-gray-100 text-gray-600 rounded">
                          {t('products.list.archived')}
                        </span>
                      )}
                    </div>

                    {product.category_name && (
                      <p className="text-xs text-gray-500 mb-2">{product.category_name}</p>
                    )}

                    <div className="flex items-center justify-between mb-2">
                      <span className="text-lg font-bold text-gray-900">
                        {formatNumber(product.selling_price, 0)}
                      </span>
                      <span
                        className={`px-2 py-1 text-xs font-medium rounded ${stockStatus.className}`}
                      >
                        {stockStatus.label}
                      </span>
                    </div>

                    <div className="text-sm text-gray-600">
                      {t('products.form.stockQuantity')}:{' '}
                      <span className="font-medium">{product.stock_quantity}</span>{' '}
                      {t('common.units', { ns: 'common' })}
                    </div>
                  </div>
                </div>
              );
            })}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center space-x-2 pt-6">
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={page === 1}
            className="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t('common.previous', { ns: 'common' })}
          </button>
          <span className="text-sm text-gray-600">
            {t('common.page', { ns: 'common' })} {page} {t('common.of', { ns: 'common' })}{' '}
            {totalPages}
          </span>
          <button
            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
            className="px-3 py-1 text-sm border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t('common.next', { ns: 'common' })}
          </button>
        </div>
      )}
    </div>
  );
};

export default ProductList;
