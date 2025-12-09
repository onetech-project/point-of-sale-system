# Tasks: 001-order-notifications

Phase 1: Setup

- [ ] T001 [P] Add Redis and SSE configuration to `backend/notification-service/.env.example` (keys: `REDIS_URL`, `SSE_REPLAY_BUFFER_TTL_SECONDS`, `SSE_MAX_CLIENTS_PER_INSTANCE`).
- [ ] T002 [P] Add developer note to `specs/004-order-notifications/quickstart.md` explaining how to run Redis locally and verify tenant channels (commands and example `redis-cli` queries).

Phase 2: Foundational (blocking prerequisites)

- [ ] T003 Create a dedupe/event-records migration `migrations/00000X_create_event_records.up.sql` to store processed `event_id` for idempotency (file path: `migrations/00000X_create_event_records.up.sql`).
 - [ ] T003 Create a dedupe/event-records migration `backend/migrations/000023_create_event_records.up.sql` to store processed `event_id` for idempotency (file path: `backend/migrations/000023_create_event_records.up.sql`).
- [ ] T004 [P] Add Redis helper and configuration to `backend/notification-service/src/providers/redis.go` (new file) to publish tenant-channel messages and support a short-lived ring buffer for replay.
- [ ] T005 Implement SSE HTTP handler in `backend/notification-service/api/sse.go` and register route in `backend/notification-service/main.go`.
- [ ] T006 [P] Add `GET /api/orders/snapshot` endpoint in `backend/order-service/src/api/snapshot.go` to return an order list snapshot for a tenant (used for client resync).
- [ ] T007 [P] Add AsyncAPI & SSE contract verification tests under `specs/004-order-notifications/tests/contract/` to validate `contracts/asyncapi-orders-events.yaml` and `contracts/sse-contract.md` are present and syntactically valid.

Phase 3: User Story Implementation

US1 - Receive immediate notification when order is paid (Priority: P1)

- [ ] T008 [US1] Implement Kafka consumer for `orders.events` in `backend/notification-service/src/queue/orders_consumer.go` that:
  - subscribes to `orders.events`
  - parses `order_paid` and `order_status_updated` events
  - performs dedupe using `EventRecord` (migration from T003)
  - persists `Notification` entries in `backend/notification-service/src/repository/notification_repository.go` (reuse existing repository)
  - publishes a lightweight in-app payload to Redis tenant channel and calls existing `sendEmail` logic when applicable

- [ ] T009 [US1] Add or update an email template `backend/notification-service/templates/order_paid.html` for paid-order emails (create if missing) and load it in `notification_service.go`'s template loader.

- [ ] T010 [US1] Add integration tests `backend/notification-service/tests/integration/order_paid_consumer_test.go` that publish a test `order_paid` event to Kafka and assert: Notification record created, Redis publish occurred, sendEmail was invoked (mocked), and no duplicate notifications when same event reprocessed.

US2 - Real-time order list updates for submitted or status-changed orders (Priority: P1)

- [ ] T011 [US2] Implement Redis publish when processing order events in `orders_consumer.go` (same as T008) to publish a concise event for SSE clients: `{ id: event_id, event: event_type, data: { order_id, tenant_id, status, reference, total_amount, timestamp } }`.

- [ ] T012 [US2] Implement SSE client handler in frontend at `frontend/src/services/sse.ts` (new) and integrate into `frontend/src/components/OrderList.tsx` (or the equivalent order list component) to:
  - open `GET /api/sse/notifications` with auth
  - handle `order_created`, `order_paid`, `order_status_updated` events and update UI state in real time
  - implement Last-Event-ID usage and fallback to `GET /api/orders/snapshot` on missed events

- [ ] T013 [US2] Add end-to-end integration test (manual / CI) documented in `specs/004-order-notifications/quickstart.md` demonstrating: Start services, push `order_created`/`order_paid` events, and observe UI updates without page refresh.

US3 - Notification preferences and role scoping (Priority: P2)

- [ ] T014 [US3] Add migration `migrations/00000Y_add_notification_preferences_to_users.up.sql` to extend users table with `notification_email_enabled` and `notification_in_app_enabled` columns (file path: `migrations/00000Y_add_notification_preferences_to_users.up.sql`).
 - [ ] T014 [US3] Add migration `backend/migrations/000024_add_notification_preferences_to_users.up.sql` to extend users table with `notification_email_enabled` and `notification_in_app_enabled` columns (file path: `backend/migrations/000024_add_notification_preferences_to_users.up.sql`).

Additional Remediation Tasks (from analysis)

- [ ] T021 [P] Verify and/or implement event emission in `backend/order-service` so `orders.events` is published on order creation/payment/status changes. Add contract tests in `backend/order-service/tests/contract/` to validate emitted event shape matches `contracts/asyncapi-orders-events.yaml`.
- [ ] T022 [CRITICAL] Resolve duplicate spec directory numeric-prefix conflict: repository currently contains multiple `specs/001-*` directories which breaks `.specify` tooling. Options:
  - Rename other `001-*` spec folders to unique numbers (e.g., `002-...`) OR
  - Consolidate related specs under a single numeric prefix if they belong to same feature.
  Document the chosen resolution in `docs/SPEC_PREFIX_CHANGE.md` and re-run `.specify/scripts/bash/check-prerequisites.sh` to confirm the fix.
- [ ] T023 [P] Decide dedupe store and retention policy (Postgres `event_records` table vs Redis with TTL). Document the choice in `specs/004-order-notifications/research.md` and update T003 and T004 implementation details accordingly.
- [ ] T024 [P] Add SSE auth details task: update `specs/004-order-notifications/contracts/sse-contract.md` with required JWT claims (tenant_id, roles), token validation rules, and reconnect behavior; add contract tests to enforce auth behavior.
- [ ] T015 [US3] Update user model `backend/user-service/src/models/user.go` to include notification preference fields and update any user creation/update paths to accept preferences.
- [ ] T016 [US3] Update notification creation logic in `backend/notification-service/src/services/notification_service.go` to respect per-user preferences (skip email/in-app send when disabled) and ensure role scoping (only Owner/Manager/Cashier receive order-paid notifications).

Cross-cutting / Final Phase

- [ ] T017 Add observability instrumentation in `backend/notification-service/src/observability/metrics.go` (new) and wire metrics: Kafka consumer lag, processed events, failed events, SSE connections, Redis publish latency, email send failures.
- [ ] T018 [P] Add runbooks and dashboards entry in `docs/` describing alert thresholds and recovery steps for consumer lag, Redis errors, and email failure spikes (file: `docs/ORDER_NOTIFICATIONS_OPERATIONS.md`).
- [ ] T019 Add contract tests and CI job configuration to validate AsyncAPI and SSE contract files on PRs (CI config under `.github/workflows/contract-tests.yml`).
- [ ] T020 [P] Create a PR for branch `001-order-notifications` containing spec, plan, contracts, and initial implementation tasks (`git push && gh pr create` or equivalent) — include reviewers from backend/frontend/ops.

Dependencies (story completion order)

- Foundational tasks (T003..T007) MUST complete before production rollout.  
- US1 tasks (T008..T010) depend on T003 (dedupe) and T004 (Redis helper) and T005 (SSE handler) for correct end-to-end behavior.  
- US2 tasks (T011..T013) depend on T004, T005, and T006 (snapshot endpoint).  
- US3 tasks (T014..T016) depend on user-service migration (T014/T015) before notification preference enforcement (T016).

Parallel execution examples

- Team A (backend consumer & email): Work in parallel on T008 (orders_consumer), T009 (templates), and T010 (integration tests). These touch different files and can be reviewed independently.  
- Team B (SSE & frontend): Work in parallel on T005 (SSE handler) and T012 (frontend SSE client). T005 provides the server endpoint; frontend work can proceed against a dev stub or the local SSE handler.  
- Team C (data & user prefs): Work in parallel on T003 (event_records migration) and T014/T015 (user notification migration and model updates).

Implementation strategy (MVP first, incremental delivery)

- MVP scope: Implement minimal path for US1 (paid-order event → notification record → email printed to logs and Redis publish) and US2 minimal SSE path for order_created/order_paid update to a single connected client. This covers payment acknowledgement and live order appearance.  
- Iteration 1 (MVP): T003, T004, T008, T009, T011, T005 (basic SSE), T012 (basic frontend client), T010 (basic integration test).  
- Iteration 2: Add dedupe robustness, snapshot endpoint (T006), replay buffer, retry policies, and production-grade metrics (T017).  
- Iteration 3: Preferences and role scoping (T014..T016), contract tests (T019), and runbooks (T018).

Task counts & summary

- Total tasks: 20  
- Tasks by story: US1: 3 tasks (T008-T010), US2: 3 tasks (T011-T013), US3: 3 tasks (T014-T016), Foundational: 5 tasks (T003-T007), Setup: 2 tasks (T001-T002), Cross-cutting: 4 tasks (T017-T020).
