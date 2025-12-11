# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Deliver a real-time in-app notification and email pipeline for paid orders and realtime order-list updates. Use the existing `notification-service` as the primary consumer for payment/order events (Kafka) to produce email notifications and to feed in-app real-time updates. Real-time in-app updates will be delivered to active clients via Server-Sent Events (SSE) (architecture decision: notification-service will host SSE endpoints unless scaling/operational constraints advise proxying via `api-gateway`).

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.23 (backend services), Node.js / Next.js 16 (frontend)  
**Primary Dependencies**: `labstack/echo` (HTTP server), `segmentio/kafka-go` for Kafka, PostgreSQL, Redis (for ephemeral state / presence), SMTP provider for email (or transactional email provider)  
**Storage**: PostgreSQL (existing migrations present), Notification table exists in `notification-service` repository.  
**Testing**: Go unit tests (`go test`), integration tests for Kafka consumers (integration harness), frontend tests using `jest` and testing-library.  
**Target Platform**: Linux server containers (Docker), orchestrated via docker-compose in repo; production likely on Kubernetes (TBD).  
**Project Type**: Web application (backend microservices + Next.js frontend).  
**Performance Goals**: Meet spec success criteria: SSE delivery to active sessions within 5s for 95% of events; email pipeline enqueue within 30s for 95% of events. Suggested initial scaling targets (MVP defaults, subject to validation by T025 load tests): concurrent SSE connections (cluster-wide) = 5,000; burst event rate = 1,000 events/s; steady-state event rate = 100 events/s. These defaults must be validated and adjusted during load testing.
**Constraints**: Preserve microservice autonomy — `notification-service` should own notification data and delivery. Ensure idempotency and deduplication for events (order payment events). Ensure tenancy scoping and least-privilege access to notification streams.  
**Scale/Scope**: Initially support single-tenant test clusters and production-scale tenants (estimate: NEEDS CLARIFICATION: expected active tenants and peak order rates).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Gates evaluated against the project constitution (see `.specify/memory/constitution.md`):

- Microservice Autonomy: PASS — we will use the existing `notification-service` to own notification data and delivery, avoiding cross-service database sharing.
- API-First Design: PASS (no breaking API changes required; SSE contract and any new endpoints will be documented as part of Phase 1 contracts).
- Test-First Development: PASS (tests will be created for the Kafka consumer, notification processing, and SSE endpoints; plan requires contract and integration tests).
- Observability & Monitoring: CONDITIONALLY PASS — plan mandates structured logs/metrics/traces for new Kafka consumers and SSE gateway; specifics (metric names, dashboards) will be added during Phase 1.
- Security by Design: PASS — tenancy scoping and auth will be enforced at the SSE endpoint (JWT/session) and Kafka topic subscriptions limited by service credentials. Implementation details will be documented in ADR if deviations needed.
- Simplicity & YAGNI: CONDITIONALLY PASS — we will prefer reusing `notification-service` and incremental approach. If scaling issues appear, we will evaluate proxying SSE through `api-gateway` or separating SSE into its own service (justified in Complexity Tracking).

No constitution gates are blocked. Observability and scaling monitoring need concrete artifacts in Phase 1.

### Post-Design Re-evaluation

After Phase 1 artifacts (data-model, contracts, quickstart) were created, the constitution check remains PASS with the following follow-ups required before implementation:

- Add API contract tests and OpenAPI/AsyncAPI artifacts (created in `contracts/`).  
- Create monitoring dashboards and alerting (Phase 1 action items).  
- Create ADR if we decide to change SSE hosting from `notification-service` to `api-gateway` or a dedicated SSE service (document complexity justification).

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
# [REMOVE IF UNUSED] Option 1: Single project (DEFAULT)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [REMOVE IF UNUSED] Option 2: Web application (when "frontend" + "backend" detected)
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [REMOVE IF UNUSED] Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure: feature modules, UI flows, platform tests]
```

**Structure Decision**: Use existing repository structure: backend microservices (including `notification-service`) will host server logic; frontend `Next.js` app consumes SSE and REST APIs. Concrete pieces:

- Backend: `backend/notification-service/` — extend to subscribe to `order_paid` and `order_status_updated` Kafka topics (or shared topic) and to emit in-app notification events and persist Notification records.
- Frontend: `frontend/` — implement SSE client that connects to `/api/v1/sse` (auth required) and updates order-list UI and shows in-app toast/notification center.
- Gateway/API: Optionally expose SSE endpoint from `api-gateway/` to proxy to `notification-service` if needed for cross-origin or auth centralization (TBD during Phase 1).

Rationale: Reuse existing `notification-service` to avoid duplicate event consumers and centralize notification templates, retries, and email sending. Evaluate scaling trade-offs in Phase 1.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
