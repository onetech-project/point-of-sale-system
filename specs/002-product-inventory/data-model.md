# Data Model: Product & Inventory Management

**Feature Branch**: `002-product-inventory`  
**Date**: 2025-12-01  
**Phase**: 1 - Design & Contracts

## Overview

This document defines the data model for Product & Inventory Management, including entities, relationships, validation rules, and state transitions.

## Entity Definitions

### 1. Product

Represents an item available for sale in the POS system.

**Table**: `products`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT uuid_generate_v4() | Unique product identifier |
| tenant_id | UUID | NOT NULL, REFERENCES tenants(id) ON DELETE CASCADE | Tenant ownership |
| sku | VARCHAR(50) | NOT NULL | Stock Keeping Unit / Barcode |
| name | VARCHAR(255) | NOT NULL | Product name |
| description | TEXT | NULL | Product description |
| category_id | UUID | NULL, REFERENCES categories(id) ON DELETE RESTRICT | Category assignment |
| selling_price | DECIMAL(10,2) | NOT NULL, CHECK (selling_price >= 0) | Price for customers |
| cost_price | DECIMAL(10,2) | NOT NULL, CHECK (cost_price >= 0) | Cost from supplier |
| tax_rate | DECIMAL(5,2) | NOT NULL, DEFAULT 0, CHECK (tax_rate >= 0 AND tax_rate <= 100) | Tax percentage |
| stock_quantity | INTEGER | NOT NULL, DEFAULT 0 | Current inventory quantity |
| photo_path | VARCHAR(500) | NULL | Relative path to product photo |
| photo_size | INTEGER | NULL | Photo file size in bytes |
| archived_at | TIMESTAMP | NULL | Timestamp when product was archived |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | Last modification timestamp |

**Indexes**:
```sql
CREATE UNIQUE INDEX idx_products_tenant_sku ON products(tenant_id, sku);
CREATE INDEX idx_products_tenant_name ON products(tenant_id, name);
CREATE INDEX idx_products_tenant_category ON products(tenant_id, category_id);
CREATE INDEX idx_products_tenant_archived ON products(tenant_id, archived_at);
CREATE INDEX idx_products_stock_quantity ON products(tenant_id, stock_quantity) WHERE archived_at IS NULL;
```

**RLS Policy**:
```sql
ALTER TABLE products ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON products
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

**Validation Rules** (FR-030 to FR-034):
- Name: 1-255 characters, required
- SKU: 1-50 alphanumeric characters, unique per tenant, required
- Selling price: Positive decimal, required
- Cost price: Non-negative decimal, required
- Tax rate: 0-100 percentage
- Stock quantity: Integer (can be negative for backorders)
- Photo: Max 5MB, formats: JPEG/PNG/WebP

**State Transitions**:
- **Active** → **Archived**: Set `archived_at = NOW()`, product removed from active catalog
- **Archived** → **Active**: Set `archived_at = NULL`, product restored to catalog
- **Active** → **Deleted**: Only allowed if no sales history, permanent removal

---

### 2. Category

Represents a grouping for organizing products (flat structure).

**Table**: `categories`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT uuid_generate_v4() | Unique category identifier |
| tenant_id | UUID | NOT NULL, REFERENCES tenants(id) ON DELETE CASCADE | Tenant ownership |
| name | VARCHAR(100) | NOT NULL | Category name |
| display_order | INTEGER | NOT NULL, DEFAULT 0 | Sort order for UI display |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | Last modification timestamp |

**Indexes**:
```sql
CREATE UNIQUE INDEX idx_categories_tenant_name ON categories(tenant_id, name);
CREATE INDEX idx_categories_tenant_order ON categories(tenant_id, display_order);
```

**RLS Policy**:
```sql
ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON categories
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

**Validation Rules**:
- Name: 1-100 characters, unique per tenant, required
- Display order: Integer, used for manual sorting in UI

**Business Rules** (FR-026, FR-027):
- Cannot delete category if products are assigned (ON DELETE RESTRICT)
- Must reassign products to another category or set to NULL before deletion
- Category names must be unique within a tenant

---

### 3. Stock Adjustment

Represents a manual change to inventory quantity with full audit trail.

**Table**: `stock_adjustments`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT uuid_generate_v4() | Unique adjustment identifier |
| tenant_id | UUID | NOT NULL, REFERENCES tenants(id) ON DELETE CASCADE | Tenant ownership |
| product_id | UUID | NOT NULL, REFERENCES products(id) ON DELETE CASCADE | Product being adjusted |
| user_id | UUID | NOT NULL, REFERENCES users(id) ON DELETE RESTRICT | User who made adjustment |
| previous_quantity | INTEGER | NOT NULL | Stock quantity before adjustment |
| new_quantity | INTEGER | NOT NULL | Stock quantity after adjustment |
| quantity_delta | INTEGER | NOT NULL | Change amount (new - previous) |
| reason | VARCHAR(50) | NOT NULL | Adjustment reason code |
| notes | TEXT | NULL | Additional details/explanation |
| created_at | TIMESTAMP | NOT NULL, DEFAULT NOW() | Adjustment timestamp |

**Indexes**:
```sql
CREATE INDEX idx_stock_adjustments_product ON stock_adjustments(product_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_tenant_date ON stock_adjustments(tenant_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_user ON stock_adjustments(user_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_reason ON stock_adjustments(tenant_id, reason, created_at DESC);
```

**RLS Policy**:
```sql
ALTER TABLE stock_adjustments ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON stock_adjustments
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
```

**Database Trigger** (Auto-calculate delta):
```sql
CREATE OR REPLACE FUNCTION calculate_quantity_delta()
RETURNS TRIGGER AS $$
BEGIN
  NEW.quantity_delta := NEW.new_quantity - NEW.previous_quantity;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_calculate_delta
  BEFORE INSERT ON stock_adjustments
  FOR EACH ROW
  EXECUTE FUNCTION calculate_quantity_delta();
```

**Reason Codes** (FR-022):
- `supplier_delivery`: Stock received from supplier
- `physical_count`: Inventory count adjustment
- `shrinkage`: Loss due to theft, damage, expiration
- `damage`: Damaged goods removal
- `return`: Customer return processed
- `correction`: Data entry correction
- `sale`: Automatic deduction from sale (logged separately)

**Business Rules** (FR-019 to FR-024):
- Immutable records (no UPDATE or DELETE operations)
- Reason field required for all adjustments
- User ID captures who made the adjustment
- All adjustments logged with complete audit trail
- Filterable by date range, user, product, reason

---

## Entity Relationships

```
┌──────────┐
│ Tenants  │
└────┬─────┘
     │
     ├─────────────────────────────┐
     │                             │
     ▼                             ▼
┌────────────┐               ┌──────────┐
│ Categories │               │  Users   │
└────┬───────┘               └────┬─────┘
     │                             │
     │ 1:N                         │
     │                             │
     ▼                             │
┌──────────────┐                   │
│   Products   │◄──────────────────┘
└──────┬───────┘                   │
       │ 1:N                       │ N:1
       │                           │
       ▼                           ▼
┌──────────────────────┐
│ Stock Adjustments    │
└──────────────────────┘
```

**Relationships**:
1. **Tenant → Products**: One-to-Many (cascade delete)
2. **Tenant → Categories**: One-to-Many (cascade delete)
3. **Category → Products**: One-to-Many (restrict delete if products exist)
4. **Product → Stock Adjustments**: One-to-Many (cascade delete)
5. **User → Stock Adjustments**: One-to-Many (restrict delete to preserve audit)

---

## Data Validation Summary

### Product Validation
- **Name**: Required, 1-255 characters
- **SKU**: Required, 1-50 alphanumeric + hyphens/underscores, unique per tenant
- **Selling Price**: Required, positive decimal
- **Cost Price**: Required, non-negative decimal
- **Tax Rate**: 0-100 percentage
- **Category**: Optional (can be NULL for uncategorized products)
- **Photo**: Optional, max 5MB, JPEG/PNG/WebP formats

### Category Validation
- **Name**: Required, 1-100 characters, unique per tenant
- **Display Order**: Integer for sorting

### Stock Adjustment Validation
- **Product ID**: Required, must exist
- **User ID**: Required, must exist
- **Previous Quantity**: Required, captured from product
- **New Quantity**: Required, provided by user
- **Reason**: Required, one of predefined codes
- **Notes**: Optional, additional context

---

## Database Migration Strategy

### Migration Files

**009_create_products_table.up.sql**:
```sql
CREATE TABLE products (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  sku VARCHAR(50) NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  category_id UUID REFERENCES categories(id) ON DELETE RESTRICT,
  selling_price DECIMAL(10,2) NOT NULL CHECK (selling_price >= 0),
  cost_price DECIMAL(10,2) NOT NULL CHECK (cost_price >= 0),
  tax_rate DECIMAL(5,2) NOT NULL DEFAULT 0 CHECK (tax_rate >= 0 AND tax_rate <= 100),
  stock_quantity INTEGER NOT NULL DEFAULT 0,
  photo_path VARCHAR(500),
  photo_size INTEGER,
  archived_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_products_tenant_sku ON products(tenant_id, sku);
CREATE INDEX idx_products_tenant_name ON products(tenant_id, name);
CREATE INDEX idx_products_tenant_category ON products(tenant_id, category_id);
CREATE INDEX idx_products_tenant_archived ON products(tenant_id, archived_at);
CREATE INDEX idx_products_stock_quantity ON products(tenant_id, stock_quantity) WHERE archived_at IS NULL;

ALTER TABLE products ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON products
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Trigger for updated_at
CREATE TRIGGER trg_products_updated_at
  BEFORE UPDATE ON products
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
```

**010_create_categories_table.up.sql**:
```sql
CREATE TABLE categories (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL,
  display_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_categories_tenant_name ON categories(tenant_id, name);
CREATE INDEX idx_categories_tenant_order ON categories(tenant_id, display_order);

ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON categories
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Trigger for updated_at
CREATE TRIGGER trg_categories_updated_at
  BEFORE UPDATE ON categories
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();
```

**011_create_stock_adjustments_table.up.sql**:
```sql
CREATE TABLE stock_adjustments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  previous_quantity INTEGER NOT NULL,
  new_quantity INTEGER NOT NULL,
  quantity_delta INTEGER NOT NULL,
  reason VARCHAR(50) NOT NULL,
  notes TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stock_adjustments_product ON stock_adjustments(product_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_tenant_date ON stock_adjustments(tenant_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_user ON stock_adjustments(user_id, created_at DESC);
CREATE INDEX idx_stock_adjustments_reason ON stock_adjustments(tenant_id, reason, created_at DESC);

ALTER TABLE stock_adjustments ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON stock_adjustments
  USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Trigger to calculate quantity delta
CREATE OR REPLACE FUNCTION calculate_quantity_delta()
RETURNS TRIGGER AS $$
BEGIN
  NEW.quantity_delta := NEW.new_quantity - NEW.previous_quantity;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_calculate_delta
  BEFORE INSERT ON stock_adjustments
  FOR EACH ROW
  EXECUTE FUNCTION calculate_quantity_delta();
```

### Rollback Files

**009_create_products_table.down.sql**:
```sql
DROP TRIGGER IF EXISTS trg_products_updated_at ON products;
DROP POLICY IF EXISTS tenant_isolation ON products;
DROP TABLE IF EXISTS products CASCADE;
```

**010_create_categories_table.down.sql**:
```sql
DROP TRIGGER IF EXISTS trg_categories_updated_at ON categories;
DROP POLICY IF EXISTS tenant_isolation ON categories;
DROP TABLE IF EXISTS categories CASCADE;
```

**011_create_stock_adjustments_table.down.sql**:
```sql
DROP TRIGGER IF EXISTS trg_calculate_delta ON stock_adjustments;
DROP FUNCTION IF EXISTS calculate_quantity_delta();
DROP POLICY IF EXISTS tenant_isolation ON stock_adjustments;
DROP TABLE IF EXISTS stock_adjustments CASCADE;
```

---

## Query Examples

### Get Active Products with Category
```sql
SELECT 
  p.id, p.sku, p.name, p.selling_price, p.stock_quantity,
  c.name AS category_name
FROM products p
LEFT JOIN categories c ON p.category_id = c.id
WHERE p.tenant_id = $1 AND p.archived_at IS NULL
ORDER BY p.name
LIMIT 50 OFFSET 0;
```

### Search Products by Name
```sql
SELECT id, sku, name, selling_price, stock_quantity
FROM products
WHERE tenant_id = $1 
  AND archived_at IS NULL
  AND name ILIKE '%' || $2 || '%'
ORDER BY name
LIMIT 50;
```

### Get Low Stock Products
```sql
SELECT id, sku, name, stock_quantity
FROM products
WHERE tenant_id = $1 
  AND archived_at IS NULL
  AND stock_quantity <= $2
ORDER BY stock_quantity ASC, name;
```

### Get Stock Adjustment History for Product
```sql
SELECT 
  sa.id, sa.previous_quantity, sa.new_quantity, sa.quantity_delta,
  sa.reason, sa.notes, sa.created_at,
  u.full_name AS adjusted_by
FROM stock_adjustments sa
JOIN users u ON sa.user_id = u.id
WHERE sa.product_id = $1
ORDER BY sa.created_at DESC
LIMIT 100;
```

### Create Stock Adjustment
```sql
-- Transaction to update stock and log adjustment
BEGIN;

UPDATE products 
SET stock_quantity = $2, updated_at = NOW()
WHERE id = $1 AND tenant_id = $3;

INSERT INTO stock_adjustments 
  (tenant_id, product_id, user_id, previous_quantity, new_quantity, reason, notes)
VALUES 
  ($3, $1, $4, $5, $2, $6, $7);

COMMIT;
```

---

## Performance Considerations

### Indexing Strategy
- All queries filtered by `tenant_id` first (RLS + explicit filters)
- Composite indexes with `tenant_id` as first column
- Indexes on frequently filtered columns (name, category, archived_at, stock_quantity)
- Descending order on `created_at` for audit logs (recent first)

### Pagination
- Default page size: 50 items
- Maximum page size: 100 items
- Use LIMIT/OFFSET for simple pagination
- Consider cursor-based pagination for large datasets (future enhancement)

### Caching
- Cache category list in Redis (5-minute TTL)
- Invalidate cache on category create/update/delete
- No caching for product list (inventory changes frequently)

### Connection Pooling
- Max 25 database connections per service
- Connection timeout: 30 seconds
- Idle connection timeout: 5 minutes

---

## Summary

Data model designed with:
- ✅ Multi-tenant isolation via RLS
- ✅ Referential integrity with foreign keys
- ✅ Audit trail for stock adjustments
- ✅ Flexible category assignment (optional)
- ✅ Support for negative stock (backorders)
- ✅ Optimized indexes for common queries
- ✅ Validation constraints at database level
- ✅ State transitions for archiving products

Ready to proceed to **API Contract Design** (contracts/).
