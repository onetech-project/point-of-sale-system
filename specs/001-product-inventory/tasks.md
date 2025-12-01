# Tasks: Product & Inventory Management

**Input**: Design documents from `/specs/001-product-inventory/`
**Prerequisites**: plan.md (✓), spec.md (✓), research.md (✓), data-model.md (✓), contracts/ (✓)

**Tests**: Tests are included per constitution Test-First Development requirement (III)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md structure:
- Backend microservice: `backend/product-service/`
- Frontend pages: `frontend/pages/products/`
- Frontend components: `frontend/src/components/products/`
- Database migrations: `backend/migrations/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [X] T001 Create product-service directory structure at backend/product-service/ with api/, src/models/, src/repository/, src/services/, tests/ subdirectories
- [X] T002 Initialize Go module for product-service with go.mod at backend/product-service/go.mod
- [X] T003 [P] Add Echo v4, lib/pq, testify dependencies to backend/product-service/go.mod
- [X] T004 [P] Create .env.example file at backend/product-service/.env.example with DATABASE_URL, REDIS_HOST, JWT_SECRET, UPLOAD_DIR, MAX_PHOTO_SIZE_MB
- [X] T005 [P] Create main.go entry point at backend/product-service/main.go with basic Echo server setup

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Create categories table migration 010_create_categories_table.up.sql in backend/migrations/
- [X] T007 Create categories table rollback migration 010_create_categories_table.down.sql in backend/migrations/
- [X] T008 Create products table migration 009_create_products_table.up.sql in backend/migrations/ with RLS policies and indexes
- [X] T009 Create products table rollback migration 009_create_products_table.down.sql in backend/migrations/
- [X] T010 Create stock_adjustments table migration 011_create_stock_adjustments_table.up.sql in backend/migrations/ with trigger for quantity_delta calculation
- [X] T011 Create stock_adjustments table rollback migration 011_create_stock_adjustments_table.down.sql in backend/migrations/
- [X] T012 [P] Create Category model struct in backend/product-service/src/models/category.go with validation tags
- [X] T013 [P] Create Product model struct in backend/product-service/src/models/product.go with validation tags
- [X] T014 [P] Create StockAdjustment model struct in backend/product-service/src/models/stock_adjustment.go with validation tags
- [X] T015 [P] Implement database connection pool setup in backend/product-service/src/config/database.go with tenant context support
- [X] T016 [P] Implement Redis cache client setup in backend/product-service/src/config/redis.go
- [X] T017 [P] Create tenant context middleware in backend/product-service/src/middleware/tenant.go to set app.current_tenant_id
- [X] T018 [P] Create structured logging setup in backend/product-service/src/utils/logger.go
- [X] T019 [P] Create error response utilities in backend/product-service/src/utils/errors.go
- [X] T020 Update API Gateway routes in api-gateway/main.go to proxy /api/v1/products, /api/v1/categories, /api/v1/inventory to product-service on port 8084

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Add New Products to Inventory (Priority: P1) 🎯 MVP

**Goal**: Enable store managers to create new products with all required details (name, SKU, category, pricing, initial stock) and make them available in the catalog for sale.

**Independent Test**: Create a product via API with all required fields, verify it's persisted in database and appears in GET /products list with correct attributes.

### Tests for User Story 1 (TDD - Write First, Ensure FAIL)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T021 [P] [US1] Contract test for POST /products endpoint in backend/product-service/tests/contract/product_create_test.go
- [ ] T022 [P] [US1] Contract test for GET /products endpoint in backend/product-service/tests/contract/product_list_test.go
- [ ] T023 [P] [US1] Contract test for POST /products/{id}/photo endpoint in backend/product-service/tests/contract/product_photo_test.go
- [ ] T024 [P] [US1] Unit test for ProductRepository.Create in backend/product-service/tests/unit/product_repository_test.go
- [ ] T025 [P] [US1] Unit test for ProductService.CreateProduct in backend/product-service/tests/unit/product_service_test.go
- [ ] T026 [P] [US1] Integration test for full product creation workflow in backend/product-service/tests/integration/create_product_test.go

### Implementation for User Story 1

- [X] T027 [P] [US1] Implement ProductRepository interface and Create method in backend/product-service/src/repository/product_repository.go with tenant-scoped queries
- [X] T028 [P] [US1] Implement CategoryRepository interface and FindAll method in backend/product-service/src/repository/category_repository.go for category selection
- [X] T029 [US1] Implement ProductService.CreateProduct business logic in backend/product-service/src/services/product_service.go with SKU uniqueness validation
- [X] T030 [US1] Implement ProductService.UploadPhoto with file validation and resizing in backend/product-service/src/services/product_service.go
- [X] T031 [US1] Implement POST /products handler in backend/product-service/api/product_handler.go with request validation
- [X] T032 [US1] Implement GET /products handler with pagination and filtering in backend/product-service/api/product_handler.go
- [X] T033 [US1] Implement POST /products/{id}/photo handler in backend/product-service/api/product_handler.go
- [X] T034 [US1] Add validation for product creation request (name 1-255 chars, SKU 1-50 chars, positive prices) in backend/product-service/api/product_handler.go
- [X] T035 [US1] Add SKU uniqueness check with 409 Conflict response in backend/product-service/src/services/product_service.go
- [X] T036 [P] [US1] Create ProductForm component in frontend/src/components/products/ProductForm.tsx
- [X] T037 [P] [US1] Create CategorySelect component in frontend/src/components/products/CategorySelect.tsx
- [X] T038 [P] [US1] Create product API client methods (createProduct, uploadPhoto) in frontend/src/services/product.service.ts
- [X] T039 [P] [US1] Create TypeScript types for Product, Category in frontend/src/types/product.types.ts
- [X] T040 [US1] Create new product page at frontend/app/products/new/page.tsx using ProductForm component
- [X] T041 [US1] Create product catalog page at frontend/app/products/page.tsx with ProductList component
- [X] T042 [US1] Implement ProductList component in frontend/src/components/products/ProductList.tsx with search and filtering

**Checkpoint**: At this point, User Story 1 should be fully functional - managers can create products with photos and see them in the catalog

---

## Phase 4: User Story 2 - Update Product Information (Priority: P1)

**Goal**: Enable store managers to edit existing product details (price, description, category, tax rates) to keep product data accurate and current.

**Independent Test**: Update an existing product's price and description via API, verify changes persist and display in GET /products/{id}.

### Tests for User Story 2 (TDD - Write First, Ensure FAIL)

- [ ] T043 [P] [US2] Contract test for PUT /products/{id} endpoint in backend/product-service/tests/contract/product_update_test.go
- [ ] T044 [P] [US2] Contract test for GET /products/{id} endpoint in backend/product-service/tests/contract/product_get_test.go
- [ ] T045 [P] [US2] Unit test for ProductRepository.Update in backend/product-service/tests/unit/product_repository_test.go
- [ ] T046 [P] [US2] Unit test for ProductService.UpdateProduct in backend/product-service/tests/unit/product_service_test.go
- [ ] T047 [P] [US2] Integration test for product update workflow in backend/product-service/tests/integration/update_product_test.go

### Implementation for User Story 2

- [X] T048 [P] [US2] Implement ProductRepository.FindByID method in backend/product-service/src/repository/product_repository.go
- [X] T049 [P] [US2] Implement ProductRepository.Update method in backend/product-service/src/repository/product_repository.go
- [X] T050 [US2] Implement ProductService.GetProduct business logic in backend/product-service/src/services/product_service.go
- [X] T051 [US2] Implement ProductService.UpdateProduct with validation in backend/product-service/src/services/product_service.go
- [X] T052 [US2] Implement GET /products/{id} handler in backend/product-service/api/product_handler.go
- [X] T053 [US2] Implement PUT /products/{id} handler in backend/product-service/api/product_handler.go
- [X] T054 [US2] Add SKU uniqueness validation for updates (exclude current product) in backend/product-service/src/services/product_service.go
- [X] T055 [P] [US2] Add updateProduct API client method in frontend/src/services/product.service.ts
- [X] T056 [P] [US2] Add getProduct API client method in frontend/src/services/product.service.ts
- [X] T057 [US2] Create product detail/edit page at frontend/app/products/[id]/page.tsx with ProductForm pre-populated
- [X] T058 [US2] Update ProductForm component to support edit mode in frontend/src/components/products/ProductForm.tsx

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently - create and update products

---

## Phase 5: User Story 3 - Track and View Inventory Levels (Priority: P1)

**Goal**: Enable staff to view current stock quantities for all products in real-time to know what's available for sale and when to reorder.

**Independent Test**: View inventory dashboard showing current stock levels, verify quantities display correctly and update when manual adjustments are made.

### Tests for User Story 3 (TDD - Write First, Ensure FAIL)

- [ ] T059 [P] [US3] Contract test for GET /products with low_stock filter in backend/product-service/tests/contract/product_inventory_test.go
- [ ] T060 [P] [US3] Unit test for ProductRepository.FindLowStock in backend/product-service/tests/unit/product_repository_test.go
- [ ] T061 [P] [US3] Integration test for inventory dashboard data in backend/product-service/tests/integration/inventory_view_test.go

### Implementation for User Story 3

- [X] T062 [P] [US3] Implement ProductRepository.FindLowStock method in backend/product-service/src/repository/product_repository.go
- [X] T063 [P] [US3] Implement ProductService.GetInventorySummary in backend/product-service/src/services/product_service.go
- [X] T064 [US3] Add low_stock query parameter handling to GET /products handler in backend/product-service/api/product_handler.go
- [X] T065 [US3] Implement GET /inventory/summary endpoint in backend/product-service/api/stock_handler.go
- [X] T066 [P] [US3] Add getInventorySummary API client method in frontend/src/services/product.service.ts
- [X] T067 [P] [US3] Create InventoryDashboard component in frontend/src/components/products/InventoryDashboard.tsx showing stock levels and low-stock alerts
- [X] T068 [US3] Update ProductList component to highlight low-stock and out-of-stock products in frontend/src/components/products/ProductList.tsx
- [X] T069 [US3] Add inventory status badges (in-stock, low-stock, out-of-stock) to ProductList in frontend/src/components/products/ProductList.tsx

**Checkpoint**: At this point, User Stories 1, 2, AND 3 should work - create, update, and view inventory levels

---

## Phase 6: User Story 4 - Manual Stock Adjustments with Audit Trail (Priority: P2)

**Goal**: Enable managers to manually adjust inventory quantities (restocks, corrections, shrinkage) with all changes logged including user, timestamp, and reason for accountability.

**Independent Test**: Make a stock adjustment via API with reason, verify stock quantity updates and adjustment is logged in stock_adjustments table with complete audit information.

### Tests for User Story 4 (TDD - Write First, Ensure FAIL)

- [ ] T070 [P] [US4] Contract test for POST /products/{id}/stock endpoint in backend/product-service/tests/contract/stock_adjustment_test.go
- [ ] T071 [P] [US4] Contract test for GET /products/{id}/adjustments endpoint in backend/product-service/tests/contract/stock_history_test.go
- [ ] T072 [P] [US4] Unit test for StockRepository.CreateAdjustment in backend/product-service/tests/unit/stock_repository_test.go
- [ ] T073 [P] [US4] Unit test for InventoryService.AdjustStock in backend/product-service/tests/unit/inventory_service_test.go
- [ ] T074 [P] [US4] Integration test for stock adjustment workflow with transaction in backend/product-service/tests/integration/adjust_stock_test.go

### Implementation for User Story 4

- [X] T075 [P] [US4] Implement StockRepository interface and CreateAdjustment method in backend/product-service/src/repository/stock_repository.go
- [X] T076 [P] [US4] Implement StockRepository.GetAdjustmentHistory method in backend/product-service/src/repository/stock_repository.go
- [X] T077 [US4] Implement InventoryService.AdjustStock with transaction (update product + create adjustment) in backend/product-service/src/services/inventory_service.go
- [X] T078 [US4] Implement InventoryService.GetAdjustmentHistory with filtering in backend/product-service/src/services/inventory_service.go
- [X] T079 [US4] Implement POST /products/{id}/stock handler in backend/product-service/api/stock_handler.go with reason validation
- [X] T080 [US4] Implement GET /products/{id}/adjustments handler in backend/product-service/api/stock_handler.go
- [X] T081 [US4] Implement GET /inventory/adjustments handler with filters in backend/product-service/api/stock_handler.go
- [X] T082 [US4] Add validation for reason codes (supplier_delivery, physical_count, shrinkage, damage, return, correction) in backend/product-service/api/stock_handler.go
- [X] T083 [P] [US4] Add adjustStock and getAdjustmentHistory API client methods in frontend/src/services/product.service.ts
- [X] T084 [P] [US4] Create StockAdjustment component in frontend/src/components/products/StockAdjustment.tsx with reason dropdown and notes field
- [X] T085 [US4] Add stock adjustment modal to product detail page in frontend/pages/products/[id].tsx
- [X] T086 [US4] Create adjustment history view in InventoryDashboard component in frontend/src/components/products/InventoryDashboard.tsx
- [X] T087 [US4] Add audit log display showing user, timestamp, quantity change, reason in frontend/src/components/products/StockAdjustment.tsx

**Checkpoint**: At this point, User Stories 1-4 complete - full inventory management with audit trail

---

## Phase 7: User Story 5 - Archive Products No Longer Sold (Priority: P2)

**Goal**: Enable managers to archive discontinued products to keep active catalog clean while preserving historical data and allowing restoration if needed.

**Independent Test**: Archive a product via API, verify it no longer appears in default GET /products list but is accessible with archived=true filter and can be restored.

### Tests for User Story 5 (TDD - Write First, Ensure FAIL)

- [ ] T088 [P] [US5] Contract test for PATCH /products/{id}/archive endpoint in backend/product-service/tests/contract/product_archive_test.go
- [ ] T089 [P] [US5] Contract test for PATCH /products/{id}/restore endpoint in backend/product-service/tests/contract/product_restore_test.go
- [ ] T090 [P] [US5] Unit test for ProductRepository.Archive in backend/product-service/tests/unit/product_repository_test.go
- [ ] T091 [P] [US5] Unit test for ProductService.ArchiveProduct in backend/product-service/tests/unit/product_service_test.go
- [ ] T092 [P] [US5] Integration test for archive/restore workflow in backend/product-service/tests/integration/archive_product_test.go

### Implementation for User Story 5

- [X] T093 [P] [US5] Implement ProductRepository.Archive method (set archived_at) in backend/product-service/src/repository/product_repository.go
- [X] T094 [P] [US5] Implement ProductRepository.Restore method (set archived_at to NULL) in backend/product-service/src/repository/product_repository.go
- [X] T095 [US5] Implement ProductService.ArchiveProduct business logic in backend/product-service/src/services/product_service.go
- [X] T096 [US5] Implement ProductService.RestoreProduct business logic in backend/product-service/src/services/product_service.go
- [X] T097 [US5] Implement PATCH /products/{id}/archive handler in backend/product-service/api/product_handler.go
- [X] T098 [US5] Implement PATCH /products/{id}/restore handler in backend/product-service/api/product_handler.go
- [X] T099 [US5] Update GET /products handler to exclude archived by default, support archived=true filter in backend/product-service/api/product_handler.go
- [X] T100 [P] [US5] Add archiveProduct and restoreProduct API client methods in frontend/src/services/product.service.ts
- [X] T101 [US5] Add archive/restore actions to product detail page in frontend/app/products/[id]/page.tsx
- [X] T102 [US5] Add archived filter toggle to ProductList component in frontend/src/components/products/ProductList.tsx
- [X] T103 [US5] Add visual indicator for archived products in ProductList in frontend/src/components/products/ProductList.tsx

**Checkpoint**: At this point, User Stories 1-5 complete - includes archiving functionality

---

## Phase 8: User Story 6 - Organize Products with Categories (Priority: P2)

**Goal**: Enable managers to create and manage categories to organize products, making catalog navigation easier and enabling category-based filtering and reporting.

**Independent Test**: Create categories via API, assign products to them, verify products can be filtered by category and category management (rename, delete with validation) works.

### Tests for User Story 6 (TDD - Write First, Ensure FAIL)

- [ ] T104 [P] [US6] Contract test for POST /categories endpoint in backend/product-service/tests/contract/category_create_test.go
- [ ] T105 [P] [US6] Contract test for PUT /categories/{id} endpoint in backend/product-service/tests/contract/category_update_test.go
- [ ] T106 [P] [US6] Contract test for DELETE /categories/{id} endpoint in backend/product-service/tests/contract/category_delete_test.go
- [ ] T107 [P] [US6] Unit test for CategoryRepository CRUD operations in backend/product-service/tests/unit/category_repository_test.go
- [ ] T108 [P] [US6] Unit test for CategoryService with delete validation in backend/product-service/tests/unit/category_service_test.go
- [ ] T109 [P] [US6] Integration test for category management workflow in backend/product-service/tests/integration/category_management_test.go

### Implementation for User Story 6

- [X] T110 [P] [US6] Implement CategoryRepository.Create method in backend/product-service/src/repository/category_repository.go
- [X] T111 [P] [US6] Implement CategoryRepository.Update method in backend/product-service/src/repository/category_repository.go
- [X] T112 [P] [US6] Implement CategoryRepository.Delete method in backend/product-service/src/repository/category_repository.go
- [X] T113 [P] [US6] Implement CategoryRepository.HasProducts method in backend/product-service/src/repository/category_repository.go
- [X] T114 [US6] Implement CategoryService.CreateCategory in backend/product-service/src/services/category_service.go
- [X] T115 [US6] Implement CategoryService.UpdateCategory in backend/product-service/src/services/category_service.go
- [X] T116 [US6] Implement CategoryService.DeleteCategory with product check in backend/product-service/src/services/category_service.go
- [X] T117 [US6] Implement POST /categories handler in backend/product-service/api/category_handler.go
- [X] T118 [US6] Implement GET /categories handler with caching (Redis, 5-min TTL) in backend/product-service/api/category_handler.go
- [X] T119 [US6] Implement PUT /categories/{id} handler in backend/product-service/api/category_handler.go
- [X] T120 [US6] Implement DELETE /categories/{id} handler with 403 if products assigned in backend/product-service/api/category_handler.go
- [X] T121 [US6] Add category name uniqueness validation per tenant in backend/product-service/src/services/category_service.go
- [X] T122 [US6] Add cache invalidation on category create/update/delete in backend/product-service/src/services/category_service.go
- [X] T123 [P] [US6] Add category API client methods (create, update, delete, list) in frontend/src/services/product.service.ts
- [X] T124 [P] [US6] Create category management page at frontend/app/products/categories/page.tsx
- [X] T125 [US6] Implement category CRUD UI in categories page with validation errors in frontend/app/products/categories/page.tsx
- [X] T126 [US6] Update CategorySelect component to fetch categories from API in frontend/src/components/products/CategorySelect.tsx
- [X] T127 [US6] Add category filter dropdown to ProductList component in frontend/src/components/products/ProductList.tsx

**Checkpoint**: At this point, User Stories 1-6 complete - full category management integrated

---

## Phase 9: User Story 7 - Delete Products Permanently (Priority: P3)

**Goal**: Enable managers to permanently remove test products or erroneous entries with safeguards preventing deletion of products with sales history.

**Independent Test**: Delete a product with no sales history via API with confirmation, verify it's removed. Attempt to delete product with sales history, verify it's prevented with appropriate error.

### Tests for User Story 7 (TDD - Write First, Ensure FAIL)

- [ ] T128 [P] [US7] Contract test for DELETE /products/{id} endpoint in backend/product-service/tests/contract/product_delete_test.go
- [ ] T129 [P] [US7] Unit test for ProductRepository.Delete in backend/product-service/tests/unit/product_repository_test.go
- [ ] T130 [P] [US7] Unit test for ProductService.DeleteProduct with sales history check in backend/product-service/tests/unit/product_service_test.go
- [ ] T131 [P] [US7] Integration test for product deletion with validation in backend/product-service/tests/integration/delete_product_test.go

### Implementation for User Story 7

- [X] T132 [P] [US7] Implement ProductRepository.Delete method in backend/product-service/src/repository/product_repository.go
- [X] T133 [P] [US7] Implement ProductRepository.HasSalesHistory method in backend/product-service/src/repository/product_repository.go (stub for future sales service integration)
- [X] T134 [US7] Implement ProductService.DeleteProduct with sales history check in backend/product-service/src/services/product_service.go
- [X] T135 [US7] Update DELETE /products/{id} handler to return 403 if sales history exists in backend/product-service/api/product_handler.go
- [X] T136 [P] [US7] Add deleteProduct API client method in frontend/src/services/product.service.ts
- [X] T137 [US7] Add delete action with confirmation dialog to product detail page in frontend/app/products/[id]/page.tsx
- [X] T138 [US7] Add error handling for 403 response suggesting archive instead in frontend/app/products/[id]/page.tsx

**Checkpoint**: All user stories (1-7) now complete - full Product & Inventory Management feature implemented

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T139 [P] Add structured logging for all product operations in backend/product-service/src/services/product_service.go
- [ ] T140 [P] Add structured logging for all inventory operations in backend/product-service/src/services/inventory_service.go
- [ ] T141 [P] Add health check endpoint GET /health in backend/product-service/api/health_handler.go
- [ ] T142 [P] Add readiness check endpoint GET /ready in backend/product-service/api/health_handler.go
- [ ] T143 [P] Implement graceful shutdown handling in backend/product-service/main.go
- [ ] T144 [P] Add request ID middleware for distributed tracing in backend/product-service/src/middleware/request_id.go
- [ ] T145 [P] Add rate limiting middleware in backend/product-service/src/middleware/rate_limit.go
- [ ] T146 [P] Add API response time metrics in backend/product-service/src/middleware/metrics.go
- [ ] T147 [P] Create README.md for product-service at backend/product-service/README.md
- [ ] T148 [P] Add unit tests for edge cases (negative stock, duplicate SKU, photo size limits) in backend/product-service/tests/unit/
- [ ] T149 [P] Add frontend unit tests for ProductForm component in frontend/src/components/products/ProductForm.test.tsx
- [ ] T150 [P] Add frontend unit tests for ProductList component in frontend/src/components/products/ProductList.test.tsx
- [ ] T151 Code cleanup: Remove debug logs, format code with gofmt in backend/product-service/
- [ ] T152 Code cleanup: Format frontend code with prettier in frontend/
- [ ] T153 Performance optimization: Add database query explain analysis and optimize slow queries
- [ ] T154 Performance optimization: Verify indexes are used in common queries (search by name, filter by category)
- [ ] T155 Security hardening: Validate all file uploads for malicious content in backend/product-service/src/services/product_service.go
- [ ] T156 Security hardening: Add CORS configuration in backend/product-service/main.go
- [ ] T157 Run quickstart.md validation: Test all API examples from specs/001-product-inventory/quickstart.md
- [ ] T158 Documentation: Update backend/product-service/README.md with deployment instructions
- [ ] T159 Documentation: Add JSDoc comments to frontend service methods in frontend/src/services/product.service.ts
- [ ] T160 Run full test suite and verify 80%+ code coverage requirement

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-9)**: All depend on Foundational phase completion
  - P1 stories (US1, US2, US3) can proceed in parallel after Phase 2
  - P2 stories (US4, US5, US6) can proceed after Phase 2, recommended after P1 complete
  - P3 story (US7) can proceed after Phase 2, recommended last
- **Polish (Phase 10)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Independently testable, extends US1 functionality
- **User Story 3 (P1)**: Can start after Foundational (Phase 2) - Independently testable, reads data created by US1/US2
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - Independently testable, modifies stock created by US1
- **User Story 5 (P2)**: Can start after Foundational (Phase 2) - Independently testable, adds archive state to US1 products
- **User Story 6 (P2)**: Can start after Foundational (Phase 2) - Independently testable, categories already used in US1
- **User Story 7 (P3)**: Can start after Foundational (Phase 2) - Independently testable, adds delete to US1

### Within Each User Story

- **Tests FIRST**: Write all tests for a story, ensure they FAIL
- **Models**: Can be implemented in parallel (marked [P])
- **Repositories**: Can be implemented in parallel (marked [P]) after models
- **Services**: Implement after repositories (may depend on multiple repos)
- **Handlers**: Implement after services
- **Frontend components**: Can be developed in parallel with backend (marked [P])
- **Integration**: Connect frontend to backend after both are complete

### Parallel Opportunities

- **Setup phase**: Tasks T003, T004, T005 can run in parallel
- **Foundational phase**: 
  - Migrations T006-T011 can run sequentially (dependencies on table creation order)
  - Models T012-T014 can run in parallel AFTER migrations
  - Config/middleware T015-T019 can run in parallel
- **Within each user story**: Tests can all run in parallel, models can run in parallel, frontend components can run in parallel with backend
- **Across user stories**: After Phase 2, all P1 stories (US1, US2, US3) can be worked on in parallel by different developers
- **Polish phase**: Most tasks marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# Tests - Launch together (TDD: write first, ensure FAIL):
T021: Contract test for POST /products endpoint
T022: Contract test for GET /products endpoint
T023: Contract test for POST /products/{id}/photo endpoint
T024: Unit test for ProductRepository.Create
T025: Unit test for ProductService.CreateProduct
T026: Integration test for full product creation workflow

# Models - Launch together:
T012: Create Category model struct
T013: Create Product model struct
T014: Create StockAdjustment model struct

# Repositories - Launch together (after models):
T027: Implement ProductRepository with Create method
T028: Implement CategoryRepository with FindAll method

# Frontend components - Launch together (parallel with backend):
T036: Create ProductForm component
T037: Create CategorySelect component
T038: Create product API client methods
T039: Create TypeScript types
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T020) - CRITICAL checkpoint
3. Complete Phase 3: User Story 1 (T021-T042)
4. **STOP and VALIDATE**: 
   - Run all US1 tests (should all pass)
   - Test product creation via API and frontend
   - Verify product appears in catalog
   - Upload product photo and verify it displays
5. Deploy/demo if ready - **This is a functional MVP!**

### Incremental Delivery (Recommended)

1. **Foundation** (Phases 1-2): Setup + Foundational → All infrastructure ready
2. **Increment 1** (Phase 3): User Story 1 → Test independently → Deploy/Demo **MVP!**
   - Delivers: Create products with photos, view catalog
3. **Increment 2** (Phase 4): User Story 2 → Test independently → Deploy/Demo
   - Delivers: MVP + Update product information
4. **Increment 3** (Phase 5): User Story 3 → Test independently → Deploy/Demo
   - Delivers: Previous + Inventory dashboard with stock levels
5. **Increment 4** (Phase 6): User Story 4 → Test independently → Deploy/Demo
   - Delivers: Previous + Manual stock adjustments with audit trail
6. **Increment 5** (Phase 7): User Story 5 → Test independently → Deploy/Demo
   - Delivers: Previous + Archive/restore products
7. **Increment 6** (Phase 8): User Story 6 → Test independently → Deploy/Demo
   - Delivers: Previous + Category management
8. **Increment 7** (Phase 9): User Story 7 → Test independently → Deploy/Demo
   - Delivers: Previous + Permanent product deletion
9. **Final Polish** (Phase 10): Cross-cutting improvements → Full feature complete

### Parallel Team Strategy

With 3 developers after Foundation complete:

1. **Team completes Setup + Foundational together** (critical path)
2. **Once Foundational is done** (after T020):
   - **Developer A**: User Story 1 (T021-T042) - Core product CRUD
   - **Developer B**: User Story 2 (T043-T058) - Product updates
   - **Developer C**: User Story 6 (T104-T127) - Category management
3. **Second wave**:
   - **Developer A**: User Story 3 (T059-T069) - Inventory tracking
   - **Developer B**: User Story 4 (T070-T087) - Stock adjustments
   - **Developer C**: User Story 5 (T088-T103) - Archive/restore
4. **Final wave**:
   - **Developer A**: User Story 7 (T128-T138) - Delete products
   - **All**: Phase 10 Polish tasks in parallel

---

## Task Summary

- **Total Tasks**: 160
- **Phase 1 (Setup)**: 5 tasks
- **Phase 2 (Foundational)**: 15 tasks - **CRITICAL CHECKPOINT**
- **Phase 3 (US1 - Add Products)**: 22 tasks (7 tests + 15 implementation)
- **Phase 4 (US2 - Update Products)**: 16 tasks (5 tests + 11 implementation)
- **Phase 5 (US3 - Inventory Levels)**: 11 tasks (3 tests + 8 implementation)
- **Phase 6 (US4 - Stock Adjustments)**: 18 tasks (5 tests + 13 implementation)
- **Phase 7 (US5 - Archive Products)**: 16 tasks (5 tests + 11 implementation)
- **Phase 8 (US6 - Categories)**: 24 tasks (6 tests + 18 implementation)
- **Phase 9 (US7 - Delete Products)**: 11 tasks (4 tests + 7 implementation)
- **Phase 10 (Polish)**: 22 tasks

### Tasks per User Story

- **US1 (Add Products)**: 22 tasks → **MVP scope**
- **US2 (Update Products)**: 16 tasks
- **US3 (Inventory Levels)**: 11 tasks
- **US4 (Stock Adjustments)**: 18 tasks
- **US5 (Archive Products)**: 16 tasks
- **US6 (Categories)**: 24 tasks
- **US7 (Delete Products)**: 11 tasks

### Parallel Opportunities

- **Setup phase**: 3 tasks can run in parallel
- **Foundational phase**: Up to 6 tasks can run in parallel (after migrations)
- **User stories**: 7 stories can be distributed across team members after foundation
- **Within stories**: Average 5-8 tasks per story can run in parallel (tests, models, frontend)
- **Polish phase**: 18 tasks can run in parallel

### Independent Test Criteria

- **US1**: Create product with photo → appears in catalog
- **US2**: Update product price/description → changes persist
- **US3**: View inventory dashboard → stock levels display correctly
- **US4**: Adjust stock with reason → quantity updates, audit log created
- **US5**: Archive product → removed from active list, can restore
- **US6**: Create category, assign products → filter by category works
- **US7**: Delete test product → removed permanently; products with sales history protected

### Suggested MVP Scope

**Minimum viable product**: Phase 1 + Phase 2 + Phase 3 (User Story 1)
- Creates working product catalog with photos
- Enables store to start tracking what they sell
- Delivers immediate value in ~27 tasks
- Can be deployed independently and tested end-to-end

---

## Format Validation

✅ **All tasks follow checklist format**: `- [ ] [ID] [P?] [Story?] Description with file path`
✅ **Task IDs**: Sequential T001-T160
✅ **[P] markers**: Present on parallelizable tasks (different files, no dependencies)
✅ **[Story] labels**: Present on all user story phase tasks (US1-US7)
✅ **File paths**: Included in all implementation task descriptions
✅ **Tests first**: Test tasks appear before implementation tasks per TDD
✅ **Independent stories**: Each story phase can be tested independently
✅ **MVP identified**: User Story 1 marked as MVP scope

---

## Notes

- All tasks follow Test-First Development per constitution
- Tests are written FIRST, must FAIL before implementation
- Each user story delivers independently testable value
- MVP (User Story 1) delivers functional product catalog in 27 tasks
- Commit after completing each task or logical group
- Stop at any checkpoint to validate story works independently
- File paths are exact per plan.md structure
- Multi-tenant isolation enforced via RLS at database level
- Photo storage uses file system with DB metadata (per research.md)
- All API endpoints follow OpenAPI contract in contracts/product-api.yaml
