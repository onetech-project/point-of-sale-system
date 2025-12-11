# Quickstart Guide: Order Email Notifications

**Feature**: 004-order-email-notifications  
**Date**: 2025-12-11  
**Audience**: Developers implementing this feature

## Overview

This guide provides step-by-step instructions to implement email notifications for paid orders. Follow the phases sequentially for a test-driven, incremental approach.

---

## Prerequisites

- Docker and docker-compose installed
- Go 1.21+ installed
- PostgreSQL client (psql) installed
- Access to project repositories
- SMTP server credentials (or use Mailhog for development)
- Kafka running (via docker-compose)

---

## Phase 1: Database Schema (30 minutes)

### 1.1 Create Migration Files

```bash
cd backend/migrations

# Create migration for user notification preferences
cat > 000023_add_order_notification_prefs.up.sql << 'EOF'
-- Add notification preference to users table
ALTER TABLE users 
ADD COLUMN receive_order_notifications BOOLEAN DEFAULT false;

-- Index for efficient queries
CREATE INDEX idx_users_order_notifications 
ON users(tenant_id, receive_order_notifications) 
WHERE receive_order_notifications = true;

COMMENT ON COLUMN users.receive_order_notifications IS 'Whether user receives email notifications for paid orders';
EOF

cat > 000023_add_order_notification_prefs.down.sql << 'EOF'
DROP INDEX IF EXISTS idx_users_order_notifications;
ALTER TABLE users DROP COLUMN IF EXISTS receive_order_notifications;
EOF

# Create migration for notification configs
cat > 000024_create_notification_configs.up.sql << 'EOF'
-- Tenant-level notification configuration
CREATE TABLE IF NOT EXISTS notification_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    order_notifications_enabled BOOLEAN NOT NULL DEFAULT true,
    test_mode BOOLEAN NOT NULL DEFAULT false,
    test_email VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

CREATE INDEX idx_notification_configs_tenant ON notification_configs(tenant_id);

COMMENT ON TABLE notification_configs IS 'Tenant-level notification behavior settings';
COMMENT ON COLUMN notification_configs.test_mode IS 'If true, all emails go to test_email only';
EOF

cat > 000024_create_notification_configs.down.sql << 'EOF'
DROP TABLE IF EXISTS notification_configs;
EOF

# Create migration for improved notification queries
cat > 000025_add_notification_indexes.up.sql << 'EOF'
-- GIN index for JSONB queries on transaction_id
CREATE INDEX idx_notifications_order_metadata 
ON notifications USING GIN (metadata jsonb_path_ops)
WHERE event_type LIKE 'order.paid.%';

COMMENT ON INDEX idx_notifications_order_metadata IS 'Fast lookup for order notifications by transaction_id';
EOF

cat > 000025_add_notification_indexes.down.sql << 'EOF'
DROP INDEX IF EXISTS idx_notifications_order_metadata;
EOF
```

### 1.2 Apply Migrations

```bash
# Run migrations
docker-compose exec postgres psql -U postgres -d pos_db -f /migrations/000023_add_order_notification_prefs.up.sql
docker-compose exec postgres psql -U postgres -d pos_db -f /migrations/000024_create_notification_configs.up.sql
docker-compose exec postgres psql -U postgres -d pos_db -f /migrations/000025_add_notification_indexes.up.sql

# Verify migrations
docker-compose exec postgres psql -U postgres -d pos_db -c "\d+ users" | grep receive_order_notifications
docker-compose exec postgres psql -U postgres -d pos_db -c "\d notification_configs"
```

### 1.3 Write Migration Tests

```bash
cd backend/migrations
mkdir -p tests

cat > tests/migration_test.go << 'EOF'
package tests

import (
    "database/sql"
    "testing"
    _ "github.com/lib/pq"
    "github.com/stretchr/testify/assert"
)

func TestMigration023_UserNotificationPrefs(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // Verify column exists
    var exists bool
    err := db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='users' AND column_name='receive_order_notifications'
        )
    `).Scan(&exists)
    
    assert.NoError(t, err)
    assert.True(t, exists, "Column receive_order_notifications should exist")
    
    // Verify index exists
    err = db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM pg_indexes 
            WHERE indexname='idx_users_order_notifications'
        )
    `).Scan(&exists)
    
    assert.NoError(t, err)
    assert.True(t, exists, "Index idx_users_order_notifications should exist")
}
EOF
```

---

## Phase 2: Email Templates (45 minutes)

### 2.1 Create Staff Notification Template

```bash
cd backend/notification-service/templates

cat > order_staff_notification.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>New Order Notification</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f9f9f9; padding: 20px; border: 1px solid #ddd; border-top: none; }
        .order-info { background-color: white; padding: 15px; margin: 15px 0; border-radius: 5px; border: 1px solid #e0e0e0; }
        .order-info h3 { margin-top: 0; color: #4CAF50; }
        .items-table { width: 100%; border-collapse: collapse; margin: 15px 0; }
        .items-table th { background-color: #f0f0f0; padding: 10px; text-align: left; border-bottom: 2px solid #ddd; }
        .items-table td { padding: 10px; border-bottom: 1px solid #eee; }
        .total-row { font-weight: bold; background-color: #f9f9f9; }
        .footer { background-color: #f0f0f0; padding: 15px; text-align: center; font-size: 12px; color: #666; border-radius: 0 0 5px 5px; }
        .cta-button { display: inline-block; padding: 12px 30px; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 5px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🛒 New Order Paid!</h1>
    </div>
    <div class="content">
        <p><strong>Order Reference:</strong> {{.OrderReference}}</p>
        <p><strong>Paid At:</strong> {{.PaidAt}}</p>
        
        <div class="order-info">
            <h3>Customer Information</h3>
            <p><strong>Name:</strong> {{.CustomerName}}</p>
            <p><strong>Phone:</strong> {{.CustomerPhone}}</p>
            {{if .CustomerEmail}}<p><strong>Email:</strong> {{.CustomerEmail}}</p>{{end}}
            <p><strong>Delivery Type:</strong> {{.DeliveryType}}</p>
            {{if eq .DeliveryType "delivery"}}<p><strong>Address:</strong> {{.DeliveryAddress}}</p>{{end}}
            {{if .TableNumber}}<p><strong>Table:</strong> {{.TableNumber}}</p>{{end}}
        </div>
        
        <h3>Order Items</h3>
        <table class="items-table">
            <thead>
                <tr>
                    <th>Item</th>
                    <th style="text-align: center;">Qty</th>
                    <th style="text-align: right;">Price</th>
                    <th style="text-align: right;">Total</th>
                </tr>
            </thead>
            <tbody>
                {{range .Items}}
                <tr>
                    <td>{{.ProductName}}</td>
                    <td style="text-align: center;">{{.Quantity}}</td>
                    <td style="text-align: right;">Rp {{formatCurrency .UnitPrice}}</td>
                    <td style="text-align: right;">Rp {{formatCurrency .TotalPrice}}</td>
                </tr>
                {{end}}
                <tr>
                    <td colspan="3" style="text-align: right;"><strong>Subtotal:</strong></td>
                    <td style="text-align: right;"><strong>Rp {{formatCurrency .SubtotalAmount}}</strong></td>
                </tr>
                {{if gt .DeliveryFee 0}}
                <tr>
                    <td colspan="3" style="text-align: right;"><strong>Delivery Fee:</strong></td>
                    <td style="text-align: right;"><strong>Rp {{formatCurrency .DeliveryFee}}</strong></td>
                </tr>
                {{end}}
                <tr class="total-row">
                    <td colspan="3" style="text-align: right;"><strong>TOTAL PAID:</strong></td>
                    <td style="text-align: right;"><strong>Rp {{formatCurrency .TotalAmount}}</strong></td>
                </tr>
            </tbody>
        </table>
        
        <p><strong>Payment Method:</strong> {{.PaymentMethod}}</p>
        <p><strong>Transaction ID:</strong> {{.TransactionID}}</p>
        
        <div style="text-align: center;">
            <a href="{{.FrontendURL}}/admin/orders/{{.OrderReference}}" class="cta-button">View Order in Dashboard</a>
        </div>
    </div>
    <div class="footer">
        <p>This is an automated notification from your POS system.</p>
        <p>To manage notification preferences, visit Admin → Settings → Notifications</p>
    </div>
</body>
</html>
EOF
```

### 2.2 Update Invoice Template with Watermark

```bash
cd backend/notification-service/templates

# Backup existing template
cp order_invoice.html order_invoice.html.backup

# Add watermark support (modify existing template)
# Add this CSS to the <style> section:
cat >> order_invoice.html << 'EOF'
.watermark {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%) rotate(-45deg);
    font-size: 120px;
    color: rgba(0, 128, 0, 0.15);
    font-weight: bold;
    pointer-events: none;
    z-index: 1000;
    text-align: center;
    line-height: 1;
}
EOF

# Add this to the <body> section (after opening tag):
cat >> order_invoice.html << 'EOF'
{{if .ShowPaidWatermark}}
<div class="watermark">PAID</div>
{{end}}
EOF
```

### 2.3 Template Testing

```bash
cd backend/notification-service/src/services

cat > notification_service_test.go << 'EOF'
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestRenderStaffNotificationTemplate(t *testing.T) {
    service := NewNotificationService(nil) // Mock DB
    
    data := StaffNotificationData{
        OrderReference: "ORD-001",
        CustomerName: "Test Customer",
        Items: []OrderItem{{ProductName: "Test", Quantity: 1, UnitPrice: 10000}},
    }
    
    html, err := service.renderTemplate("order_staff_notification", data)
    assert.NoError(t, err)
    assert.Contains(t, html, "ORD-001")
    assert.Contains(t, html, "Test Customer")
}

func TestRenderInvoiceWithWatermark(t *testing.T) {
    service := NewNotificationService(nil)
    
    data := InvoiceData{
        ShowPaidWatermark: true,
        OrderReference: "ORD-001",
    }
    
    html, err := service.renderTemplate("order_invoice", data)
    assert.NoError(t, err)
    assert.Contains(t, html, "PAID")
    assert.Contains(t, html, "watermark")
}
EOF
```

---

## Phase 3: Event Handler (1 hour)

### 3.1 Add Order Paid Event Handler

```bash
cd backend/notification-service/src/services

# Edit notification_service.go - add to HandleEvent switch statement:
cat >> notification_service.go << 'EOF'
case "order.paid":
    return s.handleOrderPaid(ctx, event)
EOF

# Implement handler:
cat >> notification_service.go << 'EOF'
func (s *NotificationService) handleOrderPaid(ctx context.Context, event models.NotificationEvent) error {
    log.Printf("Processing order.paid event for tenant: %s", event.TenantID)
    
    // 1. Check for duplicate
    transactionID := event.Metadata["transaction_id"].(string)
    isDuplicate, err := s.hasSentOrderNotification(ctx, event.TenantID, transactionID)
    if err != nil {
        return fmt.Errorf("duplicate check failed: %w", err)
    }
    if isDuplicate {
        log.Printf("Duplicate order.paid notification detected for transaction: %s", transactionID)
        return nil // Skip processing
    }
    
    // 2. Send staff notifications
    if err := s.sendStaffNotifications(ctx, event); err != nil {
        log.Printf("Error sending staff notifications: %v", err)
        // Don't return error - continue to customer receipt
    }
    
    // 3. Send customer receipt
    if customerEmail, ok := event.Metadata["customer_email"].(string); ok && customerEmail != "" {
        if err := s.sendCustomerReceipt(ctx, event); err != nil {
            log.Printf("Error sending customer receipt: %v", err)
        }
    }
    
    return nil
}
EOF
```

### 3.2 Implement Helper Functions

```bash
cat >> notification_service.go << 'EOF'
func (s *NotificationService) hasSentOrderNotification(ctx context.Context, tenantID, transactionID string) (bool, error) {
    query := `
        SELECT EXISTS(
            SELECT 1 FROM notifications 
            WHERE tenant_id = $1 
              AND event_type LIKE 'order.paid.%'
              AND metadata->>'transaction_id' = $2
              AND status IN ('sent', 'pending')
        )
    `
    var exists bool
    err := s.repo.db.QueryRowContext(ctx, query, tenantID, transactionID).Scan(&exists)
    return exists, err
}

func (s *NotificationService) sendStaffNotifications(ctx context.Context, event models.NotificationEvent) error {
    // Get staff members with notifications enabled
    staffList, err := s.getStaffForNotifications(ctx, event.TenantID)
    if err != nil {
        return fmt.Errorf("failed to get staff list: %w", err)
    }
    
    if len(staffList) == 0 {
        log.Printf("No staff members configured for order notifications (tenant: %s)", event.TenantID)
        return nil
    }
    
    // Prepare template data
    data := s.prepareStaffNotificationData(event)
    
    // Send to each staff member
    for _, staff := range staffList {
        notification := models.Notification{
            TenantID:  event.TenantID,
            UserID:    &staff.ID,
            Type:      "email",
            Status:    "pending",
            EventType: "order.paid.staff",
            Subject:   fmt.Sprintf("New Order Paid: %s", event.Metadata["order_reference"]),
            Recipient: staff.Email,
            Metadata:  event.Metadata,
        }
        
        // Render template
        body, err := s.renderTemplate("order_staff_notification", data)
        if err != nil {
            log.Printf("Failed to render template for staff %s: %v", staff.Email, err)
            continue
        }
        notification.Body = body
        
        // Save to database
        if err := s.repo.Create(ctx, &notification); err != nil {
            log.Printf("Failed to save notification for staff %s: %v", staff.Email, err)
            continue
        }
        
        // Send email
        if err := s.emailProvider.Send(staff.Email, notification.Subject, body, true); err != nil {
            log.Printf("Failed to send email to staff %s: %v", staff.Email, err)
            s.repo.UpdateStatus(ctx, notification.ID, "failed", err.Error())
            continue
        }
        
        s.repo.UpdateStatus(ctx, notification.ID, "sent", "")
        log.Printf("Sent order notification to staff: %s", staff.Email)
    }
    
    return nil
}

func (s *NotificationService) sendCustomerReceipt(ctx context.Context, event models.NotificationEvent) error {
    customerEmail := event.Metadata["customer_email"].(string)
    
    // Prepare invoice data with watermark
    data := s.prepareInvoiceData(event, true) // true = show watermark
    
    notification := models.Notification{
        TenantID:  event.TenantID,
        Type:      "email",
        Status:    "pending",
        EventType: "order.paid.customer",
        Subject:   fmt.Sprintf("Your Order Receipt: %s", event.Metadata["order_reference"]),
        Recipient: customerEmail,
        Metadata:  event.Metadata,
    }
    
    // Render invoice template
    body, err := s.renderTemplate("order_invoice", data)
    if err != nil {
        return fmt.Errorf("failed to render invoice: %w", err)
    }
    notification.Body = body
    
    // Save to database
    if err := s.repo.Create(ctx, &notification); err != nil {
        return fmt.Errorf("failed to save notification: %w", err)
    }
    
    // Send email
    if err := s.emailProvider.Send(customerEmail, notification.Subject, body, true); err != nil {
        s.repo.UpdateStatus(ctx, notification.ID, "failed", err.Error())
        return fmt.Errorf("failed to send email: %w", err)
    }
    
    s.repo.UpdateStatus(ctx, notification.ID, "sent", "")
    log.Printf("Sent order receipt to customer: %s", customerEmail)
    
    return nil
}

func (s *NotificationService) getStaffForNotifications(ctx context.Context, tenantID string) ([]StaffMember, error) {
    query := `
        SELECT id, email, name, role 
        FROM users 
        WHERE tenant_id = $1 
          AND receive_order_notifications = true 
          AND email IS NOT NULL 
          AND email != ''
        ORDER BY role, name
    `
    
    rows, err := s.repo.db.QueryContext(ctx, query, tenantID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var staff []StaffMember
    for rows.Next() {
        var member StaffMember
        if err := rows.Scan(&member.ID, &member.Email, &member.Name, &member.Role); err != nil {
            return nil, err
        }
        staff = append(staff, member)
    }
    
    return staff, rows.Err()
}
EOF
```

### 3.3 Write Event Handler Tests

```bash
cat > notification_service_order_test.go << 'EOF'
package services

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestHandleOrderPaid_Success(t *testing.T) {
    // Setup mocks
    mockDB := new(MockDatabase)
    mockEmail := new(MockEmailProvider)
    service := NewNotificationService(mockDB)
    service.emailProvider = mockEmail
    
    // Test data
    event := createTestOrderPaidEvent()
    
    // Expectations
    mockDB.On("QueryRowContext", mock.Anything, mock.Anything, mock.Anything).
        Return(mockRows(false)) // Not duplicate
    mockDB.On("QueryContext", mock.Anything, mock.Anything, mock.Anything).
        Return(mockStaffRows(), nil) // 2 staff members
    mockEmail.On("Send", mock.Anything, mock.Anything, mock.Anything, true).
        Return(nil) // Success
    
    // Execute
    err := service.handleOrderPaid(context.Background(), event)
    
    // Assert
    assert.NoError(t, err)
    mockEmail.AssertNumberOfCalls(t, "Send", 3) // 2 staff + 1 customer
}

func TestHandleOrderPaid_Duplicate(t *testing.T) {
    mockDB := new(MockDatabase)
    service := NewNotificationService(mockDB)
    
    event := createTestOrderPaidEvent()
    
    // Mock duplicate detection
    mockDB.On("QueryRowContext", mock.Anything, mock.Anything, mock.Anything).
        Return(mockRows(true)) // Is duplicate
    
    err := service.handleOrderPaid(context.Background(), event)
    
    assert.NoError(t, err)
    // Should not call Send
}
EOF
```

---

## Phase 4: Order Service Integration (45 minutes)

### 4.1 Publish Event on Order Paid

```bash
cd backend/order-service/src/services

# Edit order_service.go - find where order status updates to PAID:
cat >> order_service.go << 'EOF'
func (s *OrderService) UpdateOrderStatusToPaid(ctx context.Context, orderID, transactionID string) error {
    // Existing logic to update order status...
    
    // Get full order details
    order, err := s.repo.GetByID(ctx, orderID)
    if err != nil {
        return fmt.Errorf("failed to get order: %w", err)
    }
    
    // Publish event
    event := s.buildOrderPaidEvent(order, transactionID)
    if err := s.publishEvent(ctx, "order.paid", event); err != nil {
        log.Printf("Failed to publish order.paid event: %v", err)
        // Don't fail the order update - event publishing is best-effort
    }
    
    return nil
}

func (s *OrderService) buildOrderPaidEvent(order *models.Order, transactionID string) map[string]interface{} {
    return map[string]interface{}{
        "order_id":         order.ID,
        "order_reference":  order.OrderReference,
        "transaction_id":   transactionID,
        "customer_name":    order.CustomerName,
        "customer_phone":   order.CustomerPhone,
        "customer_email":   order.CustomerEmail,
        "delivery_type":    order.DeliveryType,
        "delivery_address": order.DeliveryAddress,
        "table_number":     order.TableNumber,
        "items":            s.formatOrderItems(order.Items),
        "subtotal_amount":  order.SubtotalAmount,
        "delivery_fee":     order.DeliveryFee,
        "total_amount":     order.TotalAmount,
        "payment_method":   "QRIS",
        "paid_at":          order.PaidAt,
    }
}
EOF
```

### 4.2 Write Integration Test

```bash
cat > order_service_event_test.go << 'EOF'
package services

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestUpdateOrderStatusToPaid_PublishesEvent(t *testing.T) {
    // Setup
    mockKafka := new(MockKafkaProducer)
    service := NewOrderService(mockDB, mockKafka)
    
    // Test
    err := service.UpdateOrderStatusToPaid(ctx, testOrderID, "txn-123")
    
    // Assert
    assert.NoError(t, err)
    mockKafka.AssertCalled(t, "Publish", "notification-events", mock.MatchedBy(func(event map[string]interface{}) bool {
        return event["event_type"] == "order.paid" &&
               event["metadata"].(map[string]interface{})["transaction_id"] == "txn-123"
    }))
}
EOF
```

---

## Phase 5: Admin API (1 hour)

### 5.1 Create User Notification Preferences API

```bash
cd backend/user-service/api

cat > notification_preferences.go << 'EOF'
package api

import (
    "net/http"
    "github.com/labstack/echo/v4"
    "github.com/point-of-sale-system/user-service/src/services"
)

type NotificationPreferencesHandler struct {
    service *services.UserService
}

func NewNotificationPreferencesHandler(service *services.UserService) *NotificationPreferencesHandler {
    return &NotificationPreferencesHandler{service: service}
}

// GET /api/v1/users/notification-preferences
func (h *NotificationPreferencesHandler) List(c echo.Context) error {
    tenantID := c.Get("tenant_id").(string)
    
    users, err := h.service.GetUsersWithNotificationPreferences(c.Request().Context(), tenantID)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "success": false,
            "error": map[string]string{
                "code": "INTERNAL_ERROR",
                "message": err.Error(),
            },
        })
    }
    
    return c.JSON(http.StatusOK, map[string]interface{}{
        "success": true,
        "data": map[string]interface{}{
            "users": users,
        },
    })
}

// PATCH /api/v1/users/:id/notification-preferences
func (h *NotificationPreferencesHandler) Update(c echo.Context) error {
    tenantID := c.Get("tenant_id").(string)
    userID := c.Param("id")
    
    var req struct {
        ReceiveOrderNotifications bool `json:"receive_order_notifications"`
    }
    
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "success": false,
            "error": map[string]string{
                "code": "INVALID_INPUT",
                "message": "Invalid request body",
            },
        })
    }
    
    updatedUser, err := h.service.UpdateNotificationPreference(
        c.Request().Context(),
        tenantID,
        userID,
        req.ReceiveOrderNotifications,
    )
    
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "success": false,
            "error": map[string]string{
                "code": "INTERNAL_ERROR",
                "message": err.Error(),
            },
        })
    }
    
    return c.JSON(http.StatusOK, map[string]interface{}{
        "success": true,
        "data": updatedUser,
    })
}
EOF

# Register routes in main.go:
cat >> ../main.go << 'EOF'
// Notification preferences routes
notifHandler := api.NewNotificationPreferencesHandler(userService)
v1.GET("/users/notification-preferences", notifHandler.List)
v1.PATCH("/users/:id/notification-preferences", notifHandler.Update)
EOF
```

---

## Phase 6: Testing & Validation (1 hour)

### 6.1 End-to-End Test

```bash
cd scripts

cat > test-order-notifications.sh << 'EOF'
#!/bin/bash
set -e

echo "=== Order Email Notifications E2E Test ==="

# 1. Enable notifications for test user
echo "1. Enabling notifications for test user..."
curl -X PATCH http://localhost:8081/api/v1/users/$TEST_USER_ID/notification-preferences \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TEST_TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"receive_order_notifications": true}'

# 2. Create and pay for an order
echo "2. Creating test order..."
ORDER_RESPONSE=$(curl -X POST http://localhost:8083/api/v1/guest/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_name": "Test Customer",
    "customer_phone": "+6281234567890",
    "customer_email": "test@example.com",
    "delivery_type": "pickup",
    "items": [{"product_id": "'$TEST_PRODUCT_ID'", "quantity": 1}]
  }')

ORDER_ID=$(echo $ORDER_RESPONSE | jq -r '.data.order_id')
echo "Order created: $ORDER_ID"

# 3. Simulate payment callback
echo "3. Simulating Midtrans payment callback..."
curl -X POST http://localhost:8083/api/v1/payments/midtrans/notification \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "'$ORDER_ID'",
    "transaction_status": "settlement",
    "transaction_id": "test-txn-'$(date +%s)'"
  }'

# 4. Wait for email processing
echo "4. Waiting for notification processing (5 seconds)..."
sleep 5

# 5. Check notification history
echo "5. Checking notification history..."
curl -X GET "http://localhost:8085/api/v1/notifications/history?order_reference=$ORDER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "X-Tenant-ID: $TEST_TENANT_ID"

echo ""
echo "=== Test Complete ==="
echo "Check Mailhog at http://localhost:8025 to see emails"
EOF

chmod +x test-order-notifications.sh
```

### 6.2 Run Test

```bash
# Start services
docker-compose up -d

# Run test
./scripts/test-order-notifications.sh

# Check emails in Mailhog
open http://localhost:8025
```

---

## Phase 7: Frontend Integration (2 hours)

### 7.1 Create Notification Settings Page

```bash
cd frontend/src/pages/admin/settings

cat > notifications.tsx << 'EOF'
import React, { useState, useEffect } from 'react';
import { getNotificationPreferences, updateNotificationPreference } from '@/services/api';

export default function NotificationSettings() {
  const [staff, setStaff] = useState([]);
  const [loading, setLoading] = useState(true);
  
  useEffect(() => {
    loadStaff();
  }, []);
  
  const loadStaff = async () => {
    const response = await getNotificationPreferences();
    setStaff(response.data.users);
    setLoading(false);
  };
  
  const handleToggle = async (userId: string, enabled: boolean) => {
    await updateNotificationPreference(userId, enabled);
    loadStaff(); // Reload
  };
  
  if (loading) return <div>Loading...</div>;
  
  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Order Notification Settings</h1>
      
      <div className="bg-white rounded-lg shadow">
        <table className="min-w-full">
          <thead>
            <tr className="border-b">
              <th className="px-6 py-3 text-left">Name</th>
              <th className="px-6 py-3 text-left">Email</th>
              <th className="px-6 py-3 text-left">Role</th>
              <th className="px-6 py-3 text-left">Notifications</th>
            </tr>
          </thead>
          <tbody>
            {staff.map(member => (
              <tr key={member.id} className="border-b">
                <td className="px-6 py-4">{member.name}</td>
                <td className="px-6 py-4">{member.email}</td>
                <td className="px-6 py-4">{member.role}</td>
                <td className="px-6 py-4">
                  <label className="flex items-center">
                    <input
                      type="checkbox"
                      checked={member.receive_order_notifications}
                      onChange={(e) => handleToggle(member.id, e.target.checked)}
                      className="mr-2"
                    />
                    <span>Receive emails</span>
                  </label>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
EOF
```

---

## Phase 8: Production Deployment (30 minutes)

### 8.1 Environment Configuration

```bash
# Add to .env.production
cat >> backend/notification-service/.env.production << 'EOF'
# Email Configuration
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USERNAME=apikey
SMTP_PASSWORD=your_sendgrid_api_key
SMTP_FROM=orders@yourrestaurant.com

# Kafka Configuration
KAFKA_BROKERS=kafka1:9092,kafka2:9092,kafka3:9092
KAFKA_TOPIC=notification-events
KAFKA_GROUP_ID=notification-service-group

# Frontend URL for links in emails
FRONTEND_DOMAIN=https://yourrestaurant.com
EOF
```

### 8.2 Deployment Checklist

```bash
cat > DEPLOYMENT_CHECKLIST.md << 'EOF'
# Order Email Notifications - Deployment Checklist

## Pre-Deployment
- [ ] All unit tests passing
- [ ] All integration tests passing
- [ ] Contract tests passing
- [ ] Email templates rendered correctly in test
- [ ] SMTP credentials configured in production
- [ ] Kafka topics created
- [ ] Database migrations applied to production

## Deployment
- [ ] Deploy notification-service with new code
- [ ] Deploy order-service with event publishing
- [ ] Deploy user-service with notification preferences API
- [ ] Deploy frontend with settings page
- [ ] Restart services in correct order

## Post-Deployment Verification
- [ ] Create test order and verify emails sent
- [ ] Check notification history in admin dashboard
- [ ] Verify email display in Gmail, Outlook, Apple Mail
- [ ] Test notification preferences toggle
- [ ] Monitor error logs for 1 hour
- [ ] Check email delivery metrics

## Rollback Plan
- [ ] Revert notification-service to previous version
- [ ] Revert order-service to skip event publishing
- [ ] Database migrations can remain (backward compatible)
EOF
```

---

## Troubleshooting

### Issue: Emails not sending

**Check**:
```bash
# Check SMTP logs
docker-compose logs notification-service | grep SMTP

# Test SMTP connection
telnet $SMTP_HOST $SMTP_PORT
```

**Fix**: Verify SMTP credentials in .env file

---

### Issue: Duplicate emails

**Check**:
```bash
# Query for duplicates
docker-compose exec postgres psql -U postgres -d pos_db -c "
SELECT metadata->>'transaction_id', COUNT(*) 
FROM notifications 
WHERE event_type LIKE 'order.paid.%' 
GROUP BY metadata->>'transaction_id' 
HAVING COUNT(*) > 1
"
```

**Fix**: Ensure transaction_id is unique and duplicate check is working

---

### Issue: Staff not receiving notifications

**Check**:
```bash
# Verify user has notifications enabled
docker-compose exec postgres psql -U postgres -d pos_db -c "
SELECT id, name, email, receive_order_notifications 
FROM users 
WHERE tenant_id = '$TENANT_ID'
"
```

**Fix**: Enable notifications for user in admin settings

---

## Success Metrics

After deployment, monitor these metrics:

- Email delivery rate: Target >98%
- Average delivery time: Target <1 minute for staff, <2 minutes for customers
- Failed notification count: Target <2%
- Duplicate notification rate: Target <0.1%

---

## Next Steps

1. Monitor production for 1 week
2. Gather user feedback on email format
3. Consider adding:
   - SMS notifications (Phase 2)
   - Push notifications (Phase 2)
   - Notification scheduling
   - Email analytics

---

**Estimated Total Time**: 6-7 hours for full implementation
**Team**: 1-2 developers
**Complexity**: Medium
