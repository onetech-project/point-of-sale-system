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
