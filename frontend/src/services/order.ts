import axios from 'axios';
import { Order, OrderItem, OrderNote } from '../types/cart';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';

// Re-export types for backward compatibility
export type { Order, OrderItem, OrderNote };

export interface OrderWithDetails {
  order: Order;
  items: OrderItem[];
  latest_note?: OrderNote;
}

export interface OrderListResponse {
  orders: OrderWithDetails[];
  pagination: {
    limit: number;
    offset: number;
    count: number;
  };
}

/**
 * Guest Order Service
 * Provides API calls for guest order operations
 */
class GuestOrderService {
  /**
   * Get order details by reference number
   * Public endpoint - no authentication required
   */
  async getOrderByReference(orderReference: string): Promise<{
    order: Order;
    items: OrderItem[];
    notes: OrderNote[];
    payment?: any;
  }> {
    try {
      const response = await axios.get<{
        order: Order;
        items: OrderItem[];
        notes: OrderNote[];
        payment?: any;
      }>(
        `${API_BASE_URL}/api/v1/public/orders/${orderReference}`
      );

      return response.data;
    } catch (error) {
      console.error('Failed to fetch order:', error);
      throw error;
    }
  }

  /**
   * Create a new order (checkout)
   * Public endpoint - no authentication required
   */
  async createOrder(
    tenantId: string,
    sessionId: string,
    checkoutData: {
      delivery_type: string;
      customer_name: string;
      customer_phone: string;
      delivery_address?: string;
      table_number?: string;
      notes?: string;
    }
  ): Promise<{
    order_reference: string;
    order_id: string;
    status: string;
    total: number;
    delivery_type: string;
    payment_url?: string;
    payment_token?: string;
    created_at: string;
  }> {
    try {
      const response = await axios.post(
        `${API_BASE_URL}/api/v1/public/${tenantId}/checkout`,
        checkoutData,
        {
          headers: {
            'X-Session-ID': sessionId,
          },
        }
      );
      return response.data;
    } catch (error) {
      console.error('Failed to create order:', error);
      throw error;
    }
  }

  // Admin operations (require authentication)

  /**
   * List orders for tenant (admin only)
   * Requires JWT authentication
   * Tenant ID is extracted from session by API Gateway
   */
  async listOrders(
    filters?: {
      status?: string;
      page?: number;
      limit?: number;
    }
  ): Promise<OrderListResponse> {
    try {
      const params = new URLSearchParams();
      if (filters?.status) params.append('status', filters.status);
      if (filters?.limit) params.append('limit', filters.limit.toString());
      // Convert page to offset for backend
      if (filters?.page && filters?.limit) {
        const offset = (filters.page - 1) * filters.limit;
        params.append('offset', offset.toString());
      }

      const response = await axios.get<OrderListResponse>(
        `${API_BASE_URL}/api/v1/admin/orders?${params.toString()}`,
        {
          withCredentials: true,
        }
      );
      return response.data;
    } catch (error) {
      console.error('Failed to list orders:', error);
      throw error;
    }
  }

  /**
   * Get order details by ID (admin only)
   * Requires JWT authentication
   */
  async getOrderById(orderId: string): Promise<Order> {
    try {
      const response = await axios.get<Order>(
        `${API_BASE_URL}/api/v1/admin/orders/${orderId}`,
        {
          withCredentials: true,
        }
      );
      return response.data;
    } catch (error) {
      console.error('Failed to fetch order:', error);
      throw error;
    }
  }

  /**
   * Update order status (admin only)
   * Requires JWT authentication
   * Tenant ID is extracted from session by API Gateway
   */
  async updateOrderStatus(
    orderId: string,
    status: string
  ): Promise<Order> {
    try {
      const response = await axios.patch<Order>(
        `${API_BASE_URL}/api/v1/admin/orders/${orderId}/status`,
        { status },
        {
          withCredentials: true,
        }
      );
      return response.data;
    } catch (error) {
      console.error('Failed to update order status:', error);
      throw error;
    }
  }

  /**
   * Add note to order (admin only)
   * Requires JWT authentication
   * Tenant ID is extracted from session by API Gateway
   */
  async addOrderNote(
    orderId: string,
    note: string
  ): Promise<void> {
    try {
      await axios.post(
        `${API_BASE_URL}/api/v1/admin/orders/${orderId}/notes`,
        { note },
        {
          withCredentials: true,
        }
      );
    } catch (error) {
      console.error('Failed to add order note:', error);
      throw error;
    }
  }
}

export const order = new GuestOrderService();
