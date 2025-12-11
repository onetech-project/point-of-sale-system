import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useTranslation } from '@/i18n/provider';
import notificationService from '../../services/notification';
import type {
  UserNotificationPreference,
  NotificationConfig,
} from '../../types/notification';

interface NotificationSettingsProps {
  authToken?: string;
}

export const NotificationSettings: React.FC<NotificationSettingsProps> = () => {
  const { t } = useTranslation(['notifications', 'common']);
  const router = useRouter();

  const [users, setUsers] = useState<UserNotificationPreference[]>([]);
  const [config, setConfig] = useState<NotificationConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [updating, setUpdating] = useState(false);
  const [testEmailDialogOpen, setTestEmailDialogOpen] = useState(false);
  const [testEmail, setTestEmail] = useState('');
  const [testNotificationType, setTestNotificationType] = useState<'staff_order_notification' | 'customer_receipt'>(
    'staff_order_notification'
  );
  const [testEmailSending, setTestEmailSending] = useState(false);
  const [testEmailSuccess, setTestEmailSuccess] = useState<string | null>(null);
  const [testEmailError, setTestEmailError] = useState<string | null>(null);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const [usersData, configData] = await Promise.all([
        notificationService.getUserNotificationPreferences(),
        notificationService.getNotificationConfig(),
      ]);

      setUsers(usersData);
      setConfig(configData);
    } catch (err: any) {
      console.error('Failed to fetch notification settings:', err);
      setError(t('notifications.settings.load_error') || 'Failed to load notification settings');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleUserNotification = async (userId: string, currentValue: boolean) => {
    try {
      setUpdating(true);
      setError(null);

      await notificationService.updateUserNotificationPreference(userId, {
        receive_order_notifications: !currentValue,
      });

      // Update local state
      setUsers(
        users.map((user) =>
          user.id === userId ? { ...user, receive_order_notifications: !currentValue } : user
        )
      );
    } catch (err: any) {
      console.error('Failed to update notification preference:', err);
      setError(t('notifications.settings.save_error') || 'Failed to update notification preference');
    } finally {
      setUpdating(false);
    }
  };

  const handleToggleGlobalNotifications = async () => {
    if (!config) return;

    try {
      setUpdating(true);
      setError(null);

      const updatedConfig = await notificationService.updateNotificationConfig({
        order_notifications_enabled: !config.order_notifications_enabled,
      });

      setConfig(updatedConfig);
    } catch (err: any) {
      console.error('Failed to update notification config:', err);
      setError(t('notifications.settings.save_error') || 'Failed to update notification settings');
    } finally {
      setUpdating(false);
    }
  };

  const handleSendTestEmail = async () => {
    if (!testEmail) {
      setTestEmailError(t('notifications.settings.testEmail.error.required') || 'Email is required');
      return;
    }

    // Simple email validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(testEmail)) {
      setTestEmailError(t('notifications.settings.testEmail.error.invalid') || 'Invalid email address');
      return;
    }

    try {
      setTestEmailSending(true);
      setTestEmailError(null);
      setTestEmailSuccess(null);

      const response = await notificationService.sendTestNotification({
        notification_type: testNotificationType,
        recipient_email: testEmail,
      });

      setTestEmailSuccess(
        t('notifications.settings.testEmail.success') || `Test email sent successfully to ${testEmail}`
      );
      setTestEmail('');

      // Close dialog after 2 seconds
      setTimeout(() => {
        setTestEmailDialogOpen(false);
        setTestEmailSuccess(null);
      }, 2000);
    } catch (err: any) {
      console.error('Failed to send test email:', err);
      if (err.response?.status === 429) {
        setTestEmailError(
          t('notifications.settings.testEmail.error.rateLimit') ||
          'Too many test emails sent. Please try again in a minute.'
        );
      } else {
        setTestEmailError(
          t('notifications.settings.testEmail.error.send') || 'Failed to send test email. Please try again.'
        );
      }
    } finally {
      setTestEmailSending(false);
    }
  };

  if (loading) {
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
    <div className="max-w-4xl mx-auto p-6">
      <div className="bg-white rounded-lg shadow-md">
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-2xl font-bold text-gray-900">
                {t('notifications.settings.title') || 'Email Notification Settings'}
              </h2>
              <p className="mt-2 text-sm text-gray-600">
                {t('notifications.settings.description') ||
                  'Configure which staff members receive email notifications when orders are paid.'}
              </p>
            </div>
            <button
              onClick={() => router.push('/settings/notifications/history')}
              className="flex items-center space-x-2 px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <span>{t('notifications.settings.viewHistory') || 'View History'}</span>
            </button>
          </div>
        </div>

        {error && (
          <div className="mx-6 mt-6 p-4 bg-red-50 border border-red-200 rounded-md">
            <p className="text-sm text-red-800">{error}</p>
          </div>
        )}

        {/* Global notification toggle */}
        <div className="p-6 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-lg font-medium text-gray-900">
                {t('notifications.settings.global.title') || 'Email Notifications'}
              </h3>
              <p className="mt-1 text-sm text-gray-600">
                {t('notifications.settings.global.description') ||
                  'Enable or disable all order notification emails for this restaurant.'}
              </p>
            </div>
            <button
              onClick={handleToggleGlobalNotifications}
              disabled={updating || !config}
              className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${config?.order_notifications_enabled ? 'bg-blue-600' : 'bg-gray-200'
                } ${updating ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              <span
                className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${config?.order_notifications_enabled ? 'translate-x-5' : 'translate-x-0'
                  }`}
              />
            </button>
          </div>
        </div>

        {/* Test email button */}
        <div className="p-6 border-b border-gray-200">
          <button
            onClick={() => setTestEmailDialogOpen(true)}
            className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 focus:ring-offset-2 disabled:opacity-50"
            disabled={!config?.order_notifications_enabled}
          >
            {t('notifications.settings.testEmail.button') || 'Send Test Email'}
          </button>
          {!config?.order_notifications_enabled && (
            <p className="mt-2 text-sm text-gray-500">
              {t('notifications.settings.testEmail.disabled') ||
                'Enable notifications to send test emails'}
            </p>
          )}
        </div>

        {/* Staff list */}
        <div className="p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">
            {t('notifications.settings.staff.title') || 'Staff Notification Preferences'}
          </h3>
          <p className="text-sm text-gray-600 mb-4">
            {t('notifications.settings.staff.description') ||
              'Select which staff members should receive email notifications for new paid orders.'}
          </p>

          <div className="space-y-3">
            {users.map((user) => (
              <div key={user.id} className="flex items-center justify-between p-4 bg-gray-50 rounded-lg">
                <div>
                  <p className="font-medium text-gray-900">{user.name}</p>
                  <p className="text-sm text-gray-600">{user.email}</p>
                  <p className="text-xs text-gray-500 mt-1">{user.role}</p>
                </div>
                <button
                  onClick={() => handleToggleUserNotification(user.id, user.receive_order_notifications)}
                  disabled={updating || !config?.order_notifications_enabled}
                  className={`relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 ${user.receive_order_notifications ? 'bg-blue-600' : 'bg-gray-200'
                    } ${updating || !config?.order_notifications_enabled ? 'opacity-50 cursor-not-allowed' : ''}`}
                >
                  <span
                    className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${user.receive_order_notifications ? 'translate-x-5' : 'translate-x-0'
                      }`}
                  />
                </button>
              </div>
            ))}

            {users.length === 0 && (
              <div className="text-center py-8 text-gray-500">
                {t('notifications.settings.staff.empty') || 'No staff members found'}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Test email dialog */}
      {testEmailDialogOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full mx-4">
            <div className="p-6">
              <h3 className="text-lg font-medium text-gray-900 mb-4">
                {t('notifications.settings.testEmail.title') || 'Send Test Email'}
              </h3>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    {t('notifications.settings.testEmail.type') || 'Notification Type'}
                  </label>
                  <select
                    value={testNotificationType}
                    onChange={(e) =>
                      setTestNotificationType(e.target.value as 'staff_order_notification' | 'customer_receipt')
                    }
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="staff_order_notification">
                      {t('notifications.settings.testEmail.types.staff') || 'Staff Order Notification'}
                    </option>
                    <option value="customer_receipt">
                      {t('notifications.settings.testEmail.types.customer') || 'Customer Receipt'}
                    </option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    {t('notifications.settings.testEmail.email') || 'Recipient Email'}
                  </label>
                  <input
                    type="email"
                    value={testEmail}
                    onChange={(e) => setTestEmail(e.target.value)}
                    placeholder="test@example.com"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                {testEmailError && (
                  <div className="p-3 bg-red-50 border border-red-200 rounded-md">
                    <p className="text-sm text-red-800">{testEmailError}</p>
                  </div>
                )}

                {testEmailSuccess && (
                  <div className="p-3 bg-green-50 border border-green-200 rounded-md">
                    <p className="text-sm text-green-800">{testEmailSuccess}</p>
                  </div>
                )}
              </div>

              <div className="mt-6 flex justify-end space-x-3">
                <button
                  onClick={() => {
                    setTestEmailDialogOpen(false);
                    setTestEmail('');
                    setTestEmailError(null);
                    setTestEmailSuccess(null);
                  }}
                  className="px-4 py-2 text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-gray-500"
                  disabled={testEmailSending}
                >
                  {t('common.cancel', { ns: 'common' }) || 'Cancel'}
                </button>
                <button
                  onClick={handleSendTestEmail}
                  disabled={testEmailSending}
                  className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500 disabled:opacity-50"
                >
                  {testEmailSending
                    ? t('notifications.settings.testEmail.sending') || 'Sending...'
                    : t('notifications.settings.testEmail.send') || 'Send Test'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
