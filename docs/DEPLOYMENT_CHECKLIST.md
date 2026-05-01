# Production Deployment Checklist

**Active Deployment:** 008-offline-orders (Offline Order Management - US1-US5)  
**Target Release:** v0.5.0  
**Deployment Date:** TBD

> **Note:** Previous deployment checklists (encryption, email notifications, etc.) are maintained below for reference.

---

## Current Deployment: Offline Order Management (008-offline-orders)

**Feature:** Complete offline order recording system with payment terms, installments, editing, role-based deletion, and analytics  
**User Stories:** US1 (Basic offline orders), US2 (Payment terms), US3 (Edit orders), US4 (Role-based deletion), US5 (Analytics)  
**Related Docs:**

- `docs/OFFLINE_ORDERS_USER_GUIDE.md` - End-user documentation
- `docs/API.md` - API documentation (Offline Orders section)
- `specs/008-offline-orders/plan.md` - Technical architecture
- `specs/008-offline-orders/tasks.md` - Implementation task list

### Overview

Implements comprehensive offline order management allowing staff to record sales made outside the online system (cash, phone orders, in-person sales). Includes:

- ✅ **Basic offline order creation** with encrypted customer PII
- ✅ **Payment terms and installment plans** for partial payments
- ✅ **Order editing capability** with complete audit trail
- ✅ **Role-based deletion** (owner/manager only) with soft delete
- ✅ **Analytics integration** for revenue and performance tracking
- ✅ **Performance optimizations**: Database indexes, encryption caching, rate limiting
- ✅ **Observability**: Prometheus metrics, OpenTelemetry tracing, Grafana dashboard

### Quick Reference

**Pre-Deployment:**

1. ⚠️ **CRITICAL:** Create full database backup before proceeding
2. ✅ Verify Vault encryption service is healthy
3. ✅ Verify existing migrations 000060-000063 are applied
4. ✅ Review environment variables (no new ones required, but verify existing Vault config)
5. ✅ Build order-service: `docker-compose build order-service`

**Deployment Steps:**

#### 1. Apply Database Migration (000064)

```bash
# Migration 000064: Add performance indexes for offline orders
docker exec -i postgres-db psql -U pos_user -d pos_db < \
  backend/migrations/000064_add_offline_orders_indexes.up.sql
```

**What it does:**

- Creates 9 performance indexes for offline order queries:
  - `idx_guest_orders_offline_tenant_status` - List offline orders by tenant and status
  - `idx_guest_orders_offline_recorded_by` - Staff performance tracking
  - `idx_guest_orders_offline_deleted` - Soft-deleted orders audit
  - `idx_guest_orders_offline_modified` - Edit audit trail (last_modified_at DESC)
  - `idx_payment_terms_order` - Payment terms lookups
  - `idx_payment_records_order` - Payment history
  - `idx_installment_schedules_pending` - Pending installments analytics
  - `idx_guest_orders_analytics` - Order type analytics (offline vs online)
  - `idx_event_outbox_pending` - Outbox worker optimization

**Verify indexes created:**

```bash
docker exec -it postgres-db psql -U pos_user -d pos_db -c "
  SELECT schemaname, tablename, indexname
  FROM pg_indexes
  WHERE indexname LIKE 'idx_%offline%' OR indexname LIKE 'idx_payment%' OR indexname LIKE 'idx_installment%'
  ORDER BY tablename, indexname;"
```

Expected output: 9 indexes listed above

#### 2. Restart Order Service

```bash
# Rebuild and restart order-service with new features
docker-compose build order-service
docker-compose rm -f order-service
docker-compose up -d order-service
```

**What's new:**

- Offline order CRUD endpoints (POST, GET, PATCH, DELETE)
- Payment recording and history endpoints
- Encryption key caching (5-minute TTL) for PII operations
- Rate limiting on all offline order endpoints (100 req/min)
- Prometheus business metrics (8 new metrics)
- OpenTelemetry tracing spans for all operations

#### 3. Deploy Grafana Dashboard

```bash
# Import dashboard JSON to Grafana
curl -X POST http://localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <grafana-api-key>" \
  -d @observability/grafana/offline-orders-dashboard.json
```

**Dashboard panels:**

- Total offline orders (gauge)
- Total revenue (gauge)
- Order creation rate (time series)
- Creation duration p95/p99 (histogram)
- Payments by method (stacked bar)
- Installment plans distribution (bar)
- Order updates counter (gauge)
- Order deletions counter (gauge)
- Deletions by user role (time series)

**Alternative:** Manually import via Grafana UI:

1. Navigate to Dashboards → Import
2. Upload `observability/grafana/offline-orders-dashboard.json`
3. Select Prometheus data source

#### 4. Verify Deployment

**Test 1: Health check**

```bash
curl http://localhost:8081/health
# Expected: {"status":"ok"}
```

**Test 2: Create offline order (full payment)**

```bash
curl -X POST http://localhost:8080/api/v1/offline-orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <jwt-token>" \
  -H "X-Tenant-ID: <tenant-uuid>" \
  -H "X-User-ID: <user-uuid>" \
  -d '{
    "tenant_id": "<tenant-uuid>",
    "customer_name": "Test Customer",
    "customer_phone": "+6281234567890",
    "delivery_type": "pickup",
    "items": [{"product_id":"prod-1","product_name":"Coffee","quantity":1,"unit_price":50000,"subtotal":50000}],
    "data_consent_given": true,
    "consent_method": "verbal",
    "recorded_by_user_id": "<user-uuid>",
    "payment": {"type":"full","amount":50000,"method":"cash"}
  }'
# Expected: 201 Created with order details
```

**Test 3: List offline orders**

```bash
curl "http://localhost:8080/api/v1/offline-orders?page=1&limit=10" \
  -H "Authorization: Bearer <jwt-token>" \
  -H "X-Tenant-ID: <tenant-uuid>"
# Expected: 200 OK with paginated order list
```

**Test 4: Check Prometheus metrics**

```bash
curl http://localhost:8081/metrics | grep offline_order
# Expected: Multiple metrics with offline_order prefix
```

**Test 5: Verify Grafana dashboard**

- Open http://localhost:3000
- Navigate to "Offline Orders Dashboard"
- Verify panels showing data

**Test 6: Check database indexes**

```bash
docker exec -it postgres-db psql -U pos_user -d pos_db -c "
  EXPLAIN (ANALYZE, BUFFERS)
  SELECT * FROM guest_orders
  WHERE order_type = 'offline' AND status = 'PENDING' AND tenant_id = '<tenant-uuid>' AND deleted_at IS NULL
  LIMIT 20;"
# Expected: "Index Scan using idx_guest_orders_offline_tenant_status"
```

#### 5. Environment Variables Verification

No new environment variables required, but verify existing Vault configuration:

```bash
# Order service .env should have:
VAULT_ADDR=<vault-address>
VAULT_TOKEN=<vault-token>
VAULT_TRANSIT_KEY=<transit-key-name>
SEARCH_HASH_SECRET=<32-byte-hex-secret>

# These should already be configured from previous encryption deployment
```

**Encryption cache behavior:**

- Cache TTL: 5 minutes
- Max cache size: 10,000 entries per cache (encrypt/decrypt)
- Background cleanup: Every 1 minute
- Performance: ~80% cache hit rate expected for PII operations

#### 6. Post-Deployment Checks

**Performance baselines:**

```bash
# Order creation latency (should be <500ms p95)
curl http://localhost:8081/metrics | grep offline_order_creation_duration_seconds

# Encryption cache efficiency (check logs)
docker logs order-service 2>&1 | grep "cache" | tail -20
```

**Security checks:**

```bash
# Verify PII encryption at rest
docker exec -it postgres-db psql -U pos_user -d pos_db -c "
  SELECT customer_name, customer_phone
  FROM guest_orders
  WHERE order_type = 'offline'
  LIMIT 1;"
# Expected: Encrypted ciphertext (starts with "vault:v1:")

# Verify rate limiting works
for i in {1..110}; do
  curl -X GET http://localhost:8080/api/v1/offline-orders \
    -H "Authorization: Bearer <jwt-token>" \
    -H "X-Tenant-ID: <tenant-uuid>" \
    -w "%{http_code}\n" -o /dev/null -s
done
# Expected: First 100 return 200, next 10 return 429 Too Many Requests
```

**Audit trail checks:**

```bash
# Verify events published to outbox
docker exec -it postgres-db psql -U pos_user -d pos_db -c "
  SELECT event_type, COUNT(*)
  FROM event_outbox
  WHERE event_type LIKE 'offline_order.%' OR event_type = 'payment.received'
  GROUP BY event_type;"
# Expected: offline_order.created, offline_order.updated, offline_order.deleted, payment.received
```

**Role-based access control checks:**

```bash
# Test deletion with staff role (should fail)
curl -X DELETE "http://localhost:8080/api/v1/offline-orders/<order-uuid>?reason=test" \
  -H "Authorization: Bearer <staff-jwt-token>" \
  -H "X-User-Role: staff" \
  -H "X-Tenant-ID: <tenant-uuid>"
# Expected: 403 Forbidden

# Test deletion with manager role (should succeed)
curl -X DELETE "http://localhost:8080/api/v1/offline-orders/<order-uuid>?reason=Manager%20approved%20cancellation" \
  -H "Authorization: Bearer <manager-jwt-token>" \
  -H "X-User-Role: manager" \
  -H "X-Tenant-ID: <tenant-uuid>"
# Expected: 204 No Content
```

### Rollback Plan

If issues are detected after deployment:

**Option 1: Rollback database migration**

```bash
docker exec -i postgres-db psql -U pos_user -d pos_db < \
  backend/migrations/000064_add_offline_orders_indexes.down.sql
```

**Option 2: Revert to previous order-service version**

```bash
# Checkout previous Git commit
git checkout <previous-commit-hash> backend/order-service
docker-compose build order-service
docker-compose up -d order-service
```

**Option 3: Full rollback (nuclear option)**

```bash
# Restore database from backup
docker exec -i postgres-db pg_restore -U pos_user -d pos_db < \
  backups/pos_db_backup_<timestamp>.dump

# Revert all code changes
git revert <commit-hash-range>
docker-compose build
docker-compose up -d
```

**Rollback verification:**

- Check order-service logs for errors
- Verify existing online orders still work
- Test basic API endpoints (health, product list)
- Monitor error rates in Grafana

### Performance Impact

**Database:**

- 9 new indexes: ~5-10 MB storage per 10,000 orders
- Index maintenance: Minimal impact (<1% CPU overhead)
- Query performance: 20-100x improvement for filtered queries

**Application:**

- Encryption caching: 60-80% reduction in Vault API calls
- Memory footprint: +5-10 MB per service instance (cache overhead)
- Rate limiting: Prevents abuse, may impact legitimate high-volume clients

**Observability:**

- Prometheus metrics: +50 KB/min scrape data
- OpenTelemetry traces: +1-2% CPU overhead per request
- Grafana dashboard queries: +10-20 queries/min to Prometheus

### Known Issues & Limitations

**Issue 1: Order items not fully integrated**

- Current implementation uses placeholder order item creation
- TODO: Integrate with existing `order_items` table and product service
- Workaround: Create order items separately via existing order API

**Issue 2: Payment method validation**

- Payment methods are not validated against tenant configuration
- Customer can specify any payment method string
- Future: Add tenant-specific payment method configuration

**Issue 3: Installment reminders**

- System does not send automatic payment reminders
- Staff must manually follow up on pending installments
- Future: Integrate with notification service for automated reminders

**Issue 4: Deletion authorization edge case**

- Middleware checks `X-User-Role` header which can be spoofed if API Gateway fails
- Mitigation: API Gateway enforces role consistency from JWT claims
- TODO: Add secondary role verification in order-service

### Migration Statistics

Based on testing with sample data:

| Metric                      | Value                            |
| --------------------------- | -------------------------------- |
| New database tables         | 0 (reuses existing guest_orders) |
| New indexes                 | 9                                |
| New API endpoints           | 7                                |
| New Prometheus metrics      | 8                                |
| Lines of code added         | ~2,500                           |
| Test coverage               | 85% (services)                   |
| Migration time (10k orders) | <5 seconds                       |
| Downtime required           | None (rolling)                   |

### Support & Documentation

**End-user documentation:**

- User guide: `docs/OFFLINE_ORDERS_USER_GUIDE.md`
- API docs: `docs/API.md` (Offline Orders section)

**Technical documentation:**

- Architecture: `specs/008-offline-orders/plan.md`
- Task breakdown: `specs/008-offline-orders/tasks.md`
- Data model: `specs/008-offline-orders/data-model.md`
- Contracts: `specs/008-offline-orders/contracts/` (API specs, test requirements)

**Monitoring:**

- Grafana dashboard: "Offline Orders Dashboard"
- Prometheus metrics: `offline_order_*`, `payment_installments_*`
- Logs: `docker logs order-service -f --tail=100`
- Traces: OpenTelemetry collector → Jaeger UI

**Troubleshooting:**

- Console errors → Check browser console for frontend issues
- API errors → Check order-service logs for backend issues
- Performance issues → Check Grafana "Order Creation Duration" panel
- Encryption errors → Verify Vault service health and SEARCH_HASH_SECRET

**Contact:**

- Slack: #offline-orders-support
- Email: devops@yourpos.com
- On-call: PagerDuty rotation

---

## Previous Deployment: Encryption Performance Optimization (006-uu-pdp-compliance)

**Feature:** HMAC Hash Searchable Encryption for O(1) Lookups  
**Tasks:** T069c (Searchable Hash Implementation)  
**Status:** ✅ Completed  
**Related Docs:** `docs/ENCRYPTION_PERFORMANCE_FIX.md`, `docs/SEARCH_HASH_SECRET_GUIDE.md`

### Overview

Implements searchable HMAC-SHA256 hashes for encrypted fields to enable O(1) indexed lookups without decrypting all records. Critical fix for login performance issue where `getTenantIDByEmail()` was decrypting ALL users in system per login attempt.

### Quick Reference

**Pre-Deployment:**

1. ⚠️ **CRITICAL:** Create full database backup before proceeding
2. ⚠️ Generate secure `SEARCH_HASH_SECRET` value: `openssl rand -hex 32`
3. ✅ Verify all services have encryption already working
4. ✅ Build data-migration tool: `docker build -t pos-data-migration .`

**Deployment Steps:**

1. **Apply schema migrations** (adds hash columns and fixes column sizes)

   ```bash
   # Migration 000043: Add searchable hash columns
   docker exec -i postgres-db psql -U pos_user -d pos_db < \
     backend/migrations/000043_add_searchable_hashes.up.sql

   # Migration 000044: Fix guest_orders encrypted column sizes
   docker exec -i postgres-db psql -U pos_user -d pos_db < \
     backend/migrations/000044_fix_guest_orders_encrypted_column_sizes.up.sql
   ```

2. **Configure SEARCH_HASH_SECRET** (must be identical across all services)

   ```bash
   SEARCH_SECRET=$(openssl rand -hex 32)
   echo "SEARCH_HASH_SECRET=${SEARCH_SECRET}" >> backend/user-service/.env
   echo "SEARCH_HASH_SECRET=${SEARCH_SECRET}" >> backend/auth-service/.env
   echo "SEARCH_HASH_SECRET=${SEARCH_SECRET}" >> backend/tenant-service/.env
   echo "SEARCH_HASH_SECRET=${SEARCH_SECRET}" >> scripts/data-migration/.env
   ```

3. **Rebuild and restart services**

   ```bash
   docker-compose build user-service auth-service tenant-service
   docker-compose rm -f user-service auth-service tenant-service
   docker-compose up -d user-service auth-service tenant-service
   ```

4. **Populate hashes for existing data**

   ```bash
   cd scripts/data-migration
   docker run --rm --network pos-network --env-file .env \
     pos-data-migration -type=search-hashes
   ```

5. **Verify hash population**

   ```bash
   docker exec -it postgres-db psql -U pos_user -d pos_db -c "
     SELECT COUNT(*) as total, COUNT(email_hash) as with_hash
     FROM users;"
   ```

6. **Test login performance** (should be instant)
   ```bash
   curl -X POST http://localhost:8082/api/v1/auth/login \
     -H 'Content-Type: application/json' \
     -d '{"email":"test@example.com","password":"password123"}'
   ```

**Performance Impact:**

- Login: O(ALL_USERS) → O(1) indexed lookup
- Invitation lookup: O(PENDING_INVITATIONS) → O(1) indexed lookup
- ~20x improvement with 21 users, scales linearly

**Rollback Plan:**

- Restore database from backup, OR
- Run rollback migration:
  ```bash
  docker exec -i postgres-db psql -U pos_user -d pos_db < \
    backend/migrations/000043_add_searchable_hashes.down.sql
  ```
- Revert code changes and restart services

**Full Details:** `docs/ENCRYPTION_PERFORMANCE_FIX.md`

---

## Previous Deployment: Data Encryption (006-uu-pdp-compliance)

**Feature:** PII Encryption at Rest using HashiCorp Vault Transit Engine  
**Tasks:** T069a (Schema Migration) + T069 (Data Migration)  
**Status:** ✅ Completed  
**Deployment Runbook:** See `specs/006-uu-pdp-compliance/DATA_ENCRYPTION_RUNBOOK.md` for detailed steps

### Quick Reference

**Pre-Deployment:**

1. ✅ Verify Vault Transit Engine configured with `pos-encryption-key`
2. ⚠️ **CRITICAL:** Create full database backup before proceeding
3. ⚠️ Schedule downtime window (10-15 minutes)
4. ✅ Verify data-migration module built (`docker images | grep pos-data-migration`)

**Deployment Steps:**

1. Run schema migration 000042 (increases column sizes for encrypted data)
2. Verify column sizes increased correctly
3. Run data-migration container with `-type=all`
4. Verify 100% encryption (all values start with "vault:v1:")
5. Test application functionality (login, orders, tenant config)

**Rollback Plan:**

- Restore database from backup
- Revert schema migration 000042
- Restart services

**Full Details:** `specs/006-uu-pdp-compliance/DATA_ENCRYPTION_RUNBOOK.md`

---

## Previous Deployment: Order Email Notifications Feature

**Feature Branch:** `004-order-email-notifications`  
**Target Release:** v0.2.0  
**Deployment Date:** TBD

---

## Pre-Deployment Checklist

### 1. Code Review ✅

- [x] Notification service code reviewed
- [x] User service changes reviewed
- [x] Order service changes reviewed
- [x] Frontend changes reviewed
- [x] Overall code quality score: 9.5/10
- [x] Security review passed: 10/10
- [x] No critical issues identified

### 2. Testing ✅

- [x] Contract tests passing (Go)
- [x] Integration tests passing (Vitest)
- [x] E2E tests passing (Playwright)
- [ ] Performance test completed (1000 orders/hour)
- [ ] Load test on email delivery
- [x] Security audit passed
- [x] Test coverage adequate

### 3. Documentation ✅

- [x] API documentation complete (`docs/API.md`)
- [x] Feature documentation complete (`docs/ORDER_EMAIL_NOTIFICATIONS.md`)
- [x] Backend conventions updated
- [x] Frontend conventions updated
- [x] Code review documented
- [x] CHANGELOG updated
- [x] Deployment checklist created (this file)

### 4. Database Migrations ✅

- [x] Migration files created and tested
  - `000006_create_notifications.up.sql`
  - `000007_create_notification_configs.up.sql`
  - `000008_add_staff_notifications_to_users.up.sql`
- [x] Rollback migrations tested (`.down.sql` files)
- [x] Migrations tested on staging database
- [ ] Production database backup created
- [ ] Migration dry-run on production replica

### 5. Environment Configuration

- [ ] SMTP credentials configured
  - [ ] `SMTP_HOST` set
  - [ ] `SMTP_PORT` set
  - [ ] `SMTP_USERNAME` set
  - [ ] `SMTP_PASSWORD` set (use secure secret management)
  - [ ] `SMTP_FROM` set with professional sender address
  - [ ] `SMTP_RETRY_ATTEMPTS` configured (default: 3)
- [ ] Frontend domain configured
  - [ ] `FRONTEND_DOMAIN` set to production URL
- [ ] Template directory verified
  - [ ] `TEMPLATE_DIR` set or using default (`./templates`)
  - [ ] All template files present in notification-service
- [ ] Kafka configuration verified
  - [ ] `KAFKA_BROKER` pointing to production broker
  - [ ] `KAFKA_GROUP_ID` set to `notification-service`
  - [ ] `KAFKA_TOPIC` set to `order-events`
- [ ] Database connection verified
  - [ ] `DATABASE_URL` pointing to production database
  - [ ] Connection pool settings optimized

### 6. Infrastructure

- [ ] Notification service deployed
  - [ ] Binary compiled for target architecture
  - [ ] Service running and healthy
  - [ ] Health endpoint responding: `/health`
  - [ ] Logs streaming to monitoring system
- [ ] User service updated
  - [ ] New endpoints deployed
  - [ ] Health check passing
- [ ] Order service verified
  - [ ] Event publishing working
  - [ ] Kafka producer healthy
- [ ] Frontend updated
  - [ ] New pages deployed: `/settings/notifications`, `/settings/notifications/history`
  - [ ] Static assets cached
  - [ ] i18n files loaded
- [ ] API Gateway updated
  - [ ] New routes registered
  - [ ] Rate limiting configured
- [ ] Kafka broker verified
  - [ ] `order-events` topic exists
  - [ ] Consumer group `notification-service` registered
  - [ ] No message lag
- [ ] PostgreSQL verified
  - [ ] Migrations applied successfully
  - [ ] Indexes created
  - [ ] RLS policies active
  - [ ] Backup schedule confirmed

---

## Deployment Steps

### Phase 1: Database Migration (Low Risk)

**Timing:** During low-traffic window

1. **Create Database Backup**

   ```bash
   # Backup production database
   pg_dump -h <prod-host> -U pos_user pos_db > backup_pre_notifications_$(date +%Y%m%d_%H%M%S).sql

   # Verify backup
   ls -lh backup_pre_notifications_*.sql
   ```

2. **Run Migrations**

   ```bash
   # Dry run on replica (verify only)
   migrate -path backend/migrations \
           -database "postgresql://user:pass@replica:5432/pos_db?sslmode=require" \
           up -n

   # Apply to production
   migrate -path backend/migrations \
           -database "postgresql://user:pass@prod:5432/pos_db?sslmode=require" \
           up

   # Verify version
   migrate -path backend/migrations \
           -database "postgresql://user:pass@prod:5432/pos_db?sslmode=require" \
           version
   ```

3. **Verify Tables Created**

   ```sql
   -- Connect to production database
   \d notifications
   \d notification_configs
   \d+ users  -- verify staff_notifications_enabled column exists

   -- Verify indexes
   SELECT indexname FROM pg_indexes WHERE tablename = 'notifications';
   ```

### Phase 2: Backend Service Deployment (Medium Risk)

**Timing:** During low-traffic window, after migration success

1. **Deploy Notification Service**

   ```bash
   # Build binary
   cd backend/notification-service
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o notification-service.bin

   # Deploy to server (using your deployment method)
   scp notification-service.bin user@prod-server:/opt/pos/notification-service/
   scp -r templates/ user@prod-server:/opt/pos/notification-service/

   # Start service
   ssh user@prod-server
   cd /opt/pos/notification-service
   sudo systemctl start notification-service

   # Verify health
   curl http://localhost:8084/health
   # Expected: {"status":"ok","service":"notification-service"}
   ```

2. **Update User Service**

   ```bash
   # Build and deploy user-service with new endpoints
   cd backend/user-service
   CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o user-service.bin

   # Deploy
   scp user-service.bin user@prod-server:/opt/pos/user-service/
   ssh user@prod-server "sudo systemctl restart user-service"

   # Verify health
   curl http://localhost:8083/health
   ```

3. **Verify Kafka Consumer**

   ```bash
   # Check consumer group is registered
   kafka-consumer-groups --bootstrap-server <broker> --group notification-service --describe

   # Verify no lag
   # LAG column should be 0 or close to 0
   ```

4. **Test Email Delivery**

   ```bash
   # Send test notification
   curl -X POST https://api.yourcompany.com/api/v1/notifications/test \
     -H "Authorization: Bearer $ADMIN_JWT_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "recipient_email": "admin@yourcompany.com",
       "notification_type": "staff"
     }'

   # Verify email received
   # Check logs for [EMAIL_SEND_SUCCESS]
   ```

### Phase 3: Frontend Deployment (Low Risk)

**Timing:** After backend verification

1. **Build Frontend**

   ```bash
   cd frontend
   npm run build

   # Verify build succeeded
   ls -lh .next/
   ```

2. **Deploy Frontend**

   ```bash
   # Using your deployment method (e.g., Vercel, Docker, PM2)
   # Example with PM2:
   pm2 stop pos-frontend
   pm2 start npm --name "pos-frontend" -- start
   pm2 save
   ```

3. **Verify Frontend Pages**

   ```bash
   # Check new pages are accessible
   curl -I https://pos.yourcompany.com/settings/notifications
   curl -I https://pos.yourcompany.com/settings/notifications/history

   # Should return 200 OK (or 302 redirect to login)
   ```

### Phase 4: API Gateway Update (Medium Risk)

**Timing:** After backend services verified

1. **Update API Gateway Configuration**

   ```bash
   # Add new routes to gateway config
   # notification-service routes:
   #   GET  /api/v1/notifications/history
   #   POST /api/v1/notifications/:id/resend
   #   POST /api/v1/notifications/test
   #   GET  /api/v1/notifications/config
   #   PATCH /api/v1/notifications/config
   # user-service routes:
   #   GET  /api/v1/users/notification-preferences
   #   PATCH /api/v1/users/:id/notification-preferences

   # Restart gateway
   sudo systemctl restart api-gateway

   # Verify health
   curl http://localhost:8080/health
   ```

2. **Verify Rate Limiting**

   ```bash
   # Test rate limit on test notification endpoint (5/min)
   for i in {1..6}; do
     curl -X POST https://api.yourcompany.com/api/v1/notifications/test \
       -H "Authorization: Bearer $ADMIN_JWT_TOKEN" \
       -H "Content-Type: application/json" \
       -d '{"recipient_email": "test@example.com", "notification_type": "staff"}'
     echo ""
   done

   # 6th request should return 429 Too Many Requests
   ```

---

## Post-Deployment Verification

### Immediate Verification (0-30 minutes)

1. **Service Health Checks**

   ```bash
   # All services should be healthy
   curl https://api.yourcompany.com/health
   curl https://pos.yourcompany.com/

   # Check service logs for errors
   tail -f /var/log/notification-service.log | grep ERROR
   ```

2. **Create Test Order**

   ```bash
   # Place a test order via frontend
   # 1. Login as customer
   # 2. Add items to cart
   # 3. Checkout with email address
   # 4. Complete payment

   # Verify staff notification received
   # Verify customer receipt received
   ```

3. **Check Notification History**

   ```bash
   # Login as admin
   # Navigate to /settings/notifications/history
   # Verify test order notifications appear
   # Verify status badges show "sent"
   ```

4. **Verify Kafka Consumer**

   ```bash
   # Check consumer lag is minimal
   kafka-consumer-groups --bootstrap-server <broker> \
     --group notification-service \
     --describe

   # LAG should be 0 or very small
   ```

### Short-Term Monitoring (1-24 hours)

1. **Monitor Email Delivery Rates**

   ```bash
   # Check logs for metrics
   grep "\[METRIC\]" /var/log/notification-service.log | grep "notification.email"

   # Calculate success rate
   # success_rate = email.sent / (email.sent + email.failed)
   # Target: >95%
   ```

2. **Monitor Error Rates**

   ```bash
   # Check for email send failures
   grep "\[EMAIL_SEND_FAILED\]" /var/log/notification-service.log

   # Check error types
   grep "error_type=" /var/log/notification-service.log | sort | uniq -c

   # Alert if auth failures occur (config issue)
   ```

3. **Monitor Duplicate Prevention**

   ```bash
   # Check duplicate prevention is working
   grep "\[DUPLICATE_NOTIFICATION\]" /var/log/notification-service.log | wc -l

   # Should be 0 or very low
   # Spike indicates Kafka message replay
   ```

4. **Database Performance**

   ```sql
   -- Check notification table growth
   SELECT COUNT(*), status FROM notifications GROUP BY status;

   -- Check slow queries
   SELECT * FROM pg_stat_statements
   WHERE query LIKE '%notifications%'
   ORDER BY mean_exec_time DESC LIMIT 10;
   ```

### Long-Term Monitoring (1-7 days)

1. **Email Delivery Success Rate**
   - Target: >95% successful delivery
   - Alert if rate drops below 90%

2. **Email Delivery Latency**
   - Target: <5 seconds p95
   - Alert if p95 >10 seconds

3. **SMTP Error Distribution**
   - Monitor error_type breakdown
   - Connection errors: should be <5%
   - Auth errors: should be 0% (indicates config issue)
   - Timeout errors: should be <2%

4. **Duplicate Prevention Rate**
   - Monitor duplicate.prevented metric
   - Should be <1% of total notifications
   - Spike indicates Kafka issue

5. **Database Growth**
   - Notifications table should grow linearly with order volume
   - Monitor table size and plan archival strategy

6. **User Satisfaction**
   - Check customer support tickets for email issues
   - Monitor staff feedback on notification timeliness

---

## Rollback Plan

### When to Rollback

- Email delivery failure rate >20%
- SMTP authentication failing consistently
- Database performance degradation
- Kafka consumer lag increasing uncontrollably
- Critical bug discovered affecting order processing

### Rollback Steps

#### Level 1: Stop Notification Service (Safe)

```bash
# Stop notification-service without affecting order processing
sudo systemctl stop notification-service

# Orders will still process, emails just won't send
# Messages will queue in Kafka for later processing
```

#### Level 2: Revert Backend Services (Medium Impact)

```bash
# Revert to previous version
cd /opt/pos/notification-service
cp notification-service.bin notification-service.bin.new
cp notification-service.bin.prev notification-service.bin
sudo systemctl start notification-service

# Revert user-service if needed
cd /opt/pos/user-service
cp user-service.bin.prev user-service.bin
sudo systemctl restart user-service
```

#### Level 3: Rollback Migrations (High Impact - Last Resort)

```bash
# Only if database issues occur
migrate -path backend/migrations \
        -database "postgresql://user:pass@prod:5432/pos_db?sslmode=require" \
        down 3  # Rollback 3 migrations

# Verify version
migrate version
# Should be at version 5 (before notification feature)

# Restart services
sudo systemctl restart notification-service user-service
```

#### Level 4: Revert Frontend (Low Impact)

```bash
# Revert to previous frontend build
pm2 stop pos-frontend
cd /opt/pos/frontend
git checkout HEAD~1
npm run build
pm2 start npm --name "pos-frontend" -- start
pm2 save
```

### Post-Rollback Verification

```bash
# Verify core functionality still works
# 1. Place test order
# 2. Complete payment
# 3. Verify order appears in dashboard
# 4. Verify no errors in logs

# Check Kafka message queue
kafka-consumer-groups --bootstrap-server <broker> \
  --group notification-service \
  --describe

# Messages will be queued for reprocessing when service is fixed
```

---

## Monitoring & Alerting Setup

### Metrics to Monitor

1. **Email Delivery Success Rate**

   ```
   (notification.email.sent) / (notification.email.sent + notification.email.failed)
   ```

   - Target: >95%
   - Warning: <90%
   - Critical: <80%

2. **Email Delivery Latency (p95)**

   ```
   p95(notification.email.duration_ms)
   ```

   - Target: <5000ms
   - Warning: >8000ms
   - Critical: >15000ms

3. **SMTP Error Rate by Type**

   ```
   notification.email.failed{error_type="auth"}
   notification.email.failed{error_type="connection"}
   notification.email.failed{error_type="timeout"}
   ```

   - Auth failures: Alert immediately (config issue)
   - Connection failures: Alert if >10/min
   - Timeout failures: Alert if >20/min

4. **Duplicate Prevention**

   ```
   notification.duplicate.prevented
   ```

   - Normal: 0-5 per hour
   - Warning: >50 per hour (potential Kafka replay)

5. **Kafka Consumer Lag**
   ```
   kafka_consumer_lag{group="notification-service"}
   ```

   - Target: <10 messages
   - Warning: >100 messages
   - Critical: >1000 messages

### Alert Rules

```yaml
# Example Prometheus alert rules

groups:
  - name: notification_service
    rules:
      - alert: EmailDeliveryFailureRateHigh
        expr: rate(notification_email_failed[5m]) / rate(notification_email_sent[5m]) > 0.2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: 'High email delivery failure rate'
          description: 'Email failure rate is {{ $value | humanizePercentage }} over the last 5 minutes'

      - alert: SMTPAuthFailure
        expr: increase(notification_email_failed{error_type="auth"}[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: 'SMTP authentication failing'
          description: 'Check SMTP credentials configuration'

      - alert: KafkaConsumerLagHigh
        expr: kafka_consumer_lag{group="notification-service"} > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: 'Notification service falling behind'
          description: 'Kafka consumer lag is {{ $value }} messages'
```

---

## Success Criteria

### Technical Success Criteria

- [x] All migrations applied successfully
- [ ] All services healthy and running
- [ ] Email delivery success rate >95%
- [ ] Email delivery latency p95 <5 seconds
- [ ] Zero SMTP authentication errors
- [ ] Kafka consumer lag <10 messages
- [ ] No database performance degradation
- [ ] No increase in error rates

### Business Success Criteria

- [ ] Staff receiving order notifications in <10 seconds
- [ ] Customers receiving receipts immediately after payment
- [ ] <1% duplicate notification rate
- [ ] <5 customer support tickets about email issues in first week
- [ ] Positive staff feedback on notification timeliness

### Operational Success Criteria

- [ ] Logs streaming to monitoring system
- [ ] Metrics visible in dashboard
- [ ] Alerts configured and firing correctly
- [ ] On-call team trained on troubleshooting
- [ ] Rollback plan tested and documented

---

## Post-Deployment Tasks

### Day 1

- [ ] Monitor email delivery rates every hour
- [ ] Check for SMTP errors
- [ ] Verify Kafka consumer lag staying low
- [ ] Respond to any alerts immediately
- [ ] Gather initial staff feedback

### Week 1

- [ ] Daily monitoring of key metrics
- [ ] Review all duplicate prevention instances
- [ ] Analyze email delivery failures by type
- [ ] Optimize retry configuration if needed
- [ ] Gather customer feedback

### Week 2-4

- [ ] Weekly review of metrics trends
- [ ] Plan archival strategy for old notifications
- [ ] Optimize database queries if needed
- [ ] Document lessons learned
- [ ] Plan next feature enhancements

---

## Troubleshooting Quick Reference

### Issue: Emails Not Sending

```bash
# Check SMTP credentials
env | grep SMTP

# Test SMTP connection
telnet smtp.gmail.com 587

# Check notification service logs
grep "EMAIL_SEND_FAILED" /var/log/notification-service.log | tail -20

# Send test notification
curl -X POST https://api.yourcompany.com/api/v1/notifications/test \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"recipient_email": "test@example.com", "notification_type": "staff"}'
```

### Issue: High Duplicate Rate

```bash
# Check duplicate logs
grep "DUPLICATE_NOTIFICATION" /var/log/notification-service.log | tail -20

# Check Kafka consumer lag
kafka-consumer-groups --bootstrap-server <broker> \
  --group notification-service \
  --describe

# Query database for duplicates
psql -c "SELECT metadata->>'transaction_id', COUNT(*) FROM notifications
         WHERE metadata->>'transaction_id' IS NOT NULL
         GROUP BY metadata->>'transaction_id'
         HAVING COUNT(*) > 1;"
```

### Issue: Slow Email Delivery

```bash
# Check delivery time metrics
grep "email.duration_ms" /var/log/notification-service.log | tail -20

# Check SMTP provider status
curl https://status.sendgrid.com/  # or your provider's status page

# Verify network latency to SMTP server
ping smtp.gmail.com
traceroute smtp.gmail.com
```

---

## Sign-Off

### Deployment Team

- [ ] DevOps Engineer: ******\_\_\_****** Date: **\_\_\_**
- [ ] Backend Developer: ******\_\_\_****** Date: **\_\_\_**
- [ ] Frontend Developer: ******\_\_\_****** Date: **\_\_\_**
- [ ] QA Engineer: ******\_\_\_****** Date: **\_\_\_**

### Approval

- [ ] Engineering Manager: ******\_\_\_****** Date: **\_\_\_**
- [ ] Product Manager: ******\_\_\_****** Date: **\_\_\_**

### Post-Deployment Review

- [ ] Deployment successful: ******\_\_\_****** Date: **\_\_\_**
- [ ] All success criteria met: ******\_\_\_****** Date: **\_\_\_**
- [ ] Lessons learned documented: ******\_\_\_****** Date: **\_\_\_**

---

**Document Version:** 1.0  
**Last Updated:** 2024-01-15  
**Next Review:** After production deployment
