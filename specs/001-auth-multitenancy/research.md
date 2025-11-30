# Research: User Authentication and Multi-Tenancy

**Feature**: User Authentication and Multi-Tenancy  
**Date**: 2025-11-22  
**Status**: Complete

## Overview

This document consolidates research findings for implementing secure authentication and multi-tenant architecture using Next.js frontend, Go backend services, PostgreSQL database, API Gateway, Redis session management, and JWT tokens in HTTP-only cookies.

---

## 1. JWT Token Storage Strategy

### Decision: HTTP-Only Cookies (NOT localStorage)

**Rationale**:
- HTTP-only cookies are inaccessible to JavaScript, preventing XSS attacks from stealing tokens
- Browser automatically sends cookies with each request, eliminating manual token management
- Secure flag ensures cookies only transmitted over HTTPS
- SameSite attribute prevents CSRF attacks
- Industry best practice for session token storage in web applications

**Alternatives Considered**:
- **localStorage**: Rejected due to XSS vulnerability - any JavaScript on the page can access tokens
- **sessionStorage**: Similar XSS vulnerability as localStorage
- **Memory-only (state)**: Rejected because tokens lost on page refresh, poor UX

**Implementation Details**:
- Set cookies with flags: `HttpOnly; Secure; SameSite=Strict`
- Cookie name: `auth_token`
- Cookie max age: 15 minutes (matching session timeout from spec)
- Refresh token in separate cookie: `refresh_token` with longer expiration
- API Gateway validates JWT from cookie on every request

---

## 2. API Gateway Pattern

### Decision: Dedicated API Gateway as Single Entry Point

**Rationale**:
- Centralizes cross-cutting concerns: authentication, rate limiting, logging, CORS
- Frontend complexity reduced - only needs to know gateway URL
- Service discovery handled by gateway, services can be relocated without frontend changes
- Simplifies security model - only gateway exposed to public internet
- Standard pattern for microservices architecture

**Alternatives Considered**:
- **Frontend calls services directly**: Rejected due to complexity (service discovery, auth duplication, CORS on each service)
- **Service mesh only**: Rejected because still needs external entry point, gateway provides additional features

**Implementation Details**:
- Echo framework for gateway (consistent with backend services)
- Routes prefix by service: `/api/auth/*`, `/api/users/*`, `/api/tenants/*`
- JWT middleware validates tokens before proxying to services
- Rate limiting middleware per IP/user
- Circuit breaker pattern for resilient service calls
- Health check endpoint aggregates backend service health

---

## 3. PostgreSQL Multi-Tenancy Strategy

### Decision: Shared Database with Tenant ID Discrimination

**Rationale**:
- Cost-effective: single database cluster serves all tenants
- ACID compliance ensures data consistency for financial transactions
- Robust foreign key constraints enforce referential integrity
- Excellent performance with proper indexing on tenant_id
- Mature ecosystem with proven reliability for POS systems
- Native support for complex queries and transactions

**Alternatives Considered**:
- **Database per tenant**: Rejected due to operational complexity at scale (100+ tenants), connection pool exhaustion
- **Schema per tenant**: Rejected due to migration complexity and connection overhead
- **Separate database instances**: Rejected due to cost and operational overhead

**Implementation Details**:
- All tables include `tenant_id` column (UUID)
- Composite indexes: `(tenant_id, <query_field>)`
- Row-level security (RLS) policies enforce tenant isolation at database level
- Application-level enforcement: all queries MUST include `tenant_id` filter
- Service layer automatically injects `tenant_id` from JWT context
- Database migration scripts use Flyway or golang-migrate

**Tables**:
```
tenants: { id (PK), business_name, created_at, status, settings (JSONB) }
users: { id (PK), tenant_id (FK), email, password_hash, role, status, created_at }
sessions: { id (PK), session_id (unique), tenant_id (FK), user_id (FK), expires_at, created_at }
invitations: { id (PK), tenant_id (FK), email, token (unique), invited_by (FK), expires_at, status }
```

---

## 4. Redis Session Management

### Decision: Redis for Active Session Storage and Validation

**Rationale**:
- Sub-millisecond session lookup performance (meets <100ms constraint)
- Built-in TTL for automatic session expiration (15 minutes from spec)
- Distributed cache supports horizontal scaling
- Reduces database load for high-frequency session validation
- Industry standard for session management

**Alternatives Considered**:
- **In-memory (service-local)**: Rejected due to issues with horizontal scaling, session loss on restart
- **Database only**: Rejected due to slower query performance compared to Redis

**Implementation Details**:
- Key pattern: `session:{sessionId}` → `{userId, tenantId, expiresAt, metadata}`
- TTL set to 15 minutes (session timeout from spec)
- On each request: Gateway validates JWT → checks Redis for session validity
- Session renewal: Update TTL on successful request (sliding window expiration)
- On logout: Delete Redis key immediately
- On user deletion: Scan and delete all user's sessions

---

## 5. Echo Framework for Go Services

### Decision: Echo v4 for All Backend Services and API Gateway

**Rationale**:
- High performance: minimal overhead, fast routing
- Built-in middleware: JWT, CORS, rate limiting, logging
- Simple API design, low learning curve
- Well-documented with active community
- Standard HTTP patterns, easy to test

**Alternatives Considered**:
- **Gin**: Similar performance, chose Echo for better middleware ecosystem
- **Go standard library (net/http)**: Rejected due to need to build all middleware from scratch
- **Fiber**: Rejected due to Express-like API (less Go-idiomatic)

**Implementation Details**:
- Each service runs as independent Echo server
- Structured logging with Echo's Logger middleware
- Error handling middleware returns consistent JSON error responses
- CORS middleware on gateway (not services)
- Context propagation for tenant and user ID through request chain

---

## 6. Next.js Frontend Architecture

### Decision: Next.js 13+ with App Router and Server Components

**Rationale**:
- App Router provides better routing with layouts and nested routes
- Server Components reduce JavaScript bundle size
- Middleware for authentication checks before page render
- API route handlers for BFF pattern if needed
- Built-in optimization: image, font, code splitting
- React Server Actions for mutations (optional)

**Alternatives Considered**:
- **Next.js Pages Router**: Rejected in favor of newer App Router features
- **Create React App**: Rejected due to lack of SSR, routing, optimization features
- **Vite + React Router**: Rejected due to need to configure SSR, auth, optimization manually

**Implementation Details**:
- Route groups: `(auth)` for public pages, `(dashboard)` for protected pages
- Middleware checks auth cookie, redirects unauthorized users
- API client wrapper handles cookie-based authentication automatically
- Auth context provider for client-side auth state
- Form components for login/signup with client-side validation
- Error boundaries for graceful error handling

---

## 7. Password Security

### Decision: bcrypt for Password Hashing

**Rationale**:
- Designed for password hashing with built-in salt
- Adaptive cost factor protects against hardware improvements
- Industry standard, well-audited
- Available in Go standard crypto library

**Alternatives Considered**:
- **Argon2**: Rejected due to more complex configuration, bcrypt sufficient for POS use case
- **PBKDF2**: Rejected due to less resistance to hardware attacks than bcrypt
- **scrypt**: Rejected due to memory requirements, bcrypt more battle-tested

**Implementation Details**:
- Cost factor: 12 (balance between security and performance)
- Hash stored in user document `passwordHash` field
- Plain-text password never logged or stored
- Password validation: compare provided password with hash using bcrypt.CompareHashAndPassword

---

## 8. Rate Limiting Strategy

### Decision: Token Bucket Algorithm with Redis Backing

**Rationale**:
- Prevents brute force attacks (spec requirement: 5 attempts per 15 minutes)
- Distributed rate limiting across gateway instances
- Token bucket allows burst traffic while enforcing average rate
- Redis provides shared state and atomic operations

**Alternatives Considered**:
- **In-memory rate limiting**: Rejected due to inability to share state across gateway instances
- **Fixed window**: Rejected due to burst at window boundary
- **Sliding window**: Rejected due to higher complexity, token bucket sufficient

**Implementation Details**:
- Key pattern: `ratelimit:login:{email}:{tenantId}`
- Limit: 5 attempts per 15-minute window (from spec FR-018)
- On failed login: Increment counter with 15-minute TTL
- On successful login: Reset counter
- Return HTTP 429 (Too Many Requests) when limit exceeded
- Include Retry-After header in 429 response

### Decision: Three-Tier Testing (Contract, Integration, Unit)

**Rationale**:
- Contract tests ensure API compatibility between services and frontend
- Integration tests verify service interactions and database operations
- Unit tests verify business logic in isolation
- Test-first development mandated by constitution

**Implementation Details**:

**Contract Tests**:
- OpenAPI spec validation for each service
- Generated from `contracts/*.yaml` files
- Validates request/response schemas
- Tests run against live services in Docker Compose

**Integration Tests**:
- Test service with real PostgreSQL and Redis (Docker Compose)
- Verify tenant isolation by attempting cross-tenant access
- Test authentication flow end-to-end
- Verify session management and expiration

**Unit Tests**:
- Go: `testing` package with `testify` for assertions
- Frontend: Jest + React Testing Library
- Mock database and Redis clients
- Test business logic functions in isolation
- Coverage target: ≥80% (from constitution)

---

## 10. Observability and Monitoring

### Decision: Structured Logging + Health Checks + Request Tracing

**Rationale**:
- Constitution requires observability for all services
- Structured logs enable querying and analysis
- Health checks enable load balancer routing and orchestration
- Request tracing links operations across services
- Audit trail for authentication events (spec FR-016)

**Implementation Details**:

**Structured Logging**:
- JSON format: `{timestamp, level, service, tenantId, userId, message, context}`
- Log all authentication attempts with outcome
- Log all authorization failures
- Log session creation/deletion
- Log rate limit violations

**Health Checks**:
- Endpoint: `GET /health` returns `{status: "ok", services: {...}}`
- Readiness: `GET /ready` checks database and Redis connectivity
- Liveness: `GET /live` confirms process is running

**Request Tracing**:
- Generate request ID on gateway entry
- Propagate via `X-Request-ID` header
- Include in all logs for request correlation
- Include tenant ID and user ID in trace context

**Metrics** (Future Enhancement):
- Authentication success/failure rates
- API endpoint latency (p50, p95, p99)
- Session creation/expiration rates
- Rate limit violations

---

## 11. Internationalization (i18n) and Language Switcher

### Decision: i18next for Backend and Frontend with EN/ID Support

**Rationale**:
- Unified i18n solution across backend (API error messages) and frontend (UI)
- Supports multiple languages with easy extensibility
- Built-in language detection from browser, localStorage, or user preferences
- Namespace support for organizing translations by feature
- Seamless integration with React (react-i18next) and Next.js
- JSON-based translations are easy to maintain and version control

**Alternatives Considered**:
- **Format.js (React Intl)**: More enterprise features but heavier for two-language support
- **Custom solution**: Would require implementing locale management, fallbacks from scratch
- **Next.js built-in i18n**: Limited to routing-based locales, not suitable for dynamic switching

**Implementation Details**:

**Frontend (Next.js + react-i18next)**:
- Language switcher component in navigation bar
- Dropdown/toggle with flags: EN (English) / ID (Indonesia)
- State managed in React Context
- Persisted in localStorage for guests
- Synced to user profile in database for authenticated users

**Backend (Go services)**:
- Accept-Language header from frontend
- Return error messages in user's locale
- Store user's locale preference in database

**Translation Organization**:
```
frontend/locales/
├── en/
│   ├── common.json       # Shared: buttons, labels
│   ├── auth.json         # Login, signup
│   └── dashboard.json    # Dashboard UI
└── id/
    ├── common.json
    ├── auth.json
    └── dashboard.json

backend/locales/
├── en.json              # API error messages
└── id.json
```

**Language Switcher Component**:
```typescript
// components/LanguageSwitcher.tsx
import { useTranslation } from 'react-i18next';

export function LanguageSwitcher() {
  const { i18n } = useTranslation();
  
  const changeLanguage = async (locale: 'en' | 'id') => {
    await i18n.changeLanguage(locale);
    localStorage.setItem('locale', locale);
    
    // Update user preference if authenticated
    if (authenticated) {
      await fetch('/api/users/me/locale', {
        method: 'PATCH',
        body: JSON.stringify({ locale })
      });
    }
  };

  return (
    <select value={i18n.language} onChange={(e) => changeLanguage(e.target.value)}>
      <option value="en">English</option>
      <option value="id">Indonesia</option>
    </select>
  );
}
```

**User Model Extension**:
```sql
ALTER TABLE users ADD COLUMN locale VARCHAR(5) DEFAULT 'en';
```

**Initial Load Priority**:
1. Authenticated user: Load from database user.locale
2. Guest user: Load from localStorage
3. Fallback: Browser Accept-Language header
4. Default: 'en'

**Translation Example**:
```json
// locales/en/auth.json
{
  "login": {
    "title": "Sign In",
    "email": "Email",
    "password": "Password",
    "submit": "Sign In",
    "errors": {
      "invalidCredentials": "Invalid email or password"
    }
  }
}

// locales/id/auth.json
{
  "login": {
    "title": "Masuk",
    "email": "Email",
    "password": "Kata Sandi",
    "submit": "Masuk",
    "errors": {
      "invalidCredentials": "Email atau kata sandi tidak valid"
    }
  }
}
```

**Backend API Error Localization**:
```go
// services/auth/handler.go
func (h *Handler) Login(c echo.Context) error {
    locale := c.Request().Header.Get("Accept-Language")
    if locale == "" {
        locale = "en"
    }
    
    // ... authentication logic ...
    
    if !valid {
        msg := i18n.Translate(locale, "auth.errors.invalidCredentials")
        return c.JSON(401, ErrorResponse{Error: msg})
    }
}
```

**Performance Considerations**:
- Translation files are small (~5KB each gzipped)
- Loaded asynchronously, don't block initial render
- Cached in browser after first load
- Server-side: translations loaded into memory on startup

**Testing**:
- Snapshot tests for translation file completeness
- Component tests verify language switcher updates UI
- Integration tests verify locale persistence
- API tests verify error messages in correct locale

---

## Summary of Decisions

| Topic | Decision | Key Rationale |
|-------|----------|---------------|
| Token Storage | HTTP-only cookies | XSS protection, security best practice |
| API Gateway | Dedicated Echo gateway | Centralized auth, rate limiting, service discovery |
| Database | PostgreSQL with tenant ID | ACID compliance, mature, excellent for financial data |
| Multi-Tenancy | Shared DB with tenant ID + RLS | Cost-effective, database-enforced isolation |
| Session Management | Redis with 15-min TTL | Performance (<100ms), automatic expiration |
| Backend Framework | Echo v4 | Performance, middleware ecosystem |
| Frontend Framework | Next.js App Router | SSR, optimization, modern patterns |
| Password Hashing | bcrypt (cost 12) | Industry standard, adaptive cost |
| Rate Limiting | Token bucket + Redis | Distributed, handles bursts |
| Testing | Contract + Integration + Unit | API compatibility, isolation, coverage |
| Observability | Structured logs + health checks | Auditing, debugging, orchestration |
| Internationalization | i18next (backend + frontend) | Unified i18n, EN/ID support, easy maintenance |
| Language Switcher | React component + localStorage/DB | Persistent locale, syncs across devices |

---

**Next Phase**: Generate data-model.md and API contracts based on these decisions.
