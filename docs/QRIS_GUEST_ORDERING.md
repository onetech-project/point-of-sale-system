# QRIS Guest Ordering Feature

**Status**: ✅ Implementation Complete (107/120 tasks)  
**Version**: 1.0.0  
**Last Updated**: 2025-12-03

## Overview

The QRIS Guest Ordering System enables customers to browse a restaurant's menu, add items to cart, and complete payment using Indonesia's QRIS standard via Midtrans payment gateway—all without requiring user authentication.

### Key Features

- 🛒 **Session-Based Shopping Cart**: Persistent cart with 24-hour TTL in Redis
- 💳 **QRIS Payment Integration**: Seamless Midtrans QRIS payment flow
- 📦 **Inventory Management**: Automatic 15-minute reservations with auto-release
- 🚚 **Delivery Options**: Pickup, delivery, and dine-in support
- 📍 **Distance-Based Fees**: Google Maps Geocoding API for delivery fees
- 👨‍💼 **Admin Dashboard**: Order management with status updates
- 🏢 **Multi-Tenant**: Complete tenant isolation with per-tenant configuration

## User Stories

### User Story 1: Browse & Add to Cart (US1)
**As a** guest customer  
**I want to** browse the menu and add items to my cart  
**So that** I can prepare my order without creating an account

**Implementation**:
- Public menu page: `/menu/{tenant_id}`
- Session-based cart with cookies
- Real-time inventory checks
- 24-hour cart persistence

### User Story 2: Select Delivery Type (US2)
**As a** guest customer  
**I want to** choose between pickup, delivery, or dine-in  
**So that** I can receive my order in my preferred way

**Implementation**:
- Delivery type selector in checkout
- Table number input for dine-in
- Address input with geocoding for delivery
- Service area validation

### User Story 3: Complete Payment (US3)
**As a** guest customer  
**I want to** pay using QRIS  
**So that** I can complete my order quickly

**Implementation**:
- Midtrans Snap integration
- QRIS payment method
- Order reference generation (GO-XXXXX)
- Payment status tracking
- Auto-refresh order status page

### User Story 4: Admin Order Management (US4)
**As a** restaurant admin  
**I want to** view and manage all orders  
**So that** I can fulfill customer orders efficiently

**Implementation**:
- Order list with filters (status, date)
- Order detail modal
- Status update: PENDING → PAID → COMPLETE
- Courier tracking notes

### User Story 5: Inventory Reservations (US5)
**As a** system  
**I want to** reserve inventory during checkout  
**So that** products don't get oversold

**Implementation**:
- 15-minute TTL reservations on order creation
- Auto-release on expiration
- Convert to permanent on payment
- SELECT FOR UPDATE for race conditions

### User Story 6: Delivery Fee Calculation (US6)
**As a** guest customer  
**I want to** see delivery fees before payment  
**So that** I know the total cost upfront

**Implementation**:
- Google Maps Geocoding API
- Distance-based pricing
- Zone-based pricing (future)
- Service area validation

### User Story 7: Multi-Tenant Support (US7)
**As a** platform  
**I want to** support multiple restaurants  
**So that** each tenant operates independently

**Implementation**:
- Tenant-scoped database queries
- Per-tenant configuration
- Tenant branding display
- Isolated cart namespaces

## Architecture

### Backend Services

#### Order Service (Port 8084)
**Technology**: Go 1.23+, Echo framework  
**Responsibilities**:
- Cart management (session-based)
- Order creation and tracking
- Payment webhook handling
- Inventory reservation management
- Delivery fee calculation

**Endpoints**:

**Public (No Auth)**:
- `GET /api/v1/public/:tenantId/cart` - Get cart
- `POST /api/v1/public/:tenantId/cart/items` - Add item
- `PATCH /api/v1/public/:tenantId/cart/items/:productId` - Update quantity
- `DELETE /api/v1/public/:tenantId/cart/items/:productId` - Remove item
- `DELETE /api/v1/public/:tenantId/cart` - Clear cart
- `POST /api/v1/public/:tenantId/checkout` - Create order & payment
- `GET /api/v1/public/:tenantId/orders/:orderRef` - Get order status

**Admin (JWT Auth)**:
- `GET /api/v1/admin/orders` - List orders
- `GET /api/v1/admin/orders/:orderId` - Get order details
- `PATCH /api/v1/admin/orders/:orderId/status` - Update status
- `POST /api/v1/admin/orders/:orderId/notes` - Add notes

**Webhook**:
- `POST /api/v1/payments/midtrans/notification` - Midtrans callback

#### Dependencies
- **PostgreSQL 14**: Order storage, inventory reservations, payment transactions
- **Redis 6**: Cart sessions (24-hour TTL), inventory cache
- **Midtrans**: Payment processing (sandbox/production)
- **Google Maps API**: Geocoding for delivery addresses

### Frontend Components

#### Guest Flow Components

**1. PublicMenu.tsx**
- Product catalog display
- Tenant branding (logo, name, description)
- Add to cart functionality
- Real-time inventory display
- Category filtering

**2. Cart.tsx**
- Cart items list
- Quantity adjustment
- Remove items
- Subtotal calculation
- Proceed to checkout button

**3. CheckoutForm.tsx**
- Delivery type selection (pickup/delivery/dine-in)
- Address input with geocoding
- Delivery fee display
- Order notes
- Customer info (name, phone)
- Place order button

**4. AddressInput.tsx**
- Google Maps autocomplete
- Geocoding validation
- Service area checking
- Error messaging

**5. PaymentReturn.tsx**
- Midtrans callback handler
- Transaction status mapping
- Success/failure/pending display
- Auto-redirect to order page

**6. OrderConfirmation.tsx**
- Order details display
- Status indicators (PENDING/PAID/COMPLETE/CANCELLED)
- Delivery information
- Order items with prices
- Print receipt button

**7. Order Status Page** (`/orders/[orderReference]`)
- Public order tracking
- Auto-refresh every 10 seconds (for PENDING/PAID)
- Order history

#### Admin Flow Components

**8. OrderManagement.tsx**
- Order list with pagination
- Status filter (all/PENDING/PAID/COMPLETE/CANCELLED)
- Search by order reference
- Order detail modal
- Status update dialog
- Add notes dialog

**9. Admin Orders Page** (`/admin/orders`)
- Authentication check
- JWT token handling
- OrderManagement integration

### Database Schema

#### guest_orders
```sql
CREATE TABLE guest_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_reference VARCHAR(20) UNIQUE NOT NULL,  -- GO-XXXXX
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',  -- PENDING, PAID, COMPLETE, CANCELLED
    subtotal_amount INTEGER NOT NULL CHECK (subtotal_amount >= 0),
    delivery_fee INTEGER NOT NULL DEFAULT 0 CHECK (delivery_fee >= 0),
    total_amount INTEGER NOT NULL CHECK (total_amount >= 0),
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(20) NOT NULL,
    delivery_type VARCHAR(20) NOT NULL,  -- pickup, delivery, dine_in
    table_number VARCHAR(50),
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMP,
    completed_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    session_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT
);

-- Indexes
CREATE INDEX idx_guest_orders_tenant_status ON guest_orders(tenant_id, status);
CREATE INDEX idx_guest_orders_order_reference ON guest_orders(order_reference);
CREATE INDEX idx_guest_orders_created_at ON guest_orders(created_at DESC);
CREATE INDEX idx_guest_orders_session_id ON guest_orders(session_id);
```

#### order_items
```sql
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
    subtotal INTEGER NOT NULL CHECK (subtotal >= 0),
    notes TEXT
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
```

#### inventory_reservations
```sql
CREATE TABLE inventory_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'active',  -- active, expired, converted, released
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,  -- created_at + 15 minutes
    released_at TIMESTAMP
);

CREATE INDEX idx_inventory_reservations_product_status ON inventory_reservations(product_id, status);
CREATE INDEX idx_inventory_reservations_expires_at ON inventory_reservations(expires_at);
CREATE INDEX idx_inventory_reservations_status_expires ON inventory_reservations(status, expires_at);
```

#### payment_transactions
```sql
CREATE TABLE payment_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    transaction_id VARCHAR(255) UNIQUE NOT NULL,  -- Midtrans transaction_id
    payment_type VARCHAR(50) NOT NULL,  -- qris, gopay, etc.
    gross_amount INTEGER NOT NULL CHECK (gross_amount > 0),
    transaction_status VARCHAR(50) NOT NULL,  -- settlement, pending, deny, cancel, expire
    transaction_time TIMESTAMP NOT NULL,
    settlement_time TIMESTAMP,
    signature_key VARCHAR(255),
    status_code VARCHAR(10),
    raw_response JSONB
);

CREATE INDEX idx_payment_transactions_order_id ON payment_transactions(order_id);
CREATE INDEX idx_payment_transactions_transaction_id ON payment_transactions(transaction_id);
CREATE INDEX idx_payment_transactions_status ON payment_transactions(transaction_status);
```

#### delivery_addresses
```sql
CREATE TABLE delivery_addresses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    address_line TEXT NOT NULL,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    distance_km DECIMAL(10, 2),
    notes TEXT
);

CREATE INDEX idx_delivery_addresses_order_id ON delivery_addresses(order_id);
```

#### tenant_configs
```sql
CREATE TABLE tenant_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    enabled_delivery_types TEXT[] NOT NULL DEFAULT '{pickup}',
    service_area_type VARCHAR(20),  -- radius, polygon
    service_area_data JSONB,
    enable_delivery_fee_calculation BOOLEAN DEFAULT true,
    delivery_fee_type VARCHAR(20),  -- distance, zone, flat
    delivery_fee_config JSONB,
    inventory_reservation_ttl_minutes INTEGER DEFAULT 15 CHECK (inventory_reservation_ttl_minutes >= 5),
    min_order_amount INTEGER DEFAULT 0 CHECK (min_order_amount >= 0),
    location_lat DECIMAL(10, 8),
    location_lng DECIMAL(11, 8),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_configs_tenant_id ON tenant_configs(tenant_id);
```

## Payment Flow

### 1. Checkout Initiation
```
Guest → Frontend: Click "Place Order"
Frontend → Order Service: POST /api/v1/public/:tenantId/checkout
  {
    "customer_name": "John Doe",
    "customer_phone": "081234567890",
    "delivery_type": "delivery",
    "address": {
      "address_line": "Jl. Sudirman No. 123, Jakarta",
      "latitude": -6.208763,
      "longitude": 106.845599
    },
    "notes": "Please ring the bell"
  }
```

### 2. Order Creation
```
Order Service:
1. Validate cart items exist and have inventory
2. Create inventory reservations (15-min TTL)
3. Calculate delivery fee (if applicable)
4. Create guest_order record (status: PENDING)
5. Create order_items records
6. Create delivery_address record (if delivery)
7. Call Midtrans API to create Snap transaction
8. Store payment_transaction record
9. Return payment URL to frontend
```

### 3. Payment Redirect
```
Frontend → Guest: Redirect to Midtrans Snap URL
Guest → Midtrans: Scan QRIS code and pay
```

### 4. Webhook Processing
```
Midtrans → Order Service: POST /api/v1/payments/midtrans/notification
  {
    "transaction_status": "settlement",
    "order_id": "GO-ABC123",
    "transaction_id": "mid-123456789",
    "gross_amount": "150000.00",
    "signature_key": "sha512_hash...",
    "status_code": "200",
    "transaction_time": "2025-12-03 10:30:00"
  }

Order Service:
1. Verify signature: SHA512(order_id + status_code + gross_amount + server_key)
2. Check idempotency (prevent duplicate processing)
3. Validate gross_amount matches order total_amount
4. Update guest_order status to PAID
5. Set paid_at timestamp
6. Convert inventory reservations to permanent (status: converted)
7. Decrement product stock_quantity
8. Update inventory cache in Redis
9. Return 200 OK
```

### 5. Payment Callback
```
Midtrans → Guest Browser: Redirect to /payment/return?order_id=GO-ABC123&status_code=200&transaction_status=settlement

Frontend:
1. Parse query parameters
2. Map transaction_status to display:
   - settlement/capture → Success (green)
   - pending → Waiting for payment (yellow)
   - deny/cancel/expire → Failed (red)
3. Wait 2 seconds
4. Redirect to /orders/GO-ABC123
```

### 6. Order Tracking
```
Guest → Frontend: Visit /orders/GO-ABC123
Frontend → Order Service: GET /api/v1/public/:tenantId/orders/GO-ABC123
Order Service → Frontend: Return order details

Frontend:
- Display order status, items, total
- If status is PENDING or PAID: Auto-refresh every 10 seconds
- If status is COMPLETE or CANCELLED: Stop auto-refresh
```

## Configuration

### Environment Variables

**.env (order-service)**:
```bash
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/pos_db?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379/0

# Midtrans
MIDTRANS_SERVER_KEY=your_server_key_here
MIDTRANS_CLIENT_KEY=your_client_key_here
MIDTRANS_ENVIRONMENT=sandbox  # or production

# Google Maps
GOOGLE_MAPS_API_KEY=your_google_maps_api_key

# Service Configuration
PORT=8084
CORS_ALLOWED_ORIGINS=http://localhost:3000

# Session & Cart
SESSION_TTL_HOURS=24
CART_TTL_HOURS=24
INVENTORY_RESERVATION_TTL_MINUTES=15

# Logging
LOG_LEVEL=info
ENVIRONMENT=development
```

**.env.local (frontend)**:
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080/api
NEXT_PUBLIC_ORDER_SERVICE_URL=http://localhost:8084/api/v1
```

### Docker Compose

```yaml
order-service:
  build: ./backend/order-service
  ports:
    - "8080:8080"
  environment:
    - DATABASE_URL=postgresql://pos_user:pos_password@postgres:5432/pos_db?sslmode=disable
    - REDIS_URL=redis://redis:6379
    - MIDTRANS_SERVER_KEY=${MIDTRANS_SERVER_KEY}
    - GOOGLE_MAPS_API_KEY=${GOOGLE_MAPS_API_KEY}
  depends_on:
    postgres:
      condition: service_healthy
    redis:
      condition: service_healthy
  healthcheck:
    test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
    interval: 30s
    timeout: 10s
    retries: 3
```

## Testing

### Unit Tests
```bash
cd backend/order-service
go test ./src/services/... -v
go test ./src/repository/... -v
```

### Integration Tests
```bash
go test ./tests/integration/... -v
```

### Contract Tests
```bash
go test ./tests/contract/... -v
```

### Frontend Tests
```bash
cd frontend
npm run test
npm run test:e2e
```

## Deployment

### Prerequisites
1. PostgreSQL 14+ database
2. Redis 6+ instance
3. Midtrans account with production keys
4. Google Maps API key with Geocoding enabled
5. Domain with SSL certificate

### Steps

1. **Database Migration**:
   ```bash
   migrate -path backend/migrations \
           -database "$DATABASE_URL" \
           up
   ```

2. **Configure Webhooks**:
   - Midtrans Dashboard → Settings → Configuration → Notification URL
   - Set to: `https://yourdomain.com/api/v1/payments/midtrans/notification`

3. **Build & Deploy**:
   ```bash
   docker-compose up -d
   ```

4. **Verify Health**:
   ```bash
   curl https://yourdomain.com/health
   ```

## Monitoring

### Key Metrics
- Order creation rate
- Payment success rate
- Average checkout time
- Cart abandonment rate
- Inventory reservation expiration rate
- Webhook processing time
- API response times

### Logs
All services use structured logging with zerolog:
```json
{
  "level": "info",
  "service": "order-service",
  "tenant_id": "uuid",
  "order_reference": "GO-ABC123",
  "action": "order_created",
  "duration_ms": 234,
  "timestamp": "2025-12-03T10:30:00Z"
}
```

## Troubleshooting

### Order Stuck in PENDING
**Symptom**: Order created but never transitions to PAID  
**Causes**:
1. Webhook not received (check Midtrans logs)
2. Signature verification failed (check server key)
3. Amount mismatch (check order total)

**Solution**:
```sql
-- Check payment_transactions for this order
SELECT * FROM payment_transactions WHERE order_id = 'uuid';

-- Manually mark as paid (if payment confirmed in Midtrans)
UPDATE guest_orders SET status = 'PAID', paid_at = NOW() WHERE id = 'uuid';
```

### Inventory Oversold
**Symptom**: Stock goes negative  
**Cause**: Race condition in reservation or conversion

**Solution**:
1. Check for concurrent transactions
2. Verify SELECT FOR UPDATE is used
3. Review cleanup job logs
4. Manually adjust stock:
   ```sql
   UPDATE products SET stock_quantity = stock_quantity + X WHERE id = 'uuid';
   ```

### Cart Not Persisting
**Symptom**: Cart disappears on page refresh  
**Causes**:
1. Redis connection lost
2. Session cookie not set
3. TTL expired

**Solution**:
```bash
# Check Redis connectivity
redis-cli PING

# Check cart key exists
redis-cli KEYS "cart:*"

# Check TTL
redis-cli TTL "cart:tenant_id:session_id"
```

## Support

- **Documentation**: See `/docs` folder
- **API Reference**: See `quickstart.md`
- **Issues**: GitHub Issues
- **Contact**: dev@yourcompany.com

## Changelog

### v1.0.0 (2025-12-03)
- ✅ Initial release
- ✅ Session-based cart
- ✅ QRIS payment via Midtrans
- ✅ Inventory reservations
- ✅ Delivery fee calculation
- ✅ Admin order management
- ✅ Multi-tenant support

### Future Enhancements
- 🔜 Guest order history (optional email)
- 🔜 Push notifications for order updates
- 🔜 Multiple payment methods (GoPay, OVO, etc.)
- 🔜 Zone-based delivery pricing
- 🔜 Scheduled orders
- 🔜 Promotions and discounts
