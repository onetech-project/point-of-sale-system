import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { order, Order, OrderWithDetails } from '../../services/order';
import { OrderItem, OrderNote } from '../../types/cart';
import { formatCurrency, renderTextWithLinks } from '../../utils/text';

interface OrderManagementProps {
  // Removed tenantId - API Gateway extracts it from session
  authToken?: string;
}

export const OrderManagement: React.FC<OrderManagementProps> = ({
  authToken,
}) => {
  const { t } = useTranslation();

  const [ordersWithDetails, setOrdersWithDetails] = useState<OrderWithDetails[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedOrderDetails, setSelectedOrderDetails] = useState<OrderWithDetails | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [page, setPage] = useState(1);
  const [totalOrders, setTotalOrders] = useState(0);
  const [updating, setUpdating] = useState(false);
  const [note, setNote] = useState('');
  const [showNoteDialog, setShowNoteDialog] = useState(false);
  const [showStatusDialog, setShowStatusDialog] = useState(false);
  const [newStatus, setNewStatus] = useState('');

  const ITEMS_PER_PAGE = 20;

  useEffect(() => {
    fetchOrders();
  }, [statusFilter, page]);

  const fetchOrders = async () => {
    try {
      setLoading(true);
      setError(null);

      const filters: any = {
        page,
        limit: ITEMS_PER_PAGE,
      };

      if (statusFilter !== 'all') {
        filters.status = statusFilter;
      }

      const response = await order.listOrders(filters);

      setOrdersWithDetails(response.orders || []);
      // Backend doesn't return total, use count from current page
      // If we get fewer items than limit, we're on the last page
      setTotalOrders(response.pagination?.count || (response.orders?.length || 0));
    } catch (err: any) {
      console.error('Failed to fetch orders:', err);
      setError('Failed to load orders. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleOrderClick = (orderDetails: OrderWithDetails) => {
    setSelectedOrderDetails(orderDetails);
  };

  const handleCloseDetail = () => {
    setSelectedOrderDetails(null);
    setNote('');
    setShowNoteDialog(false);
    setShowStatusDialog(false);
  };

  // T097: Status update functionality with confirmation dialog
  const handleStatusUpdateRequest = (status: string) => {
    setNewStatus(status);
    setShowStatusDialog(true);
  };

  const confirmStatusUpdate = async () => {
    if (!selectedOrderDetails) return;

    try {
      setUpdating(true);
      await order.updateOrderStatus(
        selectedOrderDetails.order.id,
        newStatus
      );

      // Update local state
      setOrdersWithDetails(
        ordersWithDetails.map((od) =>
          od.order.id === selectedOrderDetails.order.id
            ? { ...od, order: { ...od.order, status: newStatus as Order['status'] } }
            : od
        )
      );
      setSelectedOrderDetails({
        ...selectedOrderDetails,
        order: { ...selectedOrderDetails.order, status: newStatus as Order['status'] }
      });
      setShowStatusDialog(false);
    } catch (err: any) {
      console.error('Failed to update order status:', err);
      alert('Failed to update order status. Please try again.');
    } finally {
      setUpdating(false);
    }
  };

  // T098: Notes/comments functionality
  const handleAddNote = async () => {
    if (!selectedOrderDetails || !note.trim()) return;

    try {
      setUpdating(true);
      await order.addOrderNote(selectedOrderDetails.order.id, note.trim());
      alert('Note added successfully!');
      setNote('');
      setShowNoteDialog(false);
      // Refresh order details
      await fetchOrders();
    } catch (err: any) {
      console.error('Failed to add note:', err);
      alert('Failed to add note. Please try again.');
    } finally {
      setUpdating(false);
    }
  };

  const formatDate = (dateString: string): string => {
    return new Date(dateString).toLocaleString('id-ID', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'PAID':
        return 'bg-green-100 text-green-800';
      case 'PENDING':
        return 'bg-yellow-100 text-yellow-800';
      case 'COMPLETE':
        return 'bg-blue-100 text-blue-800';
      case 'CANCELLED':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const getDeliveryTypeLabel = (type: string): string => {
    switch (type) {
      case 'delivery':
        return 'Delivery';
      case 'pickup':
        return 'Pickup';
      case 'dine_in':
        return 'Dine In';
      default:
        return type;
    }
  };

  if (loading && ordersWithDetails.length === 0) {
    return (
      <div className="flex justify-center items-center p-12">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header with Filters */}
      <div className="bg-white rounded-lg shadow p-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
          <h2 className="text-2xl font-bold text-gray-900">Order Management</h2>
          <button
            onClick={fetchOrders}
            disabled={loading}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400"
          >
            {loading ? 'Refreshing...' : 'Refresh'}
          </button>
        </div>

        {/* Status Filter */}
        <div className="flex items-center gap-2">
          <label className="text-sm font-medium text-gray-700">Filter:</label>
          <select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">All Orders</option>
            <option value="PENDING">Pending Payment</option>
            <option value="PAID">Paid</option>
            <option value="COMPLETE">Completed</option>
            <option value="CANCELLED">Cancelled</option>
          </select>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-800">{error}</p>
        </div>
      )}

      {/* Orders List */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        {ordersWithDetails.length === 0 ? (
          <div className="p-12 text-center">
            <p className="text-gray-500">No orders found</p>
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Order Reference
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Customer
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Type
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Total
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Created
                    </th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {ordersWithDetails.map((orderDetails) => (
                    <tr
                      key={orderDetails.order.id}
                      className="hover:bg-gray-50 cursor-pointer"
                      onClick={() => handleOrderClick(orderDetails)}
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="font-mono text-sm font-medium text-gray-900">
                          {orderDetails.order.order_reference}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="text-sm text-gray-900">
                          {orderDetails.order.customer_name}
                        </div>
                        <div className="text-sm text-gray-500">
                          {orderDetails.order.customer_phone}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="text-sm text-gray-900">
                          {getDeliveryTypeLabel(orderDetails.order.delivery_type)}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="text-sm font-medium text-gray-900">
                          {formatCurrency(orderDetails.order.total_amount)}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span
                          className={`px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full ${getStatusColor(
                            orderDetails.order.status
                          )}`}
                        >
                          {orderDetails.order.status}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDate(orderDetails.order.created_at)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleOrderClick(orderDetails);
                          }}
                          className="text-blue-600 hover:text-blue-900"
                        >
                          View Details
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            <div className="bg-gray-50 px-6 py-3 flex items-center justify-between border-t">
              <div className="text-sm text-gray-700">
                Showing {ordersWithDetails.length > 0 ? (page - 1) * ITEMS_PER_PAGE + 1 : 0} to{' '}
                {(page - 1) * ITEMS_PER_PAGE + ordersWithDetails.length} of orders
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => setPage(page - 1)}
                  disabled={page === 1}
                  className="px-4 py-2 border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
                >
                  Previous
                </button>
                <button
                  onClick={() => setPage(page + 1)}
                  disabled={ordersWithDetails.length < ITEMS_PER_PAGE}
                  className="px-4 py-2 border rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
                >
                  Next
                </button>
              </div>
            </div>
          </>
        )}
      </div>

      {/* Order Detail Modal */}
      {selectedOrderDetails && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-3xl w-full max-h-[90vh] overflow-y-auto">
            {/* Modal Header */}
            <div className="sticky top-0 bg-white border-b px-6 py-4 flex justify-between items-center">
              <h3 className="text-xl font-bold">
                Order {selectedOrderDetails.order.order_reference}
              </h3>
              <button
                onClick={handleCloseDetail}
                className="text-gray-400 hover:text-gray-600"
              >
                <svg
                  className="w-6 h-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>

            {/* Modal Content */}
            <div className="p-6 space-y-6">
              {/* Customer Info */}
              <div>
                <h4 className="font-semibold mb-2">Customer Information</h4>
                <div className="bg-gray-50 rounded p-4 space-y-2">
                  <div className="flex justify-between">
                    <span className="text-gray-600">Name:</span>
                    <span className="font-medium">
                      {selectedOrderDetails.order.customer_name}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Phone:</span>
                    <span className="font-medium">
                      {selectedOrderDetails.order.customer_phone}
                    </span>
                  </div>
                  {selectedOrderDetails.order.delivery_address && (
                    <div className="flex justify-between">
                      <span className="text-gray-600">Address:</span>
                      <span className="font-medium text-right max-w-xs">
                        {selectedOrderDetails.order.delivery_address}
                      </span>
                    </div>
                  )}
                </div>
              </div>

              {/* Order Items */}
              {selectedOrderDetails.items && selectedOrderDetails.items.length > 0 && (
                <div>
                  <h4 className="font-semibold mb-2">Order Items</h4>
                  <div className="border rounded overflow-hidden">
                    {selectedOrderDetails.items.map((item) => (
                      <div
                        key={item.id}
                        className="flex justify-between p-4 border-b last:border-b-0"
                      >
                        <div>
                          <p className="font-medium">{item.product_name}</p>
                          <p className="text-sm text-gray-500">
                            {item.quantity} × {formatCurrency(item.unit_price)}
                          </p>
                        </div>
                        <p className="font-medium">
                          {formatCurrency(item.total_price)}
                        </p>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Latest Note */}
              {selectedOrderDetails.latest_note && (
                <div>
                  <h4 className="font-semibold mb-2">Latest Note</h4>
                  <div className="bg-yellow-50 border border-yellow-200 rounded p-4">
                    <p className="text-sm text-gray-700">{selectedOrderDetails.latest_note.note}</p>
                    <div className="mt-2 text-xs text-gray-500">
                      By {selectedOrderDetails.latest_note.created_by_name || 'Admin'} • {formatDate(selectedOrderDetails.latest_note.created_at)}
                    </div>
                  </div>
                </div>
              )}

              {/* Order Summary */}
              <div>
                <h4 className="font-semibold mb-2">Order Summary</h4>
                <div className="bg-gray-50 rounded p-4 space-y-2">
                  <div className="flex justify-between">
                    <span>Subtotal:</span>
                    <span>{formatCurrency(selectedOrderDetails.order.subtotal_amount)}</span>
                  </div>
                  {selectedOrderDetails.order.delivery_fee > 0 && (
                    <div className="flex justify-between">
                      <span>Delivery Fee:</span>
                      <span>{formatCurrency(selectedOrderDetails.order.delivery_fee)}</span>
                    </div>
                  )}
                  <div className="flex justify-between font-bold text-lg border-t pt-2">
                    <span>Total:</span>
                    <span>{formatCurrency(selectedOrderDetails.order.total_amount)}</span>
                  </div>
                </div>
              </div>

              {/* Status Update Actions - T097 */}
              <div>
                <h4 className="font-semibold mb-2">Update Status</h4>
                <div className="flex flex-wrap gap-2">
                  {selectedOrderDetails.order.status === 'PAID' && (
                    <button
                      onClick={() => handleStatusUpdateRequest('COMPLETE')}
                      disabled={updating}
                      className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400"
                    >
                      Mark as Complete
                    </button>
                  )}
                  {(selectedOrderDetails.order.status === 'PENDING' ||
                    selectedOrderDetails.order.status === 'PAID') && (
                      <button
                        onClick={() => handleStatusUpdateRequest('CANCELLED')}
                        disabled={updating}
                        className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:bg-gray-400"
                      >
                        Cancel Order
                      </button>
                    )}
                </div>
              </div>

              {/* Add Note Button - T098 */}
              <div>
                <button
                  onClick={() => setShowNoteDialog(true)}
                  className="px-4 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700"
                >
                  Add Note / Courier Info
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Status Update Confirmation Dialog - T097 */}
      {showStatusDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-md w-full p-6">
            <h3 className="text-lg font-bold mb-4">Confirm Status Update</h3>
            <p className="text-gray-600 mb-6">
              Are you sure you want to update the order status to{' '}
              <strong>{newStatus}</strong>?
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowStatusDialog(false)}
                className="px-4 py-2 border rounded-lg hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={confirmStatusUpdate}
                disabled={updating}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400"
              >
                {updating ? 'Updating...' : 'Confirm'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Note Dialog - T098 */}
      {showNoteDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-md w-full p-6">
            <h3 className="text-lg font-bold mb-4">Add Note</h3>
            <textarea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              rows={4}
              className="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 resize-none"
              placeholder="Enter courier tracking info or other notes..."
            />
            <div className="flex justify-end gap-3 mt-4">
              <button
                onClick={() => {
                  setShowNoteDialog(false);
                  setNote('');
                }}
                className="px-4 py-2 border rounded-lg hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleAddNote}
                disabled={updating || !note.trim()}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400"
              >
                {updating ? 'Adding...' : 'Add Note'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default OrderManagement;
