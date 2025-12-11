# SSE Contract - Auth & Tenant Scoping

This document describes the contract expectations for the SSE endpoint `/api/v1/sse`.

Required JWT claims for subscription:

- `tenant_id` (string): the tenant UUID the client is subscribing for
- `roles` (array): roles assigned to the user (one of `Owner`, `Manager`, `Cashier` required for order notifications)

Authorization header: `Authorization: Bearer <token>` where `<token>` is a JWT signed with `TEST_JWT_SECRET` for local tests.

Tests in this folder should verify:

- Missing `Authorization` → 401 Unauthorized
- Malformed token → 401 Unauthorized
- Valid token but `tenant_id` does not match allowed tenant in server → 403 Forbidden
- Valid token and tenant match → 200 OK and SSE stream established

The project provides service-level contract tests in `backend/notification-service/tests/e2e` which exercise these expectations in-process.
