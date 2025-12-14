# Tasks: Product Photo Storage in Object Storage

**Input**: Design documents from `/specs/005-product-photo-storage/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.md

**Feature**: Migrate product photo storage from local filesystem to S3-compatible object storage with multi-tenant isolation, automatic optimization, and graceful fallback handling.

**Tests**: Tests are NOT explicitly requested in the specification, so test tasks are EXCLUDED from this implementation plan. TDD approach will be followed during implementation by developers.

**Organization**: Tasks are grouped by user story (P1, P2, P3) to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- All tasks include exact file paths

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, dependencies, and basic structure

- [X] T001 Add minio-go v7 dependency to backend/product-service/go.mod
- [X] T002 Add MinIO service to docker-compose.yml with ports 9000 (API) and 9001 (console)
- [X] T003 [P] Update backend/product-service/.env.example with S3 configuration variables
- [X] T004 [P] Create placeholder image assets in frontend/public/assets/ for product photo fallbacks

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story implementation

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Create database migration backend/migrations/007_add_product_photos.up.sql
- [X] T006 Create database migration backend/migrations/007_add_product_photos.down.sql
- [X] T007 Run database migration to create product_photos table and add storage tracking to tenants
- [X] T008 [P] Create ProductPhoto model struct in backend/product-service/src/models/product_photo.go
- [X] T009 [P] Create storage configuration loader in backend/product-service/src/config/storage.go
- [X] T010 [P] Update Tenant model in backend/product-service/src/models/tenant.go with storage quota fields (BLOCKED: tenant model in tenant-service)
- [X] T011 Create StorageService with S3 client initialization in backend/product-service/src/services/storage_service.go
- [X] T012 [P] Create PhotoRepository for database operations in backend/product-service/src/repository/photo_repository.go
- [X] T013 Create ImageProcessor service for validation/optimization in backend/product-service/src/services/image_processor.go
- [X] T014 Initialize MinIO bucket creation script or manual setup documentation

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Upload Product Photos (Priority: P1) 🎯 MVP

**Goal**: Enable tenant administrators to upload product photos that are stored in S3 and accessible for display

**Independent Test**: Upload a photo when creating/editing a product and verify it appears when viewing product details

### Implementation for User Story 1

- [X] T015 [P] [US1] Implement UploadPhoto method in StorageService (backend/product-service/src/services/storage_service.go)
- [X] T016 [P] [US1] Implement GetPhotoURL method in StorageService (backend/product-service/src/services/storage_service.go)
- [X] T017 [P] [US1] Implement ValidateImage method in ImageProcessor (backend/product-service/src/services/image_processor.go)
- [X] T018 [P] [US1] Implement OptimizeImage method in ImageProcessor (backend/product-service/src/services/image_processor.go)
- [X] T019 [US1] Create PhotoService with UploadPhoto business logic in backend/product-service/src/services/photo_service.go
- [X] T020 [US1] Implement Create method in PhotoRepository (backend/product-service/src/repository/photo_repository.go)
- [X] T021 [US1] Implement GetByProduct method in PhotoRepository (backend/product-service/src/repository/photo_repository.go)
- [X] T022 [US1] Implement UpdateTenantStorageUsage method in PhotoRepository (backend/product-service/src/repository/photo_repository.go)
- [X] T023 [US1] Create PhotoHandler with UploadPhoto endpoint handler in backend/product-service/api/photo_handler.go
- [X] T024 [US1] Register POST /api/v1/products/:product_id/photos route in backend/product-service/main.go
- [X] T025 [US1] Add file upload validation middleware (size, type, dimensions) in backend/product-service/api/photo_handler.go
- [X] T026 [US1] Add storage quota check before upload in PhotoService (backend/product-service/src/services/photo_service.go)
- [X] T027 [US1] Implement photo replacement logic (delete old, upload new) in PhotoService
- [X] T028 [US1] Add error handling and logging for upload operations in PhotoHandler
- [X] T029 [P] [US1] Create PhotoUpload component in frontend/src/components/products/PhotoUpload.tsx
- [X] T030 [P] [US1] Create photoService API client in frontend/src/services/photoService.ts
- [ ] T031 [US1] Integrate PhotoUpload component into product create/edit forms in frontend
- [ ] T032 [US1] Add upload progress indicator and error messages in PhotoUpload component

**Checkpoint**: At this point, User Story 1 should be fully functional - can upload photos and view them in product details

---

## Phase 4: User Story 3 - Access Product Photos Across System (Priority: P1)

**Goal**: Ensure product photos load quickly (<2s) and reliably across all system contexts (catalog, orders, receipts)

**Independent Test**: View products in different contexts (web catalog, order history) and verify photos load within 2 seconds

**Note**: Implementing this before US2 because it's P1 (critical) vs US2 is P2 (enhancement)

### Implementation for User Story 3

- [X] T033 [P] [US3] Implement GetPhotoByID method in PhotoRepository (backend/product-service/src/repository/photo_repository.go)
- [X] T034 [US3] Implement ListPhotos method in PhotoHandler (backend/product-service/api/photo_handler.go)
- [X] T035 [US3] Register GET /api/v1/products/:product_id/photos route in backend/product-service/main.go
- [X] T036 [US3] Implement GetPhoto method in PhotoHandler for single photo retrieval
- [X] T037 [US3] Register GET /api/v1/products/:product_id/photos/:photo_id route in backend/product-service/main.go
- [ ] T038 [US3] Add Redis caching for presigned URLs (7-day TTL for public, 24h for private) in StorageService
- [ ] T039 [US3] Enhance GET /api/v1/products/:id endpoint with include_photos=true query parameter
- [ ] T040 [US3] Enhance GET /api/v1/products endpoint with include_primary_photo=true query parameter
- [ ] T041 [P] [US3] Create PhotoGallery component for displaying photos in frontend/src/components/products/PhotoGallery.tsx
- [ ] T042 [US3] Integrate PhotoGallery into product detail pages in frontend
- [ ] T043 [US3] Add photo display to product list/catalog views in frontend
- [ ] T044 [US3] Add photo display to order detail views in frontend
- [ ] T045 [US3] Implement lazy loading for photo gallery to optimize performance

**Checkpoint**: At this point, photos load efficiently across all system contexts

---

## Phase 5: User Story 4 - Tenant Data Isolation (Priority: P1)

**Goal**: Ensure each tenant's photos are stored separately and securely with no cross-tenant access

**Independent Test**: Upload photos as two different tenants and verify each can only access their own photos

### Implementation for User Story 4

- [X] T046 [US4] Implement tenant-prefixed storage key generation in StorageService (photos/{tenant_id}/{product_id}/{photo_id})
- [X] T047 [US4] Add tenant_id validation in all PhotoRepository methods
- [X] T048 [US4] Add tenant_id validation in all PhotoHandler endpoints (verify JWT tenant_id matches request)
- [X] T049 [US4] Implement GetTenantStorageQuota endpoint in PhotoHandler
- [X] T050 [US4] Register GET /api/v1/tenants/storage-quota route in backend/product-service/main.go
- [X] T051 [US4] Add tenant_id check in presigned URL generation (only allow access to own tenant's photos)
- [ ] T052 [US4] Implement cascade delete for tenant deletion (remove all tenant photos from S3)
- [ ] T053 [US4] Add audit logging for all photo operations with tenant_id in backend/product-service/src/services/photo_service.go
- [ ] T054 [P] [US4] Add storage quota display in frontend tenant dashboard
- [ ] T055 [US4] Add storage quota warning when approaching limit (80%, 90%, 95%) in frontend

**Checkpoint**: Tenant isolation is fully enforced - no cross-tenant access possible

---

## Phase 6: User Story 2 - Multiple Photos per Product (Priority: P2)

**Goal**: Allow uploading and managing multiple photos (up to 5) per product with ordering

**Independent Test**: Upload 3-5 photos to a product and verify all are stored, retrievable, and displayed in correct order

### Implementation for User Story 2

- [X] T056 [US2] Add check for maximum photos (5) before upload in PhotoService
- [X] T057 [US2] Implement display_order auto-assignment logic in PhotoService
- [X] T058 [US2] Implement UpdatePhotoMetadata method in PhotoRepository (backend/product-service/src/repository/photo_repository.go)
- [X] T059 [US2] Implement PATCH /api/v1/products/:product_id/photos/:photo_id handler in PhotoHandler
- [X] T060 [US2] Register PATCH route for photo metadata updates in backend/product-service/main.go
- [X] T061 [US2] Implement ReorderPhotos method in PhotoService for bulk display order updates
- [X] T062 [US2] Implement PUT /api/v1/products/:product_id/photos/reorder handler in PhotoHandler
- [X] T063 [US2] Register PUT route for photo reordering in backend/product-service/main.go
- [X] T064 [US2] Add primary photo toggle logic (ensure only one primary per product) in PhotoService
- [ ] T065 [P] [US2] Create PhotoManager component with reorder/delete UI in frontend/src/components/products/PhotoManager.tsx
- [ ] T066 [US2] Integrate PhotoManager into product edit page in frontend
- [ ] T067 [US2] Add drag-and-drop reordering functionality in PhotoManager component
- [ ] T068 [US2] Add primary photo selection toggle in PhotoManager component

**Checkpoint**: Multiple photos per product fully functional with reordering

---

## Phase 7: User Story 5 - Graceful Fallbacks (Priority: P3)

**Goal**: Display placeholder images when photos are missing or fail to load

**Independent Test**: Remove a photo from storage and verify the system shows a placeholder instead of broken image

### Implementation for User Story 5

- [X] T069 [P] [US5] Create default placeholder images (multiple sizes) in frontend/public/assets/
- [X] T070 [US5] Implement DeletePhoto method in StorageService (backend/product-service/src/services/storage_service.go)
- [X] T071 [US5] Implement Delete method in PhotoRepository with soft delete marker
- [X] T072 [US5] Implement DELETE /api/v1/products/:product_id/photos/:photo_id handler in PhotoHandler
- [X] T073 [US5] Register DELETE route for photo deletion in backend/product-service/main.go
- [ ] T074 [US5] Add background job/retry logic for failed S3 deletions in PhotoService
- [ ] T075 [US5] Implement circuit breaker pattern for S3 operations in StorageService
- [ ] T076 [US5] Add fallback to placeholder URL on S3 error in StorageService.GetPhotoURL
- [ ] T077 [P] [US5] Create ImagePlaceholder component in frontend/src/components/common/ImagePlaceholder.tsx
- [ ] T078 [US5] Add error handling with placeholder fallback in PhotoGallery component
- [ ] T079 [US5] Add loading indicators during photo upload/load in frontend components
- [ ] T080 [US5] Add error logging and monitoring for photo failures in backend

**Checkpoint**: System handles photo errors gracefully without breaking UI

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T081 [P] Update API documentation with photo endpoints in docs/API.md
- [ ] T082 [P] Add photo storage feature documentation in docs/
- [ ] T083 Code cleanup and refactoring in backend photo services
- [ ] T084 Add error response standardization across all photo endpoints
- [ ] T085 Add rate limiting for photo upload endpoint (10 uploads/minute per tenant)
- [ ] T086 Add monitoring and alerts for storage quota violations
- [ ] T087 Add health check for S3/MinIO connectivity in backend/product-service
- [ ] T088 Performance optimization: add database indexes if needed
- [ ] T089 Security audit: verify tenant isolation and input validation
- [ ] T090 Run through quickstart.md validation steps to ensure developer documentation is accurate

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup (Phase 1) - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2) - MVP deliverable
- **User Story 3 (Phase 4)**: Depends on Foundational (Phase 2) and US1 (Phase 3) - Requires photos to exist for display
- **User Story 4 (Phase 5)**: Depends on Foundational (Phase 2) and US1 (Phase 3) - Enhances security of existing upload/access
- **User Story 2 (Phase 6)**: Depends on Foundational (Phase 2) and US1 (Phase 3) - Builds on single photo upload
- **User Story 5 (Phase 7)**: Depends on US1 and US3 - Adds resilience to existing photo operations
- **Polish (Phase 8)**: Depends on desired user stories being complete

### User Story Completion Order (By Priority)

**MVP Path** (P1 stories only):
1. Phase 1: Setup
2. Phase 2: Foundational ✅ GATE
3. Phase 3: User Story 1 (Upload) ✅ MVP CHECKPOINT
4. Phase 4: User Story 3 (Access/Display) ✅ COMPLETE P1
5. Phase 5: User Story 4 (Isolation) ✅ COMPLETE P1 SECURITY

**Full Feature Path** (All stories):
1-5. (As above)
6. Phase 6: User Story 2 (Multiple Photos) ✅ COMPLETE P2
7. Phase 7: User Story 5 (Fallbacks) ✅ COMPLETE P3
8. Phase 8: Polish ✅ PRODUCTION READY

### Within Each User Story

For User Story 1 (Upload):
- T001-T014: Foundation must complete first
- T015-T018: Can run in parallel (different services)
- T019: Depends on T015-T018 (PhotoService uses StorageService and ImageProcessor)
- T020-T022: Can run in parallel (different repository methods)
- T023-T028: Sequential (handler → route → validation → error handling)
- T029-T030: Can run in parallel (frontend component + API client)
- T031-T032: Sequential (integration → polish)

For User Story 3 (Access):
- Depends on US1 being complete (photos must exist to display)
- T033-T034: Can start in parallel
- T038: Caching enhancement (can be done independently)
- T041-T045: Frontend work can parallelize with backend T033-T040

For User Story 4 (Isolation):
- Security enhancements to existing US1/US3 code
- T046-T053: Backend security (sequential to ensure correctness)
- T054-T055: Frontend quota display (parallel with backend work)

For User Story 2 (Multiple Photos):
- Extends US1 upload capability
- T056-T064: Backend features (some parallelizable)
- T065-T068: Frontend features (can start after backend endpoints ready)

For User Story 5 (Fallbacks):
- Error handling for US1/US3
- T069-T076: Backend resilience (some parallelizable)
- T077-T080: Frontend fallbacks (can parallel with backend)

### Parallel Opportunities

**Phase 1 (Setup)**: All 4 tasks can run in parallel

**Phase 2 (Foundational)**: Tasks T008, T009, T010, T012 can run in parallel

**Within User Stories**:
- User Story 1: T015-T018 (services), T020-T022 (repo), T029-T030 (frontend)
- User Story 2: T056-T058 (backend logic), T065 (frontend component)
- User Story 5: T069 (assets), T077 (component)

**Parallel Execution Example for User Story 1**:

```bash
# After Foundation (T001-T014) complete, start US1:

# Batch 1 - Core services (parallel):
T015: Implement UploadPhoto in StorageService
T016: Implement GetPhotoURL in StorageService  
T017: Implement ValidateImage in ImageProcessor
T018: Implement OptimizeImage in ImageProcessor

# Batch 2 - Repository (parallel):
T020: Implement Create in PhotoRepository
T021: Implement GetByProduct in PhotoRepository
T022: Implement UpdateTenantStorageUsage in PhotoRepository

# Batch 3 - PhotoService (depends on Batch 1+2):
T019: Create PhotoService with UploadPhoto logic

# Batch 4 - API Layer (sequential):
T023: PhotoHandler
T024: Register route
T025-T028: Validation, quota, error handling

# Batch 5 - Frontend (parallel with Batch 3-4):
T029: PhotoUpload component
T030: photoService API client

# Batch 6 - Integration (depends on everything):
T031-T032: Integration and polish
```

---

## Implementation Strategy

### MVP First (P1 Stories: US1, US3, US4)

1. ✅ Complete Phase 1: Setup (T001-T004)
2. ✅ Complete Phase 2: Foundational (T005-T014) - **CRITICAL GATE**
3. ✅ Complete Phase 3: User Story 1 - Upload (T015-T032)
   - **STOP and VALIDATE**: Upload a photo, verify it's stored and viewable
4. ✅ Complete Phase 4: User Story 3 - Access (T033-T045)
   - **STOP and VALIDATE**: View photos in catalog, orders - verify <2s load time
5. ✅ Complete Phase 5: User Story 4 - Isolation (T046-T055)
   - **STOP and VALIDATE**: Test cross-tenant access blocked, quotas enforced
6. **MVP READY FOR DEPLOYMENT**

### Full Feature (Add P2 and P3)

7. ✅ Complete Phase 6: User Story 2 - Multiple Photos (T056-T068)
   - **STOP and VALIDATE**: Upload 5 photos, reorder, set primary
8. ✅ Complete Phase 7: User Story 5 - Fallbacks (T069-T080)
   - **STOP and VALIDATE**: Simulate S3 outage, verify placeholders shown
9. ✅ Complete Phase 8: Polish (T081-T090)
10. **FEATURE COMPLETE - PRODUCTION READY**

### Incremental Delivery Milestones

| Milestone | Tasks | Value Delivered | Can Deploy? |
|-----------|-------|-----------------|-------------|
| Foundation | T001-T014 | Infrastructure ready | No (no user features) |
| MVP Core | +T015-T032 | Upload photos | Yes (basic functionality) |
| MVP Display | +T033-T045 | View photos system-wide | Yes (usable feature) |
| MVP Secure | +T046-T055 | Tenant isolation enforced | Yes (production-safe) |
| Enhanced | +T056-T068 | Multiple photos per product | Yes (full P1+P2) |
| Resilient | +T069-T080 | Graceful error handling | Yes (production-hardened) |
| Polished | +T081-T090 | Documentation, monitoring | Yes (production-ready) |

### Parallel Team Strategy

With 3 developers after Foundational phase:
- **Developer A**: User Story 1 (Upload) → User Story 2 (Multiple)
- **Developer B**: User Story 3 (Access) → User Story 5 (Fallbacks)
- **Developer C**: User Story 4 (Isolation) → Phase 8 (Polish)

Stories integrate independently without blocking each other.

---

## Task Count Summary

- **Total Tasks**: 90
- **Phase 1 (Setup)**: 4 tasks
- **Phase 2 (Foundational)**: 10 tasks ⚠️ BLOCKS all stories
- **Phase 3 (US1 - Upload, P1)**: 18 tasks 🎯 MVP
- **Phase 4 (US3 - Access, P1)**: 13 tasks
- **Phase 5 (US4 - Isolation, P1)**: 10 tasks
- **Phase 6 (US2 - Multiple, P2)**: 13 tasks
- **Phase 7 (US5 - Fallbacks, P3)**: 12 tasks
- **Phase 8 (Polish)**: 10 tasks

**Parallel Opportunities**: 15+ tasks marked [P] can run simultaneously

**MVP Scope** (P1 only): 55 tasks (Phases 1-5)  
**Full Feature**: 90 tasks (All phases)

---

## Notes

- **[P] marker**: Tasks that can run in parallel (different files, no blocking dependencies)
- **[Story] marker**: Maps task to specific user story for traceability and independent delivery
- **Test-First**: Although test tasks not included, developers should write tests before implementation per TDD principle
- **Checkpoints**: Stop after each user story phase to validate independently
- **File paths**: All tasks include exact file locations for clarity
- **MVP first**: Implement P1 stories (US1, US3, US4) before P2/P3 for fastest time to value
- **Independent stories**: Each user story should work independently once its phase is complete
- **Tenant isolation**: US4 tasks enhance security of US1/US3 - can be done after upload/display work

---

## Format Validation

✅ All tasks follow checklist format: `- [ ] [TaskID] [P?] [Story?] Description with file path`  
✅ Task IDs are sequential (T001-T090)  
✅ [P] marker only on parallelizable tasks  
✅ [Story] marker on user story phase tasks only (US1-US5)  
✅ All task descriptions include specific file paths  
✅ Tasks organized by user story phases  
✅ Dependencies section shows story completion order  
✅ MVP scope clearly identified (P1 stories)
