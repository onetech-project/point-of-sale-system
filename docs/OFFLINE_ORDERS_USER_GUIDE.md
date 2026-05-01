# Offline Orders User Guide

## Table of Contents

1. [Overview](#overview)
2. [Getting Started](#getting-started)
3. [Recording Basic Offline Orders](#recording-basic-offline-orders)
4. [Managing Payment Terms and Installments](#managing-payment-terms-and-installments)
5. [Editing Offline Orders](#editing-offline-orders)
6. [Deleting Offline Orders](#deleting-offline-orders)
7. [Viewing Analytics](#viewing-analytics)
8. [Troubleshooting](#troubleshooting)

---

## Overview

The Offline Orders feature allows your staff to record sales that happen outside your online system. This includes:

- **Cash transactions** at your physical store
- **Phone orders** taken by staff
- **In-person sales** without online payment
- **Partial payments** with installment plans
- **Customer orders with personalized terms**

### Key Features

✅ **Simple order recording** - Quick entry of customer details and order items  
✅ **Payment flexibility** - Accept full payment or create installment plans  
✅ **Edit capability** - Update pending orders with full audit trail  
✅ **Role-based deletion** - Only managers and owners can delete orders  
✅ **Analytics integration** - Track revenue and performance metrics  
✅ **Data privacy** - Customer PII encrypted and protected per regulations

---

## Getting Started

### Prerequisites

- Active staff account with appropriate role:
  - **Cashier/Staff**: Can create, view, and edit orders
  - **Manager/Owner**: All staff permissions + delete orders
- Access to POS system via API Gateway
- Authentication token (JWT) from login

### Required Information

When recording an offline order, you'll need:

1. **Customer Information** (required):
   - Full name (min 2 characters)
   - Phone number with country code (e.g., +6281234567890)
   - Email address (optional)

2. **Order Details**:
   - Delivery type: Pickup / Delivery / Dine-in
   - Table number (for dine-in orders)
   - Order items with prices
   - Special notes (optional)

3. **Data Consent** (mandatory):
   - Customer must give explicit consent for data collection
   - Consent method: Verbal / Written / Digital Signature
   - Required for Indonesian UU PDP compliance

4. **Payment Information** (optional):
   - Payment method: Cash / Bank Transfer / QRIS / Credit Card / etc.
   - Payment amount
   - Receipt number (if available)

---

## Recording Basic Offline Orders

### Scenario 1: Cash Payment at Store

**Use Case**: A customer walks into your coffee shop and pays cash for their order.

**Steps**:

1. **Prepare Customer Information**
   - Ask customer: "May I have your name and phone number for our records?"
   - Explain: "We need your consent to store this information per data protection laws."
   - Record consent method (usually "verbal" for in-person)

2. **Send API Request**

```bash
POST /offline-orders
Content-Type: application/json
Authorization: Bearer <your-jwt-token>
X-Tenant-ID: <your-tenant-id>
X-User-ID: <your-user-id>

{
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_name": "John Doe",
  "customer_phone": "+6281234567890",
  "customer_email": "john.doe@example.com",
  "delivery_type": "pickup",
  "notes": "Extra hot",
  "items": [
    {
      "product_id": "prod-001",
      "product_name": "Cappuccino Large",
      "quantity": 2,
      "unit_price": 45000,
      "subtotal": 90000
    },
    {
      "product_id": "prod-002",
      "product_name": "Croissant",
      "quantity": 1,
      "unit_price": 35000,
      "subtotal": 35000
    }
  ],
  "data_consent_given": true,
  "consent_method": "verbal",
  "recorded_by_user_id": "user-uuid-123",
  "payment": {
    "type": "full",
    "amount": 125000,
    "method": "cash"
  }
}
```

3. **Expected Response**

```json
{
  "id": "order-uuid-789",
  "order_reference": "GO-001234",
  "status": "PAID",
  "total_amount": 125000,
  "created_at": "2024-01-15T10:30:00Z"
}
```

4. **What Happens Next**
   - ✅ Order saved with encrypted customer PII
   - ✅ Order reference generated (GO-001234)
   - ✅ Status automatically set to "PAID"
   - ✅ Audit event published for compliance
   - ✅ Revenue metrics updated

### Scenario 2: Phone Order for Later Pickup

**Use Case**: A customer calls to place an order for pickup later today.

**Steps**:

1. Take customer details over the phone
2. Record consent: "I have your verbal consent to store this information, correct?"
3. Create order with `delivery_type: "pickup"` and omit `payment` section
4. Order status will be "PENDING" until customer pays at pickup

```json
{
  "delivery_type": "pickup",
  "notes": "Customer will pick up at 3 PM",
  "data_consent_given": true,
  "consent_method": "verbal"
  // No payment section - order stays PENDING
}
```

---

## Managing Payment Terms and Installments

### Scenario 3: Installment Payment Plan

**Use Case**: A customer wants to buy an expensive coffee maker but pay in monthly installments.

**Business Rules**:

- Down payment: 30-50% recommended
- Installment period: 2-12 months
- Each installment must be equal amount
- Customer must agree to schedule

**Steps**:

1. **Calculate Payment Terms**
   - Total price: IDR 3,000,000
   - Down payment (30%): IDR 1,000,000
   - Remaining: IDR 2,000,000
   - Installments: 4 months × IDR 500,000

2. **Create Order with Payment Plan**

```bash
POST /offline-orders
```

```json
{
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_name": "Jane Smith",
  "customer_phone": "+6281234567891",
  "customer_email": "jane@example.com",
  "delivery_type": "delivery",
  "notes": "Deliver to office address",
  "items": [
    {
      "product_id": "prod-coffee-maker-01",
      "product_name": "Premium Coffee Maker Deluxe",
      "quantity": 1,
      "unit_price": 3000000,
      "subtotal": 3000000
    }
  ],
  "data_consent_given": true,
  "consent_method": "written",
  "recorded_by_user_id": "user-uuid-456",
  "payment": {
    "type": "installment",
    "down_payment_amount": 1000000,
    "down_payment_method": "bank_transfer",
    "installment_count": 4,
    "installment_amount": 500000,
    "payment_schedule": [
      {
        "installment_number": 1,
        "due_date": "2024-02-15",
        "amount": 500000,
        "status": "pending"
      },
      {
        "installment_number": 2,
        "due_date": "2024-03-15",
        "amount": 500000,
        "status": "pending"
      },
      {
        "installment_number": 3,
        "due_date": "2024-04-15",
        "amount": 500000,
        "status": "pending"
      },
      {
        "installment_number": 4,
        "due_date": "2024-05-15",
        "amount": 500000,
        "status": "pending"
      }
    ]
  }
}
```

3. **Expected Response**

```json
{
  "id": "order-uuid-890",
  "order_reference": "GO-001235",
  "status": "PENDING",
  "total_amount": 3000000,
  "payment_terms": {
    "remaining_balance": 2000000,
    "next_due_date": "2024-02-15"
  }
}
```

### Recording Installment Payments

When the customer makes an installment payment:

```bash
POST /offline-orders/order-uuid-890/payments
```

```json
{
  "amount_paid": 500000,
  "payment_method": "bank_transfer",
  "notes": "Payment for installment #1",
  "receipt_number": "RCPT-2024-001"
}
```

**System automatically**:

- Updates remaining balance: IDR 1,500,000 → IDR 1,000,000
- Marks installment #1 as "paid"
- Updates next due date to February 15th
- When last payment made, changes status to "PAID"

### Viewing Payment History

Check all payments for an order:

```bash
GET /offline-orders/order-uuid-890/payments
```

```json
{
  "payments": [
    {
      "payment_number": 0,
      "amount_paid": 1000000,
      "payment_method": "bank_transfer",
      "payment_date": "2024-01-15T10:30:00Z",
      "remaining_balance_after": 2000000,
      "notes": "Down payment"
    },
    {
      "payment_number": 1,
      "amount_paid": 500000,
      "payment_method": "bank_transfer",
      "payment_date": "2024-02-15T14:30:00Z",
      "remaining_balance_after": 1500000,
      "notes": "Payment for installment #1"
    }
  ],
  "payment_terms": {
    "remaining_balance": 1500000,
    "next_due_date": "2024-03-15"
  }
}
```

---

## Editing Offline Orders

### When Can You Edit?

✅ **Can edit**: Orders with status `PENDING`  
❌ **Cannot edit**: Orders with status `PAID`, `COMPLETE`, or `CANCELLED`

**Why?** Once payment is received, order becomes financial record that cannot be altered for accounting compliance.

### Scenario 4: Customer Changes Order Before Pickup

**Use Case**: Customer calls to change quantity from 1 to 2 coffee makers.

**Steps**:

```bash
PATCH /offline-orders/order-uuid-890
Content-Type: application/json
X-User-ID: user-uuid-789
```

```json
{
  "items": [
    {
      "product_id": "prod-coffee-maker-01",
      "product_name": "Premium Coffee Maker Deluxe",
      "quantity": 2,
      "unit_price": 3000000,
      "subtotal": 6000000
    }
  ],
  "notes": "Customer increased order to 2 units"
}
```

**What Changes Are Tracked?**

The system records:

- **Before**: quantity = 1, total = IDR 3,000,000
- **After**: quantity = 2, total = IDR 6,000,000
- **Modified by**: user-uuid-789
- **Modified at**: 2024-01-15T11:00:00Z

**Audit Trail**: All changes publish `offline_order.updated` event with complete diff:

```json
{
  "changes": {
    "items": {
      "old": [{ "quantity": 1, "subtotal": 3000000 }],
      "new": [{ "quantity": 2, "subtotal": 6000000 }]
    },
    "total_amount": {
      "old": 3000000,
      "new": 6000000
    }
  }
}
```

### Common Edit Scenarios

**Change customer phone number**:

```json
{
  "customer_phone": "+6281234567899"
}
```

**Update delivery type from pickup to delivery**:

```json
{
  "delivery_type": "delivery",
  "notes": "Customer requested home delivery"
}
```

**Add table number for dine-in**:

```json
{
  "table_number": "A5"
}
```

---

## Deleting Offline Orders

### Who Can Delete?

🔐 **Authorization Required**:

- **Owner role**: Can delete any offline order
- **Manager role**: Can delete any offline order
- **Staff/Cashier role**: ❌ Cannot delete orders

### When Can You Delete?

✅ **Can delete**: Orders with status `PENDING` or `CANCELLED`  
❌ **Cannot delete**: Orders with status `PAID` or `COMPLETE`

**Why?** Paid orders are financial records that must be retained for accounting and tax compliance.

### Scenario 5: Customer Cancels Order

**Use Case**: Customer calls to cancel their pending order.

**Steps**:

```bash
DELETE /offline-orders/order-uuid-890?reason=Customer%20requested%20cancellation
Authorization: Bearer <manager-or-owner-jwt-token>
X-User-ID: manager-user-id
X-User-Role: manager
```

**Query Parameters**:

- `reason` (required): Explanation for deletion (5-500 characters)

**Expected Response**: `204 No Content`

**What Happens**:

- ✅ Order marked as deleted (soft delete)
- ✅ `deleted_at` timestamp recorded
- ✅ `deleted_by_user_id` stored for audit
- ✅ Deletion reason saved
- ✅ Event published: `offline_order.deleted`
- ❌ Order NOT removed from database (compliance)

### Viewing Deleted Orders

Deleted orders are excluded from default list queries but can be retrieved with special filters in database for audit purposes.

**Audit Query Example**:

```sql
SELECT * FROM guest_orders
WHERE deleted_at IS NOT NULL
  AND tenant_id = 'your-tenant-id'
ORDER BY deleted_at DESC;
```

---

## Viewing Analytics

### Dashboard Metrics

Access Grafana dashboard: **Offline Orders Dashboard**

**Available Metrics**:

1. **Total Offline Orders** (gauge)
   - Count of pending orders
   - Real-time updates

2. **Total Revenue** (gauge)
   - Cumulative revenue from offline orders
   - Displayed in IDR

3. **Order Creation Rate** (time series)
   - Orders per 5-minute window
   - Grouped by status (PENDING/PAID)

4. **Creation Duration** (histogram)
   - p95 and p99 latency percentiles
   - Identifies performance issues

5. **Payments by Method** (stacked bar)
   - Distribution: Cash / Bank Transfer / QRIS / etc.
   - Helps plan payment acceptance

6. **Installment Plans** (bar chart)
   - Distribution by installment count (2/3/4/6/12 months)
   - Identifies popular payment terms

7. **Order Updates** (gauge)
   - Total edit operations
   - Monitors order modification frequency

8. **Order Deletions** (gauge + time series)
   - Total deletions with role breakdown
   - Monitors deletion patterns by manager/owner

### Using Prometheus Queries

**Query Examples**:

**Total pending orders**:

```promql
sum(offline_orders_total{status="PENDING"})
```

**Revenue by tenant**:

```promql
sum by(tenant_id) (offline_order_revenue)
```

**Payment methods distribution**:

```promql
sum by(payment_method) (offline_order_payments_total)
```

**Average order creation time (last hour)**:

```promql
rate(offline_order_creation_duration_seconds_sum[1h])
/
rate(offline_order_creation_duration_seconds_count[1h])
```

---

## Troubleshooting

### Common Issues

#### Issue 1: "Data consent is required"

**Error**: `422 Unprocessable Entity - data consent is required for offline orders`

**Cause**: Missing `data_consent_given: true` or `consent_method` in request.

**Solution**:

```json
{
  "data_consent_given": true,
  "consent_method": "verbal" // or "written" or "digital_signature"
}
```

**Prevention**: Always ask customer for consent before recording order.

---

#### Issue 2: "Cannot edit order with status PAID"

**Error**: `403 Forbidden - cannot edit order with status PAID`

**Cause**: Attempting to edit an order that has already been paid.

**Solution**: Paid orders cannot be edited for compliance reasons. If customer needs changes:

1. Create a new order for additional items
2. Process refund separately outside system
3. Contact manager/owner for special handling

---

#### Issue 3: "User does not have owner/manager role"

**Error**: `403 Forbidden - insufficient permissions to delete orders`

**Cause**: Staff or cashier attempting to delete order.

**Solution**: Only managers and owners can delete orders. Contact your manager if order must be canceled.

---

#### Issue 4: "Payment amount exceeds remaining balance"

**Error**: `400 Bad Request - payment amount exceeds remaining balance`

**Cause**: Recording installment payment larger than remaining balance.

**Example**:

- Remaining balance: IDR 500,000
- Payment attempt: IDR 600,000

**Solution**: Check payment history first:

```bash
GET /offline-orders/<order-id>/payments
```

Record correct amount:

```json
{
  "amount_paid": 500000 // Maximum remaining balance
}
```

---

#### Issue 5: Slow Performance Creating Orders

**Symptoms**: Order creation takes >5 seconds

**Possible Causes**:

1. Database connection pool exhausted
2. Vault encryption service slow
3. Network latency to external services

**Solution**:

1. Check database indexes exist (run migration 000064)
2. Verify Vault service health and encryption key caching
3. Review Grafana dashboard → "Order Creation Duration" panel
4. Check OpenTelemetry traces for bottlenecks

**Performance Optimization Checklist**:

- ✅ Database indexes installed
- ✅ Encryption key caching enabled (5-min TTL)
- ✅ Connection pooling configured
- ✅ Rate limiting not exceeded (100 req/min)

---

### Getting Help

**Contact Support**:

- Email: support@yourpos.com
- Slack: #offline-orders-support
- Documentation: https://docs.yourpos.com

**Before Contacting Support**:

1. Check error message in API response
2. Verify user role and permissions
3. Review order status (can only edit PENDING orders)
4. Check audit logs for recent changes
5. Test with minimal request (remove optional fields)

**Include in Support Request**:

- Order ID or order reference (GO-XXXXXX)
- Tenant ID
- User ID attempting operation
- Full error message from API response
- Timestamp of issue
- Steps to reproduce

---

## Best Practices

### 1. Data Consent

Always explain to customers why you need their information:

> "We're required to collect your name and phone number per Indonesian data protection laws. Do we have your consent to store this information securely?"

### 2. Payment Documentation

- Record receipt numbers for all payments
- Take photos of payment confirmations
- Note payment method accurately (cash vs transfer)

### 3. Order Accuracy

Double-check order details before submitting:

```
✅ Customer phone number correct?
✅ Delivery type accurate?
✅ Order items match customer request?
✅ Payment amount matches total?
```

### 4. Edit Audit Trail

When editing orders, always add meaningful notes:

```json
{
  "notes": "Customer requested extra hot - updated at 2PM on customer callback"
}
```

### 5. Deletion Reasons

Provide clear, specific deletion reasons:

- ❌ "mistake"
- ❌ "customer cancel"
- ✅ "Customer called at 2:15 PM to cancel order due to schedule conflict"
- ✅ "Duplicate order - customer already paid via online system (ref: GO-001240)"

### 6. Installment Plans

Document customer agreement:

- Save signed installment agreement (PDF/photo)
- Note agreement method in order notes
- Remind customer of due dates
- Set calendar reminders for payment follow-ups

---

## Appendix: Quick Reference

### API Endpoints

| Method | Endpoint                     | Purpose                   | Role Required  |
| ------ | ---------------------------- | ------------------------- | -------------- |
| POST   | /offline-orders              | Create order              | Staff, Cashier |
| GET    | /offline-orders              | List orders               | Staff, Cashier |
| GET    | /offline-orders/:id          | Get order details         | Staff, Cashier |
| PATCH  | /offline-orders/:id          | Edit order (PENDING only) | Staff, Cashier |
| DELETE | /offline-orders/:id          | Delete order              | Manager, Owner |
| POST   | /offline-orders/:id/payments | Record payment            | Staff, Cashier |
| GET    | /offline-orders/:id/payments | Get payment history       | Staff, Cashier |

### Order Status Lifecycle

```
PENDING → (full payment) → PAID → COMPLETE
         ↓
      CANCELLED (manual cancellation)
```

### Payment Methods

- `cash` - Cash payment
- `bank_transfer` - Bank transfer
- `qris` - QRIS payment
- `credit_card` - Credit card
- `debit_card` - Debit card
- `e_wallet` - E-wallet (GoPay, OVO, Dana, etc.)

### Delivery Types

- `pickup` - Customer picks up order
- `delivery` - Order delivered to customer
- `dine_in` - Customer dines in restaurant

### Consent Methods

- `verbal` - Customer gave verbal consent (in-person/phone)
- `written` - Customer signed physical form
- `digital_signature` - Customer signed via digital signature

---

**Version**: 1.0  
**Last Updated**: January 2024  
**Feedback**: Send suggestions to docs@yourpos.com
