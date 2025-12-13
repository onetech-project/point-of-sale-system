# Implementation Plan: Product Photo Storage in Object Storage

**Branch**: `005-product-photo-storage` | **Date**: December 12, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/005-product-photo-storage/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Migrate product photo storage from local filesystem to S3-compatible object storage (MinIO for development, S3 for production) to enable scalable, reliable, and multi-tenant isolated photo management. The product-service will be enhanced with object storage integration, supporting upload, retrieval, and deletion of multiple photos per product with tenant isolation, automatic image optimization, and graceful fallback handling.

## Technical Context

**Language/Version**: Go 1.23.0  
**Primary Dependencies**: Echo v4.12.0 (web framework), MinIO Go SDK (object storage client), lib/pq (PostgreSQL driver), disintegration/imaging v1.6.2 (image processing)  
**Storage**: PostgreSQL 14.19 (metadata), MinIO/S3 (object storage for photos), Redis 8.0.5 (caching)  
**Testing**: Go testing package with testify v1.9.0, sqlmock v1.5.2 for database mocking  
**Target Platform**: Linux server (Docker containerized microservice)  
**Project Type**: Web application (microservices architecture) - backend service enhancement  
**Performance Goals**: Photo upload <10s for 10MB files, photo retrieval <2s (95th percentile), 100 concurrent uploads supported  
**Constraints**: Multi-tenant isolation required, 5GB default storage quota per tenant, photos <10MB, dimensions <4096x4096px  
**Scale/Scope**: Existing product-service microservice (~115 lines main.go), add photo management module with S3 integration (~500-800 new lines estimated)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Microservice Autonomy
✅ **PASS** - Enhancing existing product-service microservice. No new service created. Photo storage ownership remains with product-service.

### Principle II: API-First Design
✅ **PASS** - Will define API contracts for photo upload/retrieval endpoints before implementation (Phase 1 contracts/).

### Principle III: Test-First Development
✅ **PASS** - Plan includes test strategy: unit tests for storage operations, contract tests for API endpoints, integration tests for S3 interaction.

### Principle IV: Observability & Monitoring
✅ **PASS** - Will log all photo operations (upload, delete, errors), emit metrics for storage usage, implement health checks for S3 connectivity.

### Principle V: Security by Design
✅ **PASS** - Tenant isolation via storage prefixes/buckets, authentication for photo access, input validation (file size, format, dimensions), audit logging.

### Principle VI: Simplicity First (KISS + DRY + YAGNI)
✅ **PASS** - Using existing MinIO client library (proven technology), minimal abstraction (storage service layer), implementing only required features (no premature optimization like CDN integration).

### Engineering Best Practices Assessment
- **TDD**: Will write tests before implementation ✅
- **MVP Approach**: Phase 1 implements core upload/retrieval, Phase 2+ adds enhancements ✅
- **SOLID Principles**: Storage operations encapsulated in service layer, dependency injection for S3 client ✅
- **Fail Fast**: Input validation at API boundary, early error detection ✅

### Architecture Constraints
- **Service Communication**: REST/HTTP for photo upload/retrieval ✅
- **Data Management**: Product-service owns photo metadata in its PostgreSQL schema ✅
- **Database per Service**: No shared database, photos stored in object storage ✅

**Overall Status**: ✅ ALL GATES PASSED - Proceed to Phase 0

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
backend/product-service/
├── main.go                          # Service entry point (register photo routes)
├── go.mod                           # Add minio-go v7 dependency
├── src/
│   ├── config/
│   │   └── storage.go              # NEW: S3/MinIO configuration
│   ├── models/
│   │   └── product_photo.go        # NEW: ProductPhoto model
│   ├── repository/
│   │   └── photo_repository.go     # NEW: Photo database operations
│   ├── services/
│   │   ├── storage_service.go      # NEW: S3 upload/download/delete
│   │   ├── photo_service.go        # NEW: Photo business logic
│   │   └── image_processor.go      # NEW: Resize, validate, optimize
│   └── middleware/
│       └── quota_middleware.go     # NEW: Storage quota enforcement
├── api/
│   └── photo_handler.go            # NEW: Photo upload/list/delete endpoints
└── tests/
    ├── unit/
    │   ├── storage_service_test.go
    │   ├── photo_service_test.go
    │   └── image_processor_test.go
    ├── integration/
    │   ├── photo_api_test.go
    │   └── s3_integration_test.go
    └── contract/
        └── photo_contract_test.go

backend/migrations/
└── 007_add_product_photos.up.sql   # NEW: Database schema migration

docker-compose.yml                   # Add MinIO service

frontend/src/
├── components/
│   ├── products/
│   │   ├── PhotoUpload.tsx         # NEW: Photo upload component
│   │   ├── PhotoGallery.tsx        # NEW: Photo carousel display
│   │   └── PhotoManager.tsx        # NEW: Photo list with reorder/delete
│   └── common/
│       └── ImagePlaceholder.tsx    # NEW: Fallback placeholder image
├── services/
│   └── photoService.ts             # NEW: API client for photo endpoints
└── hooks/
    └── usePhotoUpload.ts           # NEW: Upload hook with progress

specs/005-product-photo-storage/
├── spec.md                          # Feature specification
├── plan.md                          # This file
├── research.md                      # Technology decisions
├── data-model.md                    # Database schema and models
├── quickstart.md                    # Developer setup guide
├── contracts/
│   └── api.md                       # API contract documentation
└── checklists/
    └── requirements.md              # Specification quality checklist
```

**Structure Decision**: This feature enhances the existing product-service microservice in the web application architecture. No new services are created. The implementation adds:
- Backend: Photo management module in product-service (6 new files, 1 migration)
- Frontend: Photo upload and display components (5 new files)
- Infrastructure: MinIO service in docker-compose.yml

The structure follows the existing pattern where product-service owns product-related functionality, including photo storage. Frontend components are added to the Next.js application under the products feature directory.

## Complexity Tracking

**No violations detected** - All constitution principles followed. No complexity justification required.

---

## Phase 0: Research (COMPLETED)

✅ **Status**: Complete  
📄 **Output**: [research.md](./research.md)

### Decisions Made

1. **Object Storage**: MinIO (dev) + AWS S3 (prod) with S3-compatible API
2. **Multi-Tenancy**: Single bucket with tenant-prefixed keys (`photos/{tenant_id}/...`)
3. **Go SDK**: minio-go v7 for S3 operations
4. **Image Processing**: disintegration/imaging (already in dependencies)
5. **Database Schema**: New `product_photos` table with metadata
6. **Error Handling**: Circuit breaker pattern + graceful degradation with placeholders
7. **Storage Quota**: Database-tracked usage with enforcement at upload time
8. **Photo URLs**: Presigned URLs with Redis caching (7-day for public, 24h for private)
9. **Migration**: Manual script provided separately, not in feature scope

All "NEEDS CLARIFICATION" items from Technical Context have been resolved with documented rationale and alternatives considered.

---

## Phase 1: Design (COMPLETED)

✅ **Status**: Complete  
📄 **Outputs**: 
- [data-model.md](./data-model.md)
- [contracts/api.md](./contracts/api.md)
- [quickstart.md](./quickstart.md)

### Data Model Summary

**New Table**: `product_photos`
- Stores metadata: storage key, filename, size, dimensions, display order
- Foreign keys: product_id, tenant_id with cascade delete
- Constraints: unique display_order per product, only one primary photo
- Indexes: product_id, tenant_id for efficient queries

**Modified Table**: `tenants`
- Added: `storage_used_bytes`, `storage_quota_bytes`
- Default quota: 5GB per tenant

**Storage Key Format**: `photos/{tenant_id}/{product_id}/{photo_id}_{timestamp}.{ext}`

### API Endpoints

1. `POST /api/v1/products/{id}/photos` - Upload photo
2. `GET /api/v1/products/{id}/photos` - List photos
3. `GET /api/v1/products/{id}/photos/{photo_id}` - Get single photo
4. `PATCH /api/v1/products/{id}/photos/{photo_id}` - Update metadata
5. `DELETE /api/v1/products/{id}/photos/{photo_id}` - Delete photo
6. `PUT /api/v1/products/{id}/photos/reorder` - Bulk reorder
7. `GET /api/v1/tenants/storage-quota` - Get quota info
8. Enhanced product endpoints with `?include_photos=true`

All endpoints require authentication and tenant isolation enforcement.

### Developer Onboarding

Quickstart guide provides:
- MinIO setup in Docker Compose
- Database migration instructions
- Step-by-step implementation guide
- Testing procedures
- Troubleshooting tips
- Estimated 4-6 hours for MVP implementation

---

## Phase 1: Agent Context Update (COMPLETED)

✅ **Status**: Complete  
📄 **Updated**: `.github/agents/copilot-instructions.md`

Added to Active Technologies:
- Go 1.23.0 + Echo v4.12.0 (web framework)
- MinIO Go SDK (object storage client)
- lib/pq (PostgreSQL driver)
- disintegration/imaging v1.6.2 (image processing)
- PostgreSQL 14.19 (metadata)
- MinIO/S3 (object storage for photos)
- Redis 8.0.5 (caching)

---

## Constitution Check Re-evaluation (Phase 1 Complete)

**Re-check Status**: ✅ ALL GATES STILL PASSING

### Post-Design Validation

**Principle I: Microservice Autonomy**  
✅ **PASS** - Design confirms photo storage module remains within product-service. No new service dependencies introduced.

**Principle II: API-First Design**  
✅ **PASS** - Complete API contracts documented in `contracts/api.md` with request/response schemas, error codes, and versioning before any implementation.

**Principle III: Test-First Development**  
✅ **PASS** - Quickstart includes test strategy: unit tests for storage/image processing, contract tests for APIs, integration tests for S3. Tests defined before implementation code.

**Principle IV: Observability & Monitoring**  
✅ **PASS** - Design includes structured logging for all photo operations, storage usage metrics, health checks for S3 connectivity, and circuit breaker for resilience.

**Principle V: Security by Design**  
✅ **PASS** - Multi-tenant isolation enforced at database (tenant_id foreign key) and storage (tenant-prefixed keys) levels. Authentication required for all endpoints. Input validation at API boundary. Presigned URLs with expiration. Audit logging for all operations.

**Principle VI: Simplicity First (KISS + DRY + YAGNI)**  
✅ **PASS** - MVP approach: core upload/retrieval only, no premature optimization. Reuses existing imaging library. Simple storage service abstraction. No speculative features (CDN integration deferred to future).

### Engineering Best Practices Re-check

- **SOLID Principles**: Service layer separates concerns, dependency injection for storage client ✅
- **MVP Approach**: Phase 1 delivers core value (upload/retrieve/delete), enhancements deferred ✅
- **Fail Fast**: Input validation at API boundary, early detection of quota/size violations ✅
- **Design for Testability**: Services accept interfaces, database mocked in unit tests ✅

**Final Verdict**: ✅ **READY FOR IMPLEMENTATION**

No constitution violations. All principles and best practices satisfied. Proceed to `/speckit.tasks` to generate implementation tasks.

---

## Next Steps

1. **Run `/speckit.tasks`** - Generate atomic implementation tasks from this plan
2. **Create Feature Branch** - Already on `005-product-photo-storage`
3. **Setup Environment** - Follow `quickstart.md` to add MinIO and run migrations
4. **Implement TDD** - Write tests first, implement to pass tests
5. **Code Review** - Verify constitution compliance, test coverage ≥80%
6. **Deploy to Staging** - Test with real MinIO/S3
7. **Production Deploy** - Switch to AWS S3 endpoint

---

## Summary

**Feature**: Product Photo Storage in Object Storage  
**Branch**: `005-product-photo-storage`  
**Status**: Planning Complete - Ready for Tasks Generation  
**Estimated Effort**: 4-6 hours core implementation + 2-4 hours testing/polish = 6-10 hours total

**Deliverables**:
- ✅ research.md - Technology decisions and best practices
- ✅ data-model.md - Database schema and Go models
- ✅ contracts/api.md - Complete REST API specification
- ✅ quickstart.md - Developer setup and implementation guide
- ✅ Agent context updated with new technologies
- ✅ Constitution check passed (all principles satisfied)

**Key Technical Decisions**:
1. MinIO for dev, S3 for prod (seamless migration via S3 API compatibility)
2. Single bucket with tenant-prefixed keys (simple, scalable, cost-effective)
3. minio-go v7 SDK (proven, feature-complete)
4. Database-tracked quota enforcement (accurate, efficient)
5. Presigned URLs with Redis caching (performance + security)
6. Graceful degradation with placeholders (resilient to storage outages)

**Next Command**: `/speckit.tasks` to break down into atomic implementation tasks.
