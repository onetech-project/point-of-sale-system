import React, { useState, useEffect } from 'react';
import { useTranslation } from '@/i18n/provider';
import auditService from '../../services/audit';
import type { AuditEvent, AuditQueryFilters } from '../../types/audit';

export const AuditLog: React.FC = () => {
  const { t } = useTranslation(['audit', 'common']);

  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalEvents, setTotalEvents] = useState(0);
  const [pageSize] = useState(100);

  // Filter states
  const [actionFilter, setActionFilter] = useState('');
  const [resourceTypeFilter, setResourceTypeFilter] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  // Expanded event for before/after values
  const [expandedId, setExpandedId] = useState<string | null>(null);

  useEffect(() => {
    fetchAuditEvents();
  }, [currentPage, actionFilter, resourceTypeFilter, startDate, endDate]);

  const fetchAuditEvents = async () => {
    try {
      setLoading(true);
      setError(null);

      const filters: AuditQueryFilters = {
        limit: pageSize,
        offset: (currentPage - 1) * pageSize,
      };

      if (actionFilter) filters.action = actionFilter;
      if (resourceTypeFilter) filters.resource_type = resourceTypeFilter;
      if (startDate) filters.start_time = new Date(startDate).toISOString();
      if (endDate) filters.end_time = new Date(endDate).toISOString();

      const response = await auditService.getTenantAuditEvents(filters);

      setEvents(response.events);
      setTotalEvents(response.pagination.total);
    } catch (err: any) {
      console.error('Failed to fetch audit events:', err);
      setError(t('audit.load_error') || 'Failed to load audit trail');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = () => {
    setCurrentPage(1);
    fetchAuditEvents();
  };

  const handleReset = () => {
    setActionFilter('');
    setResourceTypeFilter('');
    setStartDate('');
    setEndDate('');
    setCurrentPage(1);
  };

  const toggleExpanded = (eventId: string) => {
    setExpandedId(expandedId === eventId ? null : eventId);
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const getActionBadgeColor = (action: string) => {
    switch (action) {
      case 'CREATE':
        return 'bg-green-100 text-green-800';
      case 'READ':
        return 'bg-blue-100 text-blue-800';
      case 'UPDATE':
        return 'bg-yellow-100 text-yellow-800';
      case 'DELETE':
        return 'bg-red-100 text-red-800';
      case 'LOGIN':
        return 'bg-purple-100 text-purple-800';
      case 'LOGOUT':
        return 'bg-gray-100 text-gray-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const totalPages = Math.ceil(totalEvents / pageSize);

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h2 className="text-2xl font-bold mb-6">
        {t('audit.title') || 'Audit Trail'}
      </h2>

      {/* Filters */}
      <div className="mb-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {t('audit.filter.action') || 'Action'}
          </label>
          <select
            value={actionFilter}
            onChange={(e) => setActionFilter(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">{t('audit.filter.all_actions') || 'All Actions'}</option>
            <option value="CREATE">CREATE</option>
            <option value="READ">READ</option>
            <option value="UPDATE">UPDATE</option>
            <option value="DELETE">DELETE</option>
            <option value="LOGIN">LOGIN</option>
            <option value="LOGOUT">LOGOUT</option>
            <option value="GRANT">GRANT</option>
            <option value="REVOKE">REVOKE</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {t('audit.filter.resource_type') || 'Resource Type'}
          </label>
          <select
            value={resourceTypeFilter}
            onChange={(e) => setResourceTypeFilter(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">{t('audit.filter.all_resources') || 'All Resources'}</option>
            <option value="user">User</option>
            <option value="order">Order</option>
            <option value="product">Product</option>
            <option value="config">Configuration</option>
            <option value="session">Session</option>
            <option value="consent">Consent</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {t('audit.filter.start_date') || 'Start Date'}
          </label>
          <input
            type="datetime-local"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {t('audit.filter.end_date') || 'End Date'}
          </label>
          <input
            type="datetime-local"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>

      <div className="flex gap-2 mb-6">
        <button
          onClick={handleSearch}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          {t('common.search') || 'Search'}
        </button>
        <button
          onClick={handleReset}
          className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-gray-500"
        >
          {t('common.reset') || 'Reset'}
        </button>
      </div>

      {/* Loading state */}
      {loading && (
        <div className="text-center py-8">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <p className="mt-2 text-gray-600">{t('common.loading') || 'Loading...'}</p>
        </div>
      )}

      {/* Error state */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-md mb-4">
          {error}
        </div>
      )}

      {/* Events table */}
      {!loading && !error && (
        <>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('audit.table.timestamp') || 'Timestamp'}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('audit.table.actor') || 'Actor'}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('audit.table.action') || 'Action'}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('audit.table.resource') || 'Resource'}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('audit.table.ip_address') || 'IP Address'}
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t('audit.table.details') || 'Details'}
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {events.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-6 py-4 text-center text-gray-500">
                      {t('audit.no_events') || 'No audit events found'}
                    </td>
                  </tr>
                ) : (
                  events.map((event) => (
                    <React.Fragment key={event.event_id}>
                      <tr className="hover:bg-gray-50">
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          {formatTimestamp(event.timestamp)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          <div>
                            <span className="font-medium">{event.actor_type}</span>
                            {event.actor_email && (
                              <div className="text-xs text-gray-500">{event.actor_email}</div>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <span className={`px-2 py-1 text-xs font-semibold rounded-full ${getActionBadgeColor(event.action)}`}>
                            {event.action}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                          <div>
                            <span className="font-medium">{event.resource_type}</span>
                            <div className="text-xs text-gray-500">{event.resource_id.substring(0, 8)}...</div>
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {event.ip_address || '-'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                          <button
                            onClick={() => toggleExpanded(event.event_id)}
                            className="text-blue-600 hover:text-blue-800 focus:outline-none"
                          >
                            {expandedId === event.event_id ? (
                              t('audit.hide_details') || 'Hide'
                            ) : (
                              t('audit.show_details') || 'Show'
                            )}
                          </button>
                        </td>
                      </tr>
                      {expandedId === event.event_id && (
                        <tr>
                          <td colSpan={6} className="px-6 py-4 bg-gray-50">
                            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                              {event.before_value && Object.keys(event.before_value).length > 0 && (
                                <div>
                                  <h4 className="font-semibold text-gray-700 mb-2">
                                    {t('audit.before_value') || 'Before Value'}
                                  </h4>
                                  <pre className="bg-white p-3 rounded border border-gray-200 text-xs overflow-x-auto">
                                    {JSON.stringify(event.before_value, null, 2)}
                                  </pre>
                                </div>
                              )}
                              {event.after_value && Object.keys(event.after_value).length > 0 && (
                                <div>
                                  <h4 className="font-semibold text-gray-700 mb-2">
                                    {t('audit.after_value') || 'After Value'}
                                  </h4>
                                  <pre className="bg-white p-3 rounded border border-gray-200 text-xs overflow-x-auto">
                                    {JSON.stringify(event.after_value, null, 2)}
                                  </pre>
                                </div>
                              )}
                              {event.metadata && Object.keys(event.metadata).length > 0 && (
                                <div className={event.before_value || event.after_value ? 'md:col-span-2' : ''}>
                                  <h4 className="font-semibold text-gray-700 mb-2">
                                    {t('audit.metadata') || 'Metadata'}
                                  </h4>
                                  <pre className="bg-white p-3 rounded border border-gray-200 text-xs overflow-x-auto">
                                    {JSON.stringify(event.metadata, null, 2)}
                                  </pre>
                                </div>
                              )}
                              {event.user_agent && (
                                <div className="md:col-span-2">
                                  <h4 className="font-semibold text-gray-700 mb-2">
                                    {t('audit.user_agent') || 'User Agent'}
                                  </h4>
                                  <p className="text-sm text-gray-600">{event.user_agent}</p>
                                </div>
                              )}
                            </div>
                          </td>
                        </tr>
                      )}
                    </React.Fragment>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-6 flex items-center justify-between">
              <div className="text-sm text-gray-700">
                {t('common.showing') || 'Showing'}{' '}
                <span className="font-medium">{(currentPage - 1) * pageSize + 1}</span> -{' '}
                <span className="font-medium">
                  {Math.min(currentPage * pageSize, totalEvents)}
                </span>{' '}
                {t('common.of') || 'of'}{' '}
                <span className="font-medium">{totalEvents}</span>{' '}
                {t('audit.events') || 'events'}
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => setCurrentPage(currentPage - 1)}
                  disabled={currentPage === 1}
                  className="px-4 py-2 bg-white border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t('common.previous') || 'Previous'}
                </button>
                <button
                  onClick={() => setCurrentPage(currentPage + 1)}
                  disabled={currentPage === totalPages}
                  className="px-4 py-2 bg-white border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {t('common.next') || 'Next'}
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};
