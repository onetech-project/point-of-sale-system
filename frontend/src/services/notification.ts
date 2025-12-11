import apiClient from './api';
import type {
  UserNotificationPreference,
  UpdateNotificationPreferenceRequest,
  NotificationConfig,
  UpdateNotificationConfigRequest,
  SendTestNotificationRequest,
  TestNotificationResponse,
  NotificationHistoryItem,
  NotificationHistoryFilters,
  NotificationHistoryResponse,
} from '../types/notification';

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
