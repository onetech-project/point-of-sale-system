/**
 * Offline Order Type Definitions
 * Types for staff-recorded offline order management
 */

/**
 * Order Type distinguishes between online (self-service) and offline (staff-recorded) orders
 */
export type OrderType = 'online' | 'offline';

/**
 * Consent Method represents how customer consent was obtained for data collection
 */
export type ConsentMethod = 'verbal' | 'written' | 'digital';

/**
 * Order Status for both online and offline orders
 */
export type OrderStatus = 'PENDING' | 'PAID' | 'COMPLETE' | 'CANCELLED';

/**
 * Delivery Type for orders
 */
export type DeliveryType = 'pickup' | 'delivery' | 'dine_in';

/**
 * Payment Method for offline payment recording
 */
export type PaymentMethod = 'cash' | 'card' | 'bank_transfer' | 'check' | 'other';

/**
 * Offline Order representation (extends guest order with offline-specific fields)
 */
export interface OfflineOrder {
  id: string;
  tenant_id: string;
  order_reference: string;
  status: OrderStatus;
  order_type: OrderType;

  // Customer Information (PII - encrypted at rest)
  customer_name: string;
  customer_phone: string;
  customer_email?: string;

  // Order Details
  delivery_type: 'pickup' | 'delivery' | 'dine_in';
  table_number?: string;
  notes?: string;

  // Amounts (in smallest currency unit - cents)
  subtotal_amount: number;
  delivery_fee: number;
  total_amount: number;

  // Offline-specific fields
  data_consent_given: boolean;
  consent_method?: ConsentMethod;
  recorded_by_user_id?: string;
  last_modified_by_user_id?: string;
  last_modified_at?: string;

  // Timestamps
  created_at: string;
  paid_at?: string;
  completed_at?: string;
  cancelled_at?: string;

  // Relations
  tenant_slug?: string;
}

/**
 * Order Item representation (shared with online orders)
 */
export interface OfflineOrderItem {
  id: string;
  order_id: string;
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  total_price: number;
}

/**
 * Payment Terms for installment orders
 */
export interface PaymentTerms {
  id: string;
  order_id: string;
  total_amount: number;
  down_payment_amount?: number;
  installment_count: number;
  installment_amount: number;
  payment_schedule: PaymentSchedule[];
  total_paid: number;
  remaining_balance: number;
  created_at: string;
  created_by_user_id: string;
}

/**
 * Payment Schedule entry (single installment)
 */
export interface PaymentSchedule {
  installment_number: number;
  due_date: string; // ISO 8601 date (YYYY-MM-DD)
  amount: number;
  status: 'pending' | 'paid' | 'overdue';
}

/**
 * Payment Record (transaction log)
 */
export interface PaymentRecord {
  id: string;
  order_id: string;
  payment_terms_id?: string;
  payment_number: number; // 0 for down payment, 1+ for installments
  amount_paid: number;
  payment_date: string;
  payment_method: PaymentMethod;
  remaining_balance_after: number;
  recorded_by_user_id: string;
  notes?: string;
  receipt_number?: string;
  created_at: string;
}

/**
 * Complete offline order with related entities
 */
export interface OfflineOrderWithDetails {
  order: OfflineOrder;
  items: OfflineOrderItem[];
  payment_terms?: PaymentTerms;
  payment_records?: PaymentRecord[];
}

/**
 * Create Offline Order Request
 */
export interface CreateOfflineOrderRequest {
  // Customer Information
  customer_name: string;
  customer_phone: string;
  customer_email?: string;

  // Delivery Details
  delivery_type: 'pickup' | 'delivery' | 'dine_in';
  table_number?: string;
  notes?: string;

  // Order Items
  items: {
    product_id: string;
    product_name: string;
    quantity: number;
    unit_price: number;
  }[];

  // Data Consent
  data_consent_given: boolean;
  consent_method: ConsentMethod;

  // Payment Details (optional - for installment orders)
  payment_terms?: {
    down_payment_amount?: number;
    installment_count: number;
    installment_amount: number;
    payment_schedule: PaymentSchedule[];
  };
}

/**
 * Record Payment Request
 */
export interface RecordPaymentRequest {
  order_id: string;
  payment_terms_id?: string;
  payment_number: number;
  amount_paid: number;
  payment_method: PaymentMethod;
  notes?: string;
  receipt_number?: string;
}

/**
 * List Offline Orders Response
 */
export interface ListOfflineOrdersResponse {
  orders: OfflineOrder[];
  total_count: number;
  page: number;
  page_size: number;
}

/**
 * List Filters for offline orders
 */
export interface ListOfflineOrdersFilters {
  status?: OrderStatus;
  search?: string; // Search by order_reference
  limit?: number;
  offset?: number;
}

/**
 * Update Offline Order Request (US3)
 * For editing existing offline orders with audit trail
 */
export interface UpdateOfflineOrderRequest {
  customer_name?: string;
  customer_phone?: string;
  customer_email?: string;
  delivery_type?: DeliveryType;
  table_number?: string;
  notes?: string;
  delivery_fee?: number;
  items?: OrderItemInput[];
}

/**
 * Order Item Input for creating/updating orders
 */
export interface OrderItemInput {
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
}
