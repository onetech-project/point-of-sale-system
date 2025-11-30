# Data Model: User Authentication and Multi-Tenancy

**Feature**: User Authentication and Multi-Tenancy  
**Date**: 2025-11-22  
**Status**: Design Complete

## Overview

This document defines the data models for authentication and multi-tenancy in the point-of-sale system. All entities are stored in PostgreSQL with strict tenant isolation enforced through `tenant_id` columns and row-level security policies.

---

## Core Entities

### 1. Tenant

Represents a business organization using the platform. Each tenant is completely isolated.

**PostgreSQL Table**: `tenants`

**Schema**:
```sql
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_name VARCHAR(100) NOT NULL,
    slug VARCHAR(50) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT tenants_status_check CHECK (status IN ('active', 'suspended', 'deleted'))
);

CREATE INDEX idx_tenants_status_created ON tenants(status, created_at DESC);
CREATE INDEX idx_tenants_slug ON tenants(slug);
```

**Fields**:
- `id`: Unique tenant identifier (UUID, primary key)
- `business_name`: Display name of the business (required, 1-100 chars)
- `slug`: URL-safe unique identifier (required, lowercase, alphanumeric + hyphens)
- `status`: Tenant account status (required, enum: "active", "suspended", "deleted")
- `settings`: Custom tenant settings as JSON (optional, default: {})
  - `session_timeout`: Custom session timeout in minutes (default: 15)
  - `max_users`: Maximum users allowed (default: unlimited)
- `created_at`: Timestamp of tenant creation (auto-generated)
- `updated_at`: Timestamp of last update (auto-updated via trigger)

**Indexes**:
```sql
PRIMARY KEY (id)
UNIQUE INDEX ON slug
INDEX ON (status, created_at DESC)
```

**Validation Rules**:
- `business_name` must be unique within active tenants
- `slug` must be globally unique
- `status` cannot transition from "deleted" to any other status
- At least one owner user must exist before tenant can be "active"

**State Transitions**:
```
[new] → active (on first owner user creation)
active → suspended (admin action)
suspended → active (admin action)
active → deleted (soft delete)
suspended → deleted (soft delete)
```

---

### 2. User

Represents an individual with access to the system. Each user belongs to exactly one tenant.

**PostgreSQL Table**: `users`

**Schema**:
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    locale VARCHAR(5) NOT NULL DEFAULT 'en',
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT users_tenant_email_unique UNIQUE (tenant_id, email),
    CONSTRAINT users_role_check CHECK (role IN ('owner', 'manager', 'cashier')),
    CONSTRAINT users_status_check CHECK (status IN ('active', 'invited', 'suspended', 'deleted')),
    CONSTRAINT users_locale_check CHECK (locale IN ('en', 'id'))
);

CREATE INDEX idx_users_tenant_status_role ON users(tenant_id, status, role);
CREATE INDEX idx_users_tenant_last_login ON users(tenant_id, last_login_at DESC);
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- Row-level security for tenant isolation
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
CREATE POLICY users_tenant_isolation ON users
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

**Fields**:
- `id`: Unique user identifier (UUID, primary key)
- `tenant_id`: Reference to owning tenant (required, foreign key with cascade delete)
- `email`: User's email address (required, lowercase, valid email format)
- `password_hash`: bcrypt hash of password (required, never returned in API responses)
- `role`: User's permission level (required, enum: "owner", "manager", "cashier")
- `status`: User account status (required, enum: "active", "invited", "suspended", "deleted")
- `first_name`: User's first name (optional, 1-50 chars)
- `last_name`: User's last name (optional, 1-50 chars)
- `locale`: User's preferred language (required, enum: "en", "id", default: "en")
- `last_login_at`: Timestamp of last successful login (nullable)
- `created_at`: Timestamp of user creation (auto-generated)
- `updated_at`: Timestamp of last update (auto-updated via trigger)

**Indexes**:
```sql
PRIMARY KEY (id)
UNIQUE INDEX ON (tenant_id, email)
INDEX ON (tenant_id, status, role)
INDEX ON (tenant_id, last_login_at DESC)
INDEX ON (tenant_id)
```

**Validation Rules**:
- Email format validated with regex: `/^[^\s@]+@[^\s@]+\.[^\s@]+$/`
- Password must be ≥8 characters, contain letters and numbers
- Each tenant must have at least one "owner" user
- Email + tenant_id combination must be unique (same email allowed across tenants)
- Cannot delete last owner of a tenant

**Role Permissions**:
- **owner**: Full access to all tenant features, user management, settings
- **manager**: Operational features, reports, inventory management (no system config)
- **cashier**: Point-of-sale transactions, customer lookup (no admin features)

**State Transitions**:
```
[new] → invited (invitation sent)
invited → active (invitation accepted, password set)
active → suspended (admin action)
suspended → active (admin action)
active → deleted (soft delete)
suspended → deleted (soft delete)
```

---

### 3. Session

Represents an authenticated user's active connection to the system. Sessions are stored in both Redis (active) and PostgreSQL (audit trail).

**PostgreSQL Table**: `sessions` (audit/historical)

**Schema**:
```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    terminated_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_session_id ON sessions(session_id);
CREATE INDEX idx_sessions_tenant_user ON sessions(tenant_id, user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_created_at ON sessions(created_at DESC);
CREATE INDEX idx_sessions_tenant_id ON sessions(tenant_id);

-- Row-level security for tenant isolation
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY sessions_tenant_isolation ON sessions
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

**Redis Key**: `session:{sessionId}`

**Redis Value**:
```json
{
  "userId": "string",
  "tenantId": "string",
  "email": "string",
  "role": "string",
  "createdAt": "number"
}
```

**Fields**:
- `id`: Unique record identifier (UUID, primary key, PostgreSQL only)
- `session_id`: Unique session identifier (UUID, used as JWT jti claim)
- `tenant_id`: Reference to tenant (required, foreign key with cascade delete)
- `user_id`: Reference to user (required, foreign key with cascade delete)
- `ip_address`: Client IP address (optional, for audit)
- `user_agent`: Client user agent (optional, for audit)
- `expires_at`: Session expiration timestamp (required)
- `terminated_at`: Timestamp when session was terminated (nullable, indicates logout)
- `created_at`: Timestamp of session creation (auto-generated)

**Indexes**:
```sql
PRIMARY KEY (id)
UNIQUE INDEX ON session_id
INDEX ON (tenant_id, user_id)
INDEX ON expires_at
INDEX ON created_at DESC
INDEX ON tenant_id
```

**Redis TTL**: 15 minutes (900 seconds) - sliding window on activity

**Validation Rules**:
- Session ID must be unique globally
- User must be "active" status to create session
- Tenant must be "active" status to create session
- Expired sessions automatically removed from Redis
- PostgreSQL sessions kept for 90 days for audit

**Lifecycle**:
```
[new] → active (on successful login)
active → expired (15 min inactivity)
active → terminated (explicit logout)
```

---

### 4. Invitation

Represents a pending invitation for a user to join a tenant.

**PostgreSQL Table**: `invitations`

**Schema**:
```sql
CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    accepted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT invitations_role_check CHECK (role IN ('manager', 'cashier')),
    CONSTRAINT invitations_status_check CHECK (status IN ('pending', 'accepted', 'expired', 'revoked'))
);

CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_tenant_email_status ON invitations(tenant_id, email, status);
CREATE INDEX idx_invitations_expires_at ON invitations(expires_at);
CREATE INDEX idx_invitations_tenant_id ON invitations(tenant_id);

-- Row-level security for tenant isolation
ALTER TABLE invitations ENABLE ROW LEVEL SECURITY;
CREATE POLICY invitations_tenant_isolation ON invitations
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);
```

**Fields**:
- `id`: Unique invitation identifier (UUID, primary key)
- `tenant_id`: Reference to tenant (required, foreign key with cascade delete)
- `email`: Invitee's email address (required, lowercase)
- `role`: Role to assign upon acceptance (required, enum: "manager", "cashier")
- `token`: Secure random token for invitation link (required, unique, 32 bytes hex)
- `invited_by`: Reference to user who sent invitation (required, foreign key)
- `status`: Invitation status (required, enum: "pending", "accepted", "expired", "revoked")
- `expires_at`: Invitation expiration timestamp (required, 7 days from creation)
- `accepted_at`: Timestamp when invitation was accepted (nullable)
- `created_at`: Timestamp of invitation creation (auto-generated)

**Indexes**:
```sql
PRIMARY KEY (id)
UNIQUE INDEX ON token
INDEX ON (tenant_id, email, status)
INDEX ON expires_at
INDEX ON tenant_id
```

**Validation Rules**:
- Cannot invite existing user email for the tenant
- Cannot invite if pending invitation already exists for email
- Token must be cryptographically secure (crypto.randomBytes)
- Invitation expires after 7 days
- Only "owner" or "manager" can send invitations
- Cannot invite with "owner" role (only business creation can create owners)

**State Transitions**:
```
[new] → pending (invitation created)
pending → accepted (user accepts and creates account)
pending → expired (7 days elapsed)
pending → revoked (inviter cancels)
```

---

## Relationships

```
Tenant (1) ─────── (N) User
  │                     │
  │                     │
  │                     │
  └─ (N) Session ──────┘
  │
  │
  └─ (N) Invitation

User (invitedBy) ──→ (N) Invitation
```

**Referential Integrity**:
- Users reference tenant_id with CASCADE DELETE
- Sessions reference both tenant_id and user_id with CASCADE DELETE
- Invitations reference tenant_id and invited_by with CASCADE DELETE
- Foreign key constraints enforced at database level
- On tenant deletion: cascade deletes all related users, sessions, invitations
- On user deletion: cascade deletes all sessions, invitations sent by that user

---

## Data Access Patterns

### Authentication Flow

1. **Login Request**: Query `users` table by `WHERE tenant_id = ? AND email = ?`
2. **Password Validation**: Compare provided password with `password_hash` using bcrypt
3. **Session Creation**: 
   - Generate session ID (UUID)
   - Store in Redis: `session:{sessionId}` with 15-min TTL
   - Store in PostgreSQL: `sessions` table for audit
4. **JWT Generation**: Sign token with session ID, user ID, tenant ID, role
5. **Cookie Response**: Set HTTP-only cookie with JWT

### Session Validation

1. **Request Received**: Extract JWT from cookie
2. **JWT Validation**: Verify signature, expiration
3. **Redis Lookup**: Check `session:{sessionId}` exists in Redis
4. **Context Injection**: Add tenant_id, user_id, role to request context
5. **TTL Renewal**: Update Redis TTL to 15 minutes (sliding window)

### Tenant Isolation

**All queries MUST include tenant_id filter**:

```sql
-- Correct: Tenant-scoped query
SELECT * FROM users WHERE tenant_id = $1 AND role = 'manager';

-- WRONG: Query without tenant scope
SELECT * FROM users WHERE role = 'manager';  -- CROSS-TENANT LEAK!
```

**Service Layer Enforcement**:
- Extract tenant_id from JWT context (injected by gateway)
- Automatically prepend tenant_id filter to all queries
- Validate response entities match request tenant_id
- Log warning if query attempted without tenant_id

**Database-Level Enforcement** (Row-Level Security):
- Set session variable: `SET app.current_tenant_id = '<tenant_uuid>'`
- RLS policies automatically filter all queries by tenant_id
- Provides defense-in-depth against application bugs

---

## Validation Rules Summary

### Password Requirements
- Minimum 8 characters
- Must contain at least one letter
- Must contain at least one number
- No maximum length enforced (bcrypt handles truncation)
- Regex: `/^(?=.*[A-Za-z])(?=.*\d).{8,}$/`

### Email Requirements
- Valid email format
- Lowercase normalization
- Maximum 255 characters
- Regex: `/^[^\s@]+@[^\s@]+\.[^\s@]+$/`

### Business Name Requirements
- 1-100 characters
- Must be unique within active tenants
- Allowed characters: letters, numbers, spaces, hyphens, apostrophes
- Regex: `/^[A-Za-z0-9\s\-']{1,100}$/`

### Slug Requirements
- 3-50 characters
- Lowercase alphanumeric and hyphens only
- Must start and end with alphanumeric
- Globally unique
- Auto-generated from business name if not provided
- Regex: `/^[a-z0-9][a-z0-9\-]*[a-z0-9]$/`

---

## PostgreSQL Indexes Summary

**Critical for Performance**:
- All tables indexed on `tenant_id` (prevents full table scans)
- Composite indexes for common query patterns
- Indexes on foreign keys for join performance
- Unique constraints for data integrity

**Index Maintenance**:
- Monitor index usage with `pg_stat_user_indexes`
- Add indexes based on query patterns in production
- Remove unused indexes to reduce write overhead
- Use `EXPLAIN ANALYZE` to validate query performance

---

## Data Retention and Cleanup

### Active Data (Redis)
- Sessions: 15-minute TTL, auto-removed
- Rate limit counters: 15-minute TTL, auto-removed

### Historical Data (PostgreSQL)
- Sessions: Retain 90 days for audit, then delete
- Invitations: Retain 30 days after expiration/acceptance, then delete
- Users: Soft delete only (status='deleted'), never physically remove
- Tenants: Soft delete only (status='deleted'), never physically remove

**Cleanup Jobs** (scheduled background tasks):
- Daily: Remove expired PostgreSQL sessions older than 90 days
  ```sql
  DELETE FROM sessions WHERE created_at < NOW() - INTERVAL '90 days';
  ```
- Daily: Remove invitations older than 30 days post-status-change
  ```sql
  DELETE FROM invitations 
  WHERE status IN ('accepted', 'expired', 'revoked') 
    AND created_at < NOW() - INTERVAL '30 days';
  ```
- Weekly: Audit tenant isolation (verify no cross-tenant queries in logs)

---

## Security Considerations

### Password Storage
- Never store plain-text passwords
- Never log passwords (even masked)
- Hash with bcrypt cost factor 12
- Password field excluded from all API responses (even to owner)

### Session Security
- Session IDs are cryptographically random (UUID v4)
- JWT tokens signed with HS256 or RS256 (configure per environment)
- Tokens stored in HTTP-only cookies (not localStorage)
- Cookies set with Secure and SameSite=Strict flags
- Session data in Redis does NOT include password hash

### Tenant Isolation
- tenant_id validated on EVERY query
- Row-Level Security (RLS) enforced at database level
- Service layer enforces tenant scope automatically
- Database queries without tenant_id trigger monitoring alerts
- Regular audits of query logs for cross-tenant access attempts

---

**Next Phase**: Generate API contracts in `/contracts/` directory.
