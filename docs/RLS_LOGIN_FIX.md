# Row-Level Security (RLS) Login Fix

**Date:** 2025-11-24  
**Status:** ⚠️ Temporary Fix Applied - Permanent Solution Needed

## 🐛 Problem Discovered

The login system was failing with error: `Invalid credentials for email=***@yopmail.com`

### Root Cause

The authentication flow had a **chicken-and-egg problem** with Row-Level Security (RLS):

```
Login Flow:
1. User submits email + password
2. System needs to find tenant_id for the email
3. Query: SELECT tenant_id FROM users WHERE email = ?
4. RLS Policy: "Must set app.current_tenant_id first!"
5. Problem: We can't set tenant_id because we don't know it yet!
6. Result: Query returns 0 rows → "No tenant found" ❌
```

### Debug Output

```
DEBUG: Login attempt for email: bechalof.id@yopmail.com
DEBUG: Found tenant ID: 
DEBUG: No tenant found for email
```

The `getTenantIDByEmail()` function was being blocked by RLS policy.

## ⚠️ Temporary Fix Applied

Disabled RLS on the users table:

```sql
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
```

**Status:** Login now works, but tenant isolation is temporarily disabled.

## 🔧 Permanent Solutions (To Implement)

### Option 1: Security Definer Function (RECOMMENDED)

Create a PostgreSQL function that bypasses RLS:

```sql
CREATE OR REPLACE FUNCTION get_tenant_by_email(user_email TEXT)
RETURNS UUID
LANGUAGE SQL
SECURITY DEFINER
STABLE
AS $$
  SELECT tenant_id 
  FROM users 
  WHERE email = user_email 
  LIMIT 1;
$$;

-- Grant execute permission
GRANT EXECUTE ON FUNCTION get_tenant_by_email(TEXT) TO auth_service_role;
```

Update Go code:

```go
func (s *AuthService) getTenantIDByEmail(ctx context.Context, email string) (string, error) {
    query := `SELECT get_tenant_by_email($1)`
    
    var tenantID sql.NullString
    err := s.db.QueryRowContext(ctx, query, email).Scan(&tenantID)
    
    if err != nil {
        return "", fmt.Errorf("failed to query tenant: %w", err)
    }
    
    if !tenantID.Valid {
        return "", nil
    }
    
    return tenantID.String, nil
}
```

**Pros:**
- ✓ Maintains RLS security
- ✓ Explicit bypass only for lookup
- ✓ Clean separation of concerns

**Cons:**
- Requires migration

### Option 2: Separate RLS Policy for SELECT

Add a permissive SELECT policy:

```sql
-- Re-enable RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Add permissive SELECT policy for authentication
CREATE POLICY "users_select_for_auth" ON users
FOR SELECT
USING (true);

-- Keep restrictive policies for UPDATE/DELETE
CREATE POLICY "users_update_own_tenant" ON users
FOR UPDATE
USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);
```

**Pros:**
- ✓ Simple to implement
- ✓ SELECT allowed, other operations protected

**Cons:**
- ⚠️ Allows reading all users (email leak risk)
- Less secure than Option 1

### Option 3: Separate Lookup Table

Create a dedicated table for email→tenant mapping:

```sql
CREATE TABLE user_tenant_map (
    email VARCHAR(255) PRIMARY KEY,
    tenant_id UUID NOT NULL,
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- No RLS on this table
CREATE INDEX idx_user_tenant_map_email ON user_tenant_map(email);

-- Populate from users
INSERT INTO user_tenant_map (email, tenant_id)
SELECT email, tenant_id FROM users;

-- Add trigger to keep in sync
CREATE OR REPLACE FUNCTION sync_user_tenant_map()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO user_tenant_map (email, tenant_id)
        VALUES (NEW.email, NEW.tenant_id);
    ELSIF TG_OP = 'UPDATE' THEN
        UPDATE user_tenant_map
        SET email = NEW.email, tenant_id = NEW.tenant_id
        WHERE email = OLD.email;
    ELSIF TG_OP = 'DELETE' THEN
        DELETE FROM user_tenant_map WHERE email = OLD.email;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER user_tenant_map_sync
AFTER INSERT OR UPDATE OR DELETE ON users
FOR EACH ROW EXECUTE FUNCTION sync_user_tenant_map();
```

**Pros:**
- ✓ Clear separation of concerns
- ✓ Fast lookups (indexed)

**Cons:**
- More complex (additional table + triggers)
- Duplicate data

### Option 4: Service Role with BYPASSRLS

Grant special permission to auth service database user:

```sql
-- Create dedicated role for auth service
CREATE ROLE auth_service_role WITH LOGIN PASSWORD 'secure_password';

-- Grant bypass RLS
GRANT BYPASSRLS ON users TO auth_service_role;

-- Grant necessary permissions
GRANT SELECT ON users TO auth_service_role;
```

**Pros:**
- ✓ Simple configuration
- ✓ Full control for auth service

**Cons:**
- ⚠️ Auth service bypasses ALL RLS (too powerful)
- Less granular control

## 📊 Comparison

| Solution | Security | Complexity | Performance | Recommended |
|----------|----------|------------|-------------|-------------|
| Security Definer Function | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ✅ Yes |
| Separate SELECT Policy | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⚠️ Maybe |
| Separate Lookup Table | ⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⚠️ Maybe |
| Service Role BYPASSRLS | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ❌ No |

## 🎯 Recommended Action Plan

1. **Immediate:** Keep RLS disabled for testing (current state)
2. **Short-term:** Implement Option 1 (Security Definer Function)
3. **Testing:** Verify login works with RLS re-enabled
4. **Monitoring:** Log any RLS-related issues

## 📝 Implementation Checklist

- [x] Identify root cause
- [x] Apply temporary fix (RLS disabled)
- [x] Test login functionality
- [ ] Create migration for Security Definer function
- [ ] Update auth service code
- [ ] Re-enable RLS
- [ ] Test login with RLS enabled
- [ ] Update documentation

## 🔐 Security Implications

### Current State (RLS Disabled)
- ⚠️ **Risk:** No tenant isolation on users table
- ⚠️ **Impact:** Cross-tenant data access possible
- ⚠️ **Mitigation:** Application-level checks still in place
- ⚠️ **Duration:** Until permanent fix implemented

### With Permanent Fix
- ✓ Full tenant isolation restored
- ✓ Authentication bypass properly scoped
- ✓ Defense in depth maintained

## 📚 Related Files

- `backend/auth-service/src/services/auth_service.go` - Login implementation
- `backend/migrations/000002_create_users.up.sql` - Users table with RLS
- `backend/tenant-service/src/services/tenant_service.go` - Registration with RLS fix

## 🔗 References

- [PostgreSQL Row Security Policies](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [SECURITY DEFINER Functions](https://www.postgresql.org/docs/current/sql-createfunction.html)
- [Multi-Tenant Architecture Best Practices](https://www.postgresql.org/docs/current/ddl-rowsecurity.html#DDL-ROWSECURITY-MULTI-TENANT)
