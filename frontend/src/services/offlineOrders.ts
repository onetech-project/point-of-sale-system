/**
 * Offline Order Service
 * API client for offline order management operations
 */

import apiClient from './api';
import {
  OfflineOrder,
  OfflineOrderWithDetails,
  CreateOfflineOrderRequest,
  UpdateOfflineOrderRequest,
  RecordPaymentRequest,
  ListOfflineOrdersResponse,
  ListOfflineOrdersFilters,
  PaymentRecord,
} from '../types/offlineOrder';

class OfflineOrderService {
  /**
   * Create a new offline order
   * Requires authentication
   */
  async createOfflineOrder(request: CreateOfflineOrderRequest): Promise<{ order: OfflineOrder }> {
    try {
      const response = await apiClient.post<{ order: OfflineOrder }>(
        '/api/v1/admin/offline-orders',
        request
      );
      return response;
    } catch (error) {
      console.error('Failed to create offline order:', error);
      throw error;
    }
  }

  /**
   * List offline orders with optional filters
   * Requires authentication
   */
  async listOfflineOrders(filters?: ListOfflineOrdersFilters): Promise<ListOfflineOrdersResponse> {
    try {
      const params = new URLSearchParams();

      if (filters?.status) {
        params.append('status', filters.status);
      }
      if (filters?.search) {
        params.append('search', filters.search);
      }
      if (filters?.limit) {
        params.append('limit', filters.limit.toString());
      }
      if (filters?.offset) {
        params.append('offset', filters.offset.toString());
      }

      const response = await apiClient.get<ListOfflineOrdersResponse>(
        `/api/v1/admin/offline-orders?${params.toString()}`
      );
      return response;
    } catch (error) {
      console.error('Failed to list offline orders:', error);
      throw error;
    }
  }

  /**
   * Get offline order by ID with all related details
   * Requires authentication
   */
  async getOfflineOrderWithDetails(orderId: string): Promise<OfflineOrderWithDetails> {
    try {
      const response = await apiClient.get<{
        order: OfflineOrder;
        items: any[];
      }>(`/api/v1/admin/offline-orders/${orderId}`);
      return {
        order: response.order,
        items: (response.items || []).map(item => ({
          id: item.id,
          order_id: item.order_id,
          product_id: item.product_id,
          product_name: item.product_name,
          quantity: item.quantity,
          unit_price: item.unit_price,
          total_price: item.total_price,
        })),
      };
    } catch (error) {
      console.error('Failed to fetch offline order with details:', error);
      throw error;
    }
  }

  /**
   * Get offline order by ID (basic info only)
   * Requires authentication
   */
  async getOfflineOrderById(orderId: string): Promise<OfflineOrder> {
    try {
      const response = await apiClient.get<{
        order: OfflineOrder;
      }>(`/api/v1/admin/offline-orders/${orderId}`);
      return response.order;
    } catch (error) {
      console.error('Failed to fetch offline order:', error);
      throw error;
    }
  }

  /**
   * Record a payment for an offline order
   * Requires authentication
   */
  async recordPayment(
    orderId: string,
    request: RecordPaymentRequest
  ): Promise<{ payment: PaymentRecord }> {
    try {
      const response = await apiClient.post<{ payment: PaymentRecord }>(
        `/api/v1/admin/offline-orders/${orderId}/payments`,
        request
      );
      return response;
    } catch (error) {
      console.error('Failed to record payment:', error);
      throw error;
    }
  }

  /**
   * Get payment history for an offline order (includes payment terms if exists)
   * Requires authentication
   */
  async getPaymentHistory(orderId: string): Promise<{
    payments: PaymentRecord[];
    payment_terms?: any;
  }> {
    try {
      const response = await apiClient.get<{
        payments: PaymentRecord[];
        payment_terms?: any;
      }>(`/api/v1/admin/offline-orders/${orderId}/payments`);
      return response;
    } catch (error) {
      console.error('Failed to fetch payment history:', error);
      throw error;
    }
  }

  /**
   * Update an existing offline order (US3)
   * Requires authentication
   * Only PENDING orders can be edited
   */
  async updateOfflineOrder(
    orderId: string,
    request: UpdateOfflineOrderRequest
  ): Promise<{ order: OfflineOrder }> {
    try {
      const response = await apiClient.patch<{ order: OfflineOrder }>(
        `/api/v1/admin/offline-orders/${orderId}`,
        request
      );
      return response;
    } catch (error) {
      console.error('Failed to update offline order:', error);
      throw error;
    }
  }

  /**
   * Delete an offline order (US4 - Role-Based Deletion)
   * T098: Delete offline order with reason
   * Requires authentication and owner/manager role
   * Only PENDING and CANCELLED orders can be deleted
   */
  async deleteOfflineOrder(
    orderId: string,
    reason: string
  ): Promise<{ message: string; order_id: string }> {
    try {
      // Send reason as query parameter
      const response = await apiClient.delete<{ message: string; order_id: string }>(
        `/api/v1/admin/offline-orders/${orderId}?reason=${encodeURIComponent(reason)}`
      );
      return response;
    } catch (error) {
      console.error('Failed to delete offline order:', error);
      throw error;
    }
  }

  /**
   * Generate order reference preview
   * Helper for displaying formatted order references
   */
  formatOrderReference(reference: string): string {
    return reference.toUpperCase();
  }
}

// Export singleton instance
const offlineOrderService = new OfflineOrderService();
export default offlineOrderService;
