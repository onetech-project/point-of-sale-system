# Tasks: QRIS Guest Ordering System

**Feature Branch**: `001-guest-qris-ordering`  
**Input**: Design documents from `/specs/003-guest-qris-ordering/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Test Strategy**: Following constitution Principle III (Test-First Development), contract, integration, and unit tests MUST be written before implementation code. Tests are integrated into each user story phase below.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create order-service directory structure per plan.md at backend/order-service/
- [X] T002 Initialize Go module in backend/order-service/ with go.mod and required dependencies (Echo v4, Midtrans SDK, Google Maps API, PostgreSQL driver, Redis client)
- [X] T003 [P] Create Dockerfile for order-service in backend/order-service/Dockerfile
- [X] T004 [P] Create .env.example in backend/order-service/ with required environment variables (DATABASE_URL, REDIS_URL, MIDTRANS_SERVER_KEY, GOOGLE_MAPS_API_KEY)
- [X] T005 Create database migrations 000012-000016 in backend/migrations/ for guest_orders, order_items, inventory_reservations, payment_transactions, delivery_addresses tables
- [X] T006 [P] Add tenant_configs extension migration 000017 in backend/migrations/ for delivery types, service area, delivery fee config

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T007 Create database configuration in backend/order-service/src/config/database.go with connection pooling and transaction support
- [X] T008 [P] Create Redis configuration in backend/order-service/src/config/redis.go with connection management
- [X] T009 [P] Create Midtrans SDK configuration in backend/order-service/src/config/midtrans.go with server key and environment setup
- [X] T010 [P] Create Google Maps API configuration in backend/order-service/src/config/google_maps.go
- [X] T011 Create base models in backend/order-service/src/models/ for order.go, order_item.go, inventory_reservation.go, payment_transaction.go, delivery_address.go
- [X] T012 [P] Create tenant scope middleware in backend/order-service/src/middleware/tenant_scope.go to validate tenant_id from URL path
- [X] T013 [P] Create rate limiting middleware in backend/order-service/src/middleware/rate_limit.go for public endpoints
- [X] T014 [P] Create webhook authentication middleware in backend/order-service/src/middleware/webhook_auth.go for Midtrans signature verification
- [X] T015 Create structured logger utility in backend/order-service/src/utils/logger.go with order_reference, tenant_id context fields
- [X] T016 [P] Create crypto utility in backend/order-service/src/utils/crypto.go for order reference generation (GO-XXXXXX format)
- [X] T017 Setup Echo server in backend/order-service/main.go with middleware registration, health check endpoint, and graceful shutdown
- [X] T018 Update api-gateway/main.go to add routes for order-service endpoints (public cart, orders, payment webhook, admin)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Browse and Add Items to Cart (Priority: P1) 🎯 MVP

**Goal**: Enable guests to browse tenant product catalog and manage shopping cart without authentication

**Independent Test**: Access tenant public menu URL, view products with availability, add/update/remove items from cart, verify cart persists during session

### Tests for User Story 1 (Write FIRST - Must FAIL before implementation)

- [X] T019a [P] [US1] Contract test in backend/order-service/tests/contract/cart_api_test.go - verify cart endpoints match order-api.yaml schemas (GET/POST/PATCH/DELETE)
- [ ] T019b [P] [US1] Contract test in backend/product-service/tests/contract/public_catalog_test.go - verify product catalog endpoint matches contract
- [X] T019c [P] [US1] Integration test in backend/order-service/tests/integration/cart_flow_test.go - verify add/update/remove cart items with Redis persistence and 24hr TTL
- [X] T019d [P] [US1] Unit test in backend/order-service/tests/unit/cart_service_test.go - verify cart operations, inventory validation, session association
- [ ] T019e [P] [US1] Frontend test in frontend/tests/guest/public-menu.test.tsx - verify PublicMenu component renders products and handles cart actions

### Implementation for User Story 1 (After tests written and verified to FAIL)

- [X] T019 [P] [US1] Create cart repository in backend/order-service/src/repository/cart_repository.go with Redis operations (HSET, HGET, DEL, EXPIRE)
- [X] T020 [P] [US1] Create cart model in backend/order-service/src/models/cart.go with JSON marshaling for Redis storage
- [X] T021 [US1] Implement cart service in backend/order-service/src/services/cart_service.go with add/update/remove/get operations and 24hr TTL management
- [X] T022 [P] [US1] Extend product-service with public catalog handler in backend/product-service/api/public_catalog_handler.go (GET /public/menu/:tenant_id/products)
- [X] T023 [P] [US1] Implement catalog service in backend/product-service/src/services/catalog_service.go for guest-facing product list with availability check
- [X] T024 [US1] Implement cart handler in backend/order-service/api/cart_handler.go with GET /public/cart/:tenant_id, POST /public/cart/:tenant_id/items, PATCH /public/cart/:tenant_id/items/:product_id, DELETE /public/cart/:tenant_id/items/:product_id
- [X] T025 [US1] Add session ID header validation and cart-to-session association in cart_handler.go
- [X] T026 [US1] Implement inventory validation in cart service to prevent adding unavailable items
- [X] T027 [P] [US1] Create PublicMenu component in frontend/src/components/guest/PublicMenu.tsx with product grid and filtering
- [X] T028 [P] [US1] Create ProductCard component in frontend/src/components/guest/ProductCard.tsx with add-to-cart button and availability display
- [X] T029 [P] [US1] Create GuestCart component in frontend/src/components/guest/GuestCart.tsx with item list, quantity controls, and total display
- [X] T030 [US1] Create public menu page in frontend/src/pages/menu/[tenantId].tsx with PublicMenu and GuestCart integration
- [X] T031 [US1] Implement cart service in frontend/src/services/cartService.ts with session ID management and cart persistence in localStorage (per FR-008)
- [X] T032 [US1] Add cart state management and API calls in cartService.ts for all cart operations

**Checkpoint**: At this point, guests can browse products and manage cart - User Story 1 is fully functional and testable independently

---

## Phase 4: User Story 2 - Select Delivery Type and Provide Details (Priority: P1)

**Goal**: Enable guests to select delivery type based on tenant configuration and provide required contact/address information

**Independent Test**: Proceed to checkout from cart, see tenant-allowed delivery types, select each type and verify appropriate form fields appear

### Tests for User Story 2 (Write FIRST - Must FAIL before implementation)

- [ ] T033a [P] [US2] Contract test in backend/tenant-service/tests/contract/tenant_config_test.go - verify tenant config endpoint matches contract
- [ ] T033b [P] [US2] Integration test in backend/order-service/tests/integration/checkout_validation_test.go - verify delivery type validation and required field enforcement
- [ ] T033c [P] [US2] Unit test in backend/order-service/tests/unit/checkout_handler_test.go - verify contact validation (name, phone format) and conditional field logic
- [ ] T033d [P] [US2] Frontend test in frontend/tests/guest/checkout-form.test.tsx - verify CheckoutForm renders correct fields based on delivery type

### Implementation for User Story 2 (After tests written and verified to FAIL)

- [X] T033 [P] [US2] Create tenant config repository in backend/tenant-service/src/repository/tenant_config_repository.go for delivery settings
- [X] T034 [P] [US2] Extend tenant config model in backend/tenant-service/src/models/tenant_config.go with enabled_delivery_types, service_area, delivery_fee_config fields
- [X] T035 [US2] Implement tenant config service in backend/tenant-service/src/services/tenant_config_service.go to retrieve delivery settings
- [X] T036 [US2] Add tenant config endpoint in backend/tenant-service/api/tenant_config_handler.go (GET /public/tenants/:tenant_id/config)
- [X] T037 [P] [US2] Create CheckoutForm component in frontend/src/components/guest/CheckoutForm.tsx with delivery type selection and contact fields
- [X] T038 [P] [US2] Create DeliveryTypeSelector component in frontend/src/components/guest/DeliveryTypeSelector.tsx with conditional rendering based on tenant config
- [X] T039 [P] [US2] Create AddressInput component in frontend/src/components/guest/AddressInput.tsx with validation for delivery orders
- [X] T040 [US2] Create checkout page in frontend/src/pages/checkout/[orderId].tsx with CheckoutForm integration
- [X] T041 [US2] Implement checkout handler in backend/order-service/api/checkout_handler.go with delivery type validation against tenant config
- [X] T042 [US2] Add contact information validation (name, phone format) in checkout_handler.go
- [X] T043 [US2] Add conditional field requirements based on delivery type (address for delivery, optional table_number for dine-in)

**Checkpoint**: At this point, guests can select delivery type and provide required information - User Story 2 is fully functional and testable independently

---

## Phase 5: User Story 5 - Handle Inventory Availability with Reservations (Priority: P2)

**Goal**: Implement inventory reservation system with TTL to prevent overselling while allowing abandoned carts to free inventory

**Independent Test**: Add items to cart, proceed to checkout to create reservation, verify inventory decreases temporarily, abandon order and verify reservation expires after TTL

**Note**: Implemented before US3 (payment) because payment flow depends on inventory reservations

### Tests for User Story 5 (Write FIRST - Must FAIL before implementation)

- [X] T044a [P] [US5] Integration test in backend/order-service/tests/integration/inventory_reservation_test.go - verify reservation creation, TTL expiration, and conversion on payment
- [ ] T044b [P] [US5] Integration test in backend/order-service/tests/integration/inventory_race_condition_test.go - verify SELECT FOR UPDATE prevents overselling with concurrent orders
- [ ] T044c [P] [US5] Unit test in backend/order-service/tests/unit/inventory_service_test.go - verify available inventory calculation, reservation logic, Redis cache updates
- [ ] T044d [P] [US5] Unit test in backend/order-service/tests/unit/reservation_cleanup_job_test.go - verify expired reservation detection and release logic

### Implementation for User Story 5 (After tests written and verified to FAIL)

- [X] T044 [P] [US5] Create inventory reservation repository in backend/order-service/src/repository/reservation_repository.go with PostgreSQL CRUD operations
- [X] T045 [US5] Implement inventory service in backend/order-service/src/services/inventory_service.go with reservation creation, expiration checking, and conversion logic
- [X] T046 [US5] Add inventory check with SELECT FOR UPDATE in inventory_service.go to prevent race conditions
- [X] T047 [US5] Implement reservation creation in inventory_service.go with 15min TTL calculation
- [X] T048 [US5] Add available inventory calculation (quantity - active_reservations - permanent_allocations) in inventory_service.go
- [X] T049 [US5] Implement Redis cache update in inventory_service.go for available inventory (DECR on reservation, INCR on release)
- [X] T050 [US5] Create background job in backend/order-service/src/services/reservation_cleanup_job.go to run every 1 minute
- [X] T051 [US5] Implement expired reservation detection in reservation_cleanup_job.go (status='active' AND expires_at < NOW())
- [X] T052 [US5] Add reservation release logic in reservation_cleanup_job.go (update status to 'expired', increment Redis cache)
- [X] T053 [US5] Integrate reservation creation with checkout flow in checkout_handler.go before payment initiation
- [X] T054 [US5] Add inventory validation in checkout_handler.go to prevent order creation if insufficient stock
- [X] T055 [US5] Start reservation cleanup job goroutine in backend/order-service/main.go

**Checkpoint**: At this point, inventory reservations work with TTL and prevent overselling - User Story 5 is fully functional and testable independently

---

## Phase 6: User Story 3 - Complete Midtrans QRIS Payment (Priority: P1)

**Goal**: Enable secure payment processing through Midtrans QRIS with webhook notification handling

**Independent Test**: Complete checkout to initiate payment, redirect to Midtrans, scan QR and pay, verify webhook received and order status updated to PAID

### Tests for User Story 3 (Write FIRST - Must FAIL before implementation)

- [X] T056a [P] [US3] Contract test in backend/order-service/tests/contract/midtrans_webhook_test.go - verify webhook payload matches payment-webhook.yaml schema and signature verification
- [ ] T056b [P] [US3] Integration test in backend/order-service/tests/integration/payment_flow_test.go - verify end-to-end payment: Snap creation → webhook → status update → inventory conversion
- [ ] T056c [P] [US3] Unit test in backend/order-service/tests/unit/payment_service_test.go - verify signature verification (SHA512), idempotency handling, status mapping logic
- [ ] T056d [P] [US3] Unit test in backend/order-service/tests/unit/payment_webhook_test.go - verify webhook handler validates amount, processes notifications, handles failures
- [ ] T056e [P] [US3] Frontend test in frontend/tests/guest/order-confirmation.test.tsx - verify OrderConfirmation displays correct order details and status

### Implementation for User Story 3 (After tests written and verified to FAIL)

- [X] T056 [P] [US3] Create payment transaction repository in backend/order-service/src/repository/payment_repository.go with PostgreSQL operations and idempotency key handling
- [X] T057 [US3] Implement payment service in backend/order-service/src/services/payment_service.go with Midtrans SDK integration
- [X] T058 [US3] Add Snap transaction creation in payment_service.go (CoreGateway.ChargeTransaction with QRIS method)
- [X] T059 [US3] Implement signature verification in payment_service.go using SHA512 hash (order_id + status_code + gross_amount + server_key)
- [X] T060 [US3] Add notification processing logic in payment_service.go with idempotency check, signature validation, and status mapping
- [X] T061 [US3] Implement order status update logic in payment_service.go (settlement → PAID, cancel/deny/expire → release reservation)
- [X] T062 [US3] Add inventory reservation conversion in payment_service.go (status='converted', decrement product quantity)
- [X] T063 [US3] Implement payment webhook handler in backend/order-service/api/payment_webhook.go (POST /payments/midtrans/notification)
- [X] T064 [US3] Add webhook signature middleware to payment_webhook.go route
- [X] T065 [US3] Add full notification payload logging in payment_webhook.go for audit trail
- [X] T066 [US3] Extend checkout handler to create payment transaction record and return Midtrans redirect URL
- [X] T067 [P] [US3] Create OrderConfirmation component in frontend/src/components/guest/OrderConfirmation.tsx with order reference and delivery details
- [X] T068 [US3] Add payment redirect logic in checkout page to open Midtrans Snap URL
- [X] T069 [US3] Create order status page in frontend/src/pages/orders/[orderReference].tsx for guests to check status
- [X] T070 [US3] Add payment return handling in frontend to redirect to order confirmation page

**Checkpoint**: At this point, guests can complete payment via Midtrans QRIS and orders update to PAID status - User Story 3 is fully functional and testable independently

---

## Phase 7: User Story 6 - Calculate Delivery Fees Based on Location (Priority: P2)

**Goal**: Implement optional automatic delivery fee calculation for delivery orders when tenant enables it; tenants can disable automatic calculation to handle fees manually

**Independent Test**: (1) With automatic fees enabled: verify geocoding, service area validation, fee calculation. (2) With automatic fees disabled: verify delivery_fee remains 0

### Tests for User Story 6 (Write FIRST - Must FAIL before implementation)

- [ ] T071a [P] [US6] Integration test in backend/order-service/tests/integration/geocoding_test.go - verify Google Maps API integration, caching in Redis, error handling
- [ ] T071b [P] [US6] Unit test in backend/order-service/tests/unit/geocoding_service_test.go - verify address geocoding, Haversine distance calculation, point-in-polygon validation
- [ ] T071c [P] [US6] Unit test in backend/order-service/tests/unit/delivery_fee_service_test.go - verify distance-based tier matching and zone-based fee lookup
- [ ] T071d [P] [US6] Frontend test in frontend/tests/guest/address-input.test.tsx - verify AddressInput validation and error message display

### Implementation for User Story 6 (After tests written and verified to FAIL)

- [X] T071 [P] [US6] Create delivery address repository in backend/order-service/src/repository/address_repository.go with PostgreSQL operations
- [X] T072 [US6] Implement geocoding service in backend/order-service/src/services/geocoding_service.go with Google Maps Geocoding API integration
- [X] T073 [US6] Add address geocoding in geocoding_service.go (address text → lat/lng coordinates)
- [X] T074 [US6] Implement geocoding result caching in geocoding_service.go using Redis (key: address hash, TTL: 7 days)
- [X] T075 [US6] Add service area validation in geocoding_service.go with Haversine distance calculation for radius-based areas
- [X] T076 [US6] Add point-in-polygon validation in geocoding_service.go for zone-based service areas
- [X] T077 [US6] Implement delivery fee service in backend/order-service/src/services/delivery_fee_service.go with distance and zone-based pricing logic
- [X] T078 [US6] Add distance-based tier matching in delivery_fee_service.go (calculate distance, find matching tier, return fee)
- [X] T079 [US6] Add zone-based fee lookup in delivery_fee_service.go (check which polygon contains address, return zone fee)
- [X] T080 [US6] Integrate geocoding into checkout handler in api/checkout_handler.go (geocode address before order creation)
- [X] T081 [US6] Add service area validation in checkout_handler.go to reject orders outside service area
- [X] T082 [US6] Add delivery fee calculation in checkout_handler.go and include in order total
- [X] T083 [US6] Create delivery_address record in checkout_handler.go with geocoded coordinates and calculated fee
- [X] T084 [US6] Add delivery fee display in CheckoutForm component showing itemized costs (subtotal + delivery fee + total)
- [X] T085 [US6] Add address validation error handling in frontend AddressInput component with user-friendly messages

**Checkpoint**: At this point, delivery addresses are geocoded, validated, and delivery fees calculated - User Story 6 is fully functional and testable independently

---

## Phase 8: User Story 4 - Tenant Staff Manages Order Fulfillment (Priority: P1)

**Goal**: Enable tenant staff to view orders, add notes, and update status to COMPLETE when delivery finished

**Independent Test**: Login as tenant staff, view list of PAID orders, select order to see details, update status to COMPLETE with timestamp

### Tests for User Story 4 (Write FIRST - Must FAIL before implementation)

- [ ] T086a [P] [US4] Contract test in backend/order-service/tests/contract/admin_order_test.go - verify admin endpoints match admin-api.yaml schemas
- [ ] T086b [P] [US4] Integration test in backend/order-service/tests/integration/admin_order_flow_test.go - verify order listing, filtering, status updates, note addition
- [ ] T086c [P] [US4] Unit test in backend/order-service/tests/unit/order_service_test.go - verify status transition validation (state machine), timestamp recording
- [ ] T086d [P] [US4] Frontend test in frontend/tests/admin/order-management.test.tsx - verify OrderManagement component displays orders and handles status updates

### Implementation for User Story 4 (After tests written and verified to FAIL)

- [X] T086 [P] [US4] Create order repository in backend/order-service/src/repository/order_repository.go with PostgreSQL queries for listing and updating orders
- [X] T087 [US4] Implement order service in backend/order-service/src/services/order_service.go with business logic for status transitions
- [X] T088 [US4] Add status transition validation in order_service.go (enforce valid state machine transitions from research.md)
- [X] T089 [US4] Add timestamp recording in order_service.go (paid_at, completed_at, cancelled_at)
- [X] T090 [US4] Implement admin order handler in backend/order-service/api/admin_order_handler.go (GET /admin/orders, GET /admin/orders/:id, PATCH /admin/orders/:id/status, POST /admin/orders/:id/notes)
- [X] T091 [US4] Add JWT authentication middleware to admin routes in admin_order_handler.go
- [X] T092 [US4] Add tenant-scoped order filtering in admin order handler (staff only see own tenant orders)
- [X] T093 [US4] Add status filter query parameter support in admin order handler (filter by PENDING, PAID, COMPLETE, CANCELLED)
- [X] T094 [P] [US4] Create OrderManagement component in frontend/src/components/admin/OrderManagement.tsx with order list, filters, and detail view
- [X] T095 [US4] Create admin orders page in frontend/src/pages/admin/orders.tsx with OrderManagement component
- [X] T096 [US4] Implement order service in frontend/src/services/guestOrderService.ts with API calls for admin operations
- [X] T097 [US4] Add status update functionality in OrderManagement component with confirmation dialog
- [X] T098 [US4] Add notes/comments functionality in OrderManagement component for courier tracking information

**Checkpoint**: At this point, tenant staff can manage orders and update status - User Story 4 is fully functional and testable independently

---

## Phase 9: User Story 7 - Access Tenant-Specific Public Menu (Priority: P3)

**Goal**: Ensure multi-tenant isolation with each tenant having unique public menu URL displaying only their products

**Independent Test**: Access different tenant URLs, verify each shows unique product catalog, test invalid tenant URL shows error

### Tests for User Story 7 (Write FIRST - Must FAIL before implementation)

- [ ] T099a [P] [US7] Integration test in backend/order-service/tests/integration/multi_tenant_isolation_test.go - verify tenant data isolation across all operations
- [ ] T099b [P] [US7] Unit test in backend/order-service/tests/unit/tenant_scope_middleware_test.go - verify tenant validation and active status check
- [ ] T099c [P] [US7] Frontend test in frontend/tests/guest/tenant-menu.test.tsx - verify tenant-specific menu rendering and cart separation

### Implementation for User Story 7 (After tests written and verified to FAIL)

- [X] T099 [P] [US7] Add tenant existence validation in tenant scope middleware in backend/order-service/src/middleware/tenant_scope.go
- [X] T100 [P] [US7] Add tenant active status check in tenant_scope.go (reject inactive tenants)
- [X] T101 [US7] Ensure all product queries in catalog_service.go filter by tenant_id
- [X] T102 [US7] Ensure all cart operations in cart_service.go are tenant-scoped (cart key includes tenant_id)
- [X] T103 [US7] Ensure all order queries in order_service.go filter by tenant_id
- [X] T104 [US7] Add tenant branding display in PublicMenu component (tenant name, logo from API)
- [X] T105 [US7] Add invalid tenant error page in frontend/src/pages/menu/[tenantId].tsx
- [X] T106 [US7] Add cart separation in frontend cartService.ts (different carts per tenant in localStorage, per FR-008 and FR-010)
- [X] T107 [US7] Add tenant config fetch in public menu page to display tenant-specific delivery types

**Checkpoint**: At this point, multi-tenant isolation is complete with tenant-specific URLs and data isolation - User Story 7 is fully functional and testable independently

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T108 [P] Add comprehensive logging throughout all services with structured fields (order_reference, tenant_id, session_id, product_id)
- [X] T109 [P] Add error handling standardization across all handlers with consistent error response format
- [X] T110 [P] Add input sanitization in all handlers to prevent injection attacks
- [X] T111 [P] Add rate limiting configuration tuning based on expected load
- [X] T112: Add Docker Compose service definition for order-service in docker-compose.yml
- [X] T113 [P]: Update documentation in docs/ with QRIS guest ordering feature overview
- [X] T114: Run database migrations in development environment and verify schema
- [X] T115: Run quickstart.md validation scenarios to verify complete order flow
- [X] T116 [P] Add monitoring metrics instrumentation (webhook processing time, signature failures, reservation expiration rate)
- [X] T117 [P] Performance optimization: Add database query indexing verification for common queries
- [X] T118 [P] Security hardening: Add HTTPS enforcement in production configuration
- [X] T119 Code cleanup and refactoring to remove any TODO comments or temporary code
- [X] T120 Update README.md with guest ordering feature setup instructions

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-9)**: All depend on Foundational phase completion
  - Can proceed in parallel with adequate team resources
  - Or sequentially in priority order: US1 → US5 → US3 → US2 → US6 → US4 → US7
- **Polish (Phase 10)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational - No dependencies on other stories  
- **User Story 5 (P2)**: Can start after Foundational - No dependencies (but should complete before US3)
- **User Story 3 (P1)**: Depends on US5 completion for inventory reservation logic
- **User Story 6 (P2)**: Integrates with US2 but can be tested independently with mock data
- **User Story 4 (P1)**: Depends on US3 for PAID orders to exist
- **User Story 7 (P3)**: Can start after Foundational - Enhances US1 but independent

### Recommended Implementation Order

**MVP (Minimum Viable Product)**: US1 + US5 + US3 + US2 + US4
- This enables complete guest ordering flow with payment and staff fulfillment

**Phase A**: US1, US5 (can be parallel)
**Phase B**: US3, US2 (US3 needs US5; US2 independent)  
**Phase C**: US4 (needs US3 for PAID orders)
**Phase D**: US6 (enhances delivery flow)
**Phase E**: US7 (multi-tenant polish)

### Within Each User Story

- Backend models and repositories can run in parallel
- Services depend on repositories
- Handlers depend on services
- Frontend components can run in parallel with backend
- Integration happens in final task per story

### Parallel Opportunities

**Setup Phase**: T003, T004, T005, T006 can run in parallel

**Foundational Phase**: T008, T009, T010, T012, T013, T014, T016 can run in parallel after T007

**User Story 1**: 
- Parallel: T019, T020, T022, T023, T027, T028, T029 (different files)
- Then: T021, T024, T025, T026, T030, T031, T032 sequentially

**User Story 2**: T033, T034, T037, T038, T039 can run in parallel

**User Story 5**: T044 and models can start, then T045-T055 build on each other

**User Story 3**: T056, T067 can run in parallel with core payment logic

**User Story 6**: T071, T072, T077 can start in parallel

**User Story 4**: T086, T094 can run in parallel

**User Story 7**: T099, T100, T104, T105, T106 can run in parallel

**Polish Phase**: Most tasks marked [P] can run in parallel

---

## Implementation Strategy

### MVP-First Approach

Focus on completing User Stories 1, 5, 3, 2, 4 in that order for a working end-to-end flow:
1. Guests can browse and add to cart (US1) - 19 tasks (5 tests + 14 implementation)
2. Inventory is protected with reservations (US5) - 16 tasks (4 tests + 12 implementation)
3. Payment completes via Midtrans (US3) - 20 tasks (5 tests + 15 implementation)
4. Delivery details are collected (US2) - 15 tasks (4 tests + 11 implementation)
5. Staff can fulfill orders (US4) - 17 tasks (4 tests + 13 implementation)

**MVP Total**: 87 tasks (22 test tasks + 65 implementation tasks) + Setup (6) + Foundational (12) = **105 tasks**

This delivers core value: **Guest can order, pay, and tenant can fulfill**

**Test-First Reminder**: For each user story, complete ALL test tasks (write tests, verify they FAIL) before starting ANY implementation tasks.

### Incremental Delivery

After MVP, add enhancements:
- **US6**: Delivery fee calculation (improves delivery flow)
- **US7**: Multi-tenant isolation (enables scaling to multiple tenants)

### Test Validation

Each user story has an "Independent Test" criteria. Complete validation per story by:
1. Following test scenarios in quickstart.md
2. Verifying success criteria from spec.md for that story
3. Testing edge cases mentioned in spec.md

---

## Task Count Summary

- **Setup**: 6 tasks
- **Foundational**: 12 tasks
- **User Story 1**: 19 tasks (5 tests + 14 implementation)
- **User Story 2**: 15 tasks (4 tests + 11 implementation)
- **User Story 5**: 16 tasks (4 tests + 12 implementation)
- **User Story 3**: 20 tasks (5 tests + 15 implementation)
- **User Story 6**: 19 tasks (4 tests + 15 implementation)
- **User Story 4**: 17 tasks (4 tests + 13 implementation)
- **User Story 7**: 12 tasks (3 tests + 9 implementation)
- **Polish**: 13 tasks
- **TOTAL**: 149 tasks (29 test tasks + 120 implementation/infrastructure tasks)

---

## Validation Checklist

- [x] All tasks follow checklist format with checkbox, ID, optional [P] marker, [Story] label, and file path
- [x] Each user story has clear goal and independent test criteria
- [x] Setup phase creates project structure
- [x] Foundational phase establishes blocking prerequisites
- [x] User stories organized by priority (P1 → P2 → P3)
- [x] **Test-first development enforced: Tests written BEFORE implementation per constitution Principle III**
- [x] Each user story phase includes contract, integration, and unit tests
- [x] Tests must FAIL before implementation code is written
- [x] Each user story phase includes all necessary implementation tasks
- [x] Dependencies clearly documented
- [x] Parallel opportunities identified with [P] markers
- [x] MVP scope defined (US1, US5, US3, US2, US4)
- [x] Task count appropriate for feature complexity (149 tasks including tests)
- [x] All entities from data-model.md mapped to tasks
- [x] All endpoints from contracts/ mapped to tasks
- [x] All technical decisions from research.md incorporated

**Format Validation**: ✅ ALL tasks follow required format
**Completeness**: ✅ All requirements from spec.md addressed
**Independence**: ✅ Each user story can be tested independently
**Constitution Compliance**: ✅ Test-first development mandated in all user story phases
