# QRIS Payment Implementation Summary

## Overview

Implemented complete QRIS payment flow integration with Midtrans Core API, replacing the previous Snap API flow. This allows guests to checkout and pay via QRIS code scanning directly from the order status page.

**Implementation Date**: 2025-01-07  
**Status**: ✅ Complete - Ready for Testing

---

## Architecture Changes

### Backend Changes

#### 1. Database Schema (Migration 000020)

**File**: `backend/migrations/000020_add_qr_fields_to_payment_transactions.up.sql`

Added three new columns to `payment_transactions` table:

- `qr_code_url TEXT` - URL to QR code image from Midtrans actions array
- `qr_string TEXT` - Raw QRIS string data
- `expiry_time TIMESTAMP` - Payment expiration timestamp (default 15 minutes)

**Migration Status**: ⚠️ Not yet executed - run migration before testing

#### 2. Data Models

**File**: `backend/order-service/src/models/payment_transaction.go`

Extended `PaymentTransaction` model:

```go
type PaymentTransaction struct {
    // ... existing fields
    QRCodeURL  *string    `json:"qr_code_url,omitempty"`
    QRString   *string    `json:"qr_string,omitempty"`
    ExpiryTime *time.Time `json:"expiry_time,omitempty"`
}
```

#### 3. Midtrans Configuration

**File**: `backend/order-service/src/config/midtrans.go`

- Added Core API client initialization alongside Snap client
- Added `GetCoreAPIClient()` method
- Added `GetWebhookURL()` method to retrieve webhook endpoint from env

**Required Environment Variable**:

```bash
MIDTRANS_WEBHOOK_URL=https://your-domain.com/api/v1/webhooks/midtrans
# Defaults to http://localhost:8080/api/v1/webhooks/midtrans if not set
```

#### 4. Payment Service

**File**: `backend/order-service/src/services/payment_service.go`

**New Methods**:

a) `CreateQRISCharge(ctx context.Context, order *models.Order)`:

- Creates QRIS payment using Midtrans Core API `/v2/charge` endpoint
- Builds JSON payload with `payment_type: "qris"` and transaction details
- Sets `Authorization: Basic {base64(serverKey:)}` header
- Sets `X-Override-Notification: {webhook_url}` header for webhook delivery
- Parses response including `actions` array with QR code URL
- Returns `QRISChargeResponse` with transaction_id, expiry_time, actions

b) `SaveQRISPaymentInfo(ctx context.Context, orderID string, chargeResp *QRISChargeResponse)`:

- Parses expiry_time string to time.Time
- Extracts QR code URL from `actions[0].url`
- Creates PaymentTransaction record with all QR fields populated
- Handles idempotency via transaction_id

**Data Structures**:

```go
type QRISChargeResponse struct {
    TransactionID     string   `json:"transaction_id"`
    OrderID           string   `json:"order_id"`
    GrossAmount       string   `json:"gross_amount"`
    PaymentType       string   `json:"payment_type"`
    TransactionTime   string   `json:"transaction_time"`
    TransactionStatus string   `json:"transaction_status"`
    FraudStatus       string   `json:"fraud_status"`
    ExpiryTime        string   `json:"expiry_time"`
    Actions           []Action `json:"actions"`
    QRString          string   `json:"qr_string"`
}

type Action struct {
    Name   string `json:"name"`
    Method string `json:"method"`
    URL    string `json:"url"`
}
```

#### 5. Payment Repository

**File**: `backend/order-service/src/repository/payment_repository.go`

Updated all CRUD methods to handle new QR fields:

- `CreatePaymentTransaction`: INSERT includes qr_code_url, qr_string, expiry_time
- `GetPaymentByOrderID`: SELECT and Scan include QR fields
- `GetPaymentByTransactionID`: SELECT and Scan include QR fields
- `GetPaymentByIdempotencyKey`: SELECT and Scan include QR fields

#### 6. Checkout Handler

**File**: `backend/order-service/api/checkout_handler.go`

**Modified Checkout Flow** (CreateOrder method):

- Replaced `CreateSnapTransaction` with `CreateQRISCharge`
- Added `SaveQRISPaymentInfo` call after charge creation
- Extracts QR code URL from `qrisResp.Actions[0].URL`
- Returns `paymentURL` with QR code image URL in checkout response
- Sets `PaymentToken` to nil (not needed for QRIS)

**Enhanced Order Status Endpoint** (GetPublicOrder method):

- Added payment repository query to fetch payment by order ID
- Returns combined response with both order and payment objects:

```json
{
  "order": { ... },
  "payment": {
    "transaction_id": "uuid",
    "transaction_status": "pending|settlement|expire",
    "qr_code_url": "https://api.sandbox.midtrans.com/v2/qris/{id}/qr-code",
    "expiry_time": "2025-12-07T12:32:40Z",
    "payment_type": "qris"
  }
}
```

#### 7. Webhook Handler

**File**: `backend/order-service/src/services/payment_service.go`

**Existing handlers verified** (no changes needed):

- `ProcessNotification`: Verifies Midtrans signature and routes to handlers
- `handlePaymentSuccess`: settlement + accept → order.PAID + convert inventory
- `handlePaymentFailure`: expire + accept → order.CANCELLED + release inventory

---

### Frontend Changes

#### 1. Type Definitions

**File**: `frontend/src/types/cart.ts`

Added `PaymentInfo` interface:

```typescript
export interface PaymentInfo {
  transaction_id: string
  transaction_status: string
  qr_code_url?: string
  expiry_time?: string
  payment_type: string
}
```

Extended `Order` interface:

```typescript
export interface Order {
  // ... existing fields
  payment?: PaymentInfo
}
```

#### 2. Order Service

**File**: `frontend/src/services/order.ts`

Updated `getOrderByReference` method:

- Handles new response structure `{ order: {...}, payment: {...} }`
- Attaches payment info to order object
- Sets `payment_url` from `payment.qr_code_url` for backward compatibility

```typescript
const orderData = response.data.order
if (response.data.payment) {
  orderData.payment = response.data.payment
  if (response.data.payment.qr_code_url) {
    orderData.payment_url = response.data.payment.qr_code_url
  }
}
```

#### 3. Order Confirmation Component

**File**: `frontend/src/components/guest/OrderConfirmation.tsx`

**New Features**:

- Added `paymentInfo?: PaymentInfo` prop
- Implemented countdown timer using React useState/useEffect
- Timer updates every second showing MM:SS format
- Displays "Expired" when time runs out
- Only shows countdown for PENDING orders with expiry_time

**Timer Logic**:

```typescript
const [timeRemaining, setTimeRemaining] = useState<string>('')

useEffect(() => {
  if (!paymentInfo?.expiry_time || orderStatus !== 'PENDING') {
    setTimeRemaining('')
    return
  }

  const updateTimer = () => {
    const expiryDate = new Date(paymentInfo.expiry_time)
    const now = new Date()
    const diff = expiryDate.getTime() - now.getTime()

    if (diff <= 0) {
      setTimeRemaining('Expired')
      return
    }

    const minutes = Math.floor(diff / 60000)
    const seconds = Math.floor((diff % 60000) / 1000)
    setTimeRemaining(`${minutes}:${seconds.toString().padStart(2, '0')}`)
  }

  updateTimer()
  const interval = setInterval(updateTimer, 1000)
  return () => clearInterval(interval)
}, [paymentInfo?.expiry_time, orderStatus])
```

**UI Enhancements**:

- QR code displayed in bordered container
- Countdown timer displayed prominently below QR code
- Color changes to red when expired
- Instructions for scanning with mobile banking/e-wallet

#### 4. Order Status Page

**File**: `frontend/app/orders/[orderReference]/page.tsx`

- Added `paymentInfo={orderData.payment}` prop to OrderConfirmation
- Existing auto-refresh (10 seconds) continues to work
- Fetches updated payment status automatically

---

## API Flow

### Checkout Flow

```
1. Guest completes checkout form
   ↓
2. Frontend: POST /api/v1/public/orders/checkout
   - Creates order with items
   - Validates inventory
   ↓
3. Backend: CreateQRISCharge
   - POST https://api.sandbox.midtrans.com/v2/charge
   - Headers:
     * Authorization: Basic base64(serverKey:)
     * X-Override-Notification: webhook_url
   - Body:
     {
       "payment_type": "qris",
       "transaction_details": {
         "order_id": "order-uuid",
         "gross_amount": 50000
       }
     }
   ↓
4. Midtrans Response:
   {
     "transaction_id": "uuid",
     "transaction_status": "pending",
     "expiry_time": "2025-12-07 12:32:40",
     "actions": [
       {
         "name": "generate-qr-code",
         "method": "GET",
         "url": "https://api.sandbox.midtrans.com/v2/qris/{id}/qr-code"
       }
     ]
   }
   ↓
5. Backend: SaveQRISPaymentInfo
   - Creates payment_transactions record
   - Stores transaction_id, expiry_time, qr_code_url
   ↓
6. Frontend receives:
   {
     "order_reference": "GO-XXX",
     "payment_url": "https://.../qr-code"
   }
   ↓
7. Redirect to /orders/{order_reference}
```

### Order Status Flow

```
1. Frontend: GET /api/v1/public/orders/{reference}
   ↓
2. Backend returns:
   {
     "order": {
       "id": "uuid",
       "status": "PENDING",
       ...
     },
     "payment": {
       "transaction_id": "uuid",
       "transaction_status": "pending",
       "qr_code_url": "https://.../qr-code",
       "expiry_time": "2025-12-07T12:32:40Z",
       "payment_type": "qris"
     }
   }
   ↓
3. Frontend displays:
   - Order details
   - QR code image
   - Countdown timer
   - Payment status
   ↓
4. Auto-refresh every 10 seconds
```

### Payment Webhook Flow

```
1. Customer scans QR code and pays
   ↓
2. Midtrans: POST {webhook_url}/api/v1/webhooks/midtrans
   - Headers: signature verification
   - Body:
     {
       "transaction_status": "settlement",
       "fraud_status": "accept",
       "order_id": "order-uuid",
       "transaction_id": "uuid"
     }
   ↓
3. Backend: ProcessNotification
   - Verifies signature
   - Routes to handlePaymentSuccess
   ↓
4. handlePaymentSuccess:
   - Updates order.status = PAID
   - Updates payment_transactions.transaction_status = settlement
   - Calls inventory service to convert reserved → sold
   ↓
5. Frontend auto-refresh detects status change
   - Updates UI to show "Payment Successful"
   - Order status badge changes to PAID (green)
```

### Payment Expiry Flow

```
1. 15 minutes pass without payment
   ↓
2. Midtrans: POST {webhook_url}/api/v1/webhooks/midtrans
   - Body:
     {
       "transaction_status": "expire",
       "fraud_status": "accept"
     }
   ↓
3. Backend: handlePaymentFailure
   - Updates order.status = CANCELLED
   - Updates payment_transactions.transaction_status = expire
   - Calls inventory service to release reserved stock
   ↓
4. Frontend countdown shows "Expired"
   - Auto-refresh detects CANCELLED status
   - Updates UI to show "Payment Expired"
```

---

## Midtrans API Integration

### Endpoint

- **Sandbox**: `https://api.sandbox.midtrans.com/v2/charge`
- **Production**: `https://api.midtrans.com/v2/charge`

### Authentication

```
Authorization: Basic base64encode(ServerKey + ":")
```

### Request Headers

```json
{
  "Content-Type": "application/json",
  "Accept": "application/json",
  "Authorization": "Basic <base64_encoded_server_key>",
  "X-Override-Notification": "https://your-domain.com/api/v1/webhooks/midtrans"
}
```

### Request Body

```json
{
  "payment_type": "qris",
  "transaction_details": {
    "order_id": "order-uuid",
    "gross_amount": 50000
  }
}
```

### Response Structure

```json
{
  "status_code": "201",
  "status_message": "QRIS transaction is created",
  "transaction_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "order_id": "order-uuid",
  "merchant_id": "G999999999",
  "gross_amount": "50000.00",
  "currency": "IDR",
  "payment_type": "qris",
  "transaction_time": "2025-12-07 12:17:40",
  "transaction_status": "pending",
  "fraud_status": "accept",
  "expiry_time": "2025-12-07 12:32:40",
  "actions": [
    {
      "name": "generate-qr-code",
      "method": "GET",
      "url": "https://api.sandbox.midtrans.com/v2/qris/{id}/qr-code"
    }
  ],
  "qr_string": "00020101021226..."
}
```

### Webhook Notification

```json
{
  "transaction_time": "2025-12-07 12:20:15",
  "transaction_status": "settlement",
  "transaction_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "status_message": "midtrans payment notification",
  "status_code": "200",
  "signature_key": "abc123...",
  "payment_type": "qris",
  "order_id": "order-uuid",
  "merchant_id": "G999999999",
  "gross_amount": "50000.00",
  "fraud_status": "accept",
  "currency": "IDR"
}
```

---

## Environment Variables

### Backend (order-service)

```bash
# Midtrans Configuration
MIDTRANS_SERVER_KEY=your-server-key
MIDTRANS_CLIENT_KEY=your-client-key
MIDTRANS_ENV=sandbox  # or production

# Webhook Configuration
MIDTRANS_WEBHOOK_URL=https://your-domain.com/api/v1/webhooks/midtrans
# If not set, defaults to: http://localhost:8080/api/v1/webhooks/midtrans

# Database Configuration (for migrations)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-password
DB_NAME=pos_order_db
```

---

## Testing Checklist

### Prerequisites

- [ ] Run migration 000020: `make migrate-up` or manual SQL execution
- [ ] Verify `MIDTRANS_WEBHOOK_URL` is set to publicly accessible URL
- [ ] Ensure Midtrans sandbox credentials are configured
- [ ] Backend services running (order-service, inventory-service)
- [ ] Frontend running on port 3000

### Backend Testing

#### 1. Migration Verification

```bash
# Connect to database
psql -h localhost -U postgres -d pos_order_db

# Check schema
\d payment_transactions

# Should see:
# qr_code_url    | text
# qr_string      | text
# expiry_time    | timestamp without time zone
```

#### 2. QRIS Charge Creation

```bash
# Test checkout endpoint
curl -X POST http://localhost:8080/api/v1/public/orders/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "your-tenant-id",
    "session_id": "your-session-id",
    "delivery_type": "dine_in",
    "customer_name": "Test Customer",
    "customer_phone": "08123456789",
    "table_number": "5"
  }'

# Expected response:
{
  "order_reference": "GO-XXX",
  "order_id": "uuid",
  "status": "PENDING",
  "payment_url": "https://api.sandbox.midtrans.com/v2/qris/{id}/qr-code",
  "total": 50000,
  ...
}
```

#### 3. Order Status Endpoint

```bash
# Test public order endpoint
curl http://localhost:8080/api/v1/public/orders/GO-XXX

# Expected response:
{
  "order": {
    "id": "uuid",
    "order_reference": "GO-XXX",
    "status": "PENDING",
    ...
  },
  "payment": {
    "transaction_id": "uuid",
    "transaction_status": "pending",
    "qr_code_url": "https://...",
    "expiry_time": "2025-12-07T12:32:40Z",
    "payment_type": "qris"
  }
}
```

#### 4. Database Verification

```sql
-- Check payment_transactions record
SELECT
  order_id,
  midtrans_transaction_id,
  transaction_status,
  qr_code_url,
  expiry_time,
  created_at
FROM payment_transactions
ORDER BY created_at DESC
LIMIT 1;

-- Should show:
-- qr_code_url: https://api.sandbox.midtrans.com/v2/qris/{id}/qr-code
-- expiry_time: timestamp ~15 minutes from creation
-- transaction_status: pending
```

### Frontend Testing

#### 1. Checkout Flow

1. Navigate to `/menu/{tenant_id}`
2. Add items to cart
3. Click "Checkout"
4. Fill in customer details
5. Submit checkout form
6. **Verify**: Redirected to `/orders/GO-XXX`
7. **Verify**: QR code image displayed
8. **Verify**: Countdown timer showing MM:SS format
9. **Verify**: Order status shows "PENDING"

#### 2. QR Code Display

1. On order status page
2. **Verify**: QR code image loads (not broken image)
3. **Verify**: Image size is 256x256px with border
4. **Verify**: "Scan to Pay" heading visible
5. **Verify**: Instructions displayed below heading

#### 3. Countdown Timer

1. **Verify**: Timer displays initially (e.g., "14:59")
2. Wait 1 minute
3. **Verify**: Timer decrements (e.g., "13:59")
4. **Verify**: Timer updates every second
5. **Verify**: Format is MM:SS with leading zeros

#### 4. Auto-refresh

1. Keep order status page open
2. Wait 10 seconds
3. **Verify**: Console shows new API call
4. **Verify**: Page data refreshes without reload
5. **Verify**: "Auto-refreshing every 10 seconds..." message visible

### Integration Testing

#### 1. Successful Payment Flow

1. Create order via checkout
2. Copy QR code URL from network inspector
3. Simulate Midtrans webhook (settlement):

```bash
curl -X POST http://localhost:8080/api/v1/webhooks/midtrans \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_status": "settlement",
    "fraud_status": "accept",
    "order_id": "order-uuid",
    "transaction_id": "payment-transaction-uuid",
    "gross_amount": "50000.00",
    "signature_key": "calculated-signature"
  }'
```

4. **Verify**: Order status updates to "PAID" within 10 seconds
5. **Verify**: Payment status message changes to "Payment received!"
6. **Verify**: QR code section disappears
7. **Verify**: Inventory converted from reserved to sold

#### 2. Payment Expiry Flow

1. Create order via checkout
2. Wait for countdown to reach 0 OR simulate webhook (expire):

```bash
curl -X POST http://localhost:8080/api/v1/webhooks/midtrans \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_status": "expire",
    "fraud_status": "accept",
    "order_id": "order-uuid",
    "transaction_id": "payment-transaction-uuid"
  }'
```

3. **Verify**: Countdown shows "Expired"
4. **Verify**: Order status updates to "CANCELLED"
5. **Verify**: Status message shows "This order has been cancelled"
6. **Verify**: Reserved inventory released

#### 3. Real Midtrans Payment (Sandbox)

1. Create order
2. Use Midtrans Sandbox Simulator app or real QR scanner
3. Scan the displayed QR code
4. Complete payment in simulator
5. **Verify**: Real webhook received from Midtrans
6. **Verify**: Order updates to PAID automatically
7. **Verify**: Payment transaction record updated

---

## File Changes Summary

### Backend Files Modified

1. ✅ `backend/migrations/000020_add_qr_fields_to_payment_transactions.up.sql` - CREATED
2. ✅ `backend/order-service/src/models/payment_transaction.go` - UPDATED
3. ✅ `backend/order-service/src/config/midtrans.go` - UPDATED
4. ✅ `backend/order-service/src/services/payment_service.go` - UPDATED
5. ✅ `backend/order-service/src/repository/payment_repository.go` - UPDATED
6. ✅ `backend/order-service/api/checkout_handler.go` - UPDATED

### Frontend Files Modified

1. ✅ `frontend/src/types/cart.ts` - UPDATED
2. ✅ `frontend/src/services/order.ts` - UPDATED
3. ✅ `frontend/src/components/guest/OrderConfirmation.tsx` - UPDATED
4. ✅ `frontend/app/orders/[orderReference]/page.tsx` - UPDATED

---

## Known Limitations

1. **Migration Not Run**: Database schema changes require manual migration execution
2. **Webhook Signature**: Verify signature calculation matches Midtrans specification
3. **QR Code Expiry**: Frontend countdown is client-side only (not synced with server)
4. **Error Handling**: Limited retry logic for failed Midtrans API calls
5. **Logging**: Webhook events not yet logged to separate audit table

---

## Future Enhancements

1. **QR Code Refresh**: Allow manual QR code regeneration if expired
2. **Payment Status Polling**: Add fallback if webhook fails
3. **Multiple Payment Methods**: Support other Midtrans payment types
4. **Payment History**: Show all payment attempts for an order
5. **Admin Dashboard**: View pending QRIS payments and manual status updates
6. **Notification Service**: Send SMS/email when payment received
7. **Analytics**: Track payment success rate, expiry rate, average payment time

---

## Troubleshooting

### QR Code Not Displaying

- Check browser console for image load errors
- Verify `qr_code_url` in API response is valid
- Ensure Midtrans sandbox/production mode matches credentials
- Check CORS if loading from different domain

### Countdown Not Working

- Verify `expiry_time` is in ISO 8601 format
- Check browser console for JavaScript errors
- Ensure `paymentInfo` prop passed to OrderConfirmation
- Verify order status is "PENDING"

### Webhook Not Received

- Check `MIDTRANS_WEBHOOK_URL` is publicly accessible (use ngrok for local testing)
- Verify webhook URL registered in Midtrans dashboard
- Check firewall/security group rules
- Review webhook logs in Midtrans dashboard

### Payment Status Not Updating

- Verify webhook signature validation passes
- Check order-service logs for webhook processing errors
- Ensure database connection healthy
- Verify inventory-service is accessible for stock updates

---

## Documentation References

- **Midtrans QRIS API**: Contract at `specs/003-guest-qris-ordering/contracts/midtrans-generate-qris-api.yaml`
- **Payment Webhook**: Contract at `specs/003-guest-qris-ordering/contracts/payment-webhook.yaml`
- **Backend Conventions**: `docs/BACKEND_CONVENTIONS.md`
- **Frontend Conventions**: `docs/FRONTEND_CONVENTIONS.md`

---

## Deployment Checklist

### Pre-Deployment

- [ ] Run migration 000020 on staging database
- [ ] Set production `MIDTRANS_WEBHOOK_URL` in environment
- [ ] Switch to production Midtrans credentials
- [ ] Test webhook delivery to production URL
- [ ] Review and update CORS settings if needed

### Post-Deployment

- [ ] Monitor order-service logs for errors
- [ ] Verify QR code generation succeeds
- [ ] Test end-to-end flow in production
- [ ] Monitor webhook delivery success rate
- [ ] Set up alerts for failed payments

---

## Success Metrics

**Implementation Complete**: ✅ All code changes implemented  
**Type Safety**: ✅ No TypeScript/Go errors  
**API Contracts**: ✅ Matches Midtrans specifications  
**User Experience**: ✅ QR code display + countdown timer  
**Webhook Handling**: ✅ Settlement and expiry flows covered

**Ready for Testing**: ⚠️ Requires migration execution and environment setup

---

## Support Contacts

For issues or questions:

1. Check Midtrans documentation: https://docs.midtrans.com
2. Review implementation contracts in `specs/003-guest-qris-ordering/`
3. Check backend logs: `docker logs order-service`
4. Verify database state with queries above

---

_Last Updated: 2025-01-07_
_Implementation by: GitHub Copilot_
