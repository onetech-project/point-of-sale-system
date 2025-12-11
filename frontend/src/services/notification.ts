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
    const response = await apiClient.get<{ users: UserNotificationPreference[] }>(
      '/api/v1/users/notification-preferences'
    );
    return response.users || [];
  }

  async updateUserNotificationPreference(
    userId: string,
    data: UpdateNotificationPreferenceRequest
  ): Promise<void> {
    await apiClient.patch(`/api/v1/users/${userId}/notification-preferences`, data);
  }

  // Notification config
  async getNotificationConfig(): Promise<NotificationConfig> {
    const response = await apiClient.get<NotificationConfig>(
      '/api/v1/notifications/config'
    );
    return response;
  }

  async updateNotificationConfig(data: UpdateNotificationConfigRequest): Promise<NotificationConfig> {
    const response = await apiClient.patch<NotificationConfig>(
      '/api/v1/notifications/config',
      data
    );
    return response;
  }

  // Test notifications
  async sendTestNotification(data: SendTestNotificationRequest): Promise<TestNotificationResponse> {
    const response = await apiClient.post<TestNotificationResponse>(
      '/api/v1/notifications/test',
      data
    );
    return response;
  }

  // Notification history
  async getNotificationHistory(filters?: NotificationHistoryFilters): Promise<NotificationHistoryResponse> {
    const response = await apiClient.get<NotificationHistoryResponse>(
      '/api/v1/notifications/history',
      { params: filters }
    );
    return response;
  }

  async resendNotification(notificationId: string): Promise<void> {
    await apiClient.post(`/api/v1/notifications/${notificationId}/resend`, {});
  }
}

export default new NotificationService();
