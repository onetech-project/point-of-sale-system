# Implementation Plan: Business Insights Dashboard

**Branch**: `007-business-insights-dashboard` | **Date**: 2026-01-31 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/007-business-insights-dashboard/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Tenant owners require a comprehensive business insights dashboard providing real-time and historical analytics including sales performance, product rankings, customer spending analysis, inventory valuation, and operational task alerts. The dashboard features interactive chart visualizations with multi-level time-series filtering (daily/weekly/monthly/quarterly/yearly) and context-aware date ranges. Technical approach leverages existing microservice architecture with a new analytics service aggregating data from order, product, and tenant services, exposing RESTful APIs consumed by a Next.js frontend with chart visualization library.

## Technical Context

**Language/Version**: Go 1.24.0 (backend analytics service), TypeScript 5.9+ (frontend), Next.js 16.0.3  
**Primary Dependencies**: Echo v4.13.4 (HTTP), PostgreSQL lib/pq 1.10.9 (database), Redis go-redis v9.7.0 (caching), React 19.2.0, Chart library (TBD - Recharts vs Chart.js vs Apache ECharts)  
**Storage**: PostgreSQL 14+ (shared database with tenant isolation via RLS), Redis 7+ (caching aggregated metrics)  
**Testing**: Go testing stdlib + testify 1.11.1 (backend), Jest 30.2.0 + React Testing Library 16.3.0 (frontend)  
**Target Platform**: Linux server (Docker containers), modern web browsers (Chrome/Firefox/Safari/Edge latest 2 versions)  
**Project Type**: Web application - microservice architecture with backend analytics service + frontend dashboard UI  
**Performance Goals**: Dashboard load <2s, chart rendering <3s, support 365 daily data points, handle 50,000 monthly orders and 10,000 products per tenant  
**Constraints**: <200ms p95 for metric aggregation queries, tenant data isolation (zero leakage), read-heavy workload (analytics queries don't modify transactional data), chart updates synchronous on filter change  
**Scale/Scope**: Multi-tenant system, ~5-10 dashboard API endpoints, 4-6 complex aggregation queries, 3-5 chart components, 1 new microservice (analytics-service)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Compliance Notes |
|-----------|--------|------------------|
| **I. Microservice Autonomy** | ✅ PASS | New analytics-service will be independently deployable with its own health checks and observability. Service reads from shared database (orders, products, tenants tables) via RLS-protected queries but doesn't own data - acceptable read-only pattern. No shared database writes. Exposes well-defined REST APIs. |
| **II. API-First Design** | ✅ PASS | Will design and document OpenAPI contracts before implementation (Phase 1). All analytics endpoints (overview, trends, tasks) will have defined schemas, error codes, and pagination. Backend APIs are internal (not versioned initially since new service). |
| **III. Test-First Development** | ✅ PASS | Will write tests before implementation: (1) Contract tests for API schemas, (2) Unit tests for aggregation logic, (3) Integration tests for database queries with test data. Test-first cycle enforced in Phase 2 tasks. |
| **IV. Observability & Monitoring** | ✅ PASS | Will emit structured logs (zerolog), expose /health and /ready endpoints, integrate with existing OpenTelemetry tracing. Slow query logging for analytics queries. Dashboard access and query performance metrics to Prometheus. |
| **V. Security by Design** | ✅ PASS | Authentication enforced via existing JWT middleware from auth-service. Authorization: only tenant owners access dashboard (role check in middleware). Tenant ID from JWT enforces RLS - no cross-tenant data leakage. No PII in analytics (aggregated metrics only). Customer identification uses encrypted phone numbers already in database. |
| **VI. Simplicity First (KISS/DRY/YAGNI)** | ✅ PASS | Building only current month default + filterable historical views (no speculative features like export, scheduled reports, anomaly detection). Standard SQL aggregations (no complex ML). Leverages existing microservice patterns. Chart library selection will favor simplicity (see research phase). No premature optimization - cache only if performance tests show need. |

**GATE RESULT**: ✅ **APPROVED** - All principles compliant. No violations requiring justification.

### Post-Design Re-Evaluation (2026-01-31)

**Changes Since Initial Check**: 
- Completed research phase (chart library: Recharts)
- Completed design phase (data model, API contracts, quickstart docs)
- Decisions finalized for all technical unknowns

**Re-Evaluation**:

| Principle | Post-Design Status | Notes |
|-----------|-------------------|-------|
| **I. Microservice Autonomy** | ✅ PASS | Design confirms read-only database access pattern. No new shared state. Service fully autonomous. Contract defined in OpenAPI spec (analytics-api.yaml). |
| **II. API-First Design** | ✅ PASS | **Confirmed** - OpenAPI 3.0 spec created with 5 endpoints, complete request/response schemas, error codes, and validation rules. API design complete before implementation. |
| **III. Test-First Development** | ✅ PASS | Quickstart.md documents test strategy: unit tests for services, integration tests for database queries, contract tests against OpenAPI spec. Test data seed scripts included. |
| **IV. Observability & Monitoring** | ✅ PASS | Design includes health endpoints, query_time_ms in responses, cache_hit metrics, Prometheus metrics documented in quickstart. Performance targets defined (<100ms p95). |
| **V. Security by Design** | ✅ PASS | JWT authentication specified in OpenAPI security scheme. Tenant isolation via RLS enforced in all queries. Customer phone masking in UI defined. No cross-tenant queries possible. |
| **VI. Simplicity First** | ✅ PASS | **Confirmed** - Recharts chosen for simplicity. PostgreSQL window functions (no complex frameworks). Redis caching with simple TTL. MVP scope maintained - no export, no ML, no real-time. 7 entities defined (not over-engineered). |

**Final Assessment**: ✅ **ALL PRINCIPLES MAINTAINED** - Design phase did not introduce any complexity violations. Feature remains aligned with constitution.

**Key Simplicity Wins**:
- Chose Recharts (lightweight) over ECharts (heavy)
- No background jobs - query on demand
- No materialized views (YAGNI) - direct queries sufficient
- No new message queues - synchronous API adequate
- Reused existing authentication, tenant isolation, observability patterns

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
├── analytics-service/          # New service for dashboard analytics
│   ├── main.go
│   ├── go.mod
│   ├── .env.example
│   ├── README.md
│   ├── api/
│   │   ├── health_handler.go
│   │   ├── analytics_handler.go    # Dashboard metrics endpoints
│   │   └── tasks_handler.go        # Operational tasks endpoints
│   ├── src/
│   │   ├── config/
│   │   │   ├── database.go
│   │   │   └── redis.go
│   │   ├── middleware/
│   │   │   └── tenant_auth.go      # Reuse existing pattern
│   │   ├── models/
│   │   │   ├── analytics.go        # Metric response models
│   │   │   └── tasks.go            # Task alert models
│   │   ├── repository/
│   │   │   ├── sales_repository.go      # Sales aggregation queries
│   │   │   ├── product_repository.go    # Product performance queries
│   │   │   ├── customer_repository.go   # Top customer queries
│   │   │   └── task_repository.go       # Delayed orders, low stock
│   │   ├── services/
│   │   │   ├── analytics_service.go     # Business logic
│   │   │   └── cache_service.go         # Redis caching strategy
│   │   └── utils/
│   │       ├── time_series.go           # Date range helpers
│   │       └── formatting.go            # Number formatting (K/M)
│   └── tests/
│       ├── contract/
│       ├── integration/
│       └── unit/
│
├── order-service/              # Existing - data source
├── product-service/            # Existing - data source
└── tenant-service/             # Existing - tenant config

frontend/
├── src/
│   ├── components/
│   │   ├── dashboard/          # New dashboard components
│   │   │   ├── DashboardLayout.tsx
│   │   │   ├── MetricCard.tsx        # Sales, profit, inventory cards
│   │   │   ├── SalesChart.tsx        # Chart wrapper
│   │   │   ├── ProductRankingTable.tsx   # Top/bottom products
│   │   │   ├── CustomerRankingTable.tsx  # Top customers
│   │   │   ├── TaskAlerts.tsx        # Delayed orders, low stock
│   │   │   ├── TimeSeriesFilter.tsx  # Date range selector
│   │   │   └── QuickActions.tsx      # Invite, Settings buttons
│   │   └── charts/             # Reusable chart components
│   │       ├── LineChart.tsx
│   │       ├── BarChart.tsx
│   │       └── AreaChart.tsx
│   ├── services/
│   │   └── analyticsService.ts   # API client for analytics endpoints
│   ├── types/
│   │   └── analytics.ts          # TypeScript interfaces
│   └── app/
│       └── dashboard/
│           └── page.tsx          # Dashboard page route
│
api-gateway/                    # Existing - routing
└── (add analytics-service routes)

docker-compose.yml              # Add analytics-service container
```

**Structure Decision**: Web application (Option 2) with separate backend/ and frontend/ directories. Adding new `analytics-service` to existing backend microservices following established service structure pattern (api/, src/{config,models,repository,services,utils}, tests/). Frontend adds dashboard feature module under src/components/dashboard/ and new page route. Structure mirrors existing product-service and order-service patterns for consistency.

## Complexity Tracking

> **No violations** - Constitution Check passed all principles. No justifications needed.
