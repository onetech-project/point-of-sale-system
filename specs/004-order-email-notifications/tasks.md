# Tasks: Order Email Notifications

**Feature Branch**: `004-order-email-notifications`  
**Input**: Design documents from `/specs/004-order-email-notifications/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

---

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and database schema

- [X] T001 Create database migration 000023_add_order_notification_prefs.up.sql in backend/migrations/
- [X] T002 Create database migration 000023_add_order_notification_prefs.down.sql in backend/migrations/
- [X] T003 Create database migration 000024_create_notification_configs.up.sql in backend/migrations/
- [X] T004 Create database migration 000024_create_notification_configs.down.sql in backend/migrations/
- [X] T005 Create database migration 000025_add_notification_indexes.up.sql in backend/migrations/
- [X] T006 Create database migration 000025_add_notification_indexes.down.sql in backend/migrations/
- [X] T007 Apply migrations to development database and verify schema changes
- [ ] T008 [P] Create migration test file backend/migrations/tests/migration_test.go to validate schema
- [ ] T009 [P] Update notification-service dependencies in backend/notification-service/go.mod if needed

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T010 Create OrderPaidEvent struct in backend/notification-service/src/models/events.go
- [X] T011 Add JSON Schema validation for OrderPaidEvent in backend/notification-service/src/models/event_validation.go
- [X] T012 [P] Extend Kafka consumer to handle order.paid event type in backend/notification-service/src/queue/kafka_consumer.go
- [X] T013 [P] Create NotificationConfig repository in backend/notification-service/src/repositories/notification_config_repository.go
- [X] T014 [P] Extend User repository to query receive_order_notifications field in backend/user-service/src/repositories/user_repository.go
- [X] T015 Implement duplicate notification check by transaction_id in backend/notification-service/src/services/notification_service.go
- [X] T016 [P] Create email template helpers (formatCurrency, etc.) in backend/notification-service/src/utils/template_helpers.go
- [X] T017 [P] Add retry worker background goroutine in backend/notification-service/main.go
- [X] T018 Implement exponential backoff retry logic in backend/notification-service/src/services/retry_worker.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Staff Receives Order Notifications (Priority: P1) 🎯 MVP

**Goal**: Tenant staff receive email notification within 1 minute when order transitions to PAID status

**Independent Test**: Complete a paid order through guest ordering, verify all configured staff members receive email with order details within 1 minute

### Tests for User Story 1 (TDD: Write tests FIRST)

- [X] T019 [P] [US1] Write unit test for renderStaffNotificationTemplate() in backend/notification-service/src/services/template_service_test.go (MUST FAIL initially)
- [X] T020 [P] [US1] Write unit test for handleOrderPaidEvent() with mock Kafka message in backend/notification-service/src/handlers/order_paid_handler_test.go (MUST FAIL initially)
- [X] T021 [US1] Write integration test for end-to-end staff notification flow in backend/notification-service/tests/integration/staff_notification_test.go (MUST FAIL initially)

### Implementation for User Story 1

- [X] T022 [P] [US1] Create staff notification HTML template in backend/notification-service/templates/order_staff_notification.html
- [ ] T023 [P] [US1] Create StaffNotificationData struct in backend/notification-service/src/models/notification_data.go
- [ ] T024 [US1] Implement renderStaffNotificationTemplate() in backend/notification-service/src/services/template_service.go (verify T019 passes)
- [ ] T025 [US1] Implement handleOrderPaidEvent() to process order.paid events in backend/notification-service/src/handlers/order_paid_handler.go (verify T020 passes)
- [ ] T026 [US1] Implement queryStaffRecipients() to get users with receive_order_notifications=true in backend/notification-service/src/services/notification_service.go
- [ ] T027 [US1] Implement sendStaffNotifications() to send emails to all staff recipients in backend/notification-service/src/services/notification_service.go
- [ ] T028 [US1] Add logging for notification attempts in backend/notification-service/src/services/notification_service.go
- [ ] T029 [US1] Update order-service to publish order.paid event to Kafka in backend/order-service/src/services/order_service.go
- [ ] T030 [US1] Test staff notification email rendering in Email on Acid or Litmus across common clients (verify T021 passes)

**Checkpoint**: Staff notifications working independently - staff receive order notification emails within 1 minute of payment

---

## Phase 4: User Story 2 - Customer Receives Email Receipt (Priority: P1)

**Goal**: Guest customer receives email receipt with invoice design and PAID watermark within 2 minutes of payment

**Independent Test**: Complete a guest order with payment and customer email, verify customer receives email receipt with PAID watermark and payment details

### Tests for User Story 2 (TDD: Write tests FIRST)

- [ ] T031 [P] [US2] Write unit test for renderCustomerReceiptTemplate() with watermark in backend/notification-service/src/services/template_service_test.go (MUST FAIL initially)
- [ ] T032 [P] [US2] Write unit test for sendCustomerReceipt() with valid/invalid email in backend/notification-service/src/services/notification_service_test.go (MUST FAIL initially)
- [ ] T033 [US2] Write integration test for customer receipt with PAID watermark in backend/notification-service/tests/integration/customer_receipt_test.go (MUST FAIL initially)

### Implementation for User Story 2

- [ ] T034 [P] [US2] Backup existing invoice template to backend/notification-service/templates/order_invoice.html.backup
- [ ] T035 [US2] Add CSS watermark styles to backend/notification-service/templates/order_invoice.html
- [ ] T036 [US2] Add conditional watermark div with {{if .ShowPaidWatermark}} to backend/notification-service/templates/order_invoice.html
- [ ] T037 [P] [US2] Create CustomerReceiptData struct extending InvoiceData in backend/notification-service/src/models/notification_data.go
- [ ] T038 [US2] Implement renderCustomerReceiptTemplate() reusing invoice template in backend/notification-service/src/services/template_service.go (verify T031 passes)
- [ ] T039 [US2] Implement sendCustomerReceipt() in backend/notification-service/src/services/notification_service.go (verify T032 passes)
- [ ] T040 [US2] Update handleOrderPaidEvent() to send customer receipt if customer_email provided in backend/notification-service/src/handlers/order_paid_handler.go
- [ ] T041 [US2] Add email format validation before sending receipt in backend/notification-service/src/utils/validators.go
- [ ] T042 [US2] Test customer receipt email rendering with watermark in Email on Acid or Litmus (verify T033 passes)

**Checkpoint**: Customer receipts working independently - customers receive email receipt with PAID watermark within 2 minutes

---

## Phase 5: User Story 3 - Configure Email Notification Preferences (Priority: P2)

**Goal**: Tenant administrators configure which staff members receive order notifications through admin dashboard

**Independent Test**: Login as tenant admin, navigate to notification settings, enable/disable staff notifications, verify changes take effect on next paid order

### Tests for User Story 3 (TDD: Write tests FIRST)

- [ ] T043 [P] [US3] Write contract test for GET /users/notification-preferences in backend/user-service/tests/contract/notification_preferences_test.go (MUST FAIL initially)
- [ ] T044 [P] [US3] Write contract test for PATCH /users/:id/notification-preferences in backend/user-service/tests/contract/notification_preferences_test.go (MUST FAIL initially)
- [ ] T045 [P] [US3] Write contract test for POST /notifications/test in backend/notification-service/tests/contract/test_notification_test.go (MUST FAIL initially)
- [ ] T046 [US3] Write integration test for notification settings workflow in frontend/tests/integration/notification-settings.test.tsx (MUST FAIL initially)
- [ ] T047 [US3] Write E2E test for configuring notification preferences in frontend/tests/e2e/notification-config.spec.ts (MUST FAIL initially)

### Implementation for User Story 3

- [ ] T048 [P] [US3] Create GET /api/v1/users/notification-preferences endpoint in backend/user-service/api/handlers/notification_preferences_handler.go (verify T043 passes)
- [ ] T049 [P] [US3] Create PATCH /api/v1/users/:user_id/notification-preferences endpoint in backend/user-service/api/handlers/notification_preferences_handler.go (verify T044 passes)
- [ ] T050 [P] [US3] Create POST /api/v1/notifications/test endpoint in backend/notification-service/api/handlers/test_notification_handler.go (verify T045 passes)
- [ ] T051 [P] [US3] Create GET /api/v1/notifications/config endpoint in backend/notification-service/api/handlers/notification_config_handler.go
- [ ] T052 [P] [US3] Create PATCH /api/v1/notifications/config endpoint in backend/notification-service/api/handlers/notification_config_handler.go
- [ ] T053 [US3] Implement updateUserNotificationPreference() service method in backend/user-service/src/services/user_service.go
- [ ] T054 [US3] Implement sendTestNotification() with sample order data in backend/notification-service/src/services/notification_service.go
- [ ] T055 [US3] Implement getNotificationConfig() repository method in backend/notification-service/src/repositories/notification_config_repository.go
- [ ] T056 [US3] Implement updateNotificationConfig() repository method in backend/notification-service/src/repositories/notification_config_repository.go
- [ ] T057 [US3] Add rate limiting to test notification endpoint in backend/notification-service/api/middleware/rate_limit.go
- [ ] T058 [P] [US3] Create NotificationSettings React component in frontend/src/components/admin/NotificationSettings.tsx
- [ ] T059 [P] [US3] Create notification settings page at frontend/src/pages/admin/settings/notifications.tsx
- [ ] T060 [US3] Implement API calls to user-service notification-preferences endpoints in frontend/src/services/user-api.ts
- [ ] T061 [US3] Implement API calls to notification-service config endpoints in frontend/src/services/notification-api.ts
- [ ] T062 [US3] Add "Send Test Email" button with confirmation modal in frontend/src/components/admin/NotificationSettings.tsx (verify T046 passes)
- [ ] T063 [US3] Add staff list with toggle switches for notification preferences in frontend/src/components/admin/NotificationSettings.tsx (verify T047 passes)

**Checkpoint**: Notification preferences working - admins can configure staff notifications through dashboard

---

## Phase 6: User Story 4 - View Email Notification History (Priority: P3)

**Goal**: Tenant administrators view log of all email notifications with delivery status and ability to resend failed ones

**Independent Test**: Navigate to notification history in admin dashboard, view list of sent notifications with status, filter by order reference, resend a failed notification

### Tests for User Story 4 (TDD: Write tests FIRST)

- [ ] T064 [P] [US4] Write contract test for GET /notifications/history in backend/notification-service/tests/contract/notification_history_test.go (MUST FAIL initially)
- [ ] T065 [P] [US4] Write contract test for POST /notifications/:id/resend in backend/notification-service/tests/contract/resend_notification_test.go (MUST FAIL initially)
- [ ] T066 [US4] Write integration test for notification history workflow in frontend/tests/integration/notification-history.test.tsx (MUST FAIL initially)
- [ ] T067 [US4] Write E2E test for viewing and resending notifications in frontend/tests/e2e/notification-history.spec.ts (MUST FAIL initially)

### Implementation for User Story 4

- [ ] T068 [P] [US4] Create GET /api/v1/notifications/history endpoint with pagination in backend/notification-service/api/handlers/notification_history_handler.go (verify T064 passes)
- [ ] T069 [P] [US4] Create POST /api/v1/notifications/:notification_id/resend endpoint in backend/notification-service/api/handlers/resend_notification_handler.go (verify T065 passes)
- [ ] T070 [US4] Implement getNotificationHistory() with filtering in backend/notification-service/src/services/notification_service.go
- [ ] T071 [US4] Implement resendNotification() with retry count check in backend/notification-service/src/services/notification_service.go
- [ ] T072 [US4] Add query builder for notification history filters in backend/notification-service/src/repositories/notification_repository.go
- [ ] T073 [P] [US4] Create NotificationHistory React component in frontend/src/components/admin/NotificationHistory.tsx
- [ ] T074 [P] [US4] Create notification history page at frontend/src/pages/admin/notifications/history.tsx
- [ ] T075 [US4] Implement API calls to notification history endpoints in frontend/src/services/notification-api.ts
- [ ] T076 [US4] Add filter controls (order reference, status, date range) in frontend/src/components/admin/NotificationHistory.tsx (verify T066 passes)
- [ ] T077 [US4] Add pagination controls for notification history in frontend/src/components/admin/NotificationHistory.tsx
- [ ] T078 [US4] Add "Resend" button for failed notifications in frontend/src/components/admin/NotificationHistory.tsx (verify T067 passes)
- [ ] T079 [US4] Add status badges (sent, pending, failed) in frontend/src/components/admin/NotificationHistory.tsx

**Checkpoint**: Notification history working - admins can view audit log and resend failed notifications

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T080 [P] Add comprehensive error handling for SMTP failures in backend/notification-service/src/providers/providers.go
- [ ] T081 [P] Add monitoring metrics for notification delivery success/failure rates in backend/notification-service/src/services/notification_service.go
- [ ] T082 [P] Add logging for duplicate notification attempts with transaction_id in backend/notification-service/src/services/notification_service.go
- [ ] T083 [P] Update API documentation with notification endpoints in docs/API.md
- [ ] T084 [P] Update backend conventions documentation in docs/BACKEND_CONVENTIONS.md
- [ ] T085 [P] Update frontend conventions documentation in docs/FRONTEND_CONVENTIONS.md
- [ ] T086 [P] Create feature documentation in docs/ORDER_EMAIL_NOTIFICATIONS.md
- [ ] T087 Add i18n translations for notification settings UI in frontend/public/locales/
- [ ] T088 Code review and refactoring for notification-service changes
- [ ] T089 Code review and refactoring for user-service changes
- [ ] T090 Code review and refactoring for order-service changes
- [ ] T091 Code review and refactoring for frontend changes
- [ ] T092 Run quickstart.md validation to verify all implementation steps
- [ ] T093 Performance testing for email delivery under load (1000 orders/hour)
- [ ] T094 Security audit for notification endpoints (auth, authorization, rate limiting)
- [ ] T095 Update CHANGELOG.md with feature summary
- [ ] T096 Create deployment checklist for production rollout

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User Story 1 (P1): Can start after Foundational - No dependencies on other stories
  - User Story 2 (P1): Can start after Foundational - No dependencies on other stories (can run parallel with US1)
  - User Story 3 (P2): Can start after Foundational - No dependencies on other stories (can run parallel with US1/US2)
  - User Story 4 (P3): Can start after Foundational - No dependencies on other stories (can run parallel with US1/US2/US3)
- **Polish (Phase 7)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - Reuses invoice template but independently testable
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Manages preferences for US1/US2 but stories work without it
- **User Story 4 (P3)**: Can start after Foundational (Phase 2) - Views history of US1/US2 notifications but doesn't block them

### Within Each User Story

- Tests MUST be written first following TDD (Red → Green → Refactor cycle)
- Verify tests fail before implementing
- Models/structs before services
- Services before handlers
- Backend before frontend
- Core implementation before integration
- Verify tests pass after implementation
- Story complete before moving to next priority

### Parallel Opportunities

- **Phase 1 (Setup)**: All migration files (T001-T006) can be created in parallel, then T007 applies them, T008-T009 parallel
- **Phase 2 (Foundational)**: T010-T014 parallel (different services), T015-T018 parallel after T010-T014
- **Phase 3 (US1)**: T019-T021 parallel (write tests first), T022-T023 parallel (models/templates), T024-T030 sequential (implementation verifying tests)
- **Phase 4 (US2)**: T031-T033 parallel (write tests first), T034-T037 parallel (models/templates), T038-T042 sequential (implementation verifying tests)
- **Phase 5 (US3)**: T043-T047 parallel (write tests first), T048-T052 parallel (API endpoints), T058-T059 parallel (frontend components), T053-T063 sequential (implementation verifying tests)
- **Phase 6 (US4)**: T064-T067 parallel (write tests first), T068-T069 parallel (API endpoints), T073-T074 parallel (frontend components), T070-T079 sequential (implementation verifying tests)
- **Phase 7 (Polish)**: T080-T086 parallel (documentation), T088-T091 parallel (code reviews)
- **Cross-story parallelism**: Once Phase 2 completes, different team members can work on US1, US2, US3, US4 simultaneously

---

## Parallel Example: User Story 1

```bash
# Phase 1: Write all tests in parallel (TDD Red phase):
Task T019: "Write unit test for renderStaffNotificationTemplate()" (MUST FAIL)
Task T020: "Write unit test for handleOrderPaidEvent()" (MUST FAIL)
Task T021: "Write integration test for end-to-end staff notification flow" (MUST FAIL)

# Phase 2: Implement models/templates in parallel (TDD Green phase):
Task T022: "Create staff notification HTML template"
Task T023: "Create StaffNotificationData struct"

# Phase 3: Implement services and verify tests pass:
Task T024: "Implement renderStaffNotificationTemplate()" → Verify T019 passes
Task T025: "Implement handleOrderPaidEvent()" → Verify T020 passes
Task T029: "Update order-service to publish event"
Task T030: "Test email rendering in Email on Acid" → Verify T021 passes
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only)

1. Complete Phase 1: Setup (database migrations)
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (staff notifications)
4. Complete Phase 4: User Story 2 (customer receipts)
5. **STOP and VALIDATE**: Test both stories independently
6. Deploy/demo if ready

**MVP Scope**: Staff receive order notifications + customers receive email receipts (both P1 priorities)

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (staff notifications)
3. Add User Story 2 → Test independently → Deploy/Demo (customer receipts)
4. Add User Story 3 → Test independently → Deploy/Demo (notification preferences)
5. Add User Story 4 → Test independently → Deploy/Demo (notification history)
6. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (staff notifications)
   - Developer B: User Story 2 (customer receipts)
   - Developer C: User Story 3 (notification preferences)
   - Developer D: User Story 4 (notification history)
3. Stories complete and integrate independently

---

## Task Summary

- **Total Tasks**: 96
- **Setup Phase**: 9 tasks
- **Foundational Phase**: 9 tasks (BLOCKING)
- **User Story 1 (P1)**: 12 tasks (3 tests + 9 implementation)
- **User Story 2 (P1)**: 12 tasks (3 tests + 9 implementation)
- **User Story 3 (P2)**: 21 tasks (5 tests + 16 implementation)
- **User Story 4 (P3)**: 16 tasks (4 tests + 12 implementation)
- **Polish Phase**: 17 tasks (cross-cutting concerns)
- **Total Test Tasks**: 15 (all written FIRST per TDD)

**Parallel Opportunities**: 42 tasks marked [P] can run in parallel within their phase

**MVP Recommendation**: Complete Phases 1-4 (Setup + Foundational + US1 + US2) = 42 tasks for core notification functionality

---

## Notes

- [P] tasks = different files, no dependencies within phase
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Tests written FIRST following TDD (Constitution Principle III: NON-NEGOTIABLE)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Backend uses Go 1.21+ with Echo v4 framework
- Frontend uses Next.js + TypeScript + React
- Database: PostgreSQL with 3 new migrations
- Messaging: Kafka for event-driven communication
- Email: SMTP provider in existing notification-service
- All file paths are relative to repository root
