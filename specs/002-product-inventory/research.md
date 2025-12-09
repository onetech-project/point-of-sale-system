# Research: Product & Inventory Management

**Feature Branch**: `001-product-inventory`  
**Date**: 2025-12-01  
**Phase**: 0 - Research & Analysis

## Overview

This document consolidates research findings for implementing the Product & Inventory Management feature in a Go microservices architecture with PostgreSQL and multi-tenant isolation.

## Research Areas

### 1. Product Photo Storage Strategy

**Decision**: File system storage with database references

**Rationale**: 
- Product photos are static assets that don't require transactional consistency
- File system storage is simpler and more performant than BLOB storage in PostgreSQL
- Database stores metadata (file path, size, upload date) with foreign key to product
- Supports easy migration to cloud storage (S3, GCS) in future without schema changes
- Reduces database size and backup complexity

**Implementation**:
- Store photos in `backend/product-service/uploads/{tenant_id}/{product_id}/`
- Database stores relative path: `uploads/{tenant_id}/{product_id}/photo.jpg`
- Enforce 5MB file size limit (FR-033) at API level
- Resize/compress images on upload to optimize storage and bandwidth
- Serve via dedicated static file endpoint with proper MIME types

**Alternatives Considered**:
- PostgreSQL BYTEA/BLOB: Rejected due to database bloat, backup complexity, and performance overhead
- Cloud storage (S3): Deferred to future enhancement; file system provides migration path
- Base64 in JSON: Rejected due to 33% size overhead and poor performance

---

### 2. Inventory Tracking: Negative Quantities Handling

**Decision**: Allow negative inventory with business logic flags

**Rationale**:
- Retail scenarios include backorders, pre-orders, and consignment sales
- Hard constraint (preventing negatives) breaks valid workflows
- Business logic layer should flag negative stock and alert users
- Allows flexibility for different tenant policies (some allow backorders, others don't)

**Implementation**:
- Database: `stock_quantity` column type `INTEGER` (supports negatives)
- Service layer: Check stock before sales, return warning if insufficient
- Frontend: Display "Out of Stock" or "Backorder Available" based on quantity
- Audit log records all transitions including negative states
- Optional per-tenant setting: `allow_negative_stock` (default: true)

**Alternatives Considered**:
- Hard constraint (CHECK stock_quantity >= 0): Rejected, too restrictive
- Separate backorder table: Rejected as over-engineering for current requirements
- Unsigned integer: Rejected, doesn't support valid business scenarios

---

### 3. Multi-Tenant Row-Level Security (RLS) for Product Tables

**Decision**: PostgreSQL RLS policies with `tenant_id` on all product-related tables

**Rationale**:
- Existing architecture uses RLS for tenant isolation (auth, users tables)
- Consistency with established patterns reduces cognitive overhead
- Database-level security prevents data leaks even if application bugs exist
- All queries automatically scoped to current tenant via `SET LOCAL` statements

**Implementation**:
- Add `tenant_id UUID REFERENCES tenants(id)` to products, categories, stock_adjustments tables
- Create RLS policies: `CREATE POLICY tenant_isolation ON products USING (tenant_id = current_setting('app.current_tenant_id')::uuid)`
- Middleware sets tenant context: `SET LOCAL app.current_tenant_id = '{tenant_id}'`
- Indexes on `(tenant_id, other_columns)` for query performance

**Alternatives Considered**:
- Application-level filtering only: Rejected due to security risk
- Separate databases per tenant: Rejected due to operational complexity
- Schema-per-tenant: Rejected due to migration and maintenance overhead

---

### 4. Stock Adjustment Audit Trail Pattern

**Decision**: Immutable append-only audit log table

**Rationale**:
- Audit logs must be immutable for compliance and forensic analysis
- Append-only design prevents tampering with historical records
- Supports filtering by date, user, reason for reporting (FR-024)
- Enables full inventory reconciliation and shrinkage analysis

**Implementation**:
- `stock_adjustments` table with columns: id, product_id, tenant_id, user_id, timestamp, previous_qty, new_qty, delta, reason, created_at
- No UPDATE or DELETE operations allowed on audit records
- Database trigger auto-calculates delta: `delta = new_qty - previous_qty`
- Indexed on (product_id, created_at) and (tenant_id, created_at) for reporting
- Reason field: ENUM or TEXT with predefined values (supplier_delivery, physical_count, shrinkage, damage, return, correction)

**Alternatives Considered**:
- Single inventory_transactions table: Rejected, mixes sales and adjustments
- Soft deletes: Rejected, audit logs should never be deleted
- Event sourcing: Deferred to future enhancement, current approach supports reconstruction

---

### 5. Category Management: Hierarchy vs Flat Structure

**Decision**: Flat category structure (single-level)

**Rationale**:
- Feature spec mentions "organize products into categories" without hierarchy requirements
- Flat structure is simpler and sufficient for initial POS needs
- Most small/medium retail stores use 5-20 categories (Beverages, Snacks, Household, etc.)
- Hierarchical categories add complexity (recursive queries, path management) without current need
- Can be enhanced to support hierarchy later if needed (add parent_id column)

**Implementation**:
- `categories` table: id, tenant_id, name, display_order, created_at, updated_at
- Many-to-one relationship: products.category_id references categories(id)
- Nullable category_id to support uncategorized products (FR-011)
- ON DELETE RESTRICT to prevent category deletion with assigned products (FR-026)
- Display order for manual sorting in UI

**Alternatives Considered**:
- Hierarchical categories: Rejected as YAGNI, adds unnecessary complexity
- Tags/multi-category: Rejected, spec implies single category per product
- Hardcoded categories: Rejected, tenants need custom categories

---

### 6. SKU/Barcode Uniqueness and Format

**Decision**: Unique constraint on (tenant_id, sku) with validation

**Rationale**:
- SKUs must be unique within a tenant to prevent inventory confusion
- Different tenants can have same SKU (multi-tenant isolation)
- Barcodes (UPC, EAN) are often standardized but not guaranteed unique in POS systems
- Allow alphanumeric SKUs for flexibility (store-generated codes)

**Implementation**:
- Database: `UNIQUE INDEX ON products(tenant_id, sku)` (FR-008)
- Validation: 1-50 characters, alphanumeric plus hyphens/underscores (FR-034)
- API returns 409 Conflict if SKU already exists for tenant
- Frontend validates format before submission
- Optional auto-generation of SKUs if not provided (e.g., PROD-{UUID})

**Alternatives Considered**:
- Global SKU uniqueness: Rejected, violates multi-tenant model
- Barcode as primary identifier: Rejected, not all products have barcodes
- Composite key (SKU + tenant): Rejected, prefer surrogate UUID primary key

---

### 7. Real-Time Inventory Synchronization Strategy

**Decision**: Synchronous updates for sales, async events for cross-service notifications

**Rationale**:
- Sales transactions require immediate inventory deduction to prevent overselling
- Synchronous HTTP call from sales service to product service during checkout
- Async Kafka events for non-critical notifications (low stock alerts, reporting)
- Meets <2s inventory update requirement (SC-002)

**Implementation**:
- Sales service calls `POST /api/products/{id}/adjust-stock` with quantity delta
- Product service updates stock within transaction, returns new quantity
- On success, product service publishes `inventory.updated` event to Kafka
- Notification service consumes events for low-stock alerts
- Frontend polls /api/products/{id} or uses WebSocket for real-time dashboard updates

**Alternatives Considered**:
- Async-only (Kafka events): Rejected, risk of overselling during event lag
- WebSocket for all updates: Deferred, HTTP polling sufficient for MVP
- Two-phase commit: Rejected as over-engineering, eventual consistency acceptable for notifications

---

### 8. Price Change Handling for In-Progress Transactions

**Decision**: Transactions capture price at creation time

**Rationale**:
- Edge case defined in spec: "Current transaction uses price when it started"
- Prevents customer confusion from price changes mid-checkout
- Requires sales service to store price snapshot, not reference product price
- Product service provides current price via API, sales service owns transaction pricing

**Implementation**:
- Sales transaction stores `unit_price` at time of cart addition
- Product price updates don't affect existing carts/transactions
- Sales service responsible for price snapshot logic (out of scope for this feature)
- Product service only provides current price via GET /api/products/{id}

**Alternatives Considered**:
- Always use latest price: Rejected, violates user expectation and spec
- Lock prices during transaction: Rejected, unnecessary complexity
- Price versioning: Deferred, current approach sufficient

---

### 9. Performance Optimization for 10,000+ Products

**Decision**: Database indexing, pagination, and caching strategy

**Rationale**:
- Success criteria SC-005: Support 10,000 products without degradation
- Search and filtering are primary operations requiring optimization
- Pagination prevents memory exhaustion and improves response times

**Implementation**:
- Database indexes:
  - `(tenant_id, name)` for search
  - `(tenant_id, category_id)` for category filtering
  - `(tenant_id, sku)` for SKU lookup (already unique index)
  - `(tenant_id, archived_at)` for archive filtering
- API pagination: Default 50 items per page, max 100 (query params: page, per_page)
- Caching: Redis cache for category list (infrequently changed), 5-minute TTL
- Search: PostgreSQL full-text search on (name, description) with GIN index
- Database connection pooling: Max 25 connections per service

**Alternatives Considered**:
- Elasticsearch: Deferred, PostgreSQL full-text sufficient for 10k products
- Aggressive caching: Rejected, inventory changes frequently
- NoSQL database: Rejected, PostgreSQL handles scale and provides ACID guarantees

---

### 10. Image Upload and Processing

**Decision**: Multipart form upload with server-side validation and resizing

**Rationale**:
- Product photos enhance catalog usability but require size/format control
- 5MB limit prevents abuse and manages storage costs (FR-033)
- Resize to standard dimensions reduces bandwidth and improves load times

**Implementation**:
- API endpoint: `POST /api/products/{id}/photo` (multipart/form-data)
- Accepted formats: JPEG, PNG, WebP
- Validation: Max 5MB, min 100x100px, max 4000x4000px
- Processing: Resize to 800x800 thumbnail, maintain aspect ratio, 85% JPEG quality
- Go library: `github.com/disintegration/imaging` for image manipulation
- Store original filename in metadata for download
- Return photo URL in product response: `/api/products/{id}/photo`

**Alternatives Considered**:
- Direct upload to cloud storage: Deferred, file system for MVP
- Client-side resizing: Rejected, server validation required for security
- Multiple sizes (thumb, medium, large): Deferred, single size sufficient

---

## Technology Stack Confirmation

### Backend
- **Language**: Go 1.23.0
- **Framework**: Echo v4 (HTTP routing, middleware)
- **Database**: PostgreSQL 14+ with lib/pq driver
- **Caching**: Redis v9 (category list, session data)
- **Messaging**: Kafka (inventory update events)
- **Testing**: Go testing + testify (assertions)
- **Image Processing**: github.com/disintegration/imaging

### Frontend
- **Framework**: Next.js 16 + React 19
- **Language**: TypeScript 5.9
- **HTTP Client**: Axios
- **i18n**: next-i18next (English/Indonesian)
- **Testing**: Jest + React Testing Library

### Infrastructure
- **Deployment**: Docker containers
- **Database Migrations**: golang-migrate
- **API Documentation**: OpenAPI 3.0

---

## Best Practices Summary

### Go Microservices
- Repository pattern for database abstraction
- Service layer for business logic
- Handler layer for HTTP concerns
- Dependency injection for testability
- Structured logging (JSON format)
- Graceful shutdown handling
- Health check endpoints (/health, /ready)

### PostgreSQL
- Use prepared statements to prevent SQL injection
- Connection pooling (max 25 connections)
- Transactions for multi-statement operations
- Indexes on foreign keys and filter columns
- JSONB for flexible metadata if needed
- Avoid SELECT * in production code

### Multi-Tenancy
- Tenant ID in all queries (via RLS)
- Middleware sets tenant context from JWT
- Never trust client-provided tenant ID
- Tenant-scoped unique constraints
- Indexes include tenant_id as first column

### API Design
- RESTful endpoints with standard HTTP methods
- Versioned URLs (/api/v1/products)
- Structured error responses (code, message, details)
- Pagination for list endpoints
- 200 OK, 201 Created, 400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 409 Conflict, 500 Internal Server Error
- Request validation before business logic

### Testing
- Test-first development (TDD)
- Unit tests: 80%+ coverage requirement
- Contract tests: API request/response validation
- Integration tests: Database interactions
- Mock external dependencies (other services)
- Test data cleanup after each test

---

## Open Questions & Risks

### Resolved
- ✅ Photo storage strategy: File system with DB metadata
- ✅ Negative inventory: Allowed with business logic checks
- ✅ Category structure: Flat (single-level)
- ✅ SKU uniqueness: Per-tenant unique constraint
- ✅ Price change handling: Snapshot at transaction creation

### Future Enhancements (Out of Scope)
- Cloud storage for product photos (S3, GCS)
- Hierarchical categories (parent-child relationships)
- Barcode scanner integration
- Bulk product import/export
- Product variants (size, color)
- Low-stock notifications (Kafka consumer)
- Advanced search with Elasticsearch

---

## Summary

All research areas resolved with clear decisions aligned to constitution principles:
- **Simplicity**: File system storage, flat categories, PostgreSQL for search
- **Security**: RLS for multi-tenancy, server-side validation
- **Performance**: Indexing, pagination, caching for 10k+ products
- **Auditability**: Immutable stock adjustment logs
- **Testability**: Repository pattern, dependency injection

Ready to proceed to **Phase 1: Design & Contracts**.
