# Implementation Plan: Product & Inventory Management

**Branch**: `001-product-inventory` | **Date**: 2025-12-01 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-product-inventory/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement Product & Inventory Management feature to enable CRUD operations for products, track inventory levels with automatic sales deductions, support manual stock adjustments with audit logging, and organize products using categories. This feature provides the foundation for the POS system by defining what can be sold and tracking stock availability in real-time.

## Technical Context

**Language/Version**: Go 1.23.0 (backend services), Node.js 18+ / Next.js 16 / React 19 (frontend)  
**Primary Dependencies**: Echo v4 (HTTP framework), lib/pq (PostgreSQL driver), Redis v9, Kafka (event streaming), Axios (HTTP client)  
**Storage**: PostgreSQL 14+ with Row-Level Security (RLS) for multi-tenant isolation  
**Testing**: Go testing package with testify, Jest for frontend unit tests  
**Target Platform**: Linux server (Docker containers), Web browsers (Chrome, Firefox, Safari)  
**Project Type**: Web application (microservices backend + Next.js frontend)  
**Performance Goals**: Support 10,000 products without degradation, <2s inventory updates, <10s product search, 1000+ req/s for product reads  
**Constraints**: Multi-tenant architecture with RLS, <200ms p95 for API responses, real-time inventory sync required, 5MB max product photo size  
**Scale/Scope**: 10,000+ products per tenant, real-time inventory tracking, audit trail for all stock adjustments

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Microservice Autonomy ✅
- **Compliant**: Product service will be independently deployable with its own database schema
- **Data Ownership**: Product service owns products, categories, stock_adjustments tables
- **API Contracts**: RESTful APIs with OpenAPI documentation, async events via Kafka for inventory updates

### II. API-First Design ✅
- **Compliant**: OpenAPI contracts designed before implementation in Phase 1 (contracts/product-api.yaml)
- **Versioning**: API versioned at `/api/v1/products` with backward compatibility rules
- **Documentation**: Complete request/response schemas, error codes, and SLAs documented in contracts/

### III. Test-First Development ✅
- **Compliant**: Tests written before implementation per constitution
- **Coverage Required**: Unit tests (business logic), contract tests (API), integration tests (service interactions)
- **Sequence**: Write test → Fail → Implement → Pass → Refactor

### IV. Observability & Monitoring ✅
- **Compliant**: Structured logging for all operations, metrics for API performance
- **Audit Trail**: All stock adjustments logged with user, timestamp, reason (FR-021)
- **Health Checks**: /health and /ready endpoints for service monitoring (documented in contracts)

### V. Security by Design ✅
- **Compliant**: JWT authentication at API gateway, tenant isolation via RLS
- **Authorization**: Role-based access control for product management operations
- **Data Protection**: Product photos stored securely, file size limits enforced (FR-033)

### VI. Simplicity & YAGNI ✅
- **Compliant**: Building only required features from spec, using proven patterns
- **Technology**: Reusing existing Echo framework, PostgreSQL with RLS, no new dependencies
- **Complexity**: No premature optimization, straightforward CRUD with audit logging

**GATE STATUS: ✅ PASS** - All constitution principles satisfied post-design, no violations to justify.

**Post-Design Review**: Contracts defined in OpenAPI 3.0, data model complete with RLS policies, architecture follows existing microservices pattern.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
backend/
├── product-service/          # NEW: Product & Inventory microservice
│   ├── api/                  # HTTP handlers
│   │   ├── product_handler.go
│   │   ├── category_handler.go
│   │   └── stock_handler.go
│   ├── src/
│   │   ├── models/           # Data models
│   │   │   ├── product.go
│   │   │   ├── category.go
│   │   │   └── stock_adjustment.go
│   │   ├── repository/       # Database operations
│   │   │   ├── product_repository.go
│   │   │   ├── category_repository.go
│   │   │   └── stock_repository.go
│   │   └── services/         # Business logic
│   │       ├── product_service.go
│   │       ├── category_service.go
│   │       └── inventory_service.go
│   ├── tests/
│   │   ├── contract/         # API contract tests
│   │   ├── integration/      # Service integration tests
│   │   └── unit/            # Unit tests
│   ├── go.mod
│   └── main.go
│
├── migrations/               # Shared database migrations
│   ├── 009_create_products_table.up.sql
│   ├── 009_create_products_table.down.sql
│   ├── 010_create_categories_table.up.sql
│   ├── 010_create_categories_table.down.sql
│   ├── 011_create_stock_adjustments_table.up.sql
│   └── 011_create_stock_adjustments_table.down.sql
│
├── src/                      # Shared backend utilities (existing)
│   ├── config/
│   ├── middleware/
│   └── utils/

frontend/
├── pages/
│   └── products/             # NEW: Product management pages
│       ├── index.tsx         # Product list/catalog
│       ├── [id].tsx          # Product detail/edit
│       ├── new.tsx           # Create product
│       └── categories.tsx    # Category management
├── src/
│   ├── components/
│   │   └── products/         # NEW: Product components
│   │       ├── ProductForm.tsx
│   │       ├── ProductList.tsx
│   │       ├── CategorySelect.tsx
│   │       ├── StockAdjustment.tsx
│   │       └── InventoryDashboard.tsx
│   ├── services/
│   │   └── product.service.ts  # NEW: Product API client
│   └── types/
│       └── product.types.ts    # NEW: TypeScript types

api-gateway/                   # Existing
├── middleware/               # Add product service routing
└── main.go                   # Update routes for product service

specs/004-product-inventory/  # This feature
├── plan.md                   # This file
├── research.md               # Phase 0 output (to be created)
├── data-model.md             # Phase 1 output (to be created)
├── quickstart.md             # Phase 1 output (to be created)
└── contracts/                # Phase 1 output (to be created)
    └── product-api.yaml      # OpenAPI spec
```

**Structure Decision**: Following existing microservices pattern with web application architecture. New `product-service` microservice will be created following the same structure as existing `auth-service`, `tenant-service`, and `user-service`. Frontend pages and components added to existing Next.js application. Database migrations added to shared migrations folder with tenant-scoped RLS policies.

## Complexity Tracking

**No violations to justify** - All constitution principles are satisfied by the design.
