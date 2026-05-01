# Quickstart: Implementing Offline Order Management

**Feature**: Offline Order Management  
**Date**: February 7, 2026  
**Status**: Ready for Implementation

## Prerequisites

Before starting implementation, ensure you have:

- [x] Read [spec.md](spec.md) - Feature requirements and user stories
- [x] Read [research.md](research.md) - Architecture decisions
- [x] Read [data-model.md](data-model.md) - Database schema
- [x] Reviewed [contracts/openapi-offline-orders.yaml](contracts/openapi-offline-orders.yaml) - API contract
- [x] Reviewed [contracts/kafka-events.md](contracts/kafka-events.md) - Event schemas
- [x] Go 1.24+ installed
- [x] PostgreSQL 14+ running (via Docker)
- [x] Kafka running (for event publishing)
- [x] Vault running (for PII encryption)
- [x] Access to order-service codebase

---

## Implementation Phases

### Phase 0: Database Migrations (1-2 hours)

#### Step 1: Create Migration Files

Create four migration files in `backend/migrations/`:

**File: `000060_add_offline_orders.up.sql`**

```sql
-- Extend guest_orders table for offline orders
ALTER TABLE guest_orders
ADD COLUMN IF NOT EXISTS order_type VARCHAR(20) NOT NULL DEFAULT 'online',
ADD COLUMN IF NOT EXISTS data_consent_given BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS consent_method VARCHAR(20),
ADD COLUMN IF NOT EXISTS recorded_by_user_id UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS last_modified_by_user_id UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS last_modified_at TIMESTAMP;

-- Add constraints
ALTER TABLE guest_orders
ADD CONSTRAINT check_order_type
CHECK (order_type IN ('online', 'offline'));

ALTER TABLE guest_orders
ADD CONSTRAINT check_consent_method
CHECK (consent_method IS NULL OR consent_method IN ('verbal', 'written', 'digital'));

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_guest_orders_type_status
ON guest_orders(order_type, status, tenant_id);

CREATE INDEX IF NOT EXISTS idx_guest_orders_recorded_by
ON guest_orders(recorded_by_user_id)
WHERE order_type = 'offline';

CREATE INDEX IF NOT EXISTS idx_offline_orders_pending_payment
ON guest_orders(tenant_id, created_at DESC)
WHERE order_type = 'offline' AND status = 'PENDING';

-- Backfill existing orders
UPDATE guest_orders SET order_type = 'online' WHERE order_type IS NULL;
```

**File: `000060_add_offline_orders.down.sql`**

```sql
ALTER TABLE guest_orders
DROP COLUMN IF EXISTS order_type,
DROP COLUMN IF EXISTS data_consent_given,
DROP COLUMN IF EXISTS consent_method,
DROP COLUMN IF EXISTS recorded_by_user_id,
DROP COLUMN IF EXISTS last_modified_by_user_id,
DROP COLUMN IF EXISTS last_modified_at;

DROP INDEX IF EXISTS idx_guest_orders_type_status;
DROP INDEX IF EXISTS idx_guest_orders_recorded_by;
DROP INDEX IF EXISTS idx_offline_orders_pending_payment;
```

**File: `000061_add_payment_terms.up.sql`**

```sql
CREATE TABLE IF NOT EXISTS payment_terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL UNIQUE REFERENCES guest_orders(id) ON DELETE CASCADE,
    total_amount INTEGER NOT NULL CHECK (total_amount > 0),
    down_payment_amount INTEGER CHECK (down_payment_amount >= 0 AND down_payment_amount < total_amount),
    installment_count INTEGER CHECK (installment_count >= 0),
    installment_amount INTEGER CHECK (installment_amount >= 0),
    payment_schedule JSONB,
    total_paid INTEGER NOT NULL DEFAULT 0 CHECK (total_paid >= 0 AND total_paid <= total_amount),
    remaining_balance INTEGER NOT NULL CHECK (remaining_balance >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by_user_id UUID NOT NULL REFERENCES users(id),

    CONSTRAINT check_payment_structure
    CHECK (
        (down_payment_amount IS NULL AND installment_count = 0) OR
        (down_payment_amount >= 0 AND installment_count > 0)
    ),

    CONSTRAINT check_remaining_balance
    CHECK (remaining_balance = total_amount - total_paid)
);

CREATE INDEX idx_payment_terms_order_id ON payment_terms(order_id);
CREATE INDEX idx_payment_terms_balance ON payment_terms(remaining_balance, order_id)
WHERE remaining_balance > 0;
```

**File: `000061_add_payment_terms.down.sql`**

```sql
DROP TABLE IF EXISTS payment_terms CASCADE;
```

**File: `000062_add_payment_records.up.sql`**

```sql
CREATE TABLE IF NOT EXISTS payment_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES guest_orders(id) ON DELETE CASCADE,
    payment_terms_id UUID REFERENCES payment_terms(id) ON DELETE SET NULL,
    payment_number INTEGER NOT NULL,
    amount_paid INTEGER NOT NULL CHECK (amount_paid > 0),
    payment_date TIMESTAMP NOT NULL DEFAULT NOW(),
    payment_method VARCHAR(50) NOT NULL,
    remaining_balance_after INTEGER NOT NULL CHECK (remaining_balance_after >= 0),
    recorded_by_user_id UUID NOT NULL REFERENCES users(id),
    notes TEXT,
    receipt_number VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT check_payment_method
    CHECK (payment_method IN ('cash', 'card', 'bank_transfer', 'check', 'other'))
);

CREATE INDEX idx_payment_records_order_id ON payment_records(order_id, payment_date DESC);
CREATE INDEX idx_payment_records_date ON payment_records(payment_date DESC);
CREATE INDEX idx_payment_records_recorded_by ON payment_records(recorded_by_user_id);

-- Trigger to update payment_terms totals
CREATE OR REPLACE FUNCTION update_payment_terms_totals()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE payment_terms
    SET
        total_paid = total_paid + NEW.amount_paid,
        remaining_balance = total_amount - (total_paid + NEW.amount_paid)
    WHERE order_id = NEW.order_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_payment_totals
AFTER INSERT ON payment_records
FOR EACH ROW
EXECUTE FUNCTION update_payment_terms_totals();
```

**File: `000062_add_payment_records.down.sql`**

```sql
DROP TRIGGER IF EXISTS trigger_update_payment_totals ON payment_records;
DROP FUNCTION IF EXISTS update_payment_terms_totals();
DROP TABLE IF EXISTS payment_records CASCADE;
```

**File: `000063_add_event_outbox.up.sql`** (for transactional outbox pattern)

```sql
CREATE TABLE IF NOT EXISTS event_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    event_key VARCHAR(255) NOT NULL,
    event_payload JSONB NOT NULL,
    topic VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    last_error TEXT
);

CREATE INDEX idx_outbox_pending ON event_outbox(created_at)
WHERE published_at IS NULL;
```

**File: `000063_add_event_outbox.down.sql`**

```sql
DROP TABLE IF EXISTS event_outbox CASCADE;
```

#### Step 2: Run Migrations

```bash
cd backend
migrate -path migrations \
        -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" \
        up
```

#### Step 3: Verify Schema

```sql
-- Check new columns
\d guest_orders

-- Check new tables
\d payment_terms
\d payment_records
\d event_outbox

-- Check indexes
\di idx_guest_orders_type_status
\di idx_payment_terms_order_id
\di idx_payment_records_order_id
```

**Expected Output**: All tables and indexes created successfully.

---

### Phase 1: Backend Models (2-3 hours)

#### Step 1: Extend Order Model

**File: `backend/order-service/src/models/order.go`**

Add new fields to existing `GuestOrder` struct:

```go
type GuestOrder struct {
    // ... existing fields ...

    // Offline order specific fields
    OrderType          OrderType  `json:"order_type"`
    DataConsentGiven   bool       `json:"data_consent_given"`
    ConsentMethod      *string    `json:"consent_method,omitempty"`
    RecordedByUserID   *string    `json:"recorded_by_user_id,omitempty"`
    LastModifiedByUID  *string    `json:"last_modified_by_user_id,omitempty"`
    LastModifiedAt     *time.Time `json:"last_modified_at,omitempty"`
}

// OrderType enum
type OrderType string

const (
    OrderTypeOnline  OrderType = "online"
    OrderTypeOffline OrderType = "offline"
)
```

#### Step 2: Create Payment Models

**File: `backend/order-service/src/models/payment_terms.go`**

```go
package models

import "time"

type PaymentTerms struct {
    ID                  string          `json:"id"`
    OrderID             string          `json:"order_id"`
    TotalAmount         int             `json:"total_amount"`
    DownPaymentAmount   *int            `json:"down_payment_amount,omitempty"`
    InstallmentCount    int             `json:"installment_count"`
    InstallmentAmount   int             `json:"installment_amount"`
    PaymentSchedule     []Installment   `json:"payment_schedule"`
    TotalPaid           int             `json:"total_paid"`
    RemainingBalance    int             `json:"remaining_balance"`
    CreatedAt           time.Time       `json:"created_at"`
    CreatedByUserID     string          `json:"created_by_user_id"`
}

type Installment struct {
    InstallmentNumber int       `json:"installment_number"`
    DueDate           string    `json:"due_date"` // Format: YYYY-MM-DD
    Amount            int       `json:"amount"`
    Status            string    `json:"status"` // pending, paid, overdue
}
```

**File: `backend/order-service/src/models/payment_record.go`**

```go
package models

import "time"

type PaymentRecord struct {
    ID                     string          `json:"id"`
    OrderID                string          `json:"order_id"`
    PaymentTermsID         *string         `json:"payment_terms_id,omitempty"`
    PaymentNumber          int             `json:"payment_number"`
    AmountPaid             int             `json:"amount_paid"`
    PaymentDate            time.Time       `json:"payment_date"`
    PaymentMethod          PaymentMethod   `json:"payment_method"`
    RemainingBalanceAfter  int             `json:"remaining_balance_after"`
    RecordedByUserID       string          `json:"recorded_by_user_id"`
    Notes                  *string         `json:"notes,omitempty"`
    ReceiptNumber          *string         `json:"receipt_number,omitempty"`
    CreatedAt              time.Time       `json:"created_at"`
}

type PaymentMethod string

const (
    PaymentMethodCash         PaymentMethod = "cash"
    PaymentMethodCard         PaymentMethod = "card"
    PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
    PaymentMethodCheck        PaymentMethod = "check"
    PaymentMethodOther        PaymentMethod = "other"
)
```

---

### Phase 2: Repository Layer (4-5 hours)

#### Step 1: Extend Order Repository

**File: `backend/order-service/src/repository/offline_order_repository.go`**

```go
package repository

import (
    "context"
    "database/sql"
    "github.com/point-of-sale-system/order-service/src/models"
)

type OfflineOrderRepository struct {
    db        *sql.DB
    encryptor utils.Encryptor
}

func NewOfflineOrderRepository(db *sql.DB, encryptor utils.Encryptor) *OfflineOrderRepository {
    return &OfflineOrderRepository{
        db:        db,
        encryptor: encryptor,
    }
}

// CreateOfflineOrder creates a new offline order with items
func (r *OfflineOrderRepository) CreateOfflineOrder(ctx context.Context, order *models.GuestOrder, items []models.OrderItem) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Encrypt PII fields
    encryptedName := r.encryptor.Encrypt(order.CustomerName)
    encryptedPhone := r.encryptor.EncryptSearchable(order.CustomerPhone)
    encryptedEmail := ""
    if order.CustomerEmail != nil {
        encryptedEmail = r.encryptor.EncryptSearchable(*order.CustomerEmail)
    }

    // Insert order
    query := `
        INSERT INTO guest_orders (
            id, order_reference, tenant_id, order_type, status,
            customer_name, customer_phone, customer_email,
            subtotal_amount, delivery_fee, total_amount,
            delivery_type, notes, data_consent_given, consent_method,
            recorded_by_user_id, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW()
        )
    `

    _, err = tx.ExecContext(ctx, query,
        order.ID, order.OrderReference, order.TenantID, "offline", order.Status,
        encryptedName, encryptedPhone, encryptedEmail,
        order.SubtotalAmount, order.DeliveryFee, order.TotalAmount,
        order.DeliveryType, order.Notes, order.DataConsentGiven, order.ConsentMethod,
        order.RecordedByUserID,
    )
    if err != nil {
        return err
    }

    // Insert order items
    for _, item := range items {
        itemQuery := `
            INSERT INTO order_items (
                id, order_id, product_id, product_name, quantity, unit_price, total_price
            ) VALUES ($1, $2, $3, $4, $5, $6, $7)
        `
        _, err = tx.ExecContext(ctx, itemQuery,
            item.ID, order.ID, item.ProductID, item.ProductName,
            item.Quantity, item.UnitPrice, item.TotalPrice,
        )
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}

// GetOfflineOrderByID retrieves an offline order with decrypted PII
func (r *OfflineOrderRepository) GetOfflineOrderByID(ctx context.Context, orderID string) (*models.GuestOrder, error) {
    query := `
        SELECT id, order_reference, tenant_id, order_type, status,
               customer_name, customer_phone, customer_email,
               subtotal_amount, delivery_fee, total_amount,
               delivery_type, notes, data_consent_given, consent_method,
               recorded_by_user_id, last_modified_by_user_id, last_modified_at,
               created_at, paid_at, completed_at, cancelled_at
        FROM guest_orders
        WHERE id = $1 AND order_type = 'offline'
    `

    var order models.GuestOrder
    var encryptedName, encryptedPhone, encryptedEmail sql.NullString

    err := r.db.QueryRowContext(ctx, query, orderID).Scan(
        &order.ID, &order.OrderReference, &order.TenantID, &order.OrderType, &order.Status,
        &encryptedName, &encryptedPhone, &encryptedEmail,
        &order.SubtotalAmount, &order.DeliveryFee, &order.TotalAmount,
        &order.DeliveryType, &order.Notes, &order.DataConsentGiven, &order.ConsentMethod,
        &order.RecordedByUserID, &order.LastModifiedByUID, &order.LastModifiedAt,
        &order.CreatedAt, &order.PaidAt, &order.CompletedAt, &order.CancelledAt,
    )
    if err != nil {
        return nil, err
    }

    // Decrypt PII
    if encryptedName.Valid {
        order.CustomerName = r.encryptor.Decrypt(encryptedName.String)
    }
    if encryptedPhone.Valid {
        order.CustomerPhone = r.encryptor.DecryptSearchable(encryptedPhone.String)
    }
    if encryptedEmail.Valid {
        email := r.encryptor.DecryptSearchable(encryptedEmail.String)
        order.CustomerEmail = &email
    }

    return &order, nil
}

// UpdateOfflineOrder updates order fields (only if status != PAID)
func (r *OfflineOrderRepository) UpdateOfflineOrder(ctx context.Context, orderID string, updates map[string]interface{}) error {
    // Implementation: Build dynamic UPDATE query based on updates map
    // Check status constraint
    // Log to event_outbox for audit trail
    return nil
}

// ListOfflineOrders retrieves paginated offline orders for tenant
func (r *OfflineOrderRepository) ListOfflineOrders(ctx context.Context, tenantID string, filters map[string]interface{}, page, limit int) ([]*models.GuestOrder, int, error) {
    // Implementation: Build query with filters, pagination
    // Decrypt PII fields for each order
    return nil, 0, nil
}
```

#### Step 2: Create Payment Repository

**File: `backend/order-service/src/repository/payment_repository.go`**

```go
package repository

import (
    "context"
    "database/sql"
    "github.com/point-of-sale-system/order-service/src/models"
)

type PaymentRepository struct {
    db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
    return &PaymentRepository{db: db}
}

// CreatePaymentTerms creates payment terms for an order
func (r *PaymentRepository) CreatePaymentTerms(ctx context.Context, terms *models.PaymentTerms) error {
    query := `
        INSERT INTO payment_terms (
            id, order_id, total_amount, down_payment_amount,
            installment_count, installment_amount, payment_schedule,
            total_paid, remaining_balance, created_by_user_id
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `

    scheduleJSON, _ := json.Marshal(terms.PaymentSchedule)

    _, err := r.db.ExecContext(ctx, query,
        terms.ID, terms.OrderID, terms.TotalAmount, terms.DownPaymentAmount,
        terms.InstallmentCount, terms.InstallmentAmount, scheduleJSON,
        terms.TotalPaid, terms.RemainingBalance, terms.CreatedByUserID,
    )

    return err
}

// RecordPayment records a payment and updates totals (uses trigger)
func (r *PaymentRepository) RecordPayment(ctx context.Context, payment *models.PaymentRecord) error {
    query := `
        INSERT INTO payment_records (
            id, order_id, payment_terms_id, payment_number,
            amount_paid, payment_date, payment_method,
            remaining_balance_after, recorded_by_user_id,
            notes, receipt_number
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `

    _, err := r.db.ExecContext(ctx, query,
        payment.ID, payment.OrderID, payment.PaymentTermsID, payment.PaymentNumber,
        payment.AmountPaid, payment.PaymentDate, payment.PaymentMethod,
        payment.RemainingBalanceAfter, payment.RecordedByUserID,
        payment.Notes, payment.ReceiptNumber,
    )

    return err
}

// GetPaymentHistory retrieves all payments for an order
func (r *PaymentRepository) GetPaymentHistory(ctx context.Context, orderID string) ([]*models.PaymentRecord, error) {
    query := `
        SELECT id, order_id, payment_terms_id, payment_number,
               amount_paid, payment_date, payment_method,
               remaining_balance_after, recorded_by_user_id,
               notes, receipt_number, created_at
        FROM payment_records
        WHERE order_id = $1
        ORDER BY payment_date ASC
    `

    rows, err := r.db.QueryContext(ctx, query, orderID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var payments []*models.PaymentRecord
    for rows.Next() {
        var p models.PaymentRecord
        // Scan logic...
        payments = append(payments, &p)
    }

    return payments, nil
}
```

---

### Phase 3: Service Layer (5-6 hours)

**File: `backend/order-service/src/services/offline_order_service.go`**

```go
package services

import (
    "context"
    "github.com/google/uuid"
    "github.com/point-of-sale-system/order-service/src/models"
    "github.com/point-of-sale-system/order-service/src/repository"
)

type OfflineOrderService struct {
    orderRepo   *repository.OfflineOrderRepository
    paymentRepo *repository.PaymentRepository
    eventSvc    *EventService
}

func NewOfflineOrderService(
    orderRepo *repository.OfflineOrderRepository,
    paymentRepo *repository.PaymentRepository,
    eventSvc *EventService,
) *OfflineOrderService {
    return &OfflineOrderService{
        orderRepo:   orderRepo,
        paymentRepo: paymentRepo,
        eventSvc:    eventSvc,
    }
}

// CreateOfflineOrder creates a new offline order with payment
func (s *OfflineOrderService) CreateOfflineOrder(
    ctx context.Context,
    orderData *models.GuestOrder,
    items []models.OrderItem,
    paymentData *PaymentData,
) (*models.GuestOrder, error) {
    // Generate order reference
    orderData.ID = uuid.New().String()
    orderData.OrderReference = generateOrderReference() // GO-XXXXXX
    orderData.OrderType = models.OrderTypeOffline

    // Calculate totals
    subtotal := 0
    for _, item := range items {
        subtotal += item.TotalPrice
    }
    orderData.SubtotalAmount = subtotal
    orderData.TotalAmount = subtotal + orderData.DeliveryFee

    // Determine initial status
    if paymentData.Type == "full" {
        orderData.Status = models.OrderStatusPaid
    } else {
        orderData.Status = models.OrderStatusPending
    }

    // Create order in database
    err := s.orderRepo.CreateOfflineOrder(ctx, orderData, items)
    if err != nil {
        return nil, err
    }

    // Handle payment
    if paymentData.Type == "full" {
        err = s.recordFullPayment(ctx, orderData.ID, paymentData)
    } else {
        err = s.setupInstallmentPayment(ctx, orderData.ID, paymentData)
    }
    if err != nil {
        return nil, err
    }

    // Publish audit event
    s.eventSvc.PublishOfflineOrderCreated(ctx, orderData)

    return orderData, nil
}

// RecordPayment records an installment payment
func (s *OfflineOrderService) RecordPayment(
    ctx context.Context,
    orderID string,
    amount int,
    method models.PaymentMethod,
    recordedByUserID string,
) (*models.PaymentRecord, bool, error) {
    // Get current payment terms
    terms, _ := s.paymentRepo.GetPaymentTerms(ctx, orderID)

    // Create payment record
    payment := &models.PaymentRecord{
        ID:                    uuid.New().String(),
        OrderID:               orderID,
        PaymentTermsID:        &terms.ID,
        PaymentNumber:         s.getNextPaymentNumber(ctx, orderID),
        AmountPaid:            amount,
        PaymentMethod:         method,
        RemainingBalanceAfter: terms.RemainingBalance - amount,
        RecordedByUserID:      recordedByUserID,
    }

    err := s.paymentRepo.RecordPayment(ctx, payment)
    if err != nil {
        return nil, false, err
    }

    // Check if order is now fully paid
    statusChanged := false
    if payment.RemainingBalanceAfter == 0 {
        s.orderRepo.UpdateStatus(ctx, orderID, models.OrderStatusPaid)
        statusChanged = true
    }

    // Publish events
    s.eventSvc.PublishPaymentRecorded(ctx, payment)

    return payment, statusChanged, nil
}
```

---

### Phase 4: API Handlers (4-5 hours)

**File: `backend/order-service/api/offline_orders_handler.go`**

```go
package api

import (
    "net/http"
    "github.com/labstack/echo/v4"
    "github.com/point-of-sale-system/order-service/src/services"
)

type OfflineOrderHandler struct {
    service *services.OfflineOrderService
}

func NewOfflineOrderHandler(service *services.OfflineOrderService) *OfflineOrderHandler {
    return &OfflineOrderHandler{service: service}
}

// CreateOfflineOrder godoc
// @Summary Create offline order
// @Tags offline-orders
// @Accept json
// @Produce json
// @Param request body CreateOfflineOrderRequest true "Order data"
// @Success 201 {object} OfflineOrderResponse
// @Router /offline-orders [post]
func (h *OfflineOrderHandler) CreateOfflineOrder(c echo.Context) error {
    var req CreateOfflineOrderRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
    }

    // Validate
    if err := c.Validate(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    // Get user context
    userID := c.Get("user_id").(string)
    tenantID := c.Get("tenant_id").(string)

    // Map request to domain model
    orderData := mapToOrderModel(&req, userID, tenantID)
    items := mapToOrderItems(&req)
    paymentData := mapToPaymentData(&req)

    // Create order
    order, err := h.service.CreateOfflineOrder(c.Request().Context(), orderData, items, paymentData)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create order")
    }

    return c.JSON(http.StatusCreated, toOfflineOrderResponse(order))
}

// GetOfflineOrders godoc
// @Summary List offline orders
// @Tags offline-orders
// @Produce json
// @Param status query string false "Filter by status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page" default(20)
// @Success 200 {object} OfflineOrderListResponse
// @Router /offline-orders [get]
func (h *OfflineOrderHandler) GetOfflineOrders(c echo.Context) error {
    // Implementation
    return nil
}

// GetOfflineOrderByID godoc
// @Summary Get offline order details
// @Tags offline-orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} OfflineOrderDetailResponse
// @Router /offline-orders/{id} [get]
func (h *OfflineOrderHandler) GetOfflineOrderByID(c echo.Context) error {
    // Implementation
    return nil
}

// UpdateOfflineOrder godoc
// @Summary Update offline order
// @Tags offline-orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param request body UpdateOfflineOrderRequest true "Update data"
// @Success 200 {object} OfflineOrderResponse
// @Router /offline-orders/{id} [patch]
func (h *OfflineOrderHandler) UpdateOfflineOrder(c echo.Context) error {
    // Implementation with audit logging
    return nil
}

// DeleteOfflineOrder godoc
// @Summary Delete offline order (owner/manager only)
// @Tags offline-orders
// @Accept json
// @Param id path string true "Order ID"
// @Param request body DeleteOrderRequest true "Deletion reason"
// @Success 204
// @Router /offline-orders/{id} [delete]
func (h *OfflineOrderHandler) DeleteOfflineOrder(c echo.Context) error {
    // Check role (middleware already verified, but double-check)
    userRole := c.Get("user_role").(string)
    if userRole != "owner" && userRole != "manager" {
        return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
    }

    // Implementation with audit logging
    return nil
}
```

---

### Phase 5: Middleware & Routes (2-3 hours)

**File: `backend/order-service/src/middleware/role_check.go`**

```go
package middleware

import (
    "net/http"
    "github.com/labstack/echo/v4"
)

func RequireRole(allowedRoles ...string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            userRole := c.Get("user_role").(string)

            allowed := false
            for _, role := range allowedRoles {
                if userRole == role {
                    allowed = true
                    break
                }
            }

            if !allowed {
                return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
            }

            return next(c)
        }
    }
}
```

**File: `backend/order-service/main.go`** (add routes)

```go
// Offline Orders routes (authenticated)
offlineOrdersGroup := api.Group("/offline-orders")
offlineOrdersGroup.Use(middleware.AuthRequired()) // JWT validation
{
    offlineOrdersGroup.POST("", offlineOrderHandler.CreateOfflineOrder)
    offlineOrdersGroup.GET("", offlineOrderHandler.GetOfflineOrders)
    offlineOrdersGroup.GET("/:id", offlineOrderHandler.GetOfflineOrderByID)
    offlineOrdersGroup.PATCH("/:id", offlineOrderHandler.UpdateOfflineOrder)
    offlineOrdersGroup.DELETE("/:id", offlineOrderHandler.DeleteOfflineOrder, middleware.RequireRole("owner", "manager"))
    offlineOrdersGroup.POST("/:id/payments", offlineOrderHandler.RecordPayment)
    offlineOrdersGroup.GET("/:id/payments", offlineOrderHandler.GetPaymentHistory)
}
```

---

### Phase 6: Testing (6-8 hours)

#### Unit Tests

**File: `backend/order-service/tests/unit/offline_orders_test.go`**

```go
package unit

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCreateOfflineOrder_FullPayment(t *testing.T) {
    // Given: Order data with full payment
    // When: CreateOfflineOrder called
    // Then: Order created with status PAID
}

func TestCreateOfflineOrder_InstallmentPayment(t *testing.T) {
    // Given: Order data with down payment + installments
    // When: CreateOfflineOrder called
    // Then: Order created with status PENDING, payment_terms row created
}

func TestRecordPayment_UpdatesStatus(t *testing.T) {
    // Given: Order with outstanding balance
    // When: Final payment recorded
    // Then: Order status changes to PAID
}

func TestDeleteOfflineOrder_RequiresOwnerRole(t *testing.T) {
    // Given: Staff user attempts deletion
    // When: DeleteOfflineOrder called
    // Then: Returns 403 Forbidden
}
```

#### Integration Tests

**File: `backend/order-service/tests/integration/offline_orders_integration_test.go`**

```go
package integration

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestOfflineOrderLifecycle(t *testing.T) {
    // Setup: Test database
    // Create order → Record payments → Mark complete
    // Assert: All database rows correct, event outbox populated
}
```

#### Contract Tests

Validate API responses match OpenAPI spec using tool like Dredd or Prism.

---

### Phase 7: Frontend Components (8-10 hours)

#### Frontend Structure (Next.js 13+ App Directory)

**Component Files:**

1. `frontend/src/components/orders/OfflineOrderForm.tsx` - Form for creating/editing offline orders
2. `frontend/src/components/orders/OfflineOrderList.tsx` - List view with filters
3. `frontend/src/components/orders/OfflineOrderDetail.tsx` - Order details with payment history
4. `frontend/src/components/orders/PaymentSchedule.tsx` - Payment schedule display
5. `frontend/src/components/orders/RecordPayment.tsx` - Payment recording form
6. `frontend/src/components/orders/AuditTrail.tsx` - Audit trail display
7. `frontend/src/components/orders/DeleteOrderModal.tsx` - Deletion confirmation with reason

**Page Files (App Router):**

1. `frontend/app/orders/offline-orders/page.tsx` - List page (server component)
2. `frontend/app/orders/offline-orders/new/page.tsx` - Create page
3. `frontend/app/orders/offline-orders/[id]/page.tsx` - Detail page
4. `frontend/app/orders/offline-orders/[id]/edit/page.tsx` - Edit page
5. `frontend/app/orders/offline-orders/[id]/payments/page.tsx` - Payment recording page

**Service Files:**

1. `frontend/src/services/offlineOrders.ts` - API client
2. `frontend/src/types/offlineOrder.ts` - TypeScript interfaces

**Key Implementation Notes:**

- Use Next.js 13+ app directory with server/client component separation
- Server components for data fetching, client components for interactivity
- API routes via `/api/offline-orders` endpoints
- Implement optimistic UI updates for better UX
- Add loading states and error boundaries
- Use React Server Components for initial page loads
- Client components marked with `'use client'` directive

---

## Verification Checklist

After implementation, verify:

- [ ] Database migrations run successfully
- [ ] All unit tests pass (`go test ./... -v`)
- [ ] Integration tests pass
- [ ] API endpoints return correct schemas (match OpenAPI spec)
- [ ] RBAC enforced (non-owner/manager cannot delete)
- [ ] Audit events published to Kafka
- [ ] PII encrypted in database
- [ ] Payment balance calculations correct
- [ ] Order status transitions validated
- [ ] Performance: Order creation <200ms p95
- [ ] No degradation to online order endpoints

---

## Troubleshooting

### Issue: Migration fails with "column already exists"

**Solution**: Check if previous migration ran partially. Rollback and re-run:

```bash
migrate -path migrations -database "postgresql://..." down 1
migrate -path migrations -database "postgresql://..." up
```

### Issue: PII not encrypted in database

**Solution**: Check Vault connection and encryption service initialization:

```bash
curl http://localhost:8200/v1/sys/health
```

### Issue: Kafka events not publishing

**Solution**: Check event outbox worker is running and Kafka is reachable:

```bash
docker logs kafka
# Check outbox table for pending events
SELECT * FROM event_outbox WHERE published_at IS NULL;
```

---

## Next Steps

After completing implementation:

1. ✅ `/speckit.tasks` command already generated detailed task breakdown (see tasks.md)
2. Create feature branch from `master`
3. Implement in order: Migrations (000060-000063) → Models → Repositories → Services → Handlers → Frontend
4. Write tests alongside implementation (TDD)
5. Deploy to staging environment
6. Run acceptance tests (BDD scenarios from spec.md)
7. Monitor performance and error rates
8. Deploy to production with feature flag

---

## Additional Resources

- [Echo Framework Docs](https://echo.labstack.com/guide/)
- [PostgreSQL JSON Functions](https://www.postgresql.org/docs/current/functions-json.html)
- [Kafka Go Client](https://github.com/segmentio/kafka-go)
- [Vault Go API](https://pkg.go.dev/github.com/hashicorp/vault/api)
- [OpenAPI 3.0 Spec](https://swagger.io/specification/)
