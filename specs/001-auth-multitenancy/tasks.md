# Tasks: User Authentication and Multi-Tenancy

**Input**: Design documents from `/specs/002-auth-multitenancy/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/
**Feature Branch**: `001-auth-multitenancy`
**Date**: 2025-11-23

**Tests**: All unit tests are included per specification requirements (backend 80% coverage, frontend 70% coverage, critical paths 95% coverage).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `- [ ] [ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4, US5)
- Include exact file paths in descriptions

## Path Conventions

This project uses web application architecture:
- **Backend**: `backend/` with Go services
- **Frontend**: `frontend/` with Next.js + Tailwind CSS
- **Shared**: `docker-compose.yml`, `.env.example` at repository root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, test frameworks first (Test-First Development - Constitution Principle III)

**⚠️ TEST-FIRST ORDER**: Test frameworks MUST be set up before implementation code per constitution

### Test Framework Setup (Must Complete First)

- [x] T001 Setup Go test framework with testify in backend/go.mod (add github.com/stretchr/testify dependency)
- [x] T002 Setup Jest + React Testing Library in frontend/package.json (test dependencies)
- [x] T003 Configure Jest for frontend in frontend/jest.config.js (with jsdom, tsx support)
- [x] T004 Create Jest setup file in frontend/jest.setup.js (import @testing-library/jest-dom)
- [x] T005 Configure Go test coverage reporting in .github/workflows/test.yml or Makefile
- [x] T006 Verify test frameworks work: Create sample test in backend/tests/sample_test.go and frontend/tests/sample.test.ts

### Project Structure (After Test Frameworks Ready)

- [x] T007 Create backend directory structure per plan.md (src/models, src/services, src/api, src/database, src/utils, tests/)
- [x] T008 Create frontend directory structure per plan.md (src/components, src/pages, src/services, src/hooks, src/contexts, src/types, src/utils, src/styles, tests/)
- [x] T009 Initialize backend Go modules in backend/auth-service/go.mod
- [x] T010 Initialize backend Go modules in backend/user-service/go.mod
- [x] T011 Initialize backend Go modules in backend/tenant-service/go.mod
- [x] T012 Initialize API Gateway Go module in api-gateway/go.mod
- [x] T013 Initialize frontend Next.js project with TypeScript in frontend/package.json
- [x] T014 [P] Configure ESLint and Prettier for frontend in frontend/.eslintrc.js and frontend/.prettierrc
- [x] T015 [P] Configure Go linting with golangci-lint in .golangci.yml
- [x] T016 Setup Docker Compose for local development in docker-compose.yml (PostgreSQL, Redis)
- [x] T017 Create environment variable templates in .env.example (backend, frontend, gateway)
- [x] T018 Setup i18n with react-i18next in frontend/src/i18n/config.ts
- [x] T019 Create translation files in frontend/public/locales/en/common.json
- [x] T020 Create translation files in frontend/public/locales/id/common.json
- [x] T021 Configure Git hooks for pre-commit linting in .husky/pre-commit
- [x] T022 Setup database migrations framework with golang-migrate in backend/migrations/
- [x] T023 Create Swagger/OpenAPI documentation generator config in backend/config/swagger.config.go

**Checkpoint**: Test frameworks verified and project structure ready

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Tailwind CSS Setup (MUST Complete Before UI Components)

- [x] T024 Install Tailwind CSS dependencies in frontend/ (tailwindcss, postcss, autoprefixer)
- [x] T025 Setup Tailwind CSS with PostCSS in frontend/postcss.config.js
- [x] T026 Configure Tailwind design system in frontend/tailwind.config.js (colors, spacing, breakpoints, content paths)
- [x] T027 Create global styles with Tailwind directives in frontend/src/styles/globals.css (@tailwind base, components, utilities)
- [x] T028 Import globals.css in frontend/pages/_app.js
- [x] T029 Verify Tailwind build works: Run `npm run dev` and check styling applies

### Database & Backend Infrastructure

- [x] T030 Create PostgreSQL database schema migration 001_create_tenants in backend/migrations/000001_create_tenants.up.sql
- [x] T031 Create PostgreSQL database schema migration 002_create_users in backend/migrations/000002_create_users.up.sql
- [x] T032 Create PostgreSQL database schema migration 003_create_sessions in backend/migrations/000003_create_sessions.up.sql
- [x] T033 Create PostgreSQL database schema migration 004_create_invitations in backend/migrations/000004_create_invitations.up.sql
- [x] T034 Create PostgreSQL database schema migration 005_create_password_reset_tokens in backend/migrations/000005_create_password_reset_tokens.up.sql
- [x] T035 [P] Setup database connection pool with row-level security in backend/src/database/connection.go
- [x] T036 [P] Setup Redis client connection in backend/src/database/redis.go

### Notification Service & Event Infrastructure

- [x] T298 Create PostgreSQL database schema migration 006_create_notifications in backend/migrations/000006_create_notifications.up.sql
- [x] T299 [P] Setup Kafka client and event publisher in backend/src/queue/kafka_publisher.go
- [x] T300 [P] Implement event publisher helper methods (PublishUserRegistered, PublishUserLogin, PublishPasswordResetRequested, PublishPasswordChanged) in backend/src/queue/event_helpers.go
- [x] T301 [P] Configure notification service Kafka consumer in backend/notification-service/src/queue/kafka.go (verify existing implementation)
- [x] T302 [P] Create email template for user registration/welcome in backend/notification-service/templates/registration-email.html
- [x] T303 [P] Create email template for login alert in backend/notification-service/templates/login-alert-email.html
- [x] T304 [P] Create email template for password reset in backend/notification-service/templates/password-reset-email.html (relocated from T274)
- [x] T305 [P] Create email template for password changed confirmation in backend/notification-service/templates/password-changed-email.html
- [x] T306 [P] Update notification service with all template handlers in backend/notification-service/src/services/notification_service.go
- [x] T307 [P] Add Docker Compose configuration for Kafka and Zookeeper in docker-compose.yml
- [ ] T308 [P] Unit test for Kafka event publisher in backend/tests/unit/queue/kafka_publisher_test.go
- [ ] T309 [P] Integration test for notification service event consumption in backend/tests/integration/notification_service_test.go

### Backend Utilities

- [x] T037 [P] Implement JWT token utilities (generate, validate, decode) in backend/src/utils/token.utils.go
- [x] T038 [P] Implement bcrypt password utilities (hash, compare) in backend/src/utils/password.utils.go
- [x] T039 [P] Implement validation utilities (email, password, business name) in backend/src/utils/validation.utils.go
- [x] T040 [P] Implement structured logging utilities in backend/src/utils/logger.utils.go

### API Gateway Middleware

- [x] T041 [P] Create API Gateway JWT validation middleware in api-gateway/src/middleware/auth.middleware.go
- [x] T042 [P] Create API Gateway tenant context injection middleware in api-gateway/src/middleware/tenant.middleware.go
- [x] T043 [P] Create API Gateway rate limiting middleware with Redis in api-gateway/src/middleware/rate-limit.middleware.go
- [x] T044 [P] Create API Gateway error handling middleware in api-gateway/src/middleware/error.middleware.go
- [x] T045 [P] Create API Gateway CORS middleware in api-gateway/src/middleware/cors.middleware.go
- [x] T046 [P] Create API Gateway security headers middleware in api-gateway/src/middleware/security.middleware.go
- [x] T047 [P] Setup API Gateway service routing configuration in api-gateway/src/config/routes.go

### Frontend Infrastructure (Depends on T024-T029 Tailwind Setup)

- [x] T048 [P] Implement frontend API client with Axios and interceptors in frontend/src/services/api.service.ts
- [x] T049 [P] Create authentication context provider in frontend/src/contexts/AuthContext.tsx
- [x] T050 [P] Create toast notification context provider in frontend/src/contexts/ToastContext.tsx
- [x] T051 [P] Create useAuth custom hook in frontend/src/hooks/useAuth.ts
- [x] T052 [P] Create useForm custom hook in frontend/src/hooks/useForm.ts
- [x] T053 [P] Create useToast custom hook in frontend/src/hooks/useToast.ts
- [x] T054 [P] Implement form validation utilities in frontend/src/utils/validators.ts
- [x] T055 [P] Implement data formatting utilities in frontend/src/utils/formatters.ts
- [x] T056 [P] Implement local storage utilities in frontend/src/utils/storage.ts
- [x] T057 [P] Create TypeScript types for authentication in frontend/src/types/auth.types.ts
- [x] T058 [P] Create TypeScript types for users in frontend/src/types/user.types.ts
- [x] T059 [P] Create TypeScript types for API responses in frontend/src/types/api.types.ts

**Checkpoint**: Foundation ready (including Tailwind CSS verified) - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Business Owner Account Creation (Priority: P1) 🎯 MVP

**Goal**: Enable business owner to register and create isolated tenant account with owner as first user

**Independent Test**: Complete signup process and verify new tenant with owner user is created and isolated from other tenants

### UI Component Library (Required for US1)

- [x] T049 [P] [US1] Create Button component with Tailwind styling in frontend/src/components/ui/Button.tsx
- [x] T050 [P] [US1] Create Input component with Tailwind styling in frontend/src/components/ui/Input.tsx
- [x] T051 [P] [US1] Create Card component with Tailwind styling in frontend/src/components/ui/Card.tsx
- [x] T052 [P] [US1] Create LoadingSpinner component with Tailwind styling in frontend/src/components/ui/LoadingSpinner.tsx
- [x] T053 [P] [US1] Create Toast component with Tailwind styling in frontend/src/components/ui/Toast.tsx
- [x] T054 [P] [US1] Create FormField component with validation display in frontend/src/components/forms/FormField.tsx
- [x] T055 [P] [US1] Create AuthLayout component with responsive design in frontend/src/components/layout/AuthLayout.tsx
- [x] T056 [P] [US1] Unit test for Button component in frontend/tests/unit/components/ui/Button.test.tsx
- [x] T057 [P] [US1] Unit test for Input component in frontend/tests/unit/components/ui/Input.test.tsx
- [x] T058 [P] [US1] Unit test for FormField component in frontend/tests/unit/components/forms/FormField.test.tsx

### Backend Unit Tests for US1

- [x] T059 [P] [US1] Unit test for password hashing utility in backend/tests/unit/utils/password_test.go
- [ ] T060 [P] [US1] Unit test for email validation utility in backend/tests/unit/utils/validation_test.go
- [ ] T061 [P] [US1] Unit test for JWT token generation utility in backend/tests/unit/utils/token_test.go

### Backend Implementation for US1

- [x] T062 [P] [US1] Create Tenant model in backend/src/models/Tenant.go
- [x] T063 [P] [US1] Create User model in backend/src/models/User.go
- [x] T064 [US1] Implement TenantService.Register in backend/tenant-service/src/services/TenantService.go
- [x] T065 [US1] Implement UserService.CreateOwner in backend/user-service/src/services/UserService.go
- [x] T066 [US1] Implement POST /api/tenants/register handler in backend/tenant-service/src/api/handlers.go
- [x] T310 [US1] **Publish user.registered event to Kafka after successful registration** in backend/tenant-service/src/api/handlers.go (after T066)
- [ ] T067 [US1] Unit test for TenantService.Register in backend/tests/unit/services/TenantService_test.go
- [ ] T068 [US1] Unit test for UserService.CreateOwner in backend/tests/unit/services/UserService_test.go
- [x] T069 [US1] Add tenant registration route to API Gateway in api-gateway/src/routes.go
- [x] T070 [US1] Add structured logging for registration events in backend/tenant-service/src/api/handlers.go

### Frontend Implementation for US1

- [x] T071 [US1] Create registration form component in frontend/src/components/forms/RegistrationForm.tsx
- [x] T072 [US1] Create registration page in frontend/src/pages/auth/register.tsx
- [x] T073 [US1] Implement tenant service API calls in frontend/src/services/tenant.service.ts
- [x] T074 [US1] Add registration form translations in frontend/public/locales/en/auth.json
- [x] T075 [US1] Add registration form translations in frontend/public/locales/id/auth.json
- [ ] T076 [US1] Unit test for RegistrationForm component in frontend/tests/unit/components/forms/RegistrationForm.test.tsx
- [ ] T077 [US1] Unit test for tenant service in frontend/tests/unit/services/tenant.service.test.ts

### Integration & Accessibility Tests for US1

- [ ] T078 [US1] Integration test for tenant registration flow in backend/tests/integration/tenant_registration_test.go
- [ ] T079 [US1] Contract test for POST /api/tenants/register in backend/tests/contract/tenant_contract_test.go
- [ ] T080 [US1] Accessibility test for registration form (WCAG 2.1 AA) in frontend/tests/unit/components/forms/RegistrationForm.a11y.test.tsx
- [ ] T081 [US1] E2E test for complete registration flow in frontend/tests/e2e/registration.e2e.ts

**Checkpoint**: User Story 1 is fully functional - business owners can register and create isolated tenant accounts

---

## Phase 4: User Story 2 - User Login to Tenant Account (Priority: P1)

**Goal**: Enable authorized users to log in and access tenant-scoped data with secure session management

**Independent Test**: User with valid credentials logs in and verifies access only to their tenant's resources

### Backend Unit Tests for US2

- [ ] T082 [P] [US2] Unit test for SessionService.Create in backend/tests/unit/services/SessionService_test.go
- [ ] T083 [P] [US2] Unit test for SessionService.Validate in backend/tests/unit/services/SessionService_test.go
- [ ] T084 [P] [US2] Unit test for AuthService.Login in backend/tests/unit/services/AuthService_test.go

### Backend Implementation for US2

- [x] T085 [P] [US2] Create Session model in backend/src/models/Session.go
- [x] T086 [US2] Implement SessionService.Create in backend/auth-service/src/services/SessionService.go
- [x] T087 [US2] Implement SessionService.Validate in backend/auth-service/src/services/SessionService.go
- [x] T088 [US2] Implement SessionService.Renew in backend/auth-service/src/services/SessionService.go
- [x] T089 [US2] Implement AuthService.Login in backend/auth-service/src/services/AuthService.go
- [x] T090 [US2] Implement POST /api/auth/login handler in backend/auth-service/src/api/handlers.go
- [ ] T361 [US2] **Add email verification check to login handler** - Return 403 with EMAIL_NOT_VERIFIED code and user info if not verified in backend/auth-service/src/api/handlers.go (after T090)
- [x] T311 [US2] **Publish user.login event to Kafka after successful login** in backend/auth-service/src/api/handlers.go (after T090, includes IP and user agent)
- [x] T091 [US2] Implement GET /api/auth/session handler in backend/auth-service/src/api/handlers.go
- [x] T092 [US2] Add login route with rate limiting to API Gateway in api-gateway/src/routes.go
- [x] T093 [US2] Add security event logging for login attempts in backend/auth-service/src/api/handlers.go

### Frontend Implementation for US2

- [x] T094 [US2] Create login form component in frontend/src/components/forms/LoginForm.tsx
- [x] T095 [US2] Create login page in frontend/src/pages/auth/login.tsx
- [ ] T362 [US2] **Add unverified user redirect logic to login page** - Redirect to /auth/verification-required if EMAIL_NOT_VERIFIED error received in frontend/src/pages/auth/login.tsx (after T095)
- [ ] T363 [US2] **Create verification required page** - Show message and resend button for unverified users trying to login in frontend/src/pages/auth/verification-required.tsx
- [x] T096 [US2] Implement auth service API calls in frontend/src/services/auth.service.ts
- [x] T097 [US2] Add login form translations in frontend/public/locales/en/auth.json
- [x] T098 [US2] Add login form translations in frontend/public/locales/id/auth.json
- [ ] T099 [US2] Unit test for LoginForm component in frontend/tests/unit/components/forms/LoginForm.test.tsx
- [ ] T100 [US2] Unit test for auth service in frontend/tests/unit/services/auth.service.test.ts
- [ ] T101 [US2] Unit test for useAuth hook in frontend/tests/unit/hooks/useAuth.test.ts

### Integration & Security Tests for US2

- [ ] T102 [US2] Integration test for login with valid credentials in backend/tests/integration/auth_login_test.go
- [ ] T103 [US2] Integration test for login with invalid credentials in backend/tests/integration/auth_login_test.go
- [ ] T104 [US2] Integration test for rate limiting on login in backend/tests/integration/rate_limit_test.go
- [ ] T105 [US2] Integration test for tenant isolation verification in backend/tests/integration/tenant_isolation_test.go
- [ ] T106 [US2] Contract test for POST /api/auth/login in backend/tests/contract/auth_contract_test.go
- [ ] T107 [US2] Accessibility test for login form (WCAG 2.1 AA) in frontend/tests/unit/components/forms/LoginForm.a11y.test.tsx
- [ ] T108 [US2] E2E test for complete login flow in frontend/tests/e2e/login.e2e.ts

**Checkpoint**: User Story 2 is fully functional - users can securely log in with tenant-scoped access

---

## Phase 4.5: Password Reset Flow (FR-017 - Critical MVP Feature)

**Goal**: Implement secure password reset functionality scoped to user's tenant

**Independent Test**: User requests password reset, receives email with token, successfully resets password with new credentials

### Backend Implementation for Password Reset

- [x] T268 Create PasswordResetToken model in backend/src/models/PasswordResetToken.go (token, user_id, tenant_id, expires_at, used)
- [x] T269 Implement PasswordResetService.RequestReset in backend/auth-service/src/services/PasswordResetService.go (generate token, check tenant)
- [x] T270 Implement PasswordResetService.ValidateToken in backend/auth-service/src/services/PasswordResetService.go (check expiry, tenant match)
- [x] T271 Implement PasswordResetService.ResetPassword in backend/auth-service/src/services/PasswordResetService.go (validate token, update password, invalidate token)
- [x] T272 Implement POST /api/auth/password-reset/request handler in backend/auth-service/src/api/handlers.go (email validation, tenant scoping)
- [x] T312 **Publish password.reset_requested event to Kafka after generating reset token** in backend/auth-service/src/api/handlers.go (after T272)
- [x] T273 Implement POST /api/auth/password-reset/reset handler in backend/auth-service/src/api/handlers.go (token validation, password update)
- [x] T313 **Publish password.changed event to Kafka after successful password reset** in backend/auth-service/src/api/handlers.go (after T273)
- [ ] T274 Add password reset email template in backend/notification-service/templates/password-reset-email.html (NOTE: Already covered by T304 in Phase 2)
- [ ] T275 Unit test for PasswordResetService.RequestReset in backend/tests/unit/services/PasswordResetService_test.go
- [ ] T276 Unit test for PasswordResetService.ValidateToken in backend/tests/unit/services/PasswordResetService_test.go
- [ ] T277 Unit test for PasswordResetService.ResetPassword in backend/tests/unit/services/PasswordResetService_test.go
- [ ] T278 Integration test for password reset flow in backend/tests/integration/password_reset_test.go (request → validate → reset)

### Frontend Implementation for Password Reset

- [x] T279 [P] Create PasswordResetRequestForm component in frontend/src/components/forms/PasswordResetRequestForm.tsx (email input with Tailwind styling)
- [x] T280 [P] Create PasswordResetForm component in frontend/src/components/forms/PasswordResetForm.tsx (new password, confirm password with Tailwind styling)
- [x] T281 [P] Create password reset request page in frontend/src/pages/auth/forgot-password.tsx
- [x] T282 [P] Create password reset page in frontend/src/pages/auth/reset-password.tsx (accepts token from URL)
- [x] T283 [P] Implement password reset API calls in frontend/src/services/auth.service.ts (requestReset, resetPassword methods)
- [x] T284 [P] Add password reset translations in frontend/public/locales/en/auth.json
- [x] T285 [P] Add password reset translations in frontend/public/locales/id/auth.json
- [ ] T286 [P] Unit test for PasswordResetRequestForm in frontend/tests/unit/components/forms/PasswordResetRequestForm.test.tsx
- [ ] T287 [P] Unit test for PasswordResetForm in frontend/tests/unit/components/forms/PasswordResetForm.test.tsx

### Security & Edge Cases

- [ ] T288 Add rate limiting for password reset requests (3 requests per hour per email) in backend/auth-service/src/middleware/ratelimit.go
- [ ] T289 Implement token expiration cleanup job (delete expired tokens older than 24 hours) in backend/auth-service/src/jobs/cleanup.go
- [ ] T290 Integration test for password reset with expired token in backend/tests/integration/password_reset_edge_cases_test.go
- [ ] T291 Integration test for password reset rate limiting in backend/tests/integration/password_reset_rate_limit_test.go
- [ ] T292 E2E test for complete password reset flow in frontend/tests/e2e/password-reset.spec.ts

### Documentation

- [ ] T293 Add password reset flow to API documentation in backend/config/swagger.config.go
- [ ] T294 Add password reset user guide to documentation in docs/user-guide/password-reset.md

**Checkpoint**: Password reset flow fully functional - users can securely reset forgotten passwords with tenant-scoped validation

---

## Phase 5: User Story 5 - Session Management and Logout (Priority: P2)

**Goal**: Enable users to securely end sessions and implement automatic session expiration

**Independent Test**: User logs in, logs out, and verifies session is terminated and requires re-authentication

### Additional UI Components for US5

- [x] T109 [P] [US5] Create Modal component with Tailwind styling in frontend/src/components/ui/Modal.tsx
- [x] T110 [P] [US5] Create ErrorBoundary component in frontend/src/components/ui/ErrorBoundary.tsx
- [x] T111 [P] [US5] Create DashboardLayout component with navigation in frontend/src/components/layout/DashboardLayout.tsx
- [x] T112 [P] [US5] Create ResponsiveNav component with mobile menu in frontend/src/components/layout/ResponsiveNav.tsx
- [ ] T113 [P] [US5] Unit test for Modal component in frontend/tests/unit/components/ui/Modal.test.tsx
- [ ] T114 [P] [US5] Unit test for DashboardLayout component in frontend/tests/unit/components/layout/DashboardLayout.test.tsx

### Backend Unit Tests for US5

- [ ] T115 [P] [US5] Unit test for SessionService.Terminate in backend/tests/unit/services/SessionService_test.go
- [ ] T116 [P] [US5] Unit test for SessionService.Expire in backend/tests/unit/services/SessionService_test.go

### Backend Implementation for US5

- [x] T117 [US5] Implement SessionService.Terminate in backend/auth-service/src/services/SessionService.go
- [x] T118 [US5] Implement SessionService.ExpireInactive in backend/auth-service/src/services/SessionService.go
- [x] T119 [US5] Implement POST /api/auth/logout handler in backend/auth-service/src/api/handlers.go
- [x] T120 [US5] Add logout route to API Gateway in api-gateway/src/routes.go
- [x] T121 [US5] Add session termination logging in backend/auth-service/src/api/handlers.go

### Frontend Implementation for US5

- [x] T122 [US5] Implement logout functionality in auth service in frontend/src/services/auth.service.ts
- [x] T123 [US5] Create dashboard page with logout button in frontend/src/pages/dashboard/index.tsx
- [x] T124 [US5] Add logout translations in frontend/public/locales/en/common.json
- [x] T125 [US5] Add logout translations in frontend/public/locales/id/common.json
- [ ] T126 [US5] Unit test for logout flow in frontend/tests/unit/services/auth.service.test.ts

### Integration Tests for US5

- [ ] T127 [US5] Integration test for logout flow in backend/tests/integration/auth_logout_test.go
- [ ] T128 [US5] Integration test for session expiration in backend/tests/integration/session_expiration_test.go
- [ ] T129 [US5] Contract test for POST /api/auth/logout in backend/tests/contract/auth_contract_test.go
- [ ] T130 [US5] E2E test for logout and session expiration in frontend/tests/e2e/session-management.e2e.ts

**Checkpoint**: User Story 5 is fully functional - secure session management with logout and expiration

---

## Phase 6: User Story 3 - Business Owner Adds Team Members (Priority: P2)

**Goal**: Enable business owners to invite team members who can access tenant-scoped resources

**Independent Test**: Owner sends invitation, new user accepts and logs in with access only to their tenant

### Backend Unit Tests for US3

- [ ] T131 [P] [US3] Unit test for InvitationService.Create in backend/tests/unit/services/InvitationService_test.go
- [ ] T132 [P] [US3] Unit test for InvitationService.Accept in backend/tests/unit/services/InvitationService_test.go
- [ ] T133 [P] [US3] Unit test for UserService.CreateFromInvitation in backend/tests/unit/services/UserService_test.go

### Backend Implementation for US3

- [x] T134 [P] [US3] Create Invitation model in backend/src/models/Invitation.go
- [x] T135 [US3] Implement InvitationService.Create in backend/user-service/src/services/InvitationService.go
- [x] T136 [US3] Implement InvitationService.Accept in backend/user-service/src/services/InvitationService.go
- [x] T137 [US3] Implement InvitationService.List in backend/user-service/src/services/InvitationService.go
- [x] T138 [US3] Implement UserService.CreateFromInvitation in backend/user-service/src/services/UserService.go
- [x] T139 [US3] Implement POST /api/invitations handler in backend/user-service/src/api/handlers.go
- [x] T140 [US3] Implement GET /api/invitations handler in backend/user-service/src/api/handlers.go
- [x] T141 [US3] Implement POST /api/invitations/:token/accept handler in backend/user-service/src/api/handlers.go
- [x] T142 [US3] Add invitation routes to API Gateway in api-gateway/src/routes.go
- [x] T143 [US3] Add invitation event logging in backend/user-service/src/api/handlers.go

### Frontend Implementation for US3

- [x] T144 [US3] Create invitation form component in frontend/src/components/forms/InvitationForm.tsx
- [x] T145 [US3] Create user invite page in frontend/src/pages/users/invite.tsx
- [x] T146 [US3] Create accept invitation page in frontend/src/pages/auth/accept-invitation.tsx
- [x] T147 [US3] Implement user service API calls in frontend/src/services/user.service.ts
- [x] T148 [US3] Add invitation translations in frontend/public/locales/en/users.json
- [x] T149 [US3] Add invitation translations in frontend/public/locales/id/users.json
- [ ] T150 [US3] Unit test for InvitationForm component in frontend/tests/unit/components/forms/InvitationForm.test.tsx
- [ ] T151 [US3] Unit test for user service in frontend/tests/unit/services/user.service.test.ts

### Integration Tests for US3

- [ ] T152 [US3] Integration test for invitation creation in backend/tests/integration/invitation_create_test.go
- [ ] T153 [US3] Integration test for invitation acceptance in backend/tests/integration/invitation_accept_test.go
- [ ] T154 [US3] Integration test for team member tenant isolation in backend/tests/integration/tenant_isolation_test.go
- [ ] T155 [US3] Contract test for POST /api/invitations in backend/tests/contract/user_contract_test.go
- [ ] T156 [US3] Contract test for POST /api/invitations/:token/accept in backend/tests/contract/user_contract_test.go
- [ ] T157 [US3] Accessibility test for invitation form (WCAG 2.1 AA) in frontend/tests/unit/components/forms/InvitationForm.a11y.test.tsx
- [ ] T158 [US3] E2E test for complete invitation flow in frontend/tests/e2e/team-invitation.e2e.ts

**Checkpoint**: User Story 3 is fully functional - owners can invite team members with tenant isolation

---

## Phase 7: User Story 4 - User Role-Based Access (Priority: P3)

**Goal**: Implement role-based access control with owner, manager, and cashier roles

**Independent Test**: Assign different roles to users and verify appropriate access restrictions within tenant

### Backend Unit Tests for US4

- [ ] T159 [P] [US4] Unit test for UserService.UpdateRole in backend/tests/unit/services/UserService_test.go
- [ ] T160 [P] [US4] Unit test for UserService.List with role filter in backend/tests/unit/services/UserService_test.go
- [ ] T161 [P] [US4] Unit test for RBAC middleware in backend/tests/unit/middleware/rbac_test.go

### Backend Implementation for US4

- [ ] T162 [P] [US4] Create Role model in backend/src/models/Role.go
- [ ] T163 [US4] Implement UserService.UpdateRole in backend/user-service/src/services/UserService.go
- [ ] T164 [US4] Implement UserService.List in backend/user-service/src/services/UserService.go
- [ ] T165 [US4] Implement UserService.GetById in backend/user-service/src/services/UserService.go
- [ ] T166 [US4] Implement UserService.Update in backend/user-service/src/services/UserService.go
- [ ] T167 [US4] Implement UserService.Delete in backend/user-service/src/services/UserService.go
- [ ] T168 [US4] Implement GET /api/users handler in backend/user-service/src/api/handlers.go
- [ ] T169 [US4] Implement GET /api/users/:id handler in backend/user-service/src/api/handlers.go
- [ ] T170 [US4] Implement PUT /api/users/:id handler in backend/user-service/src/api/handlers.go
- [ ] T171 [US4] Implement PUT /api/users/:id/role handler in backend/user-service/src/api/handlers.go
- [ ] T172 [US4] Implement DELETE /api/users/:id handler in backend/user-service/src/api/handlers.go
- [ ] T173 [US4] Create RBAC middleware for role validation in api-gateway/src/middleware/rbac.middleware.go
- [ ] T174 [US4] Add user management routes with RBAC to API Gateway in api-gateway/src/routes.go
- [ ] T175 [US4] Add authorization event logging in backend/user-service/src/api/handlers.go

### Frontend Implementation for US4

- [ ] T176 [US4] Create user list page in frontend/src/pages/users/index.tsx
- [ ] T177 [US4] Create user management component in frontend/src/components/users/UserManagement.tsx
- [ ] T178 [US4] Create role selector component in frontend/src/components/users/RoleSelector.tsx
- [ ] T179 [US4] Add user management translations in frontend/public/locales/en/users.json
- [ ] T180 [US4] Add user management translations in frontend/public/locales/id/users.json
- [ ] T181 [US4] Unit test for UserManagement component in frontend/tests/unit/components/users/UserManagement.test.tsx
- [ ] T182 [US4] Unit test for RoleSelector component in frontend/tests/unit/components/users/RoleSelector.test.tsx

### Integration Tests for US4

- [ ] T183 [US4] Integration test for role assignment in backend/tests/integration/rbac_assignment_test.go
- [ ] T184 [US4] Integration test for RBAC enforcement in backend/tests/integration/rbac_enforcement_test.go
- [ ] T185 [US4] Integration test for user deletion with session invalidation in backend/tests/integration/user_deletion_test.go
- [ ] T186 [US4] Contract test for GET /api/users in backend/tests/contract/user_contract_test.go
- [ ] T187 [US4] Contract test for PUT /api/users/:id/role in backend/tests/contract/user_contract_test.go
- [ ] T188 [US4] Accessibility test for user management UI (WCAG 2.1 AA) in frontend/tests/unit/components/users/UserManagement.a11y.test.tsx
- [ ] T189 [US4] E2E test for role-based access control in frontend/tests/e2e/rbac.e2e.ts

**Checkpoint**: User Story 4 is fully functional - role-based access control enforced with three-tier roles

---

## Phase 8: Internationalization & Language Switcher

**Purpose**: Complete i18n implementation with language switcher component

- [x] T190 [P] Create LanguageSwitcher component with Tailwind styling in frontend/src/components/ui/LanguageSwitcher.tsx
- [x] T191 [P] Add LanguageSwitcher to DashboardLayout in frontend/src/components/layout/DashboardLayout.tsx
- [x] T192 [P] Add LanguageSwitcher to AuthLayout in frontend/src/components/layout/AuthLayout.tsx
- [ ] T193 [P] Implement user locale preference API in backend/user-service/src/api/handlers.go (PATCH /api/users/:id/locale)
- [ ] T194 [P] Add locale persistence logic in frontend/src/hooks/useAuth.ts
- [ ] T195 [P] Create backend i18n utilities in backend/src/utils/i18n.utils.go
- [ ] T196 [P] Create backend translation files in backend/locales/en.json
- [ ] T197 [P] Create backend translation files in backend/locales/id.json
- [ ] T198 [P] Add error message localization in backend services
- [ ] T199 [P] Unit test for LanguageSwitcher component in frontend/tests/unit/components/ui/LanguageSwitcher.test.tsx
- [ ] T200 [P] Integration test for locale switching in frontend/tests/integration/i18n.test.tsx
- [ ] T201 [P] E2E test for language switching flow in frontend/tests/e2e/language-switching.e2e.ts

---

## Phase 9: Responsive Design Implementation

**Purpose**: Ensure mobile-first responsive design across all components

- [ ] T202 [P] Add responsive breakpoints to all form components (mobile, tablet, desktop)
- [ ] T203 [P] Implement mobile navigation with hamburger menu in frontend/src/components/layout/ResponsiveNav.tsx
- [ ] T204 [P] Add responsive grid layouts to dashboard in frontend/src/pages/dashboard/index.tsx
- [ ] T205 [P] Add responsive layouts to user management in frontend/src/pages/users/index.tsx
- [ ] T206 [P] Implement touch-friendly button sizing (minimum 44x44px) in frontend/src/components/ui/Button.tsx
- [ ] T207 [P] Add responsive modal layouts in frontend/src/components/ui/Modal.tsx
- [ ] T208 [P] Test responsive design on mobile viewport (< 640px) in frontend/tests/integration/responsive.test.tsx
- [ ] T209 [P] Test responsive design on tablet viewport (768px-1024px) in frontend/tests/integration/responsive.test.tsx
- [ ] T210 [P] Test responsive design on desktop viewport (> 1024px) in frontend/tests/integration/responsive.test.tsx

---

## Phase 10: Accessibility Compliance (WCAG 2.1 AA)

**Purpose**: Ensure full accessibility compliance across all features

- [ ] T211 [P] Add ARIA labels to all form inputs in frontend/src/components/forms/
- [ ] T212 [P] Add ARIA live regions for toast notifications in frontend/src/components/ui/Toast.tsx
- [ ] T213 [P] Implement focus management in modals in frontend/src/components/ui/Modal.tsx
- [ ] T214 [P] Add keyboard navigation support (Tab, Enter, Escape) to all interactive elements
- [ ] T215 [P] Verify color contrast ratios (≥4.5:1) in Tailwind config in frontend/tailwind.config.js
- [ ] T216 [P] Add focus-visible indicators to all interactive elements
- [ ] T217 [P] Add semantic HTML landmarks (main, nav, aside) to layouts
- [ ] T218 [P] Implement screen reader announcements for auth state changes
- [ ] T219 [P] Run automated accessibility audit with axe-core in frontend/tests/a11y/audit.test.ts
- [ ] T220 [P] Manual accessibility testing with keyboard-only navigation

---

## Phase 11: Performance Optimization

**Purpose**: Optimize frontend and backend performance per specification requirements

### Frontend Optimization

- [ ] T221 [P] Implement lazy loading for non-critical routes in frontend/src/pages/
- [ ] T222 [P] Optimize images with Next.js Image component in frontend/src/components/
- [ ] T223 [P] Configure code splitting at route level in frontend/next.config.js
- [ ] T224 [P] Configure Tailwind CSS purging for production in frontend/tailwind.config.js
- [ ] T225 [P] Implement bundle size monitoring with webpack-bundle-analyzer
- [ ] T226 [P] Add response caching for static data in frontend/src/services/api.service.ts
- [ ] T227 [P] Performance test for First Contentful Paint (< 1.5s target)

### Backend Optimization

- [ ] T228 [P] Verify database connection pooling (max 20 connections) in backend/src/database/connection.go
- [ ] T229 [P] Add database query performance monitoring in backend/src/database/connection.go
- [ ] T230 [P] Implement request compression (gzip) in api-gateway/src/middleware/compression.middleware.go
- [ ] T231 [P] Add response caching for tenant settings in backend/tenant-service/
- [ ] T232 [P] Performance test for login response time (< 500ms p95 target)
- [ ] T233 [P] Performance test for session validation (< 100ms p95 target)
- [ ] T234 [P] Load test with 1000+ concurrent requests per second

---

## Phase 12: Security Implementation & Hardening

**Purpose**: Implement security controls per specification requirements

- [ ] T235 [P] Verify bcrypt cost factor 10 in password utilities in backend/src/utils/password.utils.go
- [ ] T236 [P] Implement HTTP-only cookie configuration in backend/auth-service/
- [ ] T237 [P] Implement Secure and SameSite cookie flags in backend/auth-service/
- [ ] T238 [P] Implement CSRF token validation in api-gateway/src/middleware/csrf.middleware.go
- [ ] T239 [P] Configure security headers with helmet equivalent in api-gateway/src/middleware/security.middleware.go
- [ ] T240 [P] Implement SQL injection prevention via parameterized queries verification
- [ ] T241 [P] Implement XSS protection with input sanitization in backend/src/utils/validation.utils.go
- [ ] T242 [P] Verify rate limiting per email+tenant combination (5 failed logins per 15 min) in api-gateway/src/middleware/rate-limit.middleware.go
- [ ] T243 [P] Security audit for tenant isolation enforcement in backend/tests/integration/security_audit_test.go
- [ ] T244 [P] Security test for cross-tenant data access prevention in backend/tests/integration/tenant_isolation_test.go
- [ ] T295 **[CRITICAL]** Implement automated query analyzer (static analysis) that fails if any query lacks tenant_id filter in backend/tests/analysis/query_analyzer_test.go
- [ ] T296 **[CRITICAL]** Create multi-tenant JOIN query isolation test (verify all JOIN queries filter both tables by tenant_id) in backend/tests/integration/multi_tenant_join_test.go
- [ ] T297 **[CRITICAL]** Add continuous tenant isolation verification to CI/CD pipeline in .github/workflows/security-check.yml (run T295, T296 on every commit)
- [ ] T245 [P] Penetration testing for authentication endpoints

---

## Phase 13: Documentation & Developer Experience

**Purpose**: Complete documentation and developer tooling

- [ ] T246 [P] Create API documentation with Swagger UI in backend/docs/
- [ ] T247 [P] Update README.md with setup instructions in repository root
- [ ] T248 [P] Document environment variables in .env.example
- [ ] T249 [P] Create development guide in docs/development.md
- [ ] T250 [P] Create testing guide in docs/testing.md
- [ ] T251 [P] Document API endpoints in docs/api-reference.md
- [ ] T252 [P] Create troubleshooting guide in docs/troubleshooting.md
- [ ] T253 [P] Document Tailwind design system in docs/design-system.md
- [ ] T254 [P] Document accessibility guidelines in docs/accessibility.md
- [ ] T255 [P] Document i18n translation workflow in docs/i18n.md
- [ ] T256 [P] Create deployment guide in docs/deployment.md
- [ ] T326 [P] Document notification service setup and Kafka configuration in docs/notification-service.md
- [ ] T327 [P] Document email template customization guide in docs/email-templates.md

### Notification Service Integration Verification

- [ ] T328 Integration test: Verify user.registered event triggers welcome email in backend/tests/integration/notifications/registration_notification_test.go
- [ ] T329 Integration test: Verify user.login event triggers login alert email in backend/tests/integration/notifications/login_notification_test.go
- [ ] T330 Integration test: Verify password.reset_requested event triggers reset email in backend/tests/integration/notifications/password_reset_notification_test.go
- [ ] T331 Integration test: Verify password.changed event triggers confirmation email in backend/tests/integration/notifications/password_changed_notification_test.go
- [ ] T332 E2E test: Complete registration flow with email verification in frontend/tests/e2e/registration-with-email.spec.ts
- [ ] T333 Manual test: Verify all email templates render correctly in different email clients (Gmail, Outlook, Apple Mail)

---

## Phase 14: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements affecting multiple user stories

### Change Password Feature (Authenticated User)

- [ ] T314 Implement PasswordService.ChangePassword in backend/auth-service/src/services/PasswordService.go (verify old password, update to new password)
- [ ] T315 Implement PUT /api/auth/password/change handler in backend/auth-service/src/api/handlers.go (requires authentication, validates old password)
- [ ] T316 **Publish password.changed event to Kafka after successful password change** in backend/auth-service/src/api/handlers.go (after T315)
- [ ] T317 [P] Create ChangePasswordForm component in frontend/src/components/forms/ChangePasswordForm.tsx (old password, new password, confirm with Tailwind)
- [ ] T318 [P] Create change password page in frontend/src/pages/account/change-password.tsx (requires login)
- [ ] T319 [P] Implement change password API call in frontend/src/services/auth.service.ts
- [ ] T320 [P] Add change password translations in frontend/public/locales/en/auth.json
- [ ] T321 [P] Add change password translations in frontend/public/locales/id/auth.json
- [ ] T322 Unit test for PasswordService.ChangePassword in backend/tests/unit/services/PasswordService_test.go
- [ ] T323 Integration test for change password flow in backend/tests/integration/change_password_test.go
- [ ] T324 [P] Unit test for ChangePasswordForm in frontend/tests/unit/components/forms/ChangePasswordForm.test.tsx
- [ ] T325 E2E test for change password flow in frontend/tests/e2e/change-password.spec.ts

### General Polish

- [ ] T257 [P] Code cleanup and refactoring for consistency
- [ ] T258 [P] Add comprehensive code comments for complex business logic
- [ ] T259 [P] Verify all console.log removed from production code
- [x] T260 [P] Add health check endpoints to all services in backend/*/src/api/handlers.go
- [ ] T261 [P] Add readiness check endpoints with dependency validation
- [ ] T262 [P] Implement graceful shutdown for all services
- [ ] T263 [P] Add request tracing with correlation IDs in api-gateway/src/middleware/tracing.middleware.go
- [ ] T264 [P] Add metrics collection endpoints for monitoring
- [ ] T265 [P] Final code review and quality check
- [ ] T266 [P] Validate quickstart.md against actual setup process
- [ ] T267 [P] Run full test suite with coverage report verification (backend ≥80%, frontend ≥70%, critical paths ≥95%)
- [ ] T268 [P] Final E2E testing of all user stories

---

## Phase 15: Email Verification (Optional - Post-MVP Enhancement)

**Purpose**: Add email verification to registration flow for enhanced security and user validation

**Note**: This phase is optional for MVP as per spec.md line 169. Can be implemented post-launch if needed.

### Database Schema for Email Verification

- [ ] T334 Add email verification columns to users table migration in backend/migrations/000002_create_users.up.sql (email_verified BOOLEAN DEFAULT FALSE, email_verified_at TIMESTAMPTZ, verification_token VARCHAR(255), verification_token_expires_at TIMESTAMPTZ)
- [ ] T335 Create database index on verification_token in backend/migrations/000002_create_users.up.sql

### Backend Implementation

- [ ] T336 Implement EmailVerificationService.GenerateToken in backend/auth-service/src/services/EmailVerificationService.go (generate secure token, set expiry 24 hours)
- [ ] T337 Implement EmailVerificationService.VerifyEmail in backend/auth-service/src/services/EmailVerificationService.go (validate token, check expiry, mark user as verified)
- [ ] T338 Implement EmailVerificationService.ResendVerification in backend/auth-service/src/services/EmailVerificationService.go (generate new token, publish event)
- [ ] T339 Implement GET /api/auth/verify-email?token={token} handler in backend/auth-service/src/api/handlers.go (verify token and mark user verified)
- [ ] T340 Implement POST /api/auth/resend-verification handler in backend/auth-service/src/api/handlers.go (resend verification email)
- [ ] T341 Update login handler to return user email and verification status if unverified in backend/auth-service/src/api/handlers.go (replaces T361 - consolidated)
- [ ] T342 Update user.registered event to include verification_token in backend/tenant-service/src/api/handlers.go (T310 update)
- [ ] T343 Update registration email template to include verification link in backend/notification-service/templates/registration-email.html (T302 update)
- [ ] T344 Unit test for EmailVerificationService.GenerateToken in backend/tests/unit/services/EmailVerificationService_test.go
- [ ] T345 Unit test for EmailVerificationService.VerifyEmail in backend/tests/unit/services/EmailVerificationService_test.go
- [ ] T346 Integration test for email verification flow in backend/tests/integration/email_verification_test.go (register → verify → login)

### Frontend Implementation

- [ ] T347 [P] Create email verification success page in frontend/src/pages/auth/verify-email.tsx (shows success message after clicking link)
- [ ] T348 [P] Create resend verification page in frontend/src/pages/auth/resend-verification.tsx (form to resend email)
- [ ] T349 [P] Add "Verify your email" banner component in frontend/src/components/banners/VerifyEmailBanner.tsx (shows for unverified users)
- [ ] T364 [P] **Create verification required page for login attempts** in frontend/src/pages/auth/verification-required.tsx (shows when unverified user tries to login, with resend button)
- [ ] T365 [P] **Update login page to handle EMAIL_NOT_VERIFIED error** - Redirect to verification-required page with user email in frontend/src/pages/auth/login.tsx
- [ ] T350 [P] Implement email verification API calls in frontend/src/services/auth.service.ts (verifyEmail, resendVerification methods)
- [ ] T351 [P] Add email verification translations in frontend/public/locales/en/auth.json
- [ ] T352 [P] Add email verification translations in frontend/public/locales/id/auth.json
- [ ] T353 [P] Unit test for VerifyEmailBanner in frontend/tests/unit/components/banners/VerifyEmailBanner.test.tsx
- [ ] T354 Update T332 E2E test for complete registration with verification flow in frontend/tests/e2e/registration-with-email.spec.ts

### Configuration & Security

- [ ] T355 Add email verification feature flag in .env.example (REQUIRE_EMAIL_VERIFICATION=false by default)
- [ ] T356 Add verification token security (rate limiting: max 3 resend per hour per user) in backend/auth-service/src/middleware/ratelimit.go
- [ ] T357 Integration test for expired verification token in backend/tests/integration/email_verification_edge_cases_test.go
- [ ] T358 Integration test for resend verification rate limiting in backend/tests/integration/email_verification_rate_limit_test.go

### Documentation

- [ ] T359 Document email verification setup in docs/email-verification.md
- [ ] T360 Update API documentation with verification endpoints in backend/config/swagger.config.go

**Checkpoint**: Email verification fully functional - users must verify email to access system (if feature enabled)

**Feature Flag**: Can be enabled/disabled via `REQUIRE_EMAIL_VERIFICATION` environment variable for gradual rollout

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User Story 1 (P1): Independent after Foundational
  - User Story 2 (P1): Independent after Foundational
  - User Story 5 (P2): Depends on User Story 2 (login required for logout)
  - User Story 3 (P2): Independent after Foundational
  - User Story 4 (P3): Depends on User Story 3 (requires users to manage)
- **i18n (Phase 8)**: Can proceed in parallel with user stories after Foundational
- **Responsive Design (Phase 9)**: Can proceed after UI components created in Phase 3
- **Accessibility (Phase 10)**: Can proceed after UI components created in Phase 3
- **Performance (Phase 11)**: Can proceed after core features implemented
- **Security (Phase 12)**: Can proceed in parallel with implementation (security-first approach)
- **Documentation (Phase 13)**: Can proceed in parallel with implementation
- **Polish (Phase 14)**: Depends on all desired user stories being complete

### User Story Dependencies (Execution Flow)

1. **Phase 1 (Setup)** → **Phase 2 (Foundational)** ✅ MUST COMPLETE FIRST
2. After Foundational, can proceed with:
   - **Phase 3 (US1)** → Registration (P1) - Independent ✅ MVP START
   - **Phase 4 (US2)** → Login (P1) - Independent
3. After US2 complete:
   - **Phase 5 (US5)** → Logout (P2) - Depends on US2
4. After Foundational:
   - **Phase 6 (US3)** → Team Members (P2) - Independent
5. After US3 complete:
   - **Phase 7 (US4)** → RBAC (P3) - Depends on US3
6. Phases 8-14 can proceed based on components available

### Suggested MVP Scope (Minimum Viable Product)

**MVP = Phase 1 + Phase 2 + Phase 3 + Phase 4**
- Setup + Foundational infrastructure
- User Story 1: Business owner registration
- User Story 2: User login with tenant isolation
- Result: Business owners can register, log in, and access their isolated tenant data

### Parallel Opportunities

**Within Setup (Phase 1)**:
- All tasks T008-T020 marked [P] can run in parallel

**Within Foundational (Phase 2)**:
- All tasks T025-T048 marked [P] can run in parallel

**Within Each User Story**:
- All model creation tasks marked [P]
- All unit test creation tasks marked [P]
- All UI component creation tasks marked [P]

**Across User Stories** (after Foundational complete):
- US1, US2, US3 can be worked on in parallel
- US5 can start after US2 completes
- US4 can start after US3 completes

**Cross-Cutting Phases** (Phases 8-14):
- i18n, Responsive Design, Accessibility can proceed in parallel
- Documentation can proceed continuously throughout

---

## Parallel Example: User Story 1 (Business Registration)

```bash
# Launch all UI components for User Story 1 together:
Task T049: "Create Button component with Tailwind styling"
Task T050: "Create Input component with Tailwind styling"
Task T051: "Create Card component with Tailwind styling"
Task T052: "Create LoadingSpinner component with Tailwind styling"
Task T053: "Create Toast component with Tailwind styling"
Task T054: "Create FormField component with validation display"
Task T055: "Create AuthLayout component with responsive design"

# Launch all unit tests for components together:
Task T056: "Unit test for Button component"
Task T057: "Unit test for Input component"
Task T058: "Unit test for FormField component"

# Launch all backend unit tests together:
Task T059: "Unit test for password hashing utility"
Task T060: "Unit test for email validation utility"
Task T061: "Unit test for JWT token generation utility"

# Launch all models together:
Task T062: "Create Tenant model"
Task T063: "Create User model"
```

---

## Implementation Strategy

### Test-First Development (CRITICAL)

Per constitution requirement, tests MUST be written before implementation:
1. Write unit tests for utilities and services
2. Write component tests for UI
3. Write contract tests for API endpoints
4. Write integration tests for user flows
5. Verify all tests FAIL
6. Implement code to make tests pass
7. Verify coverage targets met (backend 80%, frontend 70%, critical paths 95%)

### MVP First (User Stories 1 & 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL)
3. Complete Phase 3: User Story 1 (Business Registration)
4. Complete Phase 4: User Story 2 (User Login)
5. **STOP and VALIDATE**: Test independently
6. Deploy MVP to staging

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 + US2 → Test → Deploy (MVP! ✅)
3. Add US5 (Logout) → Test → Deploy
4. Add US3 (Team Members) → Test → Deploy
5. Add US4 (RBAC) → Test → Deploy
6. Add i18n, responsive, accessibility → Test → Deploy
7. Each increment adds value without breaking previous functionality

### Parallel Team Strategy

With multiple developers:

1. **Team**: Complete Setup + Foundational together (CRITICAL PATH)
2. Once Foundational done:
   - **Developer A**: User Story 1 (Registration) + UI components
   - **Developer B**: User Story 2 (Login) + Session management
   - **Developer C**: i18n infrastructure + Language switcher
3. After initial stories:
   - **Developer A**: User Story 3 (Team Members)
   - **Developer B**: User Story 5 (Logout)
   - **Developer C**: Responsive design + Accessibility
4. Final:
   - **Developer A**: User Story 4 (RBAC)
   - **Developer B**: Performance optimization
   - **Developer C**: Security hardening

---

## Coverage Requirements Summary

- **Backend Unit Tests**: 80% minimum coverage
- **Frontend Unit Tests**: 70% minimum coverage
- **Critical Paths (Authentication, Tenant Isolation)**: 95% minimum coverage
- **Integration Tests**: All API endpoints and user flows
- **Contract Tests**: All OpenAPI specifications
- **E2E Tests**: All critical user journeys
- **Accessibility Tests**: WCAG 2.1 AA compliance for all UI
- **Performance Tests**: Response times per specification
- **Security Tests**: Tenant isolation and access control

---

## Task Count Summary

- **Phase 1 (Setup)**: 20 tasks
- **Phase 2 (Foundational)**: 28 tasks
- **Phase 3 (US1 - Registration)**: 33 tasks (includes UI components + tests)
- **Phase 4 (US2 - Login)**: 27 tasks
- **Phase 5 (US5 - Logout)**: 22 tasks (includes additional UI components)
- **Phase 6 (US3 - Team Members)**: 28 tasks
- **Phase 7 (US4 - RBAC)**: 31 tasks
- **Phase 8 (i18n)**: 12 tasks
- **Phase 9 (Responsive)**: 9 tasks
- **Phase 10 (Accessibility)**: 10 tasks
- **Phase 11 (Performance)**: 14 tasks
- **Phase 12 (Security)**: 11 tasks
- **Phase 13 (Documentation)**: 11 tasks
- **Phase 14 (Polish)**: 12 tasks

**Total**: 268 tasks

**MVP Scope**: 108 tasks (Phase 1 + 2 + 3 + 4)

---

## Notes

- [P] tasks = different files, no dependencies, can run in parallel
- [Story] label maps task to specific user story for traceability
- All UI components use Tailwind CSS utility-first approach
- All forms implement responsive mobile-first design
- All interactive elements meet WCAG 2.1 AA accessibility standards
- All user-facing text supports EN/ID internationalization
- All tests follow test-first development methodology
- Verify test coverage targets before considering phase complete
- Stop at any checkpoint to validate story works independently
- Each user story delivers independently testable value
