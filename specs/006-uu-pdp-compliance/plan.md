# Implementation Plan: Indonesian Data Protection Compliance (UU PDP)

**Branch**: `006-uu-pdp-compliance` | **Date**: 2026-01-02 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-uu-pdp-compliance/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement comprehensive data protection compliance for Indonesian Personal Data Protection Law (UU PDP No.27 Tahun 2022) covering encryption at rest for all PII, log masking to prevent sensitive data leaks, persistent audit trail for compliance investigations, explicit consent collection and management, tenant and guest data access/deletion rights, privacy policy transparency, and automated data retention enforcement. Technical approach requires encryption library for transparent field-level encryption, centralized log masking formatter, append-only audit log storage, consent tracking with versioning, privacy-aware UI components, and automated cleanup jobs for expired data.

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.24 (backend microservices), TypeScript/Next.js 16 App Router (frontend)  
**Primary Dependencies**: Echo v4 (Go web framework), lib/pq (PostgreSQL driver), OpenTelemetry, Kafka, React 19, next-i18next  
**Storage**: PostgreSQL (primary data storage with RLS and multi-tenancy), Kafka (event streaming for audit events), HashiCorp Vault (encryption key management - Transit Engine)  
**Testing**: Go testing framework (unit/integration), Jest + React Testing Library (frontend), target ≥80% coverage per constitution  
**Target Platform**: Linux server (Docker containers), Web browsers (Chrome, Firefox, Safari)
**Project Type**: web - Multi-tenant Point-of-Sale (POS) with microservice architecture  
**Performance Goals**: Privacy Policy page <2s (p95) measured via Lighthouse (SSR reduces TTFB to <500ms, p95), encryption/decryption overhead <10% for business ops (measured via benchmark tests), guest data deletion within 24h (asynchronous job processing), API response times <100ms (p95), database query times <50ms (p95)  
**Constraints**: Multi-tenant data isolation via RLS, Midtrans payment integration (third-party consent), Indonesian language support (Bahasa Indonesia), UU PDP No.27 Tahun 2022 compliance (fines up to IDR 6 billion)  
**Scale/Scope**: Multi-tenant system, multiple businesses, guest orders (no auth), microservices: auth, user, order, product, tenant, notification + API Gateway + observability stack

**Encryption Key Storage (Resolves Analysis Finding A1 - HIGH)**: Encryption keys stored in HashiCorp Vault Transit Engine (production) and file-based with 400 permissions (development only). Keys never stored in application database. Vault handles all encrypt/decrypt operations via Transit Engine API. The ciphertext format ("vault:v1:...") includes version information directly, so Vault automatically uses the correct key version during decryption. This addresses FR-009 "secure storage" requirement with specific implementation details.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Microservice Autonomy
**Status**: ✅ PASS  
**Rationale**: Feature spans multiple existing services (auth, user, order, tenant, notification). Each service maintains complete autonomy by implementing encryption, audit, and log masking within its own codebase (src/utils/). Each service owns its encryption logic, audit event publishing to Kafka, and log masking patterns. No shared libraries or dedicated audit-service; follows existing per-service implementation pattern.

### API-First Design  
**Status**: ✅ PASS  
**Rationale**: Data management endpoints (tenant data access/update/delete, guest data access/delete) require OpenAPI contracts. Consent management and privacy policy are API-driven. Contracts will be generated in Phase 1.

### Test-First Development (NON-NEGOTIABLE)
**Status**: ✅ PASS  
**Rationale**: Feature has comprehensive acceptance scenarios (5-8 scenarios per user story). Tests can be written before implementation: encryption round-trip tests, log masking validation, consent flow tests, audit trail verification. BDD scenarios map directly to test cases.

### Observability by Default
**Status**: ✅ PASS  
**Rationale**: Existing OpenTelemetry + Prometheus + Grafana + Tempo + Loki stack supports this feature. Metrics needed: encryption/decryption latency, consent grant/revoke rates, audit log write performance, data deletion processing time. Alerts for: encryption failures, audit log write failures, expired data not cleaned up.

### Security by Default
**Status**: ✅ PASS  
**Rationale**: This feature IS security implementation (encryption at rest, audit trail, consent). Follows principle by design. Key management requires secure storage (research needed). Audit logs append-only with restricted access. Guest data deletion requires verification (order ref + email/phone). 'Comprehensive audit trail' defined as: ALL operations (create, read, update, delete) on PII-containing entities logged with complete context (actor, action, timestamp, IP address, before/after values encrypted, resource type), stored in immutable append-only table, retention ≥7 years per Indonesian compliance standards.

### KISS, DRY, YAGNI
**Status**: ✅ PASS  
**Rationale**: Feature scope is minimal for compliance: encrypt sensitive fields (MVP: identified PII only), mask logs (simple pattern matching), record consent (simple table), audit trail (append-only log), privacy policy (static page). No premature optimization. Encryption library choice should prioritize simplicity over custom crypto.

### SOLID Principles
**Status**: ✅ PASS (pending design)  
**Rationale**: Encryption logic should be single responsibility service/library. Audit trail writer should be dependency-injected. Consent manager separate from user service. Will verify in Phase 1 design review.

### Incremental Delivery
**Status**: ✅ PASS  
**Rationale**: Feature has clear phases per spec Notes: (1) Encryption + log masking first, (2) Consent + privacy policy before new registrations, (3) Audit trail in parallel, (4) Data retention cleanup last. Each phase independently deployable and testable.

### Incremental Refactoring
**Status**: ✅ PASS  
**Rationale**: Existing services will be refactored to add encryption, log masking, audit hooks. Changes are additive (new fields, new middleware) with minimal breaking changes. Password hashing already exists, excluded from encryption scope.

### MVP Approach
**Status**: ✅ PASS  
**Rationale**: Spec defines MVP: encrypt identified PII fields only (not all data), implement required consents only (not granular preference center), simple privacy policy page (not interactive), guest verification via order ref + email/phone (not advanced MFA). Out of scope explicitly lists non-MVP features.

### Design for Testability
**Status**: ✅ PASS  
**Rationale**: Encryption/decryption will be injectable interface. Audit trail writer will be dependency. Log masking will be centralized formatter (testable in isolation). Consent collection is form validation (unit testable). Each component testable without full system.

### Fail Fast
**Status**: ✅ PASS  
**Rationale**: Consent validation at form submission (fail before data processing). Encryption failures should return error (don't store unencrypted fallback). Audit log write failures should alert (don't silently drop logs). Key access failures should halt service startup.

### Documentation as Code
**Status**: ✅ PASS  
**Rationale**: This plan.md + research.md + data-model.md + contracts/ fulfill requirement. Privacy policy will be code-managed. Consent versions will be tracked in code. Encryption key rotation procedures will be in runbooks (Phase 1 quickstart.md).

**GATE DECISION**: ✅ **PROCEED TO PHASE 0**  
All constitution principles satisfied. No violations requiring justification.

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
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
# Web application structure (frontend + backend detected)
backend/
├── auth-service/         # Authentication & session management (audit auth events)
├── user-service/         # User CRUD + email verification (encrypt user PII, tenant data rights)
├── order-service/        # Order management + guest orders (encrypt order PII, guest data rights)
├── tenant-service/       # Tenant management (encrypt tenant configs, payment credentials)
├── product-service/      # Product catalog (no PII, no encryption needed)
├── notification-service/ # Email/notifications (encrypt recipient PII, message content)
└── migrations/           # PostgreSQL schema migrations (add encryption fields, audit_events, consent_records)

api-gateway/             # Request routing + middleware (log masking middleware)

frontend/
├── app/                 # Next.js App Router (not pages/)
│   ├── privacy-policy/  # Privacy policy page route
│   ├── settings/        # Account settings with privacy sub-routes
│   └── guest/           # Guest order data access routes
├── src/
│   ├── components/      # Consent forms, data management UI components
│   └── services/        # API clients for consent, data access, deletion requests (existing pattern)
└── tests/

observability/           # Existing stack (monitor encryption latency, audit log writes)

# New/modified files for this feature:
backend/migrations/
├── 000028_create_consent_purposes.up.sql       # Consent purpose configuration table
├── 000029_create_privacy_policies.up.sql       # Privacy policy versioning table
├── 000030_create_consent_records.up.sql        # Consent tracking table (tenant + guest)
├── 000031_create_audit_events.up.sql           # Audit trail table (partitioned by month)
├── 000032_create_retention_policies.up.sql     # Data retention configuration table
├── 000033-000040_add_encryption_fields.up.sql  # Add *_encrypted columns to 8 existing tables
└── 000041_audit_events_immutability.up.sql     # Trigger to enforce append-only audit logs

# Each backend service extends existing structure with:
# - src/utils/encryption.go       # Encryption/decryption utilities (Vault Transit Engine)
# - src/utils/audit.go             # Audit event publisher (Kafka client)
# - src/utils/masker.go            # Log masking formatter
# - src/repository/*_repo.go       # Repository methods with encryption/audit hooks
# - middleware/logging.go          # Logging middleware with PII masking

frontend/
├── app/privacy-policy/page.tsx                      # Privacy policy page (Indonesian, SSR)
├── app/settings/privacy/page.tsx                    # Tenant consent management
├── app/guest/[ref]/data/page.tsx                    # Guest data access + deletion
├── src/components/consent/ConsentCheckboxGroup.tsx  # Reusable consent UI
├── src/services/consent.ts                          # Consent API client (existing pattern)
├── src/services/privacy.ts                          # Privacy/data rights API client (existing pattern)
└── src/services/guest.ts                            # Guest data access API client (existing pattern)
```

**Structure Decision**: Web application (Option 2 from template). Existing backend microservices will be extended with encryption, audit hooks, and new endpoints following current per-service pattern. Each service implements its own encryption utilities (src/utils/encryption.go), audit event publishing (src/utils/audit.go), and log masking (src/utils/masker.go) within its own codebase. Frontend uses Next.js 16 App Router (app/ directory) with API clients in src/services/ following existing pattern (auth.ts, user.ts, etc.).

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**No violations identified**. All constitution principles satisfied in Phase 0 and Phase 1 design:

- ✅ API-First: Consent API and Data Rights API contracts generated (OpenAPI 3.0)
- ✅ Test-First: All components designed with testability (injectable encryption service, mockable audit logger)
- ✅ KISS: Simple encryption library, field-level masking utility, straightforward API endpoints
- ✅ Security: Encryption at rest with external KMS (Vault), append-only audit logs, consent-based data processing

**Phase 1 Design Validation**:
- Data model follows normalized database design (5NF for consent/audit tables)
- API contracts follow RESTful conventions
- No premature optimizations (e.g., caching deferred to performance testing phase)
- Encryption service interface allows easy swapping (Vault → AWS KMS in future)

**GATE DECISION**: ✅ **PROCEED TO IMPLEMENTATION**
