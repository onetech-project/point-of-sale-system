import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import '@testing-library/jest-dom';
import notificationService from '../../src/services/notification';

// Mock the notification service
vi.mock('../../src/services/notification', () => ({
  default: {
    getNotificationHistory: vi.fn(),
    resendNotification: vi.fn(),
  },
}));

// Mock i18next
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: 'en' },
  }),
}));

// Placeholder component - will be implemented in T073
const NotificationHistory = () => {
  return <div>Notification History Component</div>;
};

describe('NotificationHistory Integration Tests (T066)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it.skip('should load notification history on mount', async () => {
    const mockHistory = {
      notifications: [
        {
          id: '1',
          event_type: 'order.paid.staff',
          type: 'email',
          recipient: 'staff@restaurant.com',
          subject: 'New Order: ORD-001',
          status: 'sent' as const,
          sent_at: '2025-12-11T10:30:00Z',
          retry_count: 0,
          created_at: '2025-12-11T10:29:00Z',
          order_reference: 'ORD-001',
        },
        {
          id: '2',
          event_type: 'order.paid.customer',
          type: 'email',
          recipient: 'customer@example.com',
          subject: 'Receipt: ORD-001',
          status: 'failed' as const,
          failed_at: '2025-12-11T10:30:00Z',
          error_msg: 'SMTP timeout',
          retry_count: 1,
          created_at: '2025-12-11T10:29:00Z',
          order_reference: 'ORD-001',
        },
      ],
      pagination: {
        current_page: 1,
        page_size: 20,
        total_items: 2,
        total_pages: 1,
      },
    };

    vi.mocked(notificationService.getNotificationHistory).mockResolvedValue(mockHistory);

    render(<NotificationHistory />);

    await waitFor(() => {
      expect(notificationService.getNotificationHistory).toHaveBeenCalledWith({
        page: 1,
        page_size: 20,
      });
    });

    expect(screen.getByText(/ORD-001/)).toBeInTheDocument();
    expect(screen.getByText(/staff@restaurant.com/)).toBeInTheDocument();
    expect(screen.getByText(/customer@example.com/)).toBeInTheDocument();
  });

  it.skip('should filter by order reference', async () => {
    const mockHistory = {
      notifications: [
        {
          id: '1',
          event_type: 'order.paid.staff',
          type: 'email',
          recipient: 'staff@restaurant.com',
          subject: 'New Order: ORD-123',
          status: 'sent' as const,
          sent_at: '2025-12-11T10:30:00Z',
          retry_count: 0,
          created_at: '2025-12-11T10:29:00Z',
          order_reference: 'ORD-123',
        },
      ],
      pagination: {
        current_page: 1,
        page_size: 20,
        total_items: 1,
        total_pages: 1,
      },
    };

    vi.mocked(notificationService.getNotificationHistory).mockResolvedValue(mockHistory);

    render(<NotificationHistory />);

    const searchInput = screen.getByPlaceholderText(/search by order reference/i);
    fireEvent.change(searchInput, { target: { value: 'ORD-123' } });
    fireEvent.click(screen.getByRole('button', { name: /search/i }));

    await waitFor(() => {
      expect(notificationService.getNotificationHistory).toHaveBeenCalledWith(
        expect.objectContaining({
          order_reference: 'ORD-123',
        })
      );
    });

    expect(screen.getByText(/ORD-123/)).toBeInTheDocument();
  });

  it.skip('should filter by status', async () => {
    const mockHistory = {
      notifications: [
        {
          id: '2',
          event_type: 'order.paid.customer',
          type: 'email',
          recipient: 'customer@example.com',
          subject: 'Receipt: ORD-001',
          status: 'failed' as const,
          failed_at: '2025-12-11T10:30:00Z',
          error_msg: 'SMTP timeout',
          retry_count: 1,
          created_at: '2025-12-11T10:29:00Z',
          order_reference: 'ORD-001',
        },
      ],
      pagination: {
        current_page: 1,
        page_size: 20,
        total_items: 1,
        total_pages: 1,
      },
    };

    vi.mocked(notificationService.getNotificationHistory).mockResolvedValue(mockHistory);

    render(<NotificationHistory />);

    const statusFilter = screen.getByLabelText(/status/i);
    fireEvent.change(statusFilter, { target: { value: 'failed' } });

    await waitFor(() => {
      expect(notificationService.getNotificationHistory).toHaveBeenCalledWith(
        expect.objectContaining({
          status: 'failed',
        })
      );
    });

    expect(screen.getByText(/failed/i)).toBeInTheDocument();
  });

  it.skip('should resend failed notification', async () => {
    const mockHistory = {
      notifications: [
        {
          id: 'failed-notif-id',
          event_type: 'order.paid.customer',
          type: 'email',
          recipient: 'customer@example.com',
          subject: 'Receipt: ORD-001',
          status: 'failed' as const,
          failed_at: '2025-12-11T10:30:00Z',
          error_msg: 'SMTP timeout',
          retry_count: 1,
          created_at: '2025-12-11T10:29:00Z',
          order_reference: 'ORD-001',
        },
      ],
      pagination: {
        current_page: 1,
        page_size: 20,
        total_items: 1,
        total_pages: 1,
      },
    };

    vi.mocked(notificationService.getNotificationHistory).mockResolvedValue(mockHistory);
    vi.mocked(notificationService.resendNotification).mockResolvedValue();

    render(<NotificationHistory />);

    await waitFor(() => {
      expect(screen.getByText(/failed/i)).toBeInTheDocument();
    });

    const resendButton = screen.getByRole('button', { name: /resend/i });
    fireEvent.click(resendButton);

    await waitFor(() => {
      expect(notificationService.resendNotification).toHaveBeenCalledWith('failed-notif-id');
    });

    expect(screen.getByText(/resent successfully/i)).toBeInTheDocument();
  });

  it.skip('should handle pagination', async () => {
    const mockHistory = {
      notifications: [],
      pagination: {
        current_page: 1,
        page_size: 20,
        total_items: 100,
        total_pages: 5,
      },
    };

    vi.mocked(notificationService.getNotificationHistory).mockResolvedValue(mockHistory);

    render(<NotificationHistory />);

    await waitFor(() => {
      expect(screen.getByText(/page 1 of 5/i)).toBeInTheDocument();
    });

    const nextButton = screen.getByRole('button', { name: /next/i });
    fireEvent.click(nextButton);

    await waitFor(() => {
      expect(notificationService.getNotificationHistory).toHaveBeenCalledWith(
        expect.objectContaining({
          page: 2,
        })
      );
    });
  });

  it.skip('should handle error when loading history fails', async () => {
    vi.mocked(notificationService.getNotificationHistory).mockRejectedValue(
      new Error('Failed to load')
    );

    render(<NotificationHistory />);

    await waitFor(() => {
      expect(screen.getByText(/failed to load notification history/i)).toBeInTheDocument();
    });
  });

  it.skip('should display status badges correctly', async () => {
    const mockHistory = {
      notifications: [
        {
          id: '1',
          event_type: 'order.paid.staff',
          type: 'email',
          recipient: 'staff@restaurant.com',
          subject: 'Order',
          status: 'sent' as const,
          sent_at: '2025-12-11T10:30:00Z',
          retry_count: 0,
          created_at: '2025-12-11T10:29:00Z',
        },
        {
          id: '2',
          event_type: 'order.paid.staff',
          type: 'email',
          recipient: 'staff@restaurant.com',
          subject: 'Order',
          status: 'failed' as const,
          failed_at: '2025-12-11T10:30:00Z',
          retry_count: 1,
          created_at: '2025-12-11T10:29:00Z',
        },
        {
          id: '3',
          event_type: 'order.paid.staff',
          type: 'email',
          recipient: 'staff@restaurant.com',
          subject: 'Order',
          status: 'pending' as const,
          retry_count: 0,
          created_at: '2025-12-11T10:29:00Z',
        },
      ],
      pagination: {
        current_page: 1,
        page_size: 20,
        total_items: 3,
        total_pages: 1,
      },
    };

    vi.mocked(notificationService.getNotificationHistory).mockResolvedValue(mockHistory);

    render(<NotificationHistory />);

    await waitFor(() => {
      expect(screen.getByText('sent')).toHaveClass('bg-green-100');
      expect(screen.getByText('failed')).toHaveClass('bg-red-100');
      expect(screen.getByText('pending')).toHaveClass('bg-yellow-100');
    });
  });
});
