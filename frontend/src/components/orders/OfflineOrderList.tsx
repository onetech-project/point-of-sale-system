'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import offlineOrderService, { OfflineOrderDocumentType } from '../../services/offlineOrders';
import ActionMenu from '../ui/ActionMenu';
import { formatCurrency } from '../../utils/format';
import { documentErrorMessage } from '../../utils/documentErrors';
import { OfflineOrder, OrderStatus, ListOfflineOrdersFilters } from '../../types/offlineOrder';

interface OfflineOrderListProps {
  initialFilters?: ListOfflineOrdersFilters;
}

export const OfflineOrderList: React.FC<OfflineOrderListProps> = ({ initialFilters }) => {
  const router = useRouter();
  const [orders, setOrders] = useState<OfflineOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [totalCount, setTotalCount] = useState(0);
  const [selectedOrderIds, setSelectedOrderIds] = useState<Set<string>>(new Set());
  const [batchDocumentType, setBatchDocumentType] = useState<OfflineOrderDocumentType>('invoice');
  const [documentAction, setDocumentAction] = useState<string | null>(null);

  // Filters
  const [statusFilter, setStatusFilter] = useState<OrderStatus | 'ALL'>('ALL');
  const [searchQuery, setSearchQuery] = useState('');
  const [page, setPage] = useState(1);
  const pageSize = 20;

  useEffect(() => {
    fetchOrders();
  }, [statusFilter, page]);

  const fetchOrders = async () => {
    try {
      setLoading(true);
      setError(null);

      const filters: ListOfflineOrdersFilters = {
        limit: pageSize,
        offset: (page - 1) * pageSize,
      };

      if (statusFilter !== 'ALL') {
        filters.status = statusFilter;
      }

      if (searchQuery.trim()) {
        filters.search = searchQuery.trim();
      }

      const response = await offlineOrderService.listOfflineOrders(filters);
      setOrders(response?.orders || []);
      setTotalCount(response?.total_count || 0);
      setSelectedOrderIds(new Set());
    } catch (err: any) {
      console.error('Failed to fetch offline orders:', err);
      setError('Failed to load offline orders. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = () => {
    setPage(1); // Reset to first page
    fetchOrders();
  };

  const handleOrderClick = (orderId: string) => {
    router.push(`/orders/offline-orders/${orderId}`);
  };

  const handleCreateNew = () => {
    router.push('/orders/offline-orders/new');
  };

  const isValidCustomerEmail = (email?: string): boolean => {
    return !!email && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  };

  const canDownloadDocument = (order: OfflineOrder, documentType: OfflineOrderDocumentType): boolean => {
    if (documentType === 'invoice') {
      return order.status !== 'CANCELLED';
    }
    return order.status === 'PAID' || order.status === 'COMPLETE';
  };

  const canResendDocument = (order: OfflineOrder, documentType: OfflineOrderDocumentType): boolean => {
    return canDownloadDocument(order, documentType) && isValidCustomerEmail(order.customer_email);
  };

  const selectedOrders = orders.filter(order => selectedOrderIds.has(order.id));
  const selectedBatchIsValid =
    selectedOrders.length > 0 &&
    selectedOrders.every(order => canDownloadDocument(order, batchDocumentType));
  const allCurrentPageSelected =
    orders.length > 0 && orders.every(order => selectedOrderIds.has(order.id));

  const toggleSelectAllCurrentPage = () => {
    setSelectedOrderIds(prev => {
      const next = new Set(prev);
      if (allCurrentPageSelected) {
        orders.forEach(order => next.delete(order.id));
      } else {
        orders.forEach(order => next.add(order.id));
      }
      return next;
    });
  };

  const toggleOrderSelection = (orderId: string) => {
    setSelectedOrderIds(prev => {
      const next = new Set(prev);
      if (next.has(orderId)) {
        next.delete(orderId);
      } else {
        next.add(orderId);
      }
      return next;
    });
  };

  const downloadBlob = (blob: Blob, filename: string) => {
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  };

  const handleDownloadDocument = async (order: OfflineOrder, documentType: OfflineOrderDocumentType) => {
    try {
      setDocumentAction(`${order.id}-${documentType}-download`);
      const document = await offlineOrderService.downloadDocument(order.id, documentType);
      downloadBlob(document.blob, document.filename);
    } catch (err) {
      console.error('Failed to download offline order document:', err);
      setError(await documentErrorMessage(err, 'Failed to download document. Please try again.'));
    } finally {
      setDocumentAction(null);
    }
  };

  const handleResendDocument = async (order: OfflineOrder, documentType: OfflineOrderDocumentType) => {
    try {
      setDocumentAction(`${order.id}-${documentType}-resend`);
      await offlineOrderService.resendDocument(order.id, documentType);
      setError(null);
      alert(`${documentType === 'invoice' ? 'Invoice' : 'Receipt'} resend queued.`);
    } catch (err) {
      console.error('Failed to resend offline order document:', err);
      setError(await documentErrorMessage(err, 'Failed to resend document. Check the customer email and try again.'));
    } finally {
      setDocumentAction(null);
    }
  };

  const handleBatchDownload = async () => {
    if (!selectedBatchIsValid) return;

    try {
      setDocumentAction('batch-download');
      const document = await offlineOrderService.batchDownloadDocuments(
        selectedOrders.map(order => order.id),
        batchDocumentType
      );
      downloadBlob(document.blob, document.filename);
      setError(null);
    } catch (err) {
      console.error('Failed to batch download offline order documents:', err);
      setError(await documentErrorMessage(err, 'Failed to download selected documents. Check that every selected order is eligible.'));
    } finally {
      setDocumentAction(null);
    }
  };

  const getStatusColor = (status: OrderStatus): string => {
    switch (status) {
      case 'PENDING':
        return 'bg-yellow-100 text-yellow-800';
      case 'PAID':
        return 'bg-green-100 text-green-800';
      case 'COMPLETE':
        return 'bg-blue-100 text-blue-800';
      case 'CANCELLED':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const getStatusBadge = (status: OrderStatus) => {
    const colorClass = getStatusColor(status);
    return (
      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${colorClass}`}>{status}</span>
    );
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('id-ID', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const totalPages = Math.ceil(totalCount / pageSize);

  if (loading && (!orders || orders.length === 0)) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="text-gray-500">Loading offline orders...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold text-gray-900">Offline Orders</h2>
        <button
          onClick={handleCreateNew}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium"
        >
          + Create New Order
        </button>
      </div>

      {/* Filters */}
      <div className="bg-white p-4 rounded-lg shadow space-y-4">
        <div className="flex gap-4 flex-wrap">
          {/* Status Filter */}
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
            <select
              value={statusFilter}
              onChange={e => {
                setStatusFilter(e.target.value as any);
                setPage(1);
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="ALL">All Statuses</option>
              <option value="PENDING">Pending</option>
              <option value="PAID">Paid</option>
              <option value="COMPLETE">Complete</option>
              <option value="CANCELLED">Cancelled</option>
            </select>
          </div>

          {/* Search */}
          <div className="flex-1 min-w-[300px]">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Search by Order Reference
            </label>
            <div className="flex gap-2">
              <input
                type="text"
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                onKeyPress={e => e.key === 'Enter' && handleSearch()}
                placeholder="GO-XXXXXX"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              />
              <button
                onClick={handleSearch}
                className="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700"
              >
                Search
              </button>
            </div>
          </div>
        </div>

        <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3 border-t pt-4">
          <div className="text-sm text-gray-700">
            {selectedOrders.length} selected
            {selectedOrders.length > 0 && !selectedBatchIsValid && (
              <span className="ml-2 text-red-600">Selection contains ineligible orders.</span>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <select
              value={batchDocumentType}
              onChange={e => setBatchDocumentType(e.target.value as OfflineOrderDocumentType)}
              className="px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="invoice">Invoices</option>
              <option value="receipt">Receipts</option>
            </select>
            <button
              onClick={handleBatchDownload}
              disabled={!selectedBatchIsValid || documentAction === 'batch-download'}
              className="px-4 py-2 bg-gray-900 text-white rounded-md hover:bg-gray-800 disabled:bg-gray-400 disabled:cursor-not-allowed"
            >
              {documentAction === 'batch-download' ? 'Preparing...' : 'Download ZIP'}
            </button>
            {selectedOrders.length > 0 && (
              <button
                onClick={() => setSelectedOrderIds(new Set())}
                className="px-4 py-2 border rounded-md hover:bg-gray-50"
              >
                Clear
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {/* Orders List */}
      {!orders || orders.length === 0 ? (
        <div className="bg-white rounded-lg shadow p-8 text-center">
          <p className="text-gray-500">No offline orders found.</p>
          <button
            onClick={handleCreateNew}
            className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
          >
            Create Your First Offline Order
          </button>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left">
                    <input
                      type="checkbox"
                      checked={allCurrentPageSelected}
                      onChange={toggleSelectAllCurrentPage}
                      className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                      aria-label="Select all offline orders on this page"
                    />
                  </th>
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
                    Amount
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
                {orders.map(order => (
                  <tr
                    key={order.id}
                    className="hover:bg-gray-50 cursor-pointer"
                    onClick={() => handleOrderClick(order.id)}
                  >
                    <td className="px-6 py-4 whitespace-nowrap">
                      <input
                        type="checkbox"
                        checked={selectedOrderIds.has(order.id)}
                        onChange={() => toggleOrderSelection(order.id)}
                        onClick={e => e.stopPropagation()}
                        className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                        aria-label={`Select order ${order.order_reference}`}
                      />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">
                        {order.order_reference}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">{order.customer_name}</div>
                      <div className="text-sm text-gray-500">{order.customer_phone}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900 capitalize">
                        {order.delivery_type.replace('_', ' ')}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">
                        {formatCurrency(order.total_amount)}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">{getStatusBadge(order.status)}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {formatDate(order.created_at)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-xs font-medium">
                      <ActionMenu
                        ariaLabel={`Open actions for order ${order.order_reference}`}
                        items={[
                          {
                            label: 'View Details',
                            onSelect: () => handleOrderClick(order.id),
                          },
                          {
                            label: 'Download Invoice',
                            onSelect: () => handleDownloadDocument(order, 'invoice'),
                            disabled: !canDownloadDocument(order, 'invoice'),
                          },
                          {
                            label: 'Download Receipt',
                            onSelect: () => handleDownloadDocument(order, 'receipt'),
                            disabled: !canDownloadDocument(order, 'receipt'),
                          },
                          {
                            label: 'Resend Invoice',
                            onSelect: () => handleResendDocument(order, 'invoice'),
                            disabled: !canResendDocument(order, 'invoice'),
                          },
                          {
                            label: 'Resend Receipt',
                            onSelect: () => handleResendDocument(order, 'receipt'),
                            disabled: !canResendDocument(order, 'receipt'),
                          },
                        ]}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="bg-white px-4 py-3 flex items-center justify-between border-t border-gray-200 sm:px-6">
              <div className="flex-1 flex justify-between sm:hidden">
                <button
                  onClick={() => setPage(Math.max(1, page - 1))}
                  disabled={page === 1}
                  className="relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:bg-gray-100 disabled:cursor-not-allowed"
                >
                  Previous
                </button>
                <button
                  onClick={() => setPage(Math.min(totalPages, page + 1))}
                  disabled={page === totalPages}
                  className="ml-3 relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 disabled:bg-gray-100 disabled:cursor-not-allowed"
                >
                  Next
                </button>
              </div>
              <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
                <div>
                  <p className="text-sm text-gray-700">
                    Showing <span className="font-medium">{(page - 1) * pageSize + 1}</span> to{' '}
                    <span className="font-medium">{Math.min(page * pageSize, totalCount)}</span> of{' '}
                    <span className="font-medium">{totalCount}</span> results
                  </p>
                </div>
                <div>
                  <nav className="relative z-0 inline-flex rounded-md shadow-sm -space-x-px">
                    <button
                      onClick={() => setPage(Math.max(1, page - 1))}
                      disabled={page === 1}
                      className="relative inline-flex items-center px-2 py-2 rounded-l-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:bg-gray-100 disabled:cursor-not-allowed"
                    >
                      Previous
                    </button>
                    <span className="relative inline-flex items-center px-4 py-2 border border-gray-300 bg-white text-sm font-medium text-gray-700">
                      Page {page} of {totalPages}
                    </span>
                    <button
                      onClick={() => setPage(Math.min(totalPages, page + 1))}
                      disabled={page === totalPages}
                      className="relative inline-flex items-center px-2 py-2 rounded-r-md border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:bg-gray-100 disabled:cursor-not-allowed"
                    >
                      Next
                    </button>
                  </nav>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
