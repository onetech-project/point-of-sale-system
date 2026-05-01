'use client';

import React, { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import offlineOrderService from '@/services/offlineOrders';
import { product as productService } from '@/services/product';
import { formatCurrency } from '@/utils/format';
import type {
  OfflineOrder,
  UpdateOfflineOrderRequest,
  DeliveryType,
  OrderItemInput,
} from '@/types/offlineOrder';
import type { Product } from '@/types/product';

/**
 * Edit Offline Order Page (T082)
 * US3: Edit offline orders with audit trail
 */
export default function EditOfflineOrderPage() {
  const params = useParams();
  const router = useRouter();
  const orderId = params.id as string;

  const [order, setOrder] = useState<OfflineOrder | null>(null);
  const [items, setItems] = useState<OrderItemInput[]>([]);
  const [originalItems, setOriginalItems] = useState<OrderItemInput[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [loadingProducts, setLoadingProducts] = useState(false);
  const [newProductId, setNewProductId] = useState('');
  const [newQuantity, setNewQuantity] = useState(1);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  // Form state
  const [customerName, setCustomerName] = useState('');
  const [customerPhone, setCustomerPhone] = useState('');
  const [customerEmail, setCustomerEmail] = useState('');
  const [deliveryType, setDeliveryType] = useState<DeliveryType>('pickup');
  const [tableNumber, setTableNumber] = useState('');
  const [notes, setNotes] = useState('');
  const [deliveryFee, setDeliveryFee] = useState(0);

  const normalizeItems = (rawItems: any[]): OrderItemInput[] => {
    return (rawItems || []).map(item => ({
      product_id: item.product_id,
      product_name: item.product_name,
      quantity: Number(item.quantity) || 0,
      unit_price: Number(item.unit_price) || 0,
    }));
  };

  useEffect(() => {
    const fetchProducts = async () => {
      try {
        setLoadingProducts(true);
        const response = await productService.getProducts({ limit: 200 });
        setProducts(response.data || []);
      } catch (err) {
        console.error('Failed to load products:', err);
      } finally {
        setLoadingProducts(false);
      }
    };

    fetchProducts();
  }, []);

  useEffect(() => {
    const fetchOrder = async () => {
      if (!orderId) {
        setError('Order ID is required');
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        setError(null);

        const orderDetails = await offlineOrderService.getOfflineOrderWithDetails(orderId);
        const orderData = orderDetails.order;

        // Check if order can be edited
        if (orderData.status !== 'PENDING') {
          setError(
            `Cannot edit order with status ${orderData.status}. Only PENDING orders can be edited.`
          );
          setTimeout(() => {
            router.push(`/orders/offline-orders/${orderId}`);
          }, 2000);
          return;
        }

        setOrder(orderData);
        const loadedItems = normalizeItems(orderDetails.items || []);
        setItems(loadedItems);
        setOriginalItems(loadedItems);

        // Populate form with existing data
        setCustomerName(orderData.customer_name);
        setCustomerPhone(orderData.customer_phone);
        setCustomerEmail(orderData.customer_email || '');
        setDeliveryType(orderData.delivery_type);
        setTableNumber(orderData.table_number || '');
        setNotes(orderData.notes || '');
        setDeliveryFee(orderData.delivery_fee);
      } catch (err: any) {
        console.error('Failed to fetch order:', err);
        setError(err.response?.data?.message || 'Failed to load order details');
      } finally {
        setLoading(false);
      }
    };

    fetchOrder();
  }, [orderId, router]);

  const handleItemProductChange = (index: number, productId: string) => {
    const selectedProduct = products.find(p => p.id === productId);
    if (!selectedProduct) return;

    const nextItems = [...items];
    nextItems[index] = {
      ...nextItems[index],
      product_id: selectedProduct.id,
      product_name: selectedProduct.name,
      unit_price: Number(selectedProduct.selling_price) || 0,
    };
    setItems(nextItems);
  };

  const handleItemQuantityChange = (index: number, quantity: number) => {
    const nextItems = [...items];
    nextItems[index] = {
      ...nextItems[index],
      quantity: quantity > 0 ? quantity : 1,
    };
    setItems(nextItems);
  };

  const handleRemoveItem = (index: number) => {
    setItems(prev => prev.filter((_, i) => i !== index));
  };

  const handleAddItem = () => {
    if (!newProductId) {
      setError('Please select a product to add');
      return;
    }
    if (newQuantity < 1) {
      setError('Quantity must be at least 1');
      return;
    }

    const selectedProduct = products.find(p => p.id === newProductId);
    if (!selectedProduct) {
      setError('Selected product not found');
      return;
    }

    setItems(prev => [
      ...prev,
      {
        product_id: selectedProduct.id,
        product_name: selectedProduct.name,
        quantity: newQuantity,
        unit_price: Number(selectedProduct.selling_price) || 0,
      },
    ]);

    setNewProductId('');
    setNewQuantity(1);
    setError(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    try {
      // Build update request with only changed fields
      const updates: UpdateOfflineOrderRequest = {};

      if (order) {
        if (customerName !== order.customer_name) updates.customer_name = customerName;
        if (customerPhone !== order.customer_phone) updates.customer_phone = customerPhone;
        if (customerEmail !== (order.customer_email || '')) updates.customer_email = customerEmail;
        if (deliveryType !== order.delivery_type) updates.delivery_type = deliveryType;
        if (tableNumber !== (order.table_number || '')) updates.table_number = tableNumber;
        if (notes !== (order.notes || '')) updates.notes = notes;
        if (deliveryFee !== order.delivery_fee) updates.delivery_fee = deliveryFee;
      }

      if (items.length === 0) {
        setError('At least one order item is required');
        setSubmitting(false);
        return;
      }

      const invalidItem = items.find(
        item => !item.product_id || !item.product_name || item.quantity < 1 || item.unit_price < 0
      );
      if (invalidItem) {
        setError('Please ensure all items have a valid product and quantity');
        setSubmitting(false);
        return;
      }

      const hasItemChanges = JSON.stringify(items) !== JSON.stringify(originalItems);
      if (hasItemChanges) {
        updates.items = items;
      }

      // Check if there are any changes
      if (Object.keys(updates).length === 0) {
        setError('No changes detected');
        setSubmitting(false);
        return;
      }

      await offlineOrderService.updateOfflineOrder(orderId, updates);

      setSuccess(true);
      setTimeout(() => {
        router.push(`/orders/offline-orders/${orderId}`);
      }, 1000);
    } catch (err: any) {
      console.error('Failed to update order:', err);
      setError(err.response?.data?.error || 'Failed to update order');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = () => {
    router.push(`/orders/offline-orders/${orderId}`);
  };

  return (
    <ProtectedRoute>
      <DashboardLayout>
        <div className="space-y-6 max-w-4xl mx-auto">
          {/* Header */}
          <div className="bg-white rounded-lg shadow p-6">
            <button
              onClick={handleCancel}
              className="text-blue-600 hover:text-blue-800 mb-2 flex items-center gap-1"
            >
              ← Back to Order
            </button>
            <h1 className="text-3xl font-bold text-gray-900">Edit Order</h1>
            {order && (
              <p className="text-gray-600 mt-2">
                Order {order.order_reference} - {formatCurrency(order.total_amount)}
              </p>
            )}
          </div>

          {/* Loading State */}
          {loading && (
            <div className="bg-white rounded-lg shadow p-12 text-center">
              <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
              <p className="mt-4 text-gray-600">Loading order details...</p>
            </div>
          )}

          {/* Error State */}
          {error && !loading && (
            <div className="bg-red-50 border border-red-200 rounded-lg shadow p-6">
              <div className="flex items-center">
                <svg
                  className="h-6 w-6 text-red-600 mr-3"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                <div>
                  <h3 className="text-lg font-semibold text-red-900">Error</h3>
                  <p className="text-red-700 mt-1">{error}</p>
                </div>
              </div>
              <button
                onClick={handleCancel}
                className="mt-4 px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors"
              >
                Back to Order
              </button>
            </div>
          )}

          {/* Success State */}
          {success && (
            <div className="bg-green-50 border border-green-200 rounded-lg shadow p-6">
              <div className="flex items-center">
                <svg
                  className="h-6 w-6 text-green-600 mr-3"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
                <div>
                  <h3 className="text-lg font-semibold text-green-900">Success!</h3>
                  <p className="text-green-700 mt-1">Order updated successfully. Redirecting...</p>
                </div>
              </div>
            </div>
          )}

          {/* Edit Form */}
          {order && !loading && !error && !success && (
            <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-6 space-y-6">
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm text-blue-800">
                  <strong>Note:</strong> Only PENDING orders can be edited. All changes are recorded
                  in the audit trail.
                </p>
              </div>

              {/* Customer Information */}
              <div className="space-y-4">
                <h2 className="text-lg font-semibold text-gray-900">Customer Information</h2>

                <div>
                  <label
                    htmlFor="customerName"
                    className="block text-sm font-medium text-gray-700 mb-1"
                  >
                    Customer Name *
                  </label>
                  <input
                    type="text"
                    id="customerName"
                    value={customerName}
                    onChange={e => setCustomerName(e.target.value)}
                    required
                    minLength={2}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>

                <div>
                  <label
                    htmlFor="customerPhone"
                    className="block text-sm font-medium text-gray-700 mb-1"
                  >
                    Customer Phone *
                  </label>
                  <input
                    type="tel"
                    id="customerPhone"
                    value={customerPhone}
                    onChange={e => setCustomerPhone(e.target.value)}
                    required
                    minLength={10}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>

                <div>
                  <label
                    htmlFor="customerEmail"
                    className="block text-sm font-medium text-gray-700 mb-1"
                  >
                    Customer Email (optional)
                  </label>
                  <input
                    type="email"
                    id="customerEmail"
                    value={customerEmail}
                    onChange={e => setCustomerEmail(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              </div>

              {/* Delivery Information */}
              <div className="space-y-4">
                <h2 className="text-lg font-semibold text-gray-900">Delivery Information</h2>

                <div>
                  <label
                    htmlFor="deliveryType"
                    className="block text-sm font-medium text-gray-700 mb-1"
                  >
                    Delivery Type *
                  </label>
                  <select
                    id="deliveryType"
                    value={deliveryType}
                    onChange={e => setDeliveryType(e.target.value as DeliveryType)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  >
                    <option value="pickup">Pickup</option>
                    <option value="delivery">Delivery</option>
                    <option value="dine_in">Dine In</option>
                  </select>
                </div>

                {deliveryType === 'dine_in' && (
                  <div>
                    <label
                      htmlFor="tableNumber"
                      className="block text-sm font-medium text-gray-700 mb-1"
                    >
                      Table Number
                    </label>
                    <input
                      type="text"
                      id="tableNumber"
                      value={tableNumber}
                      onChange={e => setTableNumber(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                )}

                <div>
                  <label
                    htmlFor="deliveryFee"
                    className="block text-sm font-medium text-gray-700 mb-1"
                  >
                    Delivery Fee (IDR)
                  </label>
                  <input
                    type="number"
                    id="deliveryFee"
                    value={deliveryFee}
                    onChange={e => setDeliveryFee(parseInt(e.target.value) || 0)}
                    min="0"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>

                <div>
                  <label htmlFor="notes" className="block text-sm font-medium text-gray-700 mb-1">
                    Notes (optional)
                  </label>
                  <textarea
                    id="notes"
                    value={notes}
                    onChange={e => setNotes(e.target.value)}
                    rows={3}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                  />
                </div>
              </div>

              {/* Order Items */}
              <div className="space-y-4">
                <h2 className="text-lg font-semibold text-gray-900">Order Items</h2>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                  <div className="md:col-span-2">
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Add Product
                    </label>
                    <select
                      value={newProductId}
                      onChange={e => setNewProductId(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                      disabled={loadingProducts}
                    >
                      <option value="">
                        {loadingProducts ? 'Loading products...' : 'Select product'}
                      </option>
                      {products.map(product => (
                        <option key={product.id} value={product.id}>
                          {product.name} ({formatCurrency(product.selling_price)})
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Quantity</label>
                    <input
                      type="number"
                      min={1}
                      value={newQuantity}
                      onChange={e => setNewQuantity(parseInt(e.target.value, 10) || 1)}
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    />
                  </div>
                </div>

                <button
                  type="button"
                  onClick={handleAddItem}
                  className="px-4 py-2 bg-blue-50 text-blue-700 rounded-md hover:bg-blue-100 border border-blue-200"
                >
                  Add Item
                </button>

                {items.length > 0 ? (
                  <div className="border border-gray-200 rounded-lg overflow-hidden">
                    <table className="min-w-full divide-y divide-gray-200">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                            Product
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                            Qty
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                            Unit Price
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                            Total
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                            Actions
                          </th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200">
                        {items.map((item, index) => (
                          <tr key={`${item.product_id}-${index}`}>
                            <td className="px-6 py-3 text-sm text-gray-900">
                              <select
                                value={item.product_id}
                                onChange={e => handleItemProductChange(index, e.target.value)}
                                className="w-full px-2 py-1 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                              >
                                {products.map(product => (
                                  <option key={product.id} value={product.id}>
                                    {product.name}
                                  </option>
                                ))}
                              </select>
                            </td>
                            <td className="px-6 py-3 text-sm text-gray-900">
                              <input
                                type="number"
                                min={1}
                                value={item.quantity}
                                onChange={e =>
                                  handleItemQuantityChange(index, parseInt(e.target.value, 10) || 1)
                                }
                                className="w-24 px-2 py-1 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                              />
                            </td>
                            <td className="px-6 py-3 text-sm text-gray-900">
                              {formatCurrency(item.unit_price)}
                            </td>
                            <td className="px-6 py-3 text-sm font-medium text-gray-900">
                              {formatCurrency(item.quantity * item.unit_price)}
                            </td>
                            <td className="px-6 py-3 text-sm text-gray-900">
                              <button
                                type="button"
                                onClick={() => handleRemoveItem(index)}
                                className="text-red-600 hover:text-red-800"
                              >
                                Remove
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <p className="text-gray-500 text-sm">No items in this order.</p>
                )}
              </div>

              {/* Action Buttons */}
              <div className="flex justify-end gap-3 pt-4 border-t">
                <button
                  type="button"
                  onClick={handleCancel}
                  disabled={submitting}
                  className="px-6 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 font-medium disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium disabled:opacity-50"
                >
                  {submitting ? 'Saving Changes...' : 'Save Changes'}
                </button>
              </div>
            </form>
          )}
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
