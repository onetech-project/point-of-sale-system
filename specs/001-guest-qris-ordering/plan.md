# Implementation Plan: QRIS Guest Ordering System

**Branch**: `001-guest-qris-ordering` | **Date**: 2025-12-03 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-guest-qris-ordering/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement a guest ordering system allowing unauthenticated customers to browse products via tenant-specific public URLs, add items to cart, select delivery options, and complete QRIS payments through Midtrans integration. System manages inventory with time-limited reservations, geocodes delivery addresses for serviceability validation, optionally calculates delivery fees based on tenant configuration, and provides admin interface for tenant staff to manage order fulfillment.

## Technical Context

**Language/Version**: Go 1.23.0 (backend microservices), Next.js 16 with TypeScript (frontend)
**Primary Dependencies**: Echo v4 (API framework), PostgreSQL (persistent data), Redis (session/cart/cache), Midtrans Snap API (payment), Google Maps Geocoding API (address validation)
**Storage**: PostgreSQL for orders/inventory/tenants (6 new tables), Redis for cart sessions and geocoding cache
**Testing**: Go testing package with testify, React Testing Library with Jest
**Target Platform**: Linux servers (Docker containers), web browsers (mobile-optimized)
**Project Type**: Web application - backend microservices + frontend SPA
**Performance Goals**: <2s page load on 3G, <3s payment notification processing (95th percentile), 100 concurrent users
**Constraints**: 15-minute inventory reservation TTL, idempotent payment webhooks, PCI-DSS secure payment handling
**Scale/Scope**: 7 user stories, 85+ functional requirements, multi-tenant architecture with existing auth-service and tenant-service integration

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence/Justification |
|-----------|--------|------------------------|
| **I. Microservice Autonomy** | ✅ PASS | New order-service owns guest_orders, order_items, inventory_reservations, payment_transactions, delivery_addresses tables. Communicates with existing product-service (inventory checks), tenant-service (config), auth-service (optional staff auth). Uses Redis for cart (session-scoped, not shared DB). Midtrans webhook is async event. |
| **II. API-First Design** | ⚠️ DEFERRED | API contracts will be defined in Phase 1 (contracts/ directory) before implementation. REST endpoints for guest ordering flow, webhook endpoint for Midtrans. Versioning TBD based on existing API gateway patterns. |
| **III. Test-First Development** | ✅ PASS | Specification explicitly requires test-first workflow per tasks.md (29 test tasks across all user stories, "Write FIRST - Must FAIL before implementation"). Unit tests for business logic, integration tests for Midtrans/Google Maps, contract tests for service boundaries. |
| **IV. Observability & Monitoring** | ⚠️ DEFERRED | Will implement structured logging for all payment transactions, inventory reservations, order state changes. Health checks for order-service, circuit breakers for external APIs (Midtrans, Google Maps). Specific logging framework TBD in Phase 0 research. |
| **V. Security by Design** | ⚠️ NEEDS ATTENTION | Guest endpoints are unauthenticated by design (public menu). Admin endpoints for staff require auth-service integration. Midtrans signature verification required (FR-040, FR-073, FR-076). HTTPS enforced (FR-071). Payment data NOT stored (FR-074). Order reference numbers cryptographically secure (FR-075). Rate limiting required (FR-072). NO PCI-DSS scope (Midtrans handles payment). |
| **VI. Simplicity & YAGNI** | ✅ PASS | Building only specified features: guest ordering, cart management, payment flow, basic delivery fee calculation. No premature optimization (e.g., no caching until proven necessary, simple localStorage for cart). Delivery fee calculation can be disabled per tenant (no forced complexity). Using existing services (auth, tenant, product) rather than rebuilding. |

**Overall Gate Status**: ⚠️ **CONDITIONAL PASS** - Proceed to Phase 0 with requirement to finalize API contracts (Principle II) and observability implementation details (Principle IV) during research phase. Security implementation (Principle V) is well-defined and passes.

## Project Structure

### Documentation (this feature)

```text
specs/001-guest-qris-ordering/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── order-service-api.yaml
│   ├── midtrans-webhook.yaml
│   └── product-service-contract.yaml
├── spec.md              # Feature specification (already exists)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
backend/
├── order-service/           # NEW SERVICE for this feature
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go
│   ├── src/
│   │   ├── models/         # guest_order.go, cart.go, reservation.go
│   │   ├── repository/     # PostgreSQL data access
│   │   ├── services/       # Business logic (order, cart, inventory, payment, geocoding, delivery_fee)
│   │   ├── handlers/       # HTTP handlers (guest API, webhook)
│   │   └── middleware/     # Validation, logging, rate limiting
│   └── tests/
│       ├── unit/           # Business logic tests
│       ├── integration/    # Database, Redis, external API tests
│       └── contract/       # API contract verification tests
├── product-service/         # EXISTING - minor changes for inventory checks
├── tenant-service/          # EXISTING - config queries
├── auth-service/            # EXISTING - admin endpoint auth
└── migrations/              # NEW tables: guest_orders, order_items, inventory_reservations,
                            #            payment_transactions, delivery_addresses, tenant_configs updates

frontend/
├── src/
│   ├── components/
│   │   ├── guest/          # NEW: ProductCatalog, Cart, CheckoutForm, OrderStatus
│   │   └── admin/          # NEW: OrderManagement dashboard
│   ├── pages/
│   │   ├── [tenantSlug]/   # NEW: Public menu pages
│   │   │   ├── index.tsx   # Product catalog
│   │   │   ├── checkout.tsx
│   │   │   └── order/[ref].tsx  # Order status
│   │   └── admin/
│   │       └── orders.tsx  # Staff order management
│   ├── services/
│   │   ├── orderService.ts  # NEW: API calls to order-service
│   │   ├── cartService.ts   # NEW: localStorage cart management
│   │   └── paymentService.ts # NEW: Midtrans integration
│   └── lib/
│       └── cartStorage.ts   # NEW: localStorage utilities
└── tests/
    ├── components/
    └── integration/

api-gateway/                 # EXISTING - add routes for order-service
└── middleware/
    └── rate_limit.go        # UPDATE: Add guest endpoint limits
```

**Structure Decision**: Web application (Option 2) with backend microservices and frontend SPA. New order-service is autonomous microservice following existing pattern (auth-service, tenant-service, product-service). Frontend adds new public guest pages and admin order management. Existing services require minimal changes (product-service for inventory queries, tenant-service for config, api-gateway for routing).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**No violations requiring justification.** All constitution principles either pass or have deferred items that will be resolved in Phase 0/1 (API contracts, observability implementation details). Security principle has clear implementation path with no complexity concerns.

## Phase 0: Outline & Research

**Goal**: Resolve all "NEEDS CLARIFICATION" items from Technical Context and generate research.md with technology decisions, best practices, and integration patterns.

### Research Tasks

1. **Midtrans QRIS Integration**
   - Decision: Which Midtrans API (Snap vs Core API) for QRIS payments
   - Rationale: Snap API provides hosted payment page, Core API requires custom UI
   - Research: Webhook signature verification, idempotency handling, retry logic
   - Alternatives: Core API (more control, more complexity)

2. **Google Maps Geocoding API**
   - Decision: Geocoding API for address → coordinates conversion
   - Rationale: Industry standard, reliable, good Indonesia coverage
   - Research: API key management, rate limiting, caching strategy (Redis 7-day TTL)
   - Alternatives: Nominatim (free but less reliable), Mapbox (similar cost/features)

3. **Service Area Validation**
   - Decision: Haversine formula for radius-based, point-in-polygon for zones
   - Rationale: Simple geometry sufficient for retail delivery ranges
   - Research: PostGIS for complex polygons vs in-memory calculation
   - Alternatives: External geocoding services with built-in validation (vendor lock-in)

4. **Delivery Fee Calculation**
   - Decision: Configurable per tenant (distance tiers OR zone mappings)
   - Rationale: Different businesses have different pricing models
   - Research: JSONB storage format for flexible config, calculation algorithms
   - Alternatives: Fixed formula (inflexible), external pricing service (unnecessary complexity)

5. **Inventory Reservation Pattern**
   - Decision: PostgreSQL row-level locks with TTL-based expiration
   - Rationale: ACID guarantees for race conditions, background job for cleanup
   - Research: Pessimistic locking vs optimistic with retry, expiration job frequency
   - Alternatives: Redis distributed locks (eventual consistency issues), optimistic locking (retry storms)

6. **Cart Storage**
   - Decision: Browser localStorage for cart persistence, Redis for server-side session
   - Rationale: Survives page refresh, no server load for browsing, session for checkout
   - Research: localStorage size limits, expiration strategy, sync between client/server
   - Alternatives: Session-only (lost on refresh), server-side only (unnecessary load)

7. **Observability Stack**
   - Decision: Structured JSON logging with zerolog, Prometheus metrics, OpenTelemetry traces
   - Rationale: Aligns with existing services, proven in production
   - Research: Log levels, metric cardinality, trace sampling rates
   - Alternatives: ELK stack (heavier), simple file logging (insufficient for distributed system)

8. **Rate Limiting Strategy**
   - Decision: Token bucket algorithm via api-gateway middleware
   - Rationale: Protects against abuse, fair resource allocation
   - Research: Per-IP limits for guests, per-tenant limits, burst allowances
   - Alternatives: Fixed window (bursty), sliding window (more complex)

**Output**: `research.md` documenting all decisions with rationale and rejected alternatives

## Phase 1: Design & Contracts

**Goal**: Generate data-model.md, API contracts in contracts/, quickstart.md, and update agent context

### 1. Data Model Design

Generate `data-model.md` with PostgreSQL schema for:

**New Tables**:
- `guest_orders`: Core order records (already defined in spec.md)
- `order_items`: Line items with prices at time of order
- `inventory_reservations`: Temporary holds with TTL and status
- `payment_transactions`: Midtrans interaction log
- `delivery_addresses`: Geocoded addresses with validation status

**Updated Tables**:
- `tenant_configs`: Add enable_delivery_fee_calculation, delivery_fee_type, delivery_fee_config, service_area_type, service_area_data

**Indexes**: Query patterns for order lookups, reservation expiration, payment reconciliation

**Constraints**: Foreign keys, check constraints, unique constraints

### 2. API Contract Generation

Generate OpenAPI 3.0 specs in `contracts/`:

**order-service-api.yaml**:
- `GET /api/v1/public/:tenantSlug/products` - List products for public menu
- `POST /api/v1/public/:tenantSlug/cart` - Add to cart (returns cart state)
- `GET /api/v1/public/:tenantSlug/cart/:sessionId` - Get cart contents
- `PUT /api/v1/public/:tenantSlug/cart/:sessionId` - Update cart
- `POST /api/v1/public/:tenantSlug/orders` - Create pending order
- `POST /api/v1/public/:tenantSlug/orders/:orderRef/payment` - Initiate Midtrans payment
- `GET /api/v1/public/:tenantSlug/orders/:orderRef` - Order status lookup
- `POST /api/v1/webhooks/midtrans/notification` - Midtrans payment webhook
- `GET /api/v1/admin/orders` - List orders (auth required)
- `PATCH /api/v1/admin/orders/:orderRef/status` - Update order status (auth required)
- `POST /api/v1/admin/orders/:orderRef/notes` - Add notes (auth required)

**midtrans-webhook.yaml**: Document expected Midtrans notification payload and signature verification

**product-service-contract.yaml**: Document inventory check endpoint used by order-service

### 3. Quickstart Guide

Generate `quickstart.md` with:
- Prerequisites (Go 1.23, Node 18+, PostgreSQL, Redis, Docker)
- Environment variables (Midtrans keys, Google Maps API key, database URLs)
- Database migrations
- Running order-service locally
- Running frontend development server
- Testing payment flow with Midtrans sandbox

### 4. Agent Context Update

Run `.specify/scripts/bash/update-agent-context.sh copilot` to add:
- order-service patterns (Echo handlers, repository pattern, service layer)
- Midtrans integration best practices
- Google Maps API usage
- Cart management patterns
- Test-first workflow for this feature

**Output**: data-model.md, contracts/*.yaml, quickstart.md, updated agent context file

## Phase 1: Re-evaluate Constitution Check

After design phase, verify:
- ✅ API contracts defined (Principle II satisfied)
- ✅ Observability stack chosen (Principle IV satisfied)
- ✅ All security requirements have implementation plan (Principle V satisfied)

**Expected Result**: All principles PASS, ready for Phase 2 (task breakdown via `/speckit.tasks`)

## Notes

- Phase 2 (/speckit.tasks) will generate tasks.md from this plan + data-model.md + contracts
- Implementation should follow test-first workflow defined in tasks.md
- Existing services (product, tenant, auth) require minimal changes - coordinate via API contracts
- Midtrans sandbox testing critical before production deployment
