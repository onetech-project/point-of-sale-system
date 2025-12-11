import apiClient from './api';

// User notification preference types
export interface UserNotificationPreference {
  id: string;
  name: string;
  email: string;
  role: string;
  receive_order_notifications: boolean;
}

export interface UpdateNotificationPreferenceRequest {
  receive_order_notifications: boolean;
}

// Notification config types
export interface NotificationConfig {
  tenant_id: string;
  order_notifications_enabled: boolean;
  test_mode: boolean;
  test_email?: string;
  created_at: string;
  updated_at: string;
}

export interface UpdateNotificationConfigRequest {
  order_notifications_enabled?: boolean;
  test_mode?: boolean;
  test_email?: string;
}

// Test notification types
export interface SendTestNotificationRequest {
  notification_type: 'staff_order_notification' | 'customer_receipt';
  recipient_email: string;
}

export interface TestNotificationResponse {
  success: boolean;
  notification_id: string;
  status: string;
  sent_at: string;
  recipient: string;
  message: string;
}

// Notification history types
export interface NotificationHistoryItem {
  id: string;
  event_type: string;
  type: string;
  recipient: string;
  subject: string;
  status: 'pending' | 'sent' | 'failed' | 'cancelled';
  sent_at?: string;
  failed_at?: string;
  error_msg?: string;
  retry_count: number;
  created_at: string;
  order_reference?: string;
}

export interface NotificationHistoryFilters {
  page?: number;
  page_size?: number;
  order_reference?: string;
  status?: 'pending' | 'sent' | 'failed' | 'cancelled';
  type?: 'order_staff' | 'order_customer';
  start_date?: string;
  end_date?: string;
}

export interface NotificationHistoryResponse {
  notifications: NotificationHistoryItem[];
  pagination: {
    current_page: number;
    page_size: number;
    total_items: number;
    total_pages: number;
  };
}

class NotificationService {
  // User notification preferences
  async getUserNotificationPreferences(): Promise<UserNotificationPreference[]> {
    const response = await apiClient.get<{ success: boolean; data: { users: UserNotificationPreference[] } }>(
      '/api/v1/users/notification-preferences'
    );
    return response.data.users;
  }

  async updateUserNotificationPreference(
    userId: string,
    data: UpdateNotificationPreferenceRequest
  ): Promise<void> {
    await apiClient.patch(`/api/v1/users/${userId}/notification-preferences`, data);
  }

  // Notification config
  async getNotificationConfig(): Promise<NotificationConfig> {
    const response = await apiClient.get<{ success: boolean; data: NotificationConfig }>(
      '/api/v1/notifications/config'
    );
    return response.data;
  }

  async updateNotificationConfig(data: UpdateNotificationConfigRequest): Promise<NotificationConfig> {
    const response = await apiClient.patch<{ success: boolean; data: NotificationConfig }>(
      '/api/v1/notifications/config',
      data
    );
    return response.data;
  }

  // Test notifications
  async sendTestNotification(data: SendTestNotificationRequest): Promise<TestNotificationResponse> {
    const response = await apiClient.post<{ success: boolean; data: TestNotificationResponse }>(
      '/api/v1/notifications/test',
      data
    );
    return response.data;
  }

  // Notification history
  async getNotificationHistory(filters?: NotificationHistoryFilters): Promise<NotificationHistoryResponse> {
    const response = await apiClient.get<{ success: boolean; data: NotificationHistoryResponse }>(
      '/api/v1/notifications/history',
      { params: filters }
    );
    return response.data;
  }

  async resendNotification(notificationId: string): Promise<void> {
    await apiClient.post(`/api/v1/notifications/${notificationId}/resend`, {});
  }
}

export default new NotificationService();
