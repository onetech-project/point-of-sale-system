# Tasks: Offline Order Management

**Branch**: `008-offline-orders` | **Date**: February 7, 2026  
**Input**: Design documents from `/specs/008-offline-orders/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

**Organization**: Tasks grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4, US5)
- Include exact file paths in descriptions

## Path Conventions

This is a web application with:

- **Backend**: `backend/order-service/` (Go service with Echo framework)
- **Frontend**: `frontend/` (Next.js 13+ with app directory)
  - Pages: `frontend/app/` (app router)
  - Components: `frontend/src/components/`
  - Services: `frontend/src/services/`
- **Migrations**: `backend/migrations/` (starting from 000060)
- **Tests**: `backend/order-service/tests/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and verification

- [ ] T001 Verify Go 1.24.0 and PostgreSQL 14 are running via docker-compose
- [ ] T002 [P] Verify order-service compiles and existing tests pass
- [ ] T003 [P] Review existing order-service structure (models/, repository/, services/, api/)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Database Migrations

- [ ] T004 Create migration 000060_add_offline_orders.up.sql in backend/migrations/
- [ ] T005 Create migration 000060_add_offline_orders.down.sql in backend/migrations/
- [ ] T006 [P] Create migration 000061_add_payment_terms.up.sql in backend/migrations/
- [ ] T007 [P] Create migration 000061_add_payment_terms.down.sql in backend/migrations/
- [ ] T008 [P] Create migration 000062_add_payment_records.up.sql in backend/migrations/
- [ ] T009 [P] Create migration 000062_add_payment_records.down.sql in backend/migrations/
- [ ] T010 [P] Create migration 000063_add_event_outbox.up.sql in backend/migrations/
- [ ] T011 [P] Create migration 000063_add_event_outbox.down.sql in backend/migrations/
- [ ] T012 Run migrations and verify schema in PostgreSQL

### Base Models & Infrastructure

- [ ] T013 [P] Extend GuestOrder model with offline fields in backend/order-service/src/models/order.go
- [ ] T014 [P] Create PaymentTerms model in backend/order-service/src/models/payment_terms.go
- [ ] T015 [P] Create PaymentRecord model in backend/order-service/src/models/payment_record.go
- [ ] T016 Create EventOutbox model in backend/order-service/src/models/event_outbox.go

### Event Publishing Infrastructure

- [ ] T017 Create OutboxRepository in backend/order-service/src/repository/outbox_repository.go
- [ ] T018 Create EventPublisher service in backend/order-service/src/services/event_publisher.go
- [ ] T019 Implement background worker for event outbox processing in backend/order-service/src/workers/outbox_worker.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Record Basic Offline Order (Priority: P1) 🎯 MVP

**Goal**: Enable staff to manually record walk-in customer purchases with full payment

**Independent Test**: User with any role can create offline order with customer info and items, order appears in system with "offline" designation, all data audit-trailed

### Tests for User Story 1 (Write FIRST, ensure FAIL before implementation)

- [ ] T020 [P] [US1] Contract test for POST /offline-orders in backend/order-service/tests/contract/offline_orders_test.go
- [ ] T021 [P] [US1] Contract test for GET /offline-orders in backend/order-service/tests/contract/offline_orders_test.go
- [ ] T022 [P] [US1] Contract test for GET /offline-orders/{id} in backend/order-service/tests/contract/offline_orders_test.go
- [ ] T023 [P] [US1] Integration test for create offline order journey in backend/order-service/tests/integration/offline_order_creation_test.go
- [ ] T024 [P] [US1] Integration test for list offline orders with filters in backend/order-service/tests/integration/offline_order_list_test.go

### Implementation for User Story 1

**Backend - Repository Layer**

- [ ] T025 [US1] Create OfflineOrderRepository in backend/order-service/src/repository/offline_order_repository.go
- [ ] T026 [US1] Implement CreateOfflineOrder method with encryption in backend/order-service/src/repository/offline_order_repository.go
- [ ] T027 [US1] Implement GetOfflineOrderByID method with decryption in backend/order-service/src/repository/offline_order_repository.go
- [ ] T028 [US1] Implement ListOfflineOrders method with pagination in backend/order-service/src/repository/offline_order_repository.go

**Backend - Service Layer**

- [ ] T029 [US1] Create OfflineOrderService in backend/order-service/src/services/offline_order_service.go
- [ ] T030 [US1] Implement CreateOfflineOrder business logic with validation in backend/order-service/src/services/offline_order_service.go
- [ ] T031 [US1] Implement order reference generator (GO-XXXXXX format) in backend/order-service/src/services/offline_order_service.go
- [ ] T032 [US1] Implement GetOfflineOrderByID with authorization check in backend/order-service/src/services/offline_order_service.go
- [ ] T033 [US1] Implement ListOfflineOrders with tenant filtering in backend/order-service/src/services/offline_order_service.go
- [ ] T034 [US1] Integrate event publishing for offline_order.created in backend/order-service/src/services/offline_order_service.go

**Backend - API Handlers**

- [ ] T035 [US1] Create OfflineOrderHandler in backend/order-service/api/offline_orders_handler.go
- [ ] T036 [US1] Implement POST /offline-orders handler with request validation in backend/order-service/api/offline_orders_handler.go
- [ ] T037 [US1] Implement GET /offline-orders handler with query filters in backend/order-service/api/offline_orders_handler.go
- [ ] T038 [US1] Implement GET /offline-orders/{id} handler in backend/order-service/api/offline_orders_handler.go
- [ ] T039 [US1] Register offline order routes with JWT middleware in backend/order-service/main.go

**Frontend - UI Components**

- [ ] T040 [P] [US1] Create OfflineOrderForm component in frontend/src/components/orders/OfflineOrderForm.tsx
- [ ] T041 [P] [US1] Create OfflineOrderList component in frontend/src/components/orders/OfflineOrderList.tsx
- [ ] T042 [P] [US1] Create OfflineOrderDetail component in frontend/src/components/orders/OfflineOrderDetail.tsx
- [ ] T043 [US1] Create offline orders API client in frontend/src/services/offlineOrders.ts
- [ ] T044 [US1] Create TypeScript types for offline orders in frontend/src/types/offlineOrder.ts
- [ ] T045 [US1] Create offline orders list page in frontend/app/orders/offline-orders/page.tsx
- [ ] T046 [US1] Create new offline order page in frontend/app/orders/offline-orders/new/page.tsx
- [ ] T047 [US1] Create offline order detail page in frontend/app/orders/offline-orders/[id]/page.tsx

**Checkpoint**: At this point, User Story 1 should be fully functional - staff can record offline orders with full payment and view them in the system

---

## Phase 4: User Story 2 - Manage Payment Terms and Installments (Priority: P2)

**Goal**: Enable recording offline orders with down payments and installment schedules

**Independent Test**: Create order with 30% down payment and 2 installments, verify status is "pending", record payments individually, verify status changes to "completed" when fully paid

### Tests for User Story 2 (Write FIRST, ensure FAIL before implementation)

- [ ] T048 [P] [US2] Contract test for POST /offline-orders with installment payment in backend/order-service/tests/contract/payment_terms_test.go
- [ ] T049 [P] [US2] Contract test for POST /offline-orders/{id}/payments in backend/order-service/tests/contract/payment_records_test.go
- [ ] T050 [P] [US2] Contract test for GET /offline-orders/{id}/payments in backend/order-service/tests/contract/payment_records_test.go
- [ ] T051 [P] [US2] Integration test for installment payment lifecycle in backend/order-service/tests/integration/payment_installments_test.go
- [ ] T052 [P] [US2] Unit test for payment balance calculations in backend/order-service/tests/unit/payment_calculations_test.go

### Implementation for User Story 2

**Backend - Repository Layer**

- [ ] T053 [P] [US2] Create PaymentRepository in backend/order-service/src/repository/payment_repository.go
- [ ] T054 [US2] Implement CreatePaymentTerms method in backend/order-service/src/repository/payment_repository.go
- [ ] T055 [US2] Implement RecordPayment method with trigger integration in backend/order-service/src/repository/payment_repository.go
- [ ] T056 [US2] Implement GetPaymentHistory method in backend/order-service/src/repository/payment_repository.go
- [ ] T057 [US2] Implement GetPaymentTerms method in backend/order-service/src/repository/payment_repository.go

**Backend - Service Layer**

- [ ] T058 [US2] Extend CreateOfflineOrder to handle payment terms in backend/order-service/src/services/offline_order_service.go
- [ ] T059 [US2] Implement RecordPayment method with validation in backend/order-service/src/services/offline_order_service.go
- [ ] T060 [US2] Implement payment schedule calculator in backend/order-service/src/services/payment_calculator.go
- [ ] T061 [US2] Implement automatic status update on full payment in backend/order-service/src/services/offline_order_service.go
- [ ] T062 [US2] Integrate event publishing for payment.received in backend/order-service/src/services/offline_order_service.go

**Backend - API Handlers**

- [ ] T063 [US2] Extend POST /offline-orders handler to support installment payment in backend/order-service/api/offline_orders_handler.go
- [ ] T064 [US2] Implement POST /offline-orders/{id}/payments handler in backend/order-service/api/offline_payments_handler.go
- [ ] T065 [US2] Implement GET /offline-orders/{id}/payments handler in backend/order-service/api/offline_payments_handler.go

**Frontend - UI Components**

- [ ] T066 [P] [US2] Create PaymentSchedule component in frontend/src/components/orders/PaymentSchedule.tsx
- [ ] T067 [P] [US2] Create RecordPayment component in frontend/src/components/orders/RecordPayment.tsx
- [ ] T068 [US2] Extend OfflineOrderForm to include payment terms options in frontend/src/components/orders/OfflineOrderForm.tsx
- [ ] T069 [US2] Add payment history display to OfflineOrderDetail in frontend/src/components/orders/OfflineOrderDetail.tsx
- [ ] T070 [US2] Create record payment page in frontend/app/orders/offline-orders/[id]/payments/page.tsx

**Checkpoint**: User Stories 1 AND 2 should both work independently - can create orders with full payment (US1) or installments (US2)

---

## Phase 5: User Story 3 - Edit Offline Orders with Audit Trail (Priority: P3)

**Goal**: Allow staff to correct customer information and modify order items with complete audit trail

**Independent Test**: Create order, edit customer phone and add item, verify changes reflected and logged to audit trail with UPDATE action

### Tests for User Story 3 (Write FIRST, ensure FAIL before implementation)

- [ ] T071 [P] [US3] Contract test for PATCH /offline-orders/{id} in backend/order-service/tests/contract/offline_order_update_test.go
- [ ] T072 [P] [US3] Integration test for order modification journey in backend/order-service/tests/integration/offline_order_edit_test.go
- [ ] T073 [P] [US3] Integration test for audit trail verification in backend/order-service/tests/integration/audit_trail_test.go

### Implementation for User Story 3

**Backend - Repository Layer**

- [ ] T074 [US3] Implement UpdateOfflineOrder method with field tracking in backend/order-service/src/repository/offline_order_repository.go
- [ ] T075 [US3] Implement UpdateOrderItems method in backend/order-service/src/repository/offline_order_repository.go

**Backend - Service Layer**

- [ ] T076 [US3] Implement UpdateOfflineOrder with validation in backend/order-service/src/services/offline_order_service.go
- [ ] T077 [US3] Implement change detection and diff generation in backend/order-service/src/services/offline_order_service.go
- [ ] T078 [US3] Implement status constraint check (no edit if PAID) in backend/order-service/src/services/offline_order_service.go
- [ ] T079 [US3] Integrate event publishing for offline_order.updated with change details in backend/order-service/src/services/offline_order_service.go

**Backend - API Handlers**

- [ ] T080 [US3] Implement PATCH /offline-orders/{id} handler in backend/order-service/api/offline_orders_handler.go
- [ ] T081 [US3] Add request validation for update operations in backend/order-service/api/offline_orders_handler.go

**Frontend - UI Components**

- [ ] T082 [US3] Create edit offline order page in frontend/app/orders/offline-orders/[id]/edit/page.tsx
- [ ] T083 [US3] Add edit button and navigation to OfflineOrderDetail in frontend/src/components/orders/OfflineOrderDetail.tsx
- [ ] T084 [US3] Create AuditTrail component to display change history in frontend/src/components/orders/AuditTrail.tsx
- [ ] T085 [US3] Add audit trail display to OfflineOrderDetail page in frontend/src/components/orders/OfflineOrderDetail.tsx

**Checkpoint**: User Stories 1, 2, AND 3 work independently - can create, manage payments, and edit orders with full audit trail

---

## Phase 6: User Story 4 - Role-Based Order Deletion (Priority: P3)

**Goal**: Enable owner/manager to delete erroneous or fraudulent orders with audit trail

**Independent Test**: Verify staff/cashier cannot delete orders (button hidden/blocked), owner/manager can delete with reason logged to audit trail

### Tests for User Story 4 (Write FIRST, ensure FAIL before implementation)

- [ ] T086 [P] [US4] Integration test for RBAC on deletion (staff role denied) in backend/order-service/tests/integration/rbac_deletion_test.go
- [ ] T087 [P] [US4] Integration test for RBAC on deletion (owner role allowed) in backend/order-service/tests/integration/rbac_deletion_test.go
- [ ] T088 [P] [US4] Contract test for DELETE /offline-orders/{id} in backend/order-service/tests/contract/offline_order_delete_test.go

### Implementation for User Story 4

**Backend - Middleware**

- [ ] T089 [US4] Create RequireRole middleware in backend/order-service/src/middleware/role_check.go
- [ ] T090 [US4] Add role check unit tests in backend/order-service/tests/unit/role_check_test.go

**Backend - Repository Layer**

- [ ] T091 [US4] Implement SoftDeleteOfflineOrder method in backend/order-service/src/repository/offline_order_repository.go

**Backend - Service Layer**

- [ ] T092 [US4] Implement DeleteOfflineOrder with role validation in backend/order-service/src/services/offline_order_service.go
- [ ] T093 [US4] Integrate event publishing for offline_order.deleted with reason in backend/order-service/src/services/offline_order_service.go

**Backend - API Handlers**

- [ ] T094 [US4] Implement DELETE /offline-orders/{id} handler with reason parameter in backend/order-service/api/offline_orders_admin.go
- [ ] T095 [US4] Apply RequireRole middleware to delete route in backend/order-service/main.go

**Frontend - UI Components**

- [ ] T096 [US4] Add conditional delete button to OfflineOrderDetail (owner/manager only) in frontend/src/components/orders/OfflineOrderDetail.tsx
- [ ] T097 [US4] Create DeleteOrderModal with reason input in frontend/src/components/orders/DeleteOrderModal.tsx
- [ ] T098 [US4] Implement delete API call with reason in frontend/src/services/offlineOrders.ts

**Checkpoint**: All user stories through US4 work independently - RBAC enforced for deletion operations

---

## Phase 7: User Story 5 - View Offline Orders in Analytics (Priority: P3)

**Goal**: Include offline orders in analytics dashboard for business intelligence

**Independent Test**: Create 5 online and 3 offline orders, verify dashboard shows accurate totals for both types with proper breakdowns

### Tests for User Story 5 (Write FIRST, ensure FAIL before implementation)

- [ ] T099 [P] [US5] Integration test for analytics event publishing in backend/order-service/tests/integration/analytics_events_test.go
- [ ] T100 [P] [US5] Integration test for order type filtering in analytics in backend/analytics-service/tests/integration/offline_orders_analytics_test.go

### Implementation for User Story 5

**Backend - Analytics Service Extensions**

- [ ] T101 [US5] Extend OrderCreatedEvent consumer to handle offline orders in backend/analytics-service/src/consumers/order_consumer.go
- [ ] T102 [US5] Add order_type dimension to analytics queries in backend/analytics-service/src/repository/analytics_repository.go
- [ ] T103 [US5] Create offline order metrics endpoint GET /analytics/offline-orders in backend/analytics-service/api/analytics_handler.go

**Frontend - Dashboard Extensions**

- [ ] T104 [P] [US5] Add offline order metrics to dashboard in frontend/app/dashboard/page.tsx
- [ ] T105 [P] [US5] Create OfflineOrdersChart component in frontend/src/components/analytics/OfflineOrdersChart.tsx
- [ ] T106 [US5] Add order type filter to analytics reports in frontend/src/components/analytics/ReportFilters.tsx
- [ ] T107 [US5] Create analytics API client for offline metrics in frontend/src/services/analytics.ts

**Checkpoint**: All 5 user stories complete - offline orders fully integrated into analytics dashboard

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

### Performance & Security

- [ ] T108 [P] Add database query performance indexes verification in backend/migrations/
- [ ] T109 [P] Implement encryption key caching in backend/order-service/src/services/encryption_cache.go
- [ ] T110 [P] Add rate limiting to offline order endpoints in backend/order-service/src/middleware/rate_limiter.go
- [ ] T111 Load test offline order creation (100 concurrent users) using k6 scripts in tests/performance/

### Observability

- [ ] T112 [P] Add business metrics for offline orders to Prometheus in backend/order-service/observability/metrics.go
- [ ] T113 [P] Add OpenTelemetry tracing to all offline order operations in backend/order-service/src/services/offline_order_service.go
- [ ] T114 [P] Create Grafana dashboard for offline order metrics in observability/grafana/offline-orders-dashboard.json

### Documentation

- [ ] T115 [P] Update API documentation with offline order endpoints in docs/API.md
- [ ] T116 [P] Add offline order user guide in docs/USER_GUIDE.md
- [ ] T117 [P] Update deployment checklist in docs/DEPLOYMENT_CHECKLIST.md

### Validation & Testing

- [ ] T118 Run all unit tests and verify ≥80% coverage across offline order modules
- [ ] T119 Run all integration tests and verify end-to-end journeys pass
- [ ] T120 Run quickstart.md validation checklist and verify all items pass
- [ ] T121 Verify online order performance unchanged (no degradation >5%)
- [ ] T122 Run PII encryption compliance audit and verify zero violations

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phases 3-7)**: All depend on Foundational phase completion
  - User stories can proceed in parallel (if staffed)
  - Or sequentially in priority order (US1 → US2 → US3 → US4 → US5)
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Extends US1 but independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Works with US1/US2 orders but independently testable
- **User Story 4 (P3)**: Can start after Foundational (Phase 2) - Works with any order from US1/US2/US3 but independently testable
- **User Story 5 (P3)**: Can start after Foundational (Phase 2) - Consumes events from US1/US2/US3/US4 but independently testable

### Within Each User Story

- Tests MUST be written FIRST and FAIL before implementation
- Repository layer before service layer
- Service layer before API handlers
- Backend before frontend (or parallel if API contract agreed)
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

**Setup Phase:**

- T002 and T003 can run in parallel

**Foundational Phase:**

- After migrations created (T004-T011), T012 runs alone
- T013, T014, T015 (models) can run in parallel
- T017 can run after T016

**User Story 1 Tests:**

- T020-T024 can all run in parallel (different test files)

**User Story 1 Implementation:**

- T025-T028 (repository methods) can run in parallel after OfflineOrderRepository exists
- T040, T041, T042 (frontend components) can run in parallel
- T045-T047 (frontend pages) can run in parallel after T043-T044

**User Story 2 Tests:**

- T048-T052 can run in parallel

**User Story 2 Implementation:**

- T053-T057 (PaymentRepository methods) can run after PaymentRepository created
- T066, T067 (frontend components) can run in parallel

**User Story 3 Tests:**

- T071-T073 can run in parallel

**User Story 4 Tests:**

- T086-T088 can run in parallel

**User Story 5 Tests:**

- T099-T100 can run in parallel

**User Story 5 Implementation:**

- T104, T105 (frontend components) can run in parallel

**Polish Phase:**

- T108-T110 can run in parallel
- T112-T114 can run in parallel
- T115-T117 can run in parallel

---

## Parallel Example: User Story 1

```bash
# Phase: Write all tests for User Story 1 together (TDD - these MUST fail initially):
Task T020: "Contract test for POST /offline-orders"
Task T021: "Contract test for GET /offline-orders"
Task T022: "Contract test for GET /offline-orders/{id}"
Task T023: "Integration test for create offline order journey"
Task T024: "Integration test for list offline orders with filters"

# After repository layer exists, launch all repository methods together:
Task T026: "Implement CreateOfflineOrder method with encryption"
Task T027: "Implement GetOfflineOrderByID method with decryption"
Task T028: "Implement ListOfflineOrders method with pagination"

# Launch all frontend components together:
Task T040: "Create OfflineOrderForm component"
Task T041: "Create OfflineOrderList component"
Task T042: "Create OfflineOrderDetail component"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only) - Recommended

1. Complete Phase 1: Setup (T001-T003) - ~30 minutes
2. Complete Phase 2: Foundational (T004-T019) - ~4-6 hours
3. Complete Phase 3: User Story 1 (T020-T047) - ~12-15 hours
4. **STOP and VALIDATE**: Test User Story 1 independently using acceptance scenarios from spec.md
5. Deploy/demo if ready

**Total MVP Effort**: ~18-22 hours

### Incremental Delivery (Recommended for production)

1. Setup + Foundational (Phases 1-2) → Foundation ready (~5-7 hours)
2. User Story 1 (Phase 3) → Test independently → Deploy/Demo (~12-15 hours) 🎯 **MVP!**
3. User Story 2 (Phase 4) → Test independently → Deploy/Demo (~8-10 hours)
4. User Story 3 (Phase 5) → Test independently → Deploy/Demo (~6-8 hours)
5. User Story 4 (Phase 6) → Test independently → Deploy/Demo (~4-5 hours)
6. User Story 5 (Phase 7) → Test independently → Deploy/Demo (~4-5 hours)
7. Polish (Phase 8) → Final validation → Production release (~4-6 hours)

**Total Delivery**: ~43-56 hours across multiple increments

### Parallel Team Strategy

With 3 developers after Foundational phase complete:

1. **Team completes Setup + Foundational together** (Phases 1-2) - ~5-7 hours
2. **Once Foundational done, parallel streams:**
   - **Developer A**: User Story 1 (T020-T047) - ~12-15 hours
   - **Developer B**: User Story 2 (T048-T070) - ~8-10 hours
   - **Developer C**: User Story 4 (T086-T098) - ~4-5 hours
3. **Sequential completion:**
   - Developer C picks up User Story 3 after US4
   - Developer B picks up User Story 5 after US2
   - All converge on Polish phase

**Team Delivery**: ~20-25 hours elapsed time with 3 developers

---

## Task Summary

**Total Tasks**: 122

- **Setup**: 3 tasks
- **Foundational**: 16 tasks (BLOCKING)
- **User Story 1 (P1)**: 28 tasks (MVP)
- **User Story 2 (P2)**: 23 tasks
- **User Story 3 (P3)**: 15 tasks
- **User Story 4 (P3)**: 13 tasks
- **User Story 5 (P3)**: 9 tasks
- **Polish**: 15 tasks

**Independent Test Criteria**:

- US1: Staff can record order with customer data and items, appears with "offline" label
- US2: Can create order with installments, record payments, status updates on full payment
- US3: Can edit order, changes logged to audit trail with field-level detail
- US4: Only owner/manager can delete, staff/cashier blocked, deletion logged with reason
- US5: Offline orders appear in analytics with accurate metrics and filters

**MVP Scope (Recommended)**: Phases 1-3 only (User Story 1) = ~18-22 hours

**Parallel Opportunities**: 45 tasks marked [P] can run in parallel within their phase

---

## Notes

- All tasks follow TDD workflow: write failing test → implement → verify green → refactor
- [P] tasks marked for different files with no dependencies - can run concurrently
- [Story] labels (US1-US5) map tasks to user stories from spec.md for traceability
- Each user story independently completable and testable per acceptance scenarios
- Stop at any checkpoint to validate story independently before proceeding
- Migrations (T004-T012: 000060-000063) must complete before any code implementation begins
- Frontend uses Next.js 13+ app directory structure with server/client components
- Backend API handlers go directly to `backend/order-service/api/` (no v1 subdirectory)
- Frontend tasks can start in parallel with backend if API contracts agreed
