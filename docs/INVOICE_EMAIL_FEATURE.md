# Invoice Email Feature - Implementation Summary

## Overview
Complete end-to-end implementation of an optional customer email field on the checkout page that sends invoice emails via Kafka event-driven architecture.

## Feature Status: ✅ WORKING

### Test Results (Order: GO-KTB44T)
- ✅ Email captured on checkout form: `test-customer@example.com`
- ✅ Email stored in `guest_orders.customer_email`
- ✅ Kafka event published to `notification-events` topic (478 bytes)
- ✅ Notification service received and processed event
- ✅ Email template rendered successfully
- ✅ Notification created with status: `sent`

## Implementation Details

### 1. Frontend Changes

**File: `frontend/src/types/checkout.ts`**
- Added `customer_email?: string` to `CheckoutData` interface

**File: `frontend/src/components/guest/CheckoutForm.tsx`**
- Added optional email input field with label: "(optional for invoice)"
- Email validation: `/^[^\s@]+@[^\s@]+\.[^\s@]+$/`
- Hint text: "We'll send your invoice to this email"
- Email included in checkout submission (trimmed and validated)

### 2. Database Changes

**Migration: `backend/migrations/000018_add_customer_email_to_guest_orders.up.sql`**
```sql
ALTER TABLE guest_orders ADD COLUMN customer_email VARCHAR(255);
CREATE INDEX idx_guest_orders_customer_email ON guest_orders(customer_email) 
  WHERE customer_email IS NOT NULL;
```

### 3. Backend - Order Service

**File: `backend/order-service/src/models/order.go`**
- Added `CustomerEmail *string` to `GuestOrder` struct

**File: `backend/order-service/api/checkout_handler.go`**
- Added `CustomerEmail *string` to `CheckoutRequest`
- Added `kafkaProducer` interface to `CheckoutHandler`
- Updated `insertOrder` query to include `customer_email` (13 parameters)
- Implemented `publishInvoiceEvent()` method:
  * Prepares order items array
  * Creates event with `event_type: "order.invoice"`
  * Publishes to Kafka with order reference as key
  * Logs success/failure with debug info

**File: `backend/order-service/src/queue/kafka.go`**
- Created `KafkaProducer` struct
- Implemented `NewKafkaProducer()` constructor
- Implemented `Publish()` method with debug logging
- Added comprehensive error handling and logging

**File: `backend/order-service/main.go`**
- Initialize Kafka producer with brokers from env (`KAFKA_BROKERS` or `localhost:9092`)
- Topic: `notification-events`
- Pass `kafkaProducer` to `NewCheckoutHandler`

**Dependencies Added:**
```bash
go get github.com/segmentio/kafka-go@v0.4.49
```

### 4. Backend - Notification Service

**File: `backend/notification-service/templates/order_invoice.html`**
- Professional HTML email template
- Header with order reference badge
- Customer information section (name, email, delivery type)
- Order items table (product name, quantity, unit price, total)
- Summary section (subtotal, delivery fee, total)
- "View Order Status" CTA button linking to order page
- Responsive design with blue theme (#4F46E5)
- Indonesian Rupiah currency formatting with dot separators (e.g., "27.000")

**File: `backend/notification-service/src/services/notification_service.go`**
- Added `"order_invoice.html"` to template files list
- Added `case "order.invoice"` to event handler switch
- Implemented `handleOrderInvoice()` method:
  * Extracts customer_email, customer_name, order_reference, delivery_type, amounts
  * Converts float64 amounts to int for formatting
  * Parses items array from event data
  * Creates `formatIDR` helper function for template
  * Renders template with order data
  * Creates notification record
  * Sends email via `sendEmail()`
- Implemented `formatCurrency()` helper:
  * Formats Indonesian Rupiah with thousand separators
  * Example: 27000 → "27.000"

## Event Structure

### Kafka Event: `order.invoice`
```json
{
  "event_type": "order.invoice",
  "tenant_id": "40c8c3dd-7024-4176-9bf4-4cc706d6a2c8",
  "user_id": "",
  "data": {
    "order_id": "367e12e2-8180-404e-8523-a6fce5204c7d",
    "order_reference": "GO-KTB44T",
    "customer_name": "John Doe",
    "customer_email": "test-customer@example.com",
    "delivery_type": "pickup",
    "subtotal_amount": 54000,
    "delivery_fee": 0,
    "total_amount": 54000,
    "items": [
      {
        "product_name": "Dimsum Mentai",
        "quantity": 2,
        "unit_price": 27000,
        "total_price": 54000
      }
    ],
    "created_at": "2025-12-08T23:25:46+07:00"
  }
}
```

### Notification Record
```sql
id: 1638d496-3c51-467c-8c73-a3b045169c51
type: email
subject: Order Invoice - GO-KTB44T
recipient: test-customer@example.com
status: sent
created_at: 2025-12-08 16:25:47.262081+00
```

## Data Flow

```
1. Customer enters email on checkout form (optional, validated if provided)
   ↓
2. Frontend validates email format (regex)
   ↓
3. POST /api/v1/public/:tenantId/checkout with customer_email
   ↓
4. Order service creates order with customer_email in guest_orders table
   ↓
5. Order service publishes "order.invoice" event to Kafka (notification-events topic)
   ↓
6. Notification service consumes event from Kafka
   ↓
7. handleOrderInvoice() processes event and renders order_invoice.html template
   ↓
8. Notification record created in database
   ↓
9. Email sent to customer with invoice details
```

## Configuration

### Environment Variables

**Order Service:**
- `KAFKA_BROKERS`: Kafka broker addresses (default: `localhost:9092`)

**Notification Service:**
- `KAFKA_BROKERS`: Kafka broker addresses (default: `localhost:9092`)
- `KAFKA_TOPIC`: Kafka topic name (default: `notification-events`)
- `KAFKA_GROUP_ID`: Consumer group ID (default: `notification-service`)

### Kafka Settings

**Topic:** `notification-events`
- Auto-creation: Enabled
- Replication factor: 1 (development)
- Partitions: 1 (development)

**Producer (Order Service):**
- Async: false (synchronous writes)
- Balancer: LeastBytes
- AllowAutoTopicCreation: true

**Consumer (Notification Service):**
- GroupID: notification-service
- StartOffset: LastOffset (only reads new messages after consumer start)
- CommitInterval: 1 second
- MinBytes: 100B
- MaxBytes: 10MB

## Verification Commands

### Check Email in Database
```bash
docker exec postgres-db psql -U pos_user -d pos_db -c \
  "SELECT order_reference, customer_email FROM guest_orders \
   WHERE customer_email IS NOT NULL ORDER BY created_at DESC LIMIT 5;"
```

### Check Notification
```bash
docker exec postgres-db psql -U pos_user -d pos_db -c \
  "SELECT id, subject, recipient, status, created_at FROM notifications \
   WHERE recipient = 'test-customer@example.com' ORDER BY created_at DESC LIMIT 1;"
```

### Check Order Service Logs
```bash
tail -f /tmp/order-service.log | grep -E "invoice|Kafka"
```

### Check Notification Service Logs
```bash
tail -f /tmp/notification-service.log | grep -E "order.invoice|invoice"
```

### Check Kafka Messages
```bash
docker exec pos-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic notification-events \
  --from-beginning \
  --max-messages 10
```

## Testing

### Manual Test
```bash
# Use the test script
bash /tmp/test-invoice-email.sh

# Or manually:
SESSION_ID="test_$(date +%s)"
TENANT_ID="40c8c3dd-7024-4176-9bf4-4cc706d6a2c8"

# 1. Add item to cart
curl -X POST "http://localhost:8080/api/v1/public/${TENANT_ID}/cart/items" \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: $SESSION_ID" \
  -d '{"product_id": "<PRODUCT_ID>", "quantity": 1}'

# 2. Checkout with email
curl -X POST "http://localhost:8080/api/v1/public/${TENANT_ID}/checkout" \
  -H "Content-Type: application/json" \
  -H "X-Session-ID: $SESSION_ID" \
  -d '{
    "customer_name": "Test Customer",
    "customer_email": "test@example.com",
    "delivery_type": "pickup"
  }'
```

## Known Limitations

1. **SMTP Configuration**: Emails are marked as "sent" but actual email delivery depends on SMTP configuration in notification service
2. **Consumer Offset**: Notification service starts reading from `LastOffset`, so it only processes events published after the consumer starts
3. **No Retry Logic**: Failed email sends are not automatically retried
4. **Guest Orders Only**: Feature currently only supports guest orders (not authenticated user orders)
5. **No Email Preview**: No endpoint to preview email template before sending

## Future Enhancements

1. **Email Preview Endpoint**: Add API endpoint to preview invoice email with sample data
2. **Retry Mechanism**: Implement exponential backoff retry for failed email sends
3. **Customer Preferences**: Add opt-in/opt-out functionality for invoice emails
4. **PDF Attachments**: Generate and attach PDF invoice to email
5. **Delivery Status Tracking**: Track email delivery status (opened, bounced, etc.)
6. **Multiple Email Templates**: Support different templates for different order states:
   - Order confirmation
   - Order cancelled
   - Order completed
   - Refund processed
7. **Authenticated Users**: Extend feature to support logged-in users
8. **Email Localization**: Support multiple languages based on customer preference
9. **Dead Letter Queue**: Handle failed events with DLQ for manual review

## Troubleshooting

### Email Not Stored in Database
- **Check**: Frontend validation not passing
- **Solution**: Verify email format matches regex: `/^[^\s@]+@[^\s@]+\.[^\s@]+$/`
- **Check**: Database migration not applied
- **Solution**: Run migration: `docker exec postgres-db psql -U pos_user -d pos_db < backend/migrations/000018_add_customer_email_to_guest_orders.up.sql`

### Kafka Event Not Published
- **Check**: Order service logs for errors
- **Solution**: Verify Kafka is running: `docker logs pos-kafka`
- **Check**: Kafka producer initialization
- **Solution**: Verify `KAFKA_BROKERS` env variable is set

### Notification Not Created
- **Check**: Notification service consuming from correct topic
- **Solution**: Verify topic name in env: `KAFKA_TOPIC=notification-events`
- **Check**: Notification service logs for errors
- **Solution**: Check template loading: `grep "Loaded template: order_invoice" /tmp/notification-service.log`

### Email Not Sent
- **Check**: SMTP configuration in notification service
- **Solution**: Verify SMTP env variables (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD)
- **Check**: Notification status in database
- **Solution**: If status is "failed", check notification service logs for SMTP errors

## Related Files

### Frontend
- `/frontend/src/types/checkout.ts`
- `/frontend/src/components/guest/CheckoutForm.tsx`

### Backend - Order Service
- `/backend/order-service/main.go`
- `/backend/order-service/api/checkout_handler.go`
- `/backend/order-service/src/models/order.go`
- `/backend/order-service/src/queue/kafka.go`
- `/backend/migrations/000018_add_customer_email_to_guest_orders.up.sql`

### Backend - Notification Service
- `/backend/notification-service/src/services/notification_service.go`
- `/backend/notification-service/templates/order_invoice.html`
- `/backend/notification-service/src/queue/kafka.go`

## Conclusion

The invoice email feature has been successfully implemented and tested end-to-end. All components are working correctly:
- Frontend email capture and validation
- Database storage of customer email
- Kafka event publishing from order service
- Kafka event consumption by notification service
- Email template rendering with proper formatting
- Notification record creation with "sent" status

The feature is production-ready with the caveat that SMTP configuration needs to be properly set up for actual email delivery to external addresses.

---

**Last Updated**: December 8, 2025  
**Test Order**: GO-KTB44T  
**Status**: ✅ Verified Working
