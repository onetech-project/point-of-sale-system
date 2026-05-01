'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import offlineOrderService from '../../services/offlineOrders';
import { product as productService } from '../../services/product';
import { formatCurrency } from '../../utils/format';
import {
  CreateOfflineOrderRequest,
  ConsentMethod,
  PaymentSchedule,
} from '../../types/offlineOrder';

interface Product {
  id: string;
  name: string;
  selling_price: number;
  stock_quantity: number;
}

interface OfflineOrderFormProps {
  products?: Product[];
  onSuccess?: (orderId: string) => void;
  onCancel?: () => void;
}

interface OrderItem {
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  total_price: number;
}

export const OfflineOrderForm: React.FC<OfflineOrderFormProps> = ({
  products: initialProducts,
  onSuccess,
  onCancel,
}) => {
  const router = useRouter();
  const [products, setProducts] = useState<Product[]>(initialProducts || []);
  const [loadingProducts, setLoadingProducts] = useState(!initialProducts);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load products if not provided as props
  useEffect(() => {
    if (initialProducts) {
      setProducts(initialProducts);
      setLoadingProducts(false);
      return;
    }

    const fetchProducts = async () => {
      try {
        setLoadingProducts(true);
        const response = await productService.getProducts({ limit: 100 });
        setProducts(response.data || []);
      } catch (err: any) {
        console.error('Failed to load products:', err);
        setError('Failed to load products. Please try again.');
        setProducts([]);
      } finally {
        setLoadingProducts(false);
      }
    };

    fetchProducts();
  }, [initialProducts]);

  // Customer Information
  const [customerName, setCustomerName] = useState('');
  const [customerPhone, setCustomerPhone] = useState('');
  const [customerEmail, setCustomerEmail] = useState('');

  // Delivery Details
  const [deliveryType, setDeliveryType] = useState<'pickup' | 'delivery' | 'dine_in'>('pickup');
  const [tableNumber, setTableNumber] = useState('');
  const [notes, setNotes] = useState('');

  // Data Consent
  const [dataConsentGiven, setDataConsentGiven] = useState(false);
  const [consentMethod, setConsentMethod] = useState<ConsentMethod>('verbal');

  // Order Items
  const [orderItems, setOrderItems] = useState<OrderItem[]>([]);
  const [selectedProductId, setSelectedProductId] = useState('');
  const [quantity, setQuantity] = useState(1);

  // Payment Options
  const [paymentType, setPaymentType] = useState<'full' | 'installment'>('full');
  const [downPaymentAmount, setDownPaymentAmount] = useState(0);
  const [installmentCount, setInstallmentCount] = useState(2);

  // Calculate totals
  const subtotal = orderItems.reduce((sum, item) => sum + item.total_price, 0);
  const deliveryFee = deliveryType === 'delivery' ? 10000 : 0; // 10,000 IDR for delivery
  const total = subtotal + deliveryFee;

  const handleAddItem = () => {
    if (!selectedProductId || quantity < 1) {
      setError('Please select a product and quantity');
      return;
    }

    const product = products?.find(p => p.id === selectedProductId);
    if (!product) {
      setError('Product not found');
      return;
    }

    if (quantity > product.stock_quantity) {
      setError(`Only ${product.stock_quantity} items available in stock`);
      return;
    }

    const newItem: OrderItem = {
      product_id: product.id,
      product_name: product.name,
      quantity,
      unit_price: product.selling_price,
      total_price: product.selling_price * quantity,
    };

    setOrderItems([...orderItems, newItem]);
    setSelectedProductId('');
    setQuantity(1);
    setError(null);
  };

  const handleRemoveItem = (index: number) => {
    setOrderItems(orderItems.filter((_, i) => i !== index));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validation
    if (!customerName.trim()) {
      setError('Customer name is required');
      return;
    }
    if (!customerPhone.trim()) {
      setError('Customer phone is required');
      return;
    }
    if (orderItems.length === 0) {
      setError('Please add at least one item to the order');
      return;
    }
    if (!dataConsentGiven) {
      setError('Customer data consent is required');
      return;
    }
    if (deliveryType === 'dine_in' && !tableNumber.trim()) {
      setError('Table number is required for dine-in orders');
      return;
    }

    // Build request
    const request: CreateOfflineOrderRequest = {
      customer_name: customerName.trim(),
      customer_phone: customerPhone.trim(),
      customer_email: customerEmail.trim() || undefined,
      delivery_type: deliveryType,
      table_number: deliveryType === 'dine_in' ? tableNumber.trim() : undefined,
      notes: notes.trim() || undefined,
      items: orderItems.map(item => ({
        product_id: item.product_id,
        product_name: item.product_name,
        quantity: item.quantity,
        unit_price: item.unit_price,
      })),
      data_consent_given: dataConsentGiven,
      consent_method: consentMethod,
    };

    // Add payment terms if installment
    if (paymentType === 'installment') {
      const installmentAmount = Math.ceil((total - downPaymentAmount) / installmentCount);
      const schedule: PaymentSchedule[] = [];

      for (let i = 1; i <= installmentCount; i++) {
        const dueDate = new Date();
        dueDate.setMonth(dueDate.getMonth() + i);
        schedule.push({
          installment_number: i,
          due_date: dueDate.toISOString().split('T')[0],
          amount: installmentAmount,
          status: 'pending',
        });
      }

      request.payment_terms = {
        down_payment_amount: downPaymentAmount,
        installment_count: installmentCount,
        installment_amount: installmentAmount,
        payment_schedule: schedule,
      };
    }

    try {
      setLoading(true);
      const response = await offlineOrderService.createOfflineOrder(request);

      if (onSuccess) {
        onSuccess(response.order.id);
      } else {
        router.push(`/orders/offline-orders/${response.order.id}`);
      }
    } catch (err: any) {
      console.error('Failed to create offline order:', err);
      setError(err.response?.data?.error || 'Failed to create offline order. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="space-y-6 max-w-4xl mx-auto p-6 bg-white rounded-lg shadow"
    >
      <h2 className="text-2xl font-bold text-gray-900">Create Offline Order</h2>

      {loadingProducts && (
        <div className="bg-blue-50 border border-blue-200 text-blue-700 px-4 py-3 rounded">
          Loading products...
        </div>
      )}

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {/* Customer Information */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-800">Customer Information</h3>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Customer Name *</label>
          <input
            type="text"
            value={customerName}
            onChange={e => setCustomerName(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Customer Phone *</label>
          <input
            type="tel"
            value={customerPhone}
            onChange={e => setCustomerPhone(e.target.value)}
            placeholder="+62..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Customer Email (Optional)
          </label>
          <input
            type="email"
            value={customerEmail}
            onChange={e => setCustomerEmail(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>

      {/* Delivery Details */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-800">Delivery Details</h3>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Delivery Type *</label>
          <select
            value={deliveryType}
            onChange={e => setDeliveryType(e.target.value as any)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="pickup">Pickup</option>
            <option value="delivery">Delivery</option>
            <option value="dine_in">Dine In</option>
          </select>
        </div>

        {deliveryType === 'dine_in' && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Table Number *</label>
            <input
              type="text"
              value={tableNumber}
              onChange={e => setTableNumber(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              required
            />
          </div>
        )}

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Notes (Optional)</label>
          <textarea
            value={notes}
            onChange={e => setNotes(e.target.value)}
            rows={3}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>

      {/* Order Items */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-800">Order Items</h3>

        <div className="flex gap-2">
          <select
            value={selectedProductId}
            onChange={e => setSelectedProductId(e.target.value)}
            className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
          >
            <option value="">Select Product</option>
            {products?.map(product => (
              <option key={product.id} value={product.id}>
                {product.name} - {formatCurrency(product.selling_price)} (Stock:{' '}
                {product.stock_quantity})
              </option>
            ))}
          </select>
          <input
            type="number"
            min="1"
            value={quantity}
            onChange={e => setQuantity(parseInt(e.target.value) || 1)}
            className="w-24 px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
          />
          <button
            type="button"
            onClick={handleAddItem}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
          >
            Add
          </button>
        </div>

        {orderItems.length > 0 && (
          <div className="border border-gray-200 rounded-md overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                    Product
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                    Qty
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                    Price
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                    Total
                  </th>
                  <th className="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {orderItems.map((item, index) => (
                  <tr key={index}>
                    <td className="px-4 py-3 text-sm text-gray-900">{item.product_name}</td>
                    <td className="px-4 py-3 text-sm text-gray-900 text-right">{item.quantity}</td>
                    <td className="px-4 py-3 text-sm text-gray-900 text-right">
                      {formatCurrency(item.unit_price)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-900 text-right">
                      {formatCurrency(item.total_price)}
                    </td>
                    <td className="px-4 py-3 text-right">
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
        )}

        <div className="bg-gray-50 p-4 rounded-md space-y-2">
          <div className="flex justify-between text-sm">
            <span>Subtotal:</span>
            <span>{formatCurrency(subtotal)}</span>
          </div>
          {deliveryType === 'delivery' && (
            <div className="flex justify-between text-sm">
              <span>Delivery Fee:</span>
              <span>{formatCurrency(deliveryFee)}</span>
            </div>
          )}
          <div className="flex justify-between text-lg font-bold border-t pt-2">
            <span>Total:</span>
            <span>{formatCurrency(total)}</span>
          </div>
        </div>
      </div>

      {/* Payment Options - T068: Extend form to include payment terms options */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-800">Payment Options</h3>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Payment Type *</label>
          <div className="space-y-2">
            <label className="flex items-center">
              <input
                type="radio"
                name="paymentType"
                value="full"
                checked={paymentType === 'full'}
                onChange={() => setPaymentType('full')}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
              />
              <span className="ml-2 text-sm text-gray-700">Full Payment (Paid in full)</span>
            </label>
            <label className="flex items-center">
              <input
                type="radio"
                name="paymentType"
                value="installment"
                checked={paymentType === 'installment'}
                onChange={() => setPaymentType('installment')}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
              />
              <span className="ml-2 text-sm text-gray-700">Installment Payment (Pay in parts)</span>
            </label>
          </div>
        </div>

        {paymentType === 'installment' && (
          <div className="space-y-4 p-4 bg-blue-50 rounded-md border border-blue-200">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Down Payment Amount
              </label>
              <input
                type="number"
                value={downPaymentAmount}
                onChange={e => setDownPaymentAmount(parseInt(e.target.value) || 0)}
                min="0"
                max={total}
                step="10000"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              />
              <p className="text-xs text-gray-600 mt-1">
                {downPaymentAmount > 0 ? formatCurrency(downPaymentAmount) : 'No down payment'}
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Number of Installments
              </label>
              <select
                value={installmentCount}
                onChange={e => setInstallmentCount(parseInt(e.target.value))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              >
                <option value={2}>2 installments</option>
                <option value={3}>3 installments</option>
                <option value={4}>4 installments</option>
                <option value={6}>6 installments</option>
                <option value={12}>12 installments</option>
              </select>
            </div>

            {downPaymentAmount < total && (
              <div className="bg-white p-3 rounded-md border border-blue-300">
                <p className="text-sm font-medium text-gray-700 mb-2">Payment Summary:</p>
                <div className="space-y-1 text-sm">
                  {downPaymentAmount > 0 && (
                    <div className="flex justify-between">
                      <span className="text-gray-600">Down Payment:</span>
                      <span className="font-medium">{formatCurrency(downPaymentAmount)}</span>
                    </div>
                  )}
                  <div className="flex justify-between">
                    <span className="text-gray-600">Per Installment:</span>
                    <span className="font-medium">
                      {formatCurrency(Math.ceil((total - downPaymentAmount) / installmentCount))}
                    </span>
                  </div>
                  <div className="flex justify-between pt-1 border-t">
                    <span className="text-gray-600">Total:</span>
                    <span className="font-bold">{formatCurrency(total)}</span>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Data Consent */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-gray-800">Data Consent</h3>

        <div className="flex items-start">
          <input
            type="checkbox"
            id="dataConsent"
            checked={dataConsentGiven}
            onChange={e => setDataConsentGiven(e.target.checked)}
            className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
            required
          />
          <label htmlFor="dataConsent" className="ml-2 text-sm text-gray-700">
            Customer has given consent for data collection and storage (UU PDP compliance) *
          </label>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Consent Method *</label>
          <select
            value={consentMethod}
            onChange={e => setConsentMethod(e.target.value as ConsentMethod)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            disabled={!dataConsentGiven}
          >
            <option value="verbal">Verbal</option>
            <option value="written">Written Form</option>
            <option value="digital">Digital Signature</option>
          </select>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex gap-3 pt-4">
        <button
          type="submit"
          disabled={loading || loadingProducts || orderItems.length === 0}
          className="flex-1 px-6 py-3 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed font-medium"
        >
          {loading ? 'Creating Order...' : 'Create Order'}
        </button>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={loading || loadingProducts}
            className="px-6 py-3 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 disabled:bg-gray-50 font-medium"
          >
            Cancel
          </button>
        )}
      </div>
    </form>
  );
};
