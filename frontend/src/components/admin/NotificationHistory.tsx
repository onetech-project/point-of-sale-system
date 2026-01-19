import React, { useState, useEffect } from 'react';
import { useTranslation } from '@/i18n/provider';
import notificationService from '../../services/notification';
import type {
  NotificationHistoryItem,
  NotificationHistoryFilters,
} from '../../types/notification';

export const NotificationHistory: React.FC = () => {
  const { t } = useTranslation(['notifications', 'common']);

  const [notifications, setNotifications] = useState<NotificationHistoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [totalItems, setTotalItems] = useState(0);
  const [pageSize] = useState(20);

  // Filter states
  const [orderReference, setOrderReference] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [typeFilter, setTypeFilter] = useState<string>('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  // Resend states
  const [resendingId, setResendingId] = useState<string | null>(null);
  const [resendSuccess, setResendSuccess] = useState<string | null>(null);
  const [resendError, setResendError] = useState<string | null>(null);

  // Expanded notification for error details
  const [expandedId, setExpandedId] = useState<string | null>(null);

  useEffect(() => {
    fetchHistory();
  }, [currentPage, statusFilter, typeFilter, startDate, endDate]);

  const fetchHistory = async () => {
    try {
      setLoading(true);
      setError(null);

      const filters: NotificationHistoryFilters = {
        page: currentPage,
        page_size: pageSize,
      };

      if (orderReference) filters.order_reference = orderReference;
      if (statusFilter) filters.status = statusFilter as any;
      if (typeFilter) filters.type = typeFilter as any;
      if (startDate) filters.start_date = new Date(startDate).toISOString();
      if (endDate) filters.end_date = new Date(endDate).toISOString();

      const response = await notificationService.getNotificationHistory(filters);

      setNotifications(response.notifications);
      setTotalPages(response.pagination.total_pages);
      setTotalItems(response.pagination.total_items);
    } catch (err: any) {
      console.error('Failed to fetch notification history:', err);
      setError(t('notifications.history.load_error') || 'Failed to load notification history');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = () => {
    setCurrentPage(1);
    fetchHistory();
  };

  const handleResend = async (notificationId: string) => {
    try {
      setResendingId(notificationId);
      setResendError(null);
      setResendSuccess(null);

      await notificationService.resendNotification(notificationId);

      setResendSuccess(t('notifications.history.resend_success') || 'Notification resent successfully');

      // Refresh history
      setTimeout(() => {
        fetchHistory();
        setResendSuccess(null);
      }, 2000);
    } catch (err: any) {
      console.error('Failed to resend notification:', err);

      if (err.response?.status === 429) {
        setResendError(
          t('notifications.history.resend_rate_limit') ||
          'Too many requests. Please try again later.'
        );
      } else if (err.response?.status === 409) {
        setResendError(
          t('notifications.history.resend_already_sent') ||
          'This notification has already been sent successfully.'
        );
      } else {
        setResendError(
          t('notifications.history.resend_failed') || 'Failed to resend notification. Please try again.'
        );
      }
    } finally {
      setResendingId(null);
    }
  };

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case 'sent':
        return 'bg-green-100 text-green-800';
      case 'failed':
        return 'bg-red-100 text-red-800';
      case 'pending':
        return 'bg-yellow-100 text-yellow-800';
      case 'cancelled':
        return 'bg-gray-100 text-gray-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  if (loading && notifications.length === 0) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">{t('common.loading', { ns: 'common' }) || 'Loading...'}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto p-6">
      <div className="bg-white rounded-lg shadow-md">
        <div className="p-6 border-b border-gray-200">
          <h2 className="text-2xl font-bold text-gray-900">
            {t('notifications.history.title') || 'Notification History'}
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            {t('notifications.history.description') ||
              'View and manage email notification history for your restaurant.'}
          </p>
        </div>

        {/* Filters */}
        <div className="p-6 border-b border-gray-200 bg-gray-50">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {/* Order Reference Search */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                {t('notifications.history.filter_order_ref') || 'Order Reference'}
              </label>
              <input
                type="text"
                value={orderReference}
                onChange={(e) => setOrderReference(e.target.value)}
                placeholder="GO-XXXX"
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            {/* Status Filter */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                {t('notifications.history.filter_status') || 'Status'}
              </label>
              <select
                name="status"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">{t('notifications.history.filter_status') || 'All Statuses'}</option>
                <option value="sent">{t('notifications.history.status.sent') || 'Sent'}</option>
                <option value="failed">{t('notifications.history.status.failed') || 'Failed'}</option>
                <option value="pending">{t('notifications.history.status.pending') || 'Pending'}</option>
                <option value="cancelled">{t('notifications.history.status.cancelled') || 'Cancelled'}</option>
              </select>
            </div>

            {/* Type Filter */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                {t('notifications.history.filter_type') || 'Type'}
              </label>
              <select
                value={typeFilter}
                onChange={(e) => setTypeFilter(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">{t('notifications.history.filter_type') || 'All Types'}</option>
                <option value="order_staff">{t('notifications.history.type.staff') || 'Staff Notification'}</option>
                <option value="order_customer">{t('notifications.history.type.customer') || 'Customer Receipt'}</option>
              </select>
            </div>

            {/* Start Date */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                {t('notifications.history.filter_start_date') || 'Start Date'}
              </label>
              <input
                type="date"
                name="start_date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            {/* End Date */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                {t('notifications.history.filter_end_date') || 'End Date'}
              </label>
              <input
                type="date"
                name="end_date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            {/* Search Button */}
            <div className="flex items-end">
              <button
                onClick={handleSearch}
                className="w-full px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {t('notifications.history.search') || 'Search'}
              </button>
            </div>
          </div>
        </div>

        {/* Error/Success Messages */}
        {error && (
          <div className="mx-6 mt-6 p-4 bg-red-50 border border-red-200 rounded-md">
            <p className="text-sm text-red-800">{error}</p>
          </div>
        )}

        {resendSuccess && (
          <div className="mx-6 mt-6 p-4 bg-green-50 border border-green-200 rounded-md">
            <p className="text-sm text-green-800">{resendSuccess}</p>
          </div>
        )}

        {resendError && (
          <div className="mx-6 mt-6 p-4 bg-red-50 border border-red-200 rounded-md">
            <p className="text-sm text-red-800">{resendError}</p>
          </div>
        )}

        {/* Notification List */}
        <div className="p-6" data-testid="notification-list">
          {notifications.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-gray-500">{t('notifications.history.empty') || 'No notifications found'}</p>
            </div>
          ) : (
            <div className="space-y-4">
              {notifications.map((notification) => (
                <div
                  key={notification.id}
                  data-testid="notification-item"
                  className="border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center space-x-3">
                        <span
                          data-testid="status-badge"
                          className={`px-3 py-1 rounded-full text-xs font-semibold ${getStatusBadgeClass(
                            notification.status
                          )}`}
                        >
                          {notification.status}
                        </span>
                        {notification.order_reference && (
                          <span data-testid="order-reference" className="text-sm font-medium text-gray-700">
                            {notification.order_reference}
                          </span>
                        )}
                      </div>

                      <h3 className="mt-2 text-lg font-medium text-gray-900">{notification.subject}</h3>
                      <p className="mt-1 text-sm text-gray-600">
                        {t('notifications.history.recipient') || 'To'}: {notification.recipient}
                      </p>
                      <p className="mt-1 text-xs text-gray-500">
                        {t('notifications.history.created_at') || 'Created'}: {formatDate(notification.created_at)}
                      </p>
                      {notification.sent_at && (
                        <p className="text-xs text-gray-500">
                          {t('notifications.history.sent_at') || 'Sent'}: {formatDate(notification.sent_at)}
                        </p>
                      )}
                      {notification.failed_at && (
                        <p className="text-xs text-red-600">
                          {t('notifications.history.failed_at') || 'Failed'}: {formatDate(notification.failed_at)}
                        </p>
                      )}

                      {/* Retry count */}
                      {notification.retry_count > 0 && (
                        <p className="text-xs text-gray-500" data-testid="retry-count">
                          {t('notifications.history.retry_count') || 'Retries'}: {notification.retry_count}
                        </p>
                      )}

                      {/* Error details (expandable) */}
                      {notification.status === 'failed' && notification.error_msg && (
                        <div className="mt-2">
                          <button
                            data-testid="expand-details"
                            onClick={() => setExpandedId(expandedId === notification.id ? null : notification.id)}
                            className="text-sm text-blue-600 hover:text-blue-800 focus:outline-none"
                          >
                            {expandedId === notification.id
                              ? t('notifications.history.hide_error') || 'Hide error details'
                              : t('notifications.history.show_error') || 'Show error details'}
                          </button>
                          {expandedId === notification.id && (
                            <div data-testid="error-message" className="mt-2 p-3 bg-red-50 rounded-md">
                              <p className="text-sm text-red-800">{notification.error_msg}</p>
                            </div>
                          )}
                        </div>
                      )}
                    </div>

                    {/* Resend button */}
                    {notification.status === 'failed' && notification.retry_count < 3 && (
                      <button
                        onClick={() => handleResend(notification.id)}
                        disabled={resendingId === notification.id}
                        className="ml-4 px-4 py-2 bg-orange-600 text-white rounded-md hover:bg-orange-700 focus:outline-none focus:ring-2 focus:ring-orange-500 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        {resendingId === notification.id
                          ? t('notifications.history.resending') || 'Resending...'
                          : t('notifications.history.resend') || 'Resend'}
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="p-6 border-t border-gray-200 flex items-center justify-between" data-testid="pagination">
            <div className="text-sm text-gray-600">
              {t('notifications.history.showing') || 'Showing'}{' '}
              <span className="font-medium">
                {(currentPage - 1) * pageSize + 1}-{Math.min(currentPage * pageSize, totalItems)}
              </span>{' '}
              {t('notifications.history.of') || 'of'} <span className="font-medium">{totalItems}</span>{' '}
              {t('notifications.history.results') || 'results'}
            </div>

            <div className="flex items-center space-x-2">
              <button
                onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                disabled={currentPage === 1}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {t('common.previous', { ns: 'common' }) || 'Previous'}
              </button>

              <span className="text-sm text-gray-600" data-testid="current-page">
                {t('notifications.history.page') || 'Page'} {currentPage} {t('common.of', { ns: 'common' }) || 'of'} {totalPages}
              </span>

              <button
                onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
                disabled={currentPage === totalPages}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {t('common.next', { ns: 'common' }) || 'Next'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
