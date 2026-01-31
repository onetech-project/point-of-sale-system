# Tasks: Business Insights Dashboard

**Feature**: 007-business-insights-dashboard  
**Input**: Design documents from `/specs/007-business-insights-dashboard/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/analytics-api.yaml

**Tests**: Tests are NOT explicitly requested in feature specification, so test tasks are EXCLUDED from this task list.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [x] T001 Create analytics-service directory structure in backend/analytics-service/
- [x] T002 Initialize Go module with go.mod for analytics-service
- [x] T003 [P] Create .env.example with database and Redis config in backend/analytics-service/
- [x] T004 [P] Add analytics-service to docker-compose.yml with port 8089
- [x] T005 [P] Create README.md for analytics-service in backend/analytics-service/
- [x] T006 [P] Install Recharts and TypeScript types in frontend: npm install recharts @types/recharts

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T007 Create database connection handler in backend/analytics-service/src/config/database.go
- [x] T008 [P] Create Redis client wrapper in backend/analytics-service/src/config/redis.go
- [x] T009 [P] Create tenant context middleware to extract tenant_id from request headers (set by api-gateway) in backend/analytics-service/src/middleware/tenant_auth.go
- [x] T010 [P] Create health check handler in backend/analytics-service/api/health_handler.go
- [x] T011 Create main.go with Echo server setup, route registration, and middleware in backend/analytics-service/
- [x] T012 [P] Create TimeRange model in backend/analytics-service/src/models/time_range.go
- [x] T013 [P] Create time series utility functions (date range generation, granularity parsing) in backend/analytics-service/src/utils/time_series.go
- [x] T014 [P] Create number formatting utility (K/M abbreviations) in backend/analytics-service/src/utils/formatting.go
- [x] T015 [P] Create cache service with TTL strategy in backend/analytics-service/src/services/cache_service.go
- [x] T016 [P] Create Encryptor interface and VaultClient for field-level encryption in backend/analytics-service/src/utils/encryption.go
- [x] T017 [P] Create LogMasker utility for PII masking in logs (phone, email, name) in backend/analytics-service/src/utils/masker.go
- [x] T018 [P] Create analytics TypeScript types in frontend/src/types/analytics.ts
- [x] T019 [P] Create analytics API client service in frontend/src/services/analyticsService.ts
- [x] T020 Add analytics-service routes to api-gateway/main.go for /api/v1/analytics/\*
- [x] T021 Create database indexes: composite index on orders(tenant_id, status, created_at), index on order_items(product_id, quantity), partial index on products(tenant_id) WHERE quantity <= low_stock_threshold

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - View Current Month Sales Performance (Priority: P1) 🎯 MVP

**Goal**: Display total sales, top/bottom products, net profit, and top spending customers for current month

**Independent Test**: Log in as tenant owner, view dashboard, verify all sales metrics display accurately for current month

### Implementation for User Story 1

- [x] T022 [P] [US1] Create SalesMetrics model in backend/analytics-service/src/models/sales_metrics.go
- [x] T023 [P] [US1] Create ProductRanking model in backend/analytics-service/src/models/product_ranking.go
- [x] T024 [P] [US1] Create CustomerRanking model in backend/analytics-service/src/models/customer_ranking.go
- [x] T025 [P] [US1] Implement sales aggregation queries in backend/analytics-service/src/repository/sales_repository.go
- [x] T026 [P] [US1] Implement product ranking queries (top/bottom by quantity and sales) in backend/analytics-service/src/repository/product_repository.go
- [x] T027 [P] [US1] Implement customer ranking queries with phone decryption (top 5 by encrypted phone) in backend/analytics-service/src/repository/customer_repository.go
- [x] T028 [US1] Implement analytics service with caching logic in backend/analytics-service/src/services/analytics_service.go (depends on T025, T026, T027)
- [x] T029 [US1] Implement GET /analytics/overview handler in backend/analytics-service/api/analytics_handler.go
- [x] T030 [US1] Implement GET /analytics/top-products handler in backend/analytics-service/api/analytics_handler.go
- [x] T031 [US1] Implement GET /analytics/top-customers handler with phone masking for display in backend/analytics-service/api/analytics_handler.go
- [x] T032 [P] [US1] Create DashboardLayout component in frontend/src/components/dashboard/DashboardLayout.tsx
- [x] T033 [P] [US1] Create MetricCard component for displaying sales, profit, inventory in frontend/src/components/dashboard/MetricCard.tsx
- [x] T034 [P] [US1] Create ProductRankingTable component in frontend/src/components/dashboard/ProductRankingTable.tsx
- [x] T035 [P] [US1] Create CustomerRankingTable component displaying masked phone numbers in frontend/src/components/dashboard/CustomerRankingTable.tsx
- [x] T036 [US1] Integrate overview, top-products, top-customers API calls in frontend/app/analytics/page.tsx
- [x] T037 [US1] Add error handling and loading states for User Story 1 components in frontend/app/analytics/page.tsx

**Checkpoint**: At this point, User Story 1 should be fully functional - tenant owners can view current month sales, top/bottom products, and top customers

---

## Phase 4: User Story 3 - Manage Urgent Operational Tasks (Priority: P1)

**Goal**: Display delayed orders (>15 min) and low stock alerts with counts and actionable details

**Independent Test**: Create delayed orders and set products to low stock, verify alerts appear in dashboard with correct counts and details

### Implementation for User Story 3

- [x] T038 [P] [US3] Create DelayedOrder model in backend/analytics-service/src/models/delayed_order.go
- [x] T039 [P] [US3] Create RestockAlert model in backend/analytics-service/src/models/restock_alert.go
- [x] T040 [P] [US3] Implement delayed order query with phone/name decryption (orders > 15 minutes) in backend/analytics-service/src/repository/task_repository.go
- [x] T041 [P] [US3] Implement low stock alert query in backend/analytics-service/src/repository/task_repository.go
- [x] T042 [US3] Implement GET /analytics/tasks handler with customer PII masking in backend/analytics-service/api/tasks_handler.go (depends on T040, T041)
- [x] T043 [P] [US3] Create TaskAlerts component with delayed orders list (showing masked customer data) in frontend/src/components/dashboard/TaskAlerts.tsx
- [x] T044 [P] [US3] Add restock alert list to TaskAlerts component in frontend/src/components/dashboard/TaskAlerts.tsx
- [x] T045 [US3] Integrate tasks API call in frontend/app/analytics/page.tsx
- [x] T046 [US3] Add clickable navigation from task alerts to order/product detail pages in TaskAlerts component

**Checkpoint**: At this point, User Stories 1 AND 3 should both work independently - dashboard shows sales + urgent tasks

---

## Phase 5: User Story 5 - Analyze Trends with Data Visualizations (Priority: P1)

**Goal**: Display interactive charts for sales revenue, order count, product performance with time-series filtering

**Independent Test**: Select different time series (daily/monthly), adjust date ranges, verify charts update with accurate historical data

### Implementation for User Story 5

- [x] T047 [P] [US5] Create TimeSeriesData model in backend/analytics-service/src/models/time_series.go
- [x] T048 [P] [US5] Implement sales trend query with generate_series for complete date ranges in backend/analytics-service/src/repository/sales_repository.go
- [x] T049 [US5] Implement GET /analytics/sales-trend handler in backend/analytics-service/api/analytics_handler.go (depends on T048)
- [x] T050 [P] [US5] Create TimeSeriesFilter component (granularity selector + date range picker) in frontend/src/components/dashboard/TimeSeriesFilter.tsx
- [x] T051 [P] [US5] Create reusable LineChart component with Recharts in frontend/src/components/charts/LineChart.tsx
- [x] T052 [P] [US5] Create reusable BarChart component with Recharts in frontend/src/components/charts/BarChart.tsx
- [x] T053 [P] [US5] Create SalesChart wrapper component with chart type selection in frontend/src/components/dashboard/SalesChart.tsx
- [x] T054 [US5] Integrate sales-trend API with TimeSeriesFilter in frontend/app/analytics/page.tsx
- [x] T055 [US5] Add chart tooltips with exact values and dates using Recharts Tooltip component
- [x] T056 [US5] Handle empty states when no data available for selected period in SalesChart component
- [x] T057 [US5] Implement default view (last 30 days daily) on dashboard load in frontend/app/analytics/page.tsx

**Checkpoint**: All P1 user stories complete - dashboard has sales metrics, tasks, AND data visualization with filtering

---

## Phase 6: User Story 2 - Monitor Inventory Health and Value (Priority: P2)

**Goal**: Display total inventory value (sum of product cost × quantity) on dashboard

**Independent Test**: View dashboard inventory section, verify total value calculation matches sum of all products' cost × quantity

### Implementation for User Story 2

- [x] T058 [US2] Add inventory value calculation to sales repository query in backend/analytics-service/src/repository/sales_repository.go
- [x] T059 [US2] Update GET /analytics/overview handler to include inventory_value in response (already in schema)
- [x] T060 [US2] Display inventory value in MetricCard component in frontend/src/components/dashboard/MetricCard.tsx (metric card already created in T033)
- [x] T061 [US2] Add inventory value formatting and display logic in frontend/app/analytics/page.tsx

**Checkpoint**: Inventory value metric added - dashboard now shows total stock value

**Checkpoint**: User Stories 1, 2, 3, and 5 all complete and independently testable

---

## Phase 7: User Story 4 - Quick Access to Common Actions (Priority: P3)

**Goal**: Provide one-click navigation to team invitation and settings from dashboard

**Independent Test**: Click quick action buttons, verify navigation to correct pages (team invitation and settings)

### Implementation for User Story 4

- [x] T062 [P] [US4] Create QuickActions component with Invite and Settings buttons in frontend/src/components/dashboard/QuickActions.tsx
- [x] T063 [US4] Add navigation links to team invitation page in QuickActions component
- [x] T064 [US4] Add navigation links to tenant settings page in QuickActions component
- [x] T065 [US4] Integrate QuickActions component in dashboard layout in frontend/app/analytics/page.tsx

**Checkpoint**: All user stories complete - full dashboard functionality delivered

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T066 [P] Add query performance logging (query_time_ms) across all repository queries
- [x] T067 [P] Add Prometheus metrics for dashboard access and query latency in analytics-service
- [x] T068 [P] Add structured logging with zerolog and PII masking for all analytics endpoints
- [x] T069 [P] Verify all logs mask sensitive customer data (phone, email, name) using LogMasker utility
- [x] T070 [P] Document analytics-service API in docs/API.md
- [x] T071 [P] Update QUICK_START.md with analytics-service setup and Vault encryption configuration
- [x] T072 Verify dashboard loads in <2 seconds per performance target (SC-001)
- [x] T073 Verify charts render with 365 data points without degradation per SC-013
- [x] T074 [P] Add responsive design for dashboard components (mobile/tablet support)
- [x] T075 [P] Add loading skeletons for all dashboard sections
- [x] T076 [P] Add error boundaries for graceful error handling in dashboard
- [ ] T077 Run quickstart.md validation with test data

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational phase completion
- **User Story 3 (Phase 4)**: Depends on Foundational phase completion
- **User Story 5 (Phase 5)**: Depends on Foundational phase completion
- **User Story 2 (Phase 6)**: Depends on Foundational phase completion (can run in parallel with US1/US3/US5)
- **User Story 4 (Phase 7)**: Depends on Foundational phase completion (can run in parallel with all other stories)
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 3 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 5 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Technically depends on sales repository from US1 but can share implementation
- **User Story 4 (P3)**: Can start after Foundational (Phase 2) - No dependencies (frontend-only navigation)

### Within Each User Story

- Models before services
- Repositories before services
- Services before handlers
- Handlers before frontend components
- Components before page integration
- Story complete before moving to next priority

### Parallel Opportunities

#### Setup Phase (Phase 1)

All tasks except T001-T002 can run in parallel after directory structure is created:

- T003, T004, T005, T006 can all run in parallel

#### Foundational Phase (Phase 2)

After T007 (database) and T011 (main.go), these can run in parallel:

- T008 (Redis), T009 (auth middleware), T010 (health)
- T012-T014 (models and utils) all in parallel
- T015-T017 (services and types) in parallel
- T018-T019 (gateway routes and indexes) in parallel

#### User Story 1 (Phase 3)

- T020-T022 (all models) can run in parallel
- T023-T025 (all repositories) can run in parallel after models complete
- T027-T029 (all handlers) can run in parallel after service (T026) completes
- T030-T033 (all frontend components) can run in parallel
- T034-T035 (integration) runs after all components complete

#### User Story 3 (Phase 4)

- T036-T037 (models) can run in parallel
- T038-T039 (repositories) can run in parallel
- T041-T042 (frontend components) can run in parallel
- T043-T044 (integration) runs last

#### User Story 5 (Phase 5)

- T045-T046 (model and repository) can run sequentially but independently
- T048-T051 (all frontend chart components) can run in parallel
- T052-T055 (integration) runs after components complete

#### User Story 2 (Phase 6)

- Sequential within story (T056 → T057 → T058 → T059)

#### User Story 4 (Phase 7)

- T060-T062 can be done in single component implementation
- T063 runs last

#### Polish Phase (Phase 8)

- T064-T068 (logging, metrics, docs) can all run in parallel
- T069-T070 (performance verification) can run in parallel
- T071-T073 (UI polish) can run in parallel
- T074 (validation) runs last

### Cross-Story Parallelization

Once Phase 2 (Foundational) is complete, these user stories can be worked on in parallel by different team members:

- **Team Member 1**: User Story 1 (Sales Performance) - T020 through T035
- **Team Member 2**: User Story 3 (Operational Tasks) - T036 through T044
- **Team Member 3**: User Story 5 (Data Visualization) - T045 through T055
- **Team Member 4**: User Story 2 (Inventory) + User Story 4 (Quick Actions) - T056 through T063

---

## Parallel Example: User Story 1

```bash
# After Phase 2 completes, these can run simultaneously:
git checkout -b feature/us1-models
# Terminal 1
# T020: Create SalesMetrics model

git checkout -b feature/us1-product-ranking
# Terminal 2 (parallel)
# T021: Create ProductRanking model

git checkout -b feature/us1-customer-ranking
# Terminal 3 (parallel)
# T022: Create CustomerRanking model

# After models are merged, these can run simultaneously:
git checkout -b feature/us1-sales-repo
# Terminal 1
# T023: Implement sales repository

git checkout -b feature/us1-product-repo
# Terminal 2 (parallel)
# T024: Implement product repository

git checkout -b feature/us1-customer-repo
# Terminal 3 (parallel)
# T025: Implement customer repository

# Continue pattern for handlers and frontend components...
```

---

## Implementation Strategy

### MVP Scope (Deliver First)

**Recommended MVP**: User Story 1 ONLY (Phase 1 + Phase 2 + Phase 3)

- Delivers core value: sales insights with top/bottom products and top customers
- 35 tasks (T001-T035)
- Provides immediate business value for decision-making
- Demonstrates full stack implementation (backend service + frontend dashboard)
- Validates architecture and patterns for remaining stories

### Incremental Delivery Path

1. **Sprint 1**: Setup + Foundational (T001-T019) - Establishes infrastructure
2. **Sprint 2**: User Story 1 (T020-T035) - **MVP Release** - Core sales metrics
3. **Sprint 3**: User Story 3 (T036-T044) - Adds operational urgency (delayed orders, low stock)
4. **Sprint 4**: User Story 5 (T045-T055) - Adds data visualization and trend analysis
5. **Sprint 5**: User Story 2 + 4 (T056-T063) - Completes remaining P2/P3 features
6. **Sprint 6**: Polish (T064-T074) - Production readiness

### Quality Gates

- Phase 2 complete: All foundational infrastructure functional before user story work begins
- Each user story complete: Independently testable with acceptance criteria validated
- Polish complete: Performance targets met (dashboard <2s, charts <3s, p95 <200ms)

---

## Task Summary

- **Total Tasks**: 77
- **Setup Phase**: 6 tasks (T001-T006)
- **Foundational Phase**: 15 tasks (T007-T021) - BLOCKING (includes encryption utilities)
- **User Story 1 (P1)**: 16 tasks (T022-T037) - Sales Performance
- **User Story 3 (P1)**: 9 tasks (T038-T046) - Operational Tasks
- **User Story 5 (P1)**: 11 tasks (T047-T057) - Data Visualization
- **User Story 2 (P2)**: 4 tasks (T058-T061) - Inventory
- **User Story 4 (P3)**: 4 tasks (T062-T065) - Quick Actions
- **Polish Phase**: 12 tasks (T066-T077) - Cross-cutting concerns

### MVP Statistics

- **MVP Task Count**: 37 tasks (Phase 1 + 2 + 3)
- **MVP Delivery**: Core sales insights dashboard with encrypted customer data
- **Parallel Opportunities**: 30 tasks marked [P] can run in parallel with others
- **Independent Stories**: All 5 user stories can be tested independently once foundational phase completes
- **Security**: Field-level encryption for phone numbers, email, names using Vault Transit Engine
- **Privacy**: Log masking for all PII per UU PDP compliance

### Performance Targets (from spec.md)

- Dashboard load: <2 seconds (SC-001)
- Chart rendering: <3 seconds (SC-011)
- Query latency p95: <200ms
- Support: 365 daily data points (SC-013)
- Tenant isolation: 100% (SC-009)
