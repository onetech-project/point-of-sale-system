import apiClient from './api';

export interface CustomerInfo {
  name: string;
  phone?: string;
  email?: string;
}

export interface OrderItem {
  id: string;
  order_id: string;
  product_id: string;
  product_name: string;
  quantity: number;
  total_price: number;
  unit_price: number;
}

export interface OrderDetails {
  order_id: string;
  order_reference: string;
  total_amount: number;
  payment_method: string;
  order_type: string;
  status: string;
  created_at: string;
  items: OrderItem[];
}

export interface DeliveryAddress {
  full_address: string;
  latitude?: number;
  longitude?: number;
}

export interface GuestDataResponse {
  order_reference: string;
  customer_info: CustomerInfo;
  order_details: OrderDetails;
  delivery_address?: DeliveryAddress;
}

export interface DeleteGuestDataRequest {
  email?: string | null;
  phone?: string | null;
}

export interface DeleteGuestDataResponse {
  success: boolean;
  message: string;
}

/**
 * Guest Data Service
 * Handles guest order data access and deletion requests
 */
class GuestService {
  /**
   * Get guest order data by order reference
   * Requires email or phone verification
   */
  async getGuestOrderData(
    orderReference: string,
    credentials: { email?: string; phone?: string }
  ): Promise<GuestDataResponse> {
    const params = new URLSearchParams();
    if (credentials.email) params.set('email', credentials.email);
    if (credentials.phone) params.set('phone', credentials.phone);

    const endpoint = `/api/v1/public/orders/${orderReference}/data?${params.toString()}`;
    return apiClient.get<GuestDataResponse>(endpoint);
  }

  /**
   * Delete guest order data (GDPR right to erasure)
   * Requires email or phone verification
   */
  async deleteGuestOrderData(
    orderReference: string,
    credentials: DeleteGuestDataRequest
  ): Promise<DeleteGuestDataResponse> {
    const endpoint = `/api/v1/public/orders/${orderReference}/delete`;
    return apiClient.post<DeleteGuestDataResponse>(endpoint, credentials);
  }
}

// Export singleton instance
const guestService = new GuestService();
export default guestService;
