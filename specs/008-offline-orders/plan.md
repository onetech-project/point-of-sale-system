# Implementation Plan: Offline Order Management

**Branch**: `008-offline-orders` | **Date**: February 7, 2026 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/008-offline-orders/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Enable staff to manually record offline orders (walk-in customers, phone orders) with flexible payment terms including down payments and installments. All operations must be audit-trailed (CREATE, READ, UPDATE, DELETE, ACCESS), role-based deletion (owner/manager only), and seamlessly integrated with existing online order flow, analytics dashboard, and UU PDP/GDPR compliance framework.

## Technical Context

**Language/Version**: Go 1.24.0 (backend services), TypeScript/Next.js 16 + React 19 (frontend)  
**Primary Dependencies**: Echo v4.13 (REST framework), PostgreSQL 14 (primary storage), Redis 7 (caching), Kafka (event streaming), Vault (secrets), golang-migrate (migrations), go-redis/v9, lib/pq (PostgreSQL driver)  
**Storage**: PostgreSQL 14 with Row-Level Security (RLS) for tenant isolation, Redis for session/cache  
**Testing**: Go's testing package + testify/assert (backend unit/integration), Jest (frontend unit), contract tests for API boundaries  
**Target Platform**: Linux servers via Docker containers, deployed via docker-compose, API Gateway port 8080, order-service port 8084  
**Project Type**: Web application - microservices backend + SPA frontend (existing architecture)  
**Performance Goals**: <200ms p95 response time for order operations, support 100+ concurrent users per tenant, real-time analytics updates <5 seconds  
**Constraints**: Zero degradation to online order flow (FR-008 requirement), maintain audit trail immutability, all PII encrypted (deterministic encryption for searchable fields, AES-GCM for sensitive fields), RLS enforced at database layer  
**Scale/Scope**: Multi-tenant SaaS, 10-50 tenants initially, 100-1000 orders per tenant per month, extend existing order-service (~15 new endpoints), reuse audit-service and analytics-service

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

### Principle I: Microservice Autonomy ✅

- **Status**: PASS
- **Assessment**: Extends existing `order-service` which already owns order data and exposes REST APIs. No new service needed (YAGNI). Offline orders share domain with online orders (both are "orders"), justifying same service boundary.

### Principle II: API-First Design ✅

- **Status**: PASS
- **Assessment**: OpenAPI contracts defined in Phase 1 (`contracts/openapi-offline-orders.yaml`) with 6 REST endpoints. Kafka event schemas defined (`contracts/kafka-events.md`) for audit and analytics. Versioning via `/api/v1` prefix maintained.
- **Post-Phase 1**: ✅ VALIDATED - Contract definitions complete, no breaking changes to existing APIs, all new endpoints follow RESTful conventions

### Principle III: Test-First Development (NON-NEGOTIABLE) ✅

- **Status**: PASS (enforced in tasks phase)
- **Assessment**: All implementation tasks will follow TDD: write failing test → implement → verify pass → refactor. Test coverage target ≥80% maintained.

### Principle IV: Observability & Monitoring ✅

- **Status**: PASS
- **Assessment**: Reuse existing OpenTelemetry instrumentation in order-service (traces, metrics, logs). Add business metrics for offline order operations to existing Prometheus exporter.

### Principle V: Security by Design ✅

- **Status**: PASS
- **Assessment**:
  - Auth/authz via API Gateway (existing JWT middleware)
  - Role-based deletion enforcement (owner/manager check)
  - PII encryption using existing encryption service (Vault-backed)
  - Audit trail for all operations (CREATE, READ, UPDATE, DELETE, ACCESS)

### Principle VI: Simplicity First (KISS + DRY + YAGNI) ✅

- **Status**: PASS
- **Assessment**:
  - **KISS**: Extend existing `guest_orders` table with `order_type` enum (online/offline) vs. creating separate `offline_orders` table
  - **DRY**: Reuse existing order repository, encryption service, audit logging
  - **YAGNI**: No premature abstraction; implement only required payment terms (full, down payment, installments) without complex financial modeling

### Engineering Best Practices Check ✅

- **TDD**: Enforced in tasks phase
- **BDD**: Acceptance scenarios already defined in spec.md (Given-When-Then format)
- **MVP Approach**: P1 user story (basic order recording) delivers standalone value
- **Refactoring**: Incremental approach, refactor when adding features to poorly designed code
- **Testability**: Dependency injection pattern already used in order-service

**GATE RESULT**: ✅ **PASS** - Proceed to Phase 0 Research

---

### Post-Phase 1 Re-evaluation ✅

**Date**: February 7, 2026 | **Status**: ALL PRINCIPLES PASS

After completing research.md, data-model.md, and contract definitions, validated design against constitution:

**Principle I (Microservice Autonomy)**: ✅

- Design extends order-service only, no new services
- Payment modeling stays within bounded context (orders domain)
- Event schemas published to existing topics (no new infrastructure)

**Principle II (API-First Design)**: ✅

- OpenAPI contract: 6 endpoints, RESTful conventions, versioned `/api/v1` paths
- Kafka event schemas: Backward-compatible with existing audit-events and order-events topics
- No breaking changes to online order endpoints

**Principle III (Test-First Development)**: ✅

- quickstart.md includes comprehensive testing strategy
- Unit, integration, and contract test templates provided
- TDD workflow documented (failing test → implement → verify → refactor)

**Principle IV (Observability & Monitoring)**: ✅

- All operations emit traces via existing OpenTelemetry instrumentation
- Business metrics defined (offline order count, payment completion rate)
- Event outbox ensures reliable audit trail even during failures

**Principle V (Security by Design)**: ✅

- PII encryption specified in data-model.md (deterministic for searchable, AES-GCM for sensitive)
- Role-based deletion enforced via middleware + service-layer double-check
- All operations audit-logged with event IDs for non-repudiation
- RLS constraints maintained for tenant isolation

**Principle VI (Simplicity First)**: ✅

- **KISS**: Schema design extends 1 table, adds 2 new tables (minimal complexity)
- **DRY**: Reuses existing encryption service, audit pipeline, analytics consumers
- **YAGNI**: Payment modeling supports required features only (no speculative financial abstractions), event outbox pattern prevents over-engineering Event Sourcing

**Engineering Best Practices**: ✅

- Quickstart provides incremental implementation path (migrations → models → repos → services → handlers)
- Estimated 35-45 hours for full implementation
- No gold-plating detected in design

**Constitution Compliance**: ✅ **100% PASS** - Design ready for implementation (Phase 2: tasks.md)

## Project Structure

### Documentation (this feature)

```text
specs/008-offline-orders/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/ speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── openapi-offline-orders.yaml
│   └── kafka-events.yaml
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
backend/
├── order-service/              # EXTEND - add offline order endpoints
│   ├── api/
│   │   ├── offline_orders_handler.go    # NEW
│   │   ├── offline_payments_handler.go  # NEW
│   │   └── offline_orders_admin.go      # NEW
│   ├── src/
│   │   ├── models/
│   │   │   ├── offline_order.go           # NEW
│   │   │   └── payment_record.go          # NEW
│   │   ├── repository/
│   │   │   ├── offline_order_repository.go  # NEW
│   │   │   └── payment_repository.go        # NEW
│   │   ├── services/
│   │   │   ├── offline_order_service.go   # NEW
│   │   │   └── payment_terms_service.go   # NEW
│   │   └── middleware/
│   │       └── role_check.go              # EXTEND - add offline order deletion check
│   └── tests/
│       ├── unit/
│       │   └── offline_orders_test.go     # NEW
│       ├── integration/
│       │   └── offline_orders_integration_test.go  # NEW
│       └── contract/
│           └── offline_orders_contract_test.go     # NEW
├── audit-service/              # MINOR EXTEND - ensure offline order events captured
├── analytics-service/          # MINOR EXTEND - add offline order metrics to dashboards
├── migrations/
│   ├── 000060_add_offline_orders.up.sql      # NEW - extend guest_orders table
│   ├── 000060_add_offline_orders.down.sql
│   ├── 000061_add_payment_records.up.sql     # NEW
│   ├── 000061_add_payment_records.down.sql
│   ├── 000062_add_payment_terms.up.sql       # NEW
│   └── 000062_add_payment_terms.down.sql

frontend/
├── app/
│   └── orders/
│       ├── offline-orders/           # NEW
│       │   ├── page.tsx              # List view
│       │   ├── new/
│       │   │   └── page.tsx          # Create form
│       │   ├── [id]/
│       │   │   ├── page.tsx          # Detail view
│       │   │   └── edit/             # Edit form
│       │   │       └── page.tsx
├── src/
│   ├── components/
│   │   └── orders/
│   │       ├── OfflineOrderForm.tsx       # NEW
│   │       ├── OfflineOrderList.tsx       # NEW
│   │       ├── PaymentSchedule.tsx        # NEW
│   │       └── OfflineOrderDetail.tsx     # NEW
│   ├── services/
│   │   └── offlineOrders.ts               # NEW - API client
│   └── types/
│       └── offlineOrder.ts                # NEW - TypeScript interfaces
└── tests/
    └── components/
        └── admin/
            └── offline-orders.test.tsx    # NEW
```

**Structure Decision**: Using Option 2 (Web application) as this is an existing microservices architecture. Backend services in `backend/` directory, frontend SPA in `frontend/`. Extending existing `order-service` rather than creating new service (Simplicity First principle). Reusing existing infrastructure: API Gateway, Auth, Audit, Analytics services.

## Complexity Tracking

> **No constitution violations detected. This section intentionally left minimal.**

No violations of constitution principles. All complexity is justified:

- Extending existing `order-service` (not creating new service) - follows Simplicity First
- Reusing existing encryption, audit, analytics infrastructure - follows DRY
- Adding only required features (no speculative code) - follows YAGNI

---

## Agent Context

**Status**: Phase 1 complete | **Last Updated**: February 7, 2026

### Implementation Guidance

This section provides AI coding agents with quick navigation to design artifacts and key implementation decisions.

#### Phase 0: Research Decisions (COMPLETE)

**Artifact**: [research.md](research.md)

**Key Decisions**:

1. **Database Schema**: Extend existing `guest_orders` table with `order_type` column (vs. separate table) - follows DRY principle
2. **Payment Modeling**: Two-table approach - `payment_terms` (1:1 summary) + `payment_records` (1:N transaction log)
3. **RBAC Enforcement**: Echo middleware + service-layer double-check for deletion (owner/manager only)
4. **Audit Integration**: Reuse existing `audit-events` Kafka topic with transactional outbox pattern
5. **Analytics Integration**: Extend existing `order-events` topic (no new infrastructure)
6. **Architectural Isolation**: Separate endpoints, no inventory locks for offline orders, performance isolation
7. **Compliance Strategy**: Leverage existing encryption + consent tracking mechanisms
8. **Payment Methods**: Simple enum (cash, card, bank_transfer, check, other) - avoid over-engineering
9. **Status Lifecycle**: Extend existing order status enum with PENDING → PAID → COMPLETED flow
10. **Technology Best Practices**: Follow existing patterns in order-service codebase

**Referenced By**: data-model.md, contracts/

#### Phase 1: Data Model (COMPLETE)

**Artifact**: [data-model.md](data-model.md)

**Contents**:

- Entity-Relationship Diagram (Mermaid format)
- Database schema for 3 tables:
  - Extended `guest_orders` table (migration 000018)
  - New `payment_terms` table (migration 000019)
  - New `payment_records` table (migration 000020)
- Full migration scripts (up/down SQL)
- Query patterns with optimized indexes
- Validation rules and constraints
- Security & encryption requirements

**Key Schema Changes**:

```sql
-- guest_orders extensions
ALTER TABLE guest_orders ADD COLUMN order_type VARCHAR(20);
ALTER TABLE guest_orders ADD COLUMN recorded_by_user_id UUID;
ALTER TABLE guest_orders ADD COLUMN last_modified_by_user_id UUID;

-- New tables
CREATE TABLE payment_terms (order_id UUID UNIQUE, total_amount INT, ...);
CREATE TABLE payment_records (order_id UUID, amount_paid INT, ...);
CREATE TABLE event_outbox (event_type VARCHAR, event_payload JSONB, ...);
```

**Referenced By**: quickstart.md, contracts/

#### Phase 1: API Contracts (COMPLETE)

**Artifacts**:

- [contracts/openapi-offline-orders.yaml](contracts/openapi-offline-orders.yaml) - REST API specification
- [contracts/kafka-events.md](contracts/kafka-events.md) - Event schemas

**REST Endpoints** (6 total):

1. `POST /offline-orders` - Create order with full/installment payment
2. `GET /offline-orders` - List orders with filters (status, date range, search)
3. `GET /offline-orders/{orderId}` - Get order details with payment history
4. `PATCH /offline-orders/{orderId}` - Update customer info or items
5. `DELETE /offline-orders/{orderId}` - Soft delete (owner/manager only)
6. `POST /offline-orders/{orderId}/payments` - Record installment payment

**Event Topics**:

- `audit-events`: offline_order.created, updated, deleted, accessed, payment_recorded
- `order-events`: order.created, order.completed, payment.received

**Security**: JWT bearer tokens, tenant isolation via RLS, role-based deletion

**Referenced By**: quickstart.md, handler implementations

#### Phase 1: Implementation Guide (COMPLETE)

**Artifact**: [quickstart.md](quickstart.md)

**Contents**:

- Step-by-step implementation phases (Migrations → Models → Repositories → Services → Handlers → Frontend)
- Code examples for each layer (Go backend, TypeScript frontend)
- Testing strategy (unit, integration, contract tests)
- Verification checklist
- Troubleshooting guide

**Implementation Order**:

1. Phase 0: Database migrations (000060-000062) - 1-2 hours
2. Phase 1: Backend models (order, payment_terms, payment_record) - 2-3 hours
3. Phase 2: Repository layer (offline_order_repository, payment_repository) - 4-5 hours
4. Phase 3: Service layer (offline_order_service) - 5-6 hours
5. Phase 4: API handlers (offline_orders_handler) - 4-5 hours
6. Phase 5: Middleware & routes (role_check middleware) - 2-3 hours
7. Phase 6: Testing (unit, integration, contract) - 6-8 hours
8. Phase 7: Frontend components (OfflineOrderForm, PaymentSchedule) - 8-10 hours

**Estimated Total**: ~35-45 hours

**Referenced By**: tasks.md (next phase)

### Cross-Cutting Concerns

#### Encryption

- **Customer PII**: Deterministic encryption for searchable fields (phone, email), AES-GCM for non-searchable (name)
- **Vault Integration**: Reuse existing `EncryptionService` in order-service
- **Performance**: Batch encrypt/decrypt operations, cache encryption keys

#### Audit Trail

- **Events**: All CRUD + ACCESS operations logged to `audit-events` Kafka topic
- **Transactional Outbox**: Ensure event publishing survives database transaction failures
- **PII Protection**: Use SHA-256 hashes for change tracking, no plaintext PII in events

#### Analytics Integration

- **Order Metrics**: Offline orders included in total order count, revenue, customer acquisition
- **Payment Analytics**: Track installment payment patterns, completion rates
- **Dashboard Updates**: <5 seconds latency via Kafka streaming

#### Performance Isolation

- **Query Separation**: Offline orders use dedicated indexes, no join with online orders
- **Write Isolation**: No inventory locks for offline orders (manual sync if needed)
- **Monitoring**: Track p95 latency separately for offline vs. online operations

### Technology Context (Auto-Updated)

The following technologies were added to `.github/agents/copilot-instructions.md`:

- **Language**: Go 1.24.0 (backend services), TypeScript/Next.js 16 + React 19 (frontend)
- **Framework**: Echo v4.13 (REST framework), PostgreSQL 14 (primary storage), Redis 7 (caching), Kafka (event streaming), Vault (secrets), golang-migrate (migrations), go-redis/v9, lib/pq (PostgreSQL driver)
- **Database**: PostgreSQL 14 with Row-Level Security (RLS) for tenant isolation, Redis for session/cache

**Note**: Manual technology additions between markers are preserved by `update-agent-context.sh`.

### Next Steps (After This Phase)

Run `/speckit.tasks` command to generate `tasks.md` with:

- Detailed task breakdown (database → backend → frontend → tests)
- Dependencies between tasks
- Estimated time per task
- Acceptance criteria per task
