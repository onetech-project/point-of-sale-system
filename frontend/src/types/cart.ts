/**
 * Shared Type Definitions for Cart and Guest Ordering
 * Single source of truth for cart-related types across the application
 */

/**
 * Cart Item representation
 * Used in cart operations and order creation
 */
export interface CartItem {
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  total_price: number;
  image_url?: string;
}

/**
 * Cart representation
 * Used for guest shopping cart operations
 */
export interface Cart {
  tenant_id: string;
  session_id: string;
  items: CartItem[];
  updated_at: string;
  total: number;
  expires_at?: string;
}

/**
 * Product representation for menu display
 */
export interface Product {
  id: string;
  name: string;
  description?: string;
  price: number;
  image_url?: string;
  sku: string;
  stock: number;
  available_stock: number;  // Stock minus active reservations
  is_available: boolean;
  category_id?: string;
}

/**
 * Order Item representation
 */
export interface OrderItem {
  id: string;
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  total_price: number;
}

/**
 * Order Note representation
 */
export interface OrderNote {
  id: string;
  order_id: string;
  note: string;
  created_by_user_id?: string;
  created_by_name?: string;
  created_at: string;
}

/**
 * Payment Information from payment transaction
 */
export interface PaymentInfo {
  transaction_id: string;
  transaction_status: string;
  qr_code_url?: string;
  expiry_time?: string;
  payment_type: string;
}

/**
 * Order representation
 */
export interface Order {
  id: string;
  tenant_id: string;
  order_reference: string;
  customer_name: string;
  customer_phone: string;
  delivery_type: 'DINE_IN' | 'DELIVERY' | 'TAKEAWAY';
  delivery_address?: string;
  table_number?: string;
  subtotal_amount: number;
  delivery_fee: number;
  total_amount: number;
  status: 'PENDING' | 'PAID' | 'PREPARING' | 'READY' | 'COMPLETE' | 'CANCELLED';
  payment_method?: string;
  payment_url?: string;
  items: OrderItem[];
  created_at: string;
  updated_at: string;
  payment?: PaymentInfo;
  notes?: string;
}

/**
 * Order Data from API (includes order, items, notes, payment)
 */
export interface OrderData {
  order: Order;
  items: OrderItem[];
  notes: OrderNote[];
  payment?: PaymentInfo;
}

/**
 * Tenant Configuration
 */
export interface TenantConfig {
  id: string;
  name: string;
  slug: string;
  is_active: boolean;
  settings?: {
    delivery_fee?: number;
    min_order?: number;
    max_delivery_distance?: number;
  };
}
