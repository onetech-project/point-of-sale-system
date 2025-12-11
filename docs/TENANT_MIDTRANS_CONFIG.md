# Tenant-Specific Midtrans Configuration

**Feature**: Multi-tenant Midtrans payment gateway configuration  
**Version**: 2.0  
**Date**: December 7, 2025  
**Status**: ✅ Implemented and Deployed

## Overview

Each tenant in the POS system can now configure their own Midtrans payment gateway credentials. This enables true multi-tenant isolation where each business uses their own merchant account for payment processing.

## Architecture

### Database Schema

Migration `000021_add_midtrans_config_to_tenant_configs` adds the following columns to `tenant_configs`:

```sql
ALTER TABLE tenant_configs
ADD COLUMN midtrans_server_key TEXT,
ADD COLUMN midtrans_client_key TEXT,
ADD COLUMN midtrans_merchant_id TEXT,
ADD COLUMN midtrans_environment VARCHAR(20) DEFAULT 'sandbox' CHECK (
    midtrans_environment IN ('sandbox', 'production')
);
```

### Backend Components

#### 1. Tenant Service (Port 8084)

**Updated Files:**
- `backend/tenant-service/src/repository/tenant_config_repository.go` - Extended to handle Midtrans fields
- `backend/tenant-service/src/services/tenant_config_service.go` - Added GetMidtransConfig/UpdateMidtransConfig
- `backend/tenant-service/api/tenant_config_handler.go` - New endpoints for Midtrans config
- `backend/tenant-service/main.go` - Routes updated to match `/api/v1` prefix pattern

**API Endpoints:**
- `GET /api/v1/admin/tenants/:tenant_id/midtrans-config` - Retrieve tenant's Midtrans configuration
- `PATCH /api/v1/admin/tenants/:tenant_id/midtrans-config` - Update tenant's Midtrans credentials

**Request/Response Example:**
```json
// PATCH /api/v1/admin/tenants/123e4567-e89b-12d3-a456-426614174000/midtrans-config
{
  "server_key": "SB-Mid-server-xxx",
  "client_key": "SB-Mid-client-xxx",
  "merchant_id": "G123456789",
  "environment": "sandbox"
}

// Response
{
  "success": true,
  "message": "Midtrans configuration updated successfully"
}
```

#### 2. Order Service (Port 8087)

**Updated Files:**
- `backend/order-service/src/config/midtrans.go` - Major refactor for tenant-specific clients
- `backend/order-service/src/services/payment_service.go` - Dynamic credential fetching

**Key Changes:**

**Before (Global Configuration):**
```go
// Old: Single global client for all tenants
var SnapClient snap.Client
var CoreAPIClient coreapi.Client

func InitMidtrans() {
    SnapClient.New(serverKey, midtrans.Sandbox)
}
```

**After (Tenant-Specific Configuration):**
```go
// New: Dynamic client creation per tenant
func GetSnapClientForTenant(ctx context.Context, tenantID string) (*snap.Client, error) {
    config := fetchTenantMidtransConfig(ctx, tenantID)
    client := snap.Client{}
    client.New(config.ServerKey, config.Environment)
    return &client, nil
}

func GetCoreAPIClientForTenant(ctx context.Context, tenantID string) (*coreapi.Client, error) {
    config := fetchTenantMidtransConfig(ctx, tenantID)
    client := coreapi.Client{}
    client.New(config.ServerKey, config.Environment)
    return &client, nil
}

func fetchTenantMidtransConfig(ctx context.Context, tenantID string) (*MidtransConfig, error) {
    // HTTP call to tenant-service to fetch credentials
    resp, err := http.Get(fmt.Sprintf("%s/api/v1/admin/tenants/%s/midtrans-config", 
        tenantServiceURL, tenantID))
    // Parse response and return config
}
```

**Payment Flow:**
```go
// CreateQRISCharge now fetches tenant-specific client
func (s *PaymentService) CreateQRISCharge(ctx context.Context, order *models.Order) error {
    // Get tenant-specific Snap client
    snapClient, err := config.GetSnapClientForTenant(ctx, order.TenantID)
    
    // Create charge with tenant's credentials
    resp, err := snapClient.CreateTransaction(chargeReq)
}
```

**Webhook Verification:**
```go
// VerifySignature now uses tenant-specific server key
func (s *PaymentService) VerifySignature(ctx context.Context, tenantID, orderID, statusCode, grossAmount, serverKey string) bool {
    // Fetch tenant-specific server key from tenant-service
    tenantServerKey := config.GetMidtransServerKeyForTenant(ctx, tenantID)
    
    // Verify signature using tenant's key
    signature := sha512(orderID + statusCode + grossAmount + tenantServerKey)
    return signature == receivedSignature
}
```

#### 3. API Gateway (Port 8080)

**Updated Files:**
- `api-gateway/main.go` - Wildcard proxy routing for tenant config endpoints

**Routes:**
```go
// Admin tenant configuration routes (owner only)
adminTenantConfig := protected.Group("/api/v1/admin/tenants")
adminTenantConfig.Use(middleware.RBACMiddleware(middleware.RoleOwner))
adminTenantConfig.Any("/*", proxyWildcard(tenantServiceURL))
```

**Authentication & Authorization:**
- JWT token required (owner role only)
- Tenant scope validation via `X-Tenant-ID` header
- RBAC middleware enforces owner-only access

### Frontend Components

#### 1. Payment Settings Page

**File:** `frontend/app/settings/payment/page.tsx`

**Features:**
- Environment selector (Sandbox/Production)
- Secure input fields with show/hide toggles
- Configuration status indicator
- Real-time validation
- Save/Update functionality

**UI Preview:**
```
┌─────────────────────────────────────────┐
│ Payment Settings                         │
├─────────────────────────────────────────┤
│ Environment: [Sandbox ▼]                │
│                                          │
│ Server Key: [SB-Mid-server-***] [👁]    │
│ Client Key: [SB-Mid-client-***] [👁]    │
│ Merchant ID: [G123456789]               │
│                                          │
│ Status: ✅ Configured                   │
│                                          │
│ [Save Configuration]                    │
└─────────────────────────────────────────┘
```

**Implementation:**
```typescript
const handleSubmit = async (e: React.FormEvent) => {
  e.preventDefault();
  
  const response = await axios.patch(
    `/api/v1/admin/tenants/${tenantId}/midtrans-config`,
    {
      server_key: serverKey,
      client_key: clientKey,
      merchant_id: merchantId,
      environment: environment,
    }
  );
  
  if (response.data.success) {
    toast.success('Payment configuration saved!');
  }
};
```

#### 2. Settings Navigation

**File:** `frontend/app/settings/page.tsx`

**Updated:** Removed `comingSoon` flag from Payment Settings card to activate the link.

## Migration Guide

### 1. Database Migration

```bash
cd backend/migrations
migrate -path . -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" up
```

**Verification:**
```sql
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'tenant_configs' 
AND column_name LIKE 'midtrans%';
```

**Expected Output:**
```
      column_name        |     data_type      
-------------------------+-------------------
 midtrans_server_key     | text
 midtrans_client_key     | text
 midtrans_merchant_id    | text
 midtrans_environment    | character varying
```

### 2. Service Deployment

```bash
# Stop services
./scripts/stop-all.sh

# Rebuild affected services
cd backend/tenant-service && go build -o tenant-service.bin main.go
cd ../order-service && go build -o order-service.bin main.go
cd ../../api-gateway && go build -o api-gateway.bin main.go

# Start services
./scripts/start-all.sh
```

### 3. Rollback Procedure

If issues arise, rollback the migration:

```bash
cd backend/migrations
migrate -path . -database "postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable" down 1
```

This will remove the Midtrans columns from `tenant_configs` table.

## Configuration

### Environment Variables

**Order Service (.env):**
```env
# Tenant-specific credentials fetched from tenant-service
TENANT_SERVICE_URL=http://localhost:8084

# Fallback credentials (optional, for testing only)
MIDTRANS_SERVER_KEY=SB-Mid-server-fallback
MIDTRANS_CLIENT_KEY=SB-Mid-client-fallback
MIDTRANS_ENVIRONMENT=sandbox
```

**Tenant Service (.env):**
```env
# No Midtrans credentials needed - stored in database
PORT=8084
DATABASE_URL=postgresql://pos_user:pos_password@localhost:5432/pos_db?sslmode=disable
```

## Usage Guide

### For Tenant Owners

**Setup Payment Configuration:**

1. Login to admin dashboard
2. Navigate to **Settings → Payment Settings**
3. Select environment (Sandbox for testing, Production for live)
4. Enter your Midtrans credentials:
   - Server Key: From Midtrans dashboard → Settings → Access Keys
   - Client Key: From Midtrans dashboard → Settings → Access Keys
   - Merchant ID: From Midtrans dashboard → Settings → General Settings
5. Click **Save Configuration**

**Obtaining Midtrans Credentials:**

1. Sign up at [Midtrans Dashboard](https://dashboard.midtrans.com/)
2. Complete merchant verification
3. Navigate to Settings → Access Keys
4. Copy Server Key and Client Key
5. Navigate to Settings → General Settings for Merchant ID

### For Developers

**Fetching Tenant Credentials in Code:**

```go
import "github.com/point-of-sale-system/order-service/src/config"

// In your payment handler
snapClient, err := config.GetSnapClientForTenant(ctx, tenantID)
if err != nil {
    return fmt.Errorf("failed to get Midtrans client: %w", err)
}

// Use tenant-specific client
resp, err := snapClient.CreateTransaction(chargeReq)
```

**API Call Example:**

```bash
# Get tenant's Midtrans config (owner only)
curl -X GET \
  http://localhost:8080/api/v1/admin/tenants/123e4567-e89b-12d3-a456-426614174000/midtrans-config \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "X-Tenant-ID: 123e4567-e89b-12d3-a456-426614174000"

# Update tenant's Midtrans config
curl -X PATCH \
  http://localhost:8080/api/v1/admin/tenants/123e4567-e89b-12d3-a456-426614174000/midtrans-config \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "X-Tenant-ID: 123e4567-e89b-12d3-a456-426614174000" \
  -H "Content-Type: application/json" \
  -d '{
    "server_key": "SB-Mid-server-xxx",
    "client_key": "SB-Mid-client-xxx",
    "merchant_id": "G123456789",
    "environment": "sandbox"
  }'
```

## Security Considerations

### Production Deployment

1. **Encrypt Credentials at Rest:**
   ```go
   // Consider using encryption for sensitive fields
   encryptedKey, err := encrypt(config.ServerKey, masterKey)
   ```

2. **Audit Logging:**
   - All credential updates are logged with timestamp and user
   - Failed authentication attempts are recorded
   - Payment transactions include tenant_id for tracing

3. **Access Control:**
   - Only tenant owners can view/update credentials
   - API Gateway enforces RBAC (Role-Based Access Control)
   - Tenant isolation prevents cross-tenant access

4. **HTTPS in Production:**
   ```env
   # Use HTTPS for all API calls
   TENANT_SERVICE_URL=https://tenant-service.yourdomain.com
   ```

5. **Secret Management:**
   - Consider using HashiCorp Vault or AWS Secrets Manager
   - Rotate credentials periodically
   - Never commit credentials to version control

### Known Limitations

1. **Credential Storage:** Currently stored in plaintext in database. Recommend encrypting in production.
2. **Caching:** No caching of credentials yet. Consider Redis caching for performance.
3. **Validation:** No validation that credentials are valid Midtrans keys. Could add validation API call.

## Testing

### Unit Tests

**Test tenant-specific credential fetching:**
```go
func TestGetSnapClientForTenant(t *testing.T) {
    ctx := context.Background()
    tenantID := "test-tenant-123"
    
    client, err := config.GetSnapClientForTenant(ctx, tenantID)
    assert.NoError(t, err)
    assert.NotNil(t, client)
}
```

### Integration Tests

**Test complete payment flow with tenant credentials:**
```go
func TestPaymentFlowWithTenantCredentials(t *testing.T) {
    // 1. Create test tenant with Midtrans config
    // 2. Create order for that tenant
    // 3. Initiate payment (should use tenant's credentials)
    // 4. Simulate webhook callback
    // 5. Verify order status updated
}
```

### Manual Testing

1. **Setup Test Tenant:**
   ```sql
   UPDATE tenant_configs 
   SET midtrans_server_key = 'SB-Mid-server-test',
       midtrans_client_key = 'SB-Mid-client-test',
       midtrans_merchant_id = 'G000000001',
       midtrans_environment = 'sandbox'
   WHERE tenant_id = 'test-tenant-uuid';
   ```

2. **Create Test Order:**
   - Access tenant's public menu: `http://localhost:3000/menu/test-tenant-uuid`
   - Add items to cart
   - Proceed to checkout
   - Select delivery type and provide details

3. **Verify Payment Initiation:**
   - Check order-service logs for tenant credential fetch
   - Verify Midtrans Snap API called with tenant's server key
   - Confirm QR code generated

4. **Test Webhook:**
   ```bash
   curl -X POST http://localhost:8087/payments/midtrans/notification \
     -H "Content-Type: application/json" \
     -d '{
       "transaction_status": "settlement",
       "order_id": "GO-123456",
       "gross_amount": "100000",
       "signature_key": "calculated_signature"
     }'
   ```

## Troubleshooting

### Issue: 401 Unauthorized on Midtrans Config Endpoints

**Symptoms:** Frontend receives 401 when calling `/api/v1/admin/tenants/:tenant_id/midtrans-config`

**Solution:**
- Verify tenant-service routes include `/api/v1` prefix
- Check API Gateway wildcard routing
- Ensure JWT token is valid and user has owner role

**Verification:**
```bash
# Check tenant-service routes
curl http://localhost:8084/api/v1/admin/tenants/test-uuid/midtrans-config

# Should return 200 with auth, not 404
```

### Issue: Order Service Can't Fetch Tenant Credentials

**Symptoms:** Payment creation fails with "failed to fetch tenant config"

**Solution:**
- Verify `TENANT_SERVICE_URL` in order-service `.env`
- Check tenant-service is running and healthy
- Ensure tenant has configured Midtrans credentials

**Debug:**
```bash
# Check tenant-service health
curl http://localhost:8084/health

# Check if tenant has credentials configured
curl http://localhost:8084/api/v1/admin/tenants/TENANT_ID/midtrans-config \
  -H "Authorization: Bearer TOKEN"
```

### Issue: Webhook Signature Verification Fails

**Symptoms:** Webhook returns 400 or logs "invalid signature"

**Solution:**
- Verify webhook uses tenant-specific server key
- Check signature calculation matches Midtrans format
- Ensure order's tenant_id is correctly passed to verification

**Debug:**
```go
// Add logging in payment_service.go
log.Printf("Verifying signature for tenant: %s, order: %s", tenantID, orderID)
log.Printf("Using server key: %s***", tenantServerKey[:10])
```

## Performance Considerations

### Credential Caching

**Current:** Every payment request fetches credentials from tenant-service via HTTP

**Recommendation:** Implement Redis caching:

```go
// Pseudo-code for caching
func GetSnapClientForTenant(ctx context.Context, tenantID string) (*snap.Client, error) {
    // Check Redis cache first
    cachedConfig, err := redis.Get(ctx, fmt.Sprintf("tenant:%s:midtrans", tenantID))
    if err == nil {
        return createClientFromConfig(cachedConfig), nil
    }
    
    // Cache miss - fetch from tenant-service
    config := fetchTenantMidtransConfig(ctx, tenantID)
    
    // Cache for 5 minutes
    redis.Set(ctx, fmt.Sprintf("tenant:%s:midtrans", tenantID), config, 5*time.Minute)
    
    return createClientFromConfig(config), nil
}
```

**Benefits:**
- Reduces HTTP calls to tenant-service
- Faster payment processing
- Lower latency for customers

**Tradeoff:**
- 5-minute delay for credential updates
- Need cache invalidation on update

### Connection Pooling

Order service makes HTTP calls to tenant-service. Use connection pooling:

```go
var httpClient = &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 10 * time.Second,
}
```

## Monitoring & Observability

### Metrics to Track

1. **Credential Fetch Latency:** Time to fetch credentials from tenant-service
2. **Payment Success Rate:** Per tenant payment success rate
3. **Webhook Processing Time:** Time from webhook receipt to order status update
4. **Signature Verification Failures:** Count of failed webhook verifications

### Logging

**Structured Logging Example:**
```go
log.Info().
    Str("tenant_id", tenantID).
    Str("order_id", orderID).
    Str("environment", config.Environment).
    Msg("Creating Midtrans charge with tenant credentials")
```

**Important Log Points:**
- Credential fetch success/failure
- Payment initiation
- Webhook receipt and verification
- Order status updates

## Future Enhancements

1. **Credential Validation:** Validate Midtrans keys before saving
2. **Encryption at Rest:** Encrypt server_key in database
3. **Credential Rotation:** Support for rotating credentials without downtime
4. **Multi-Gateway Support:** Support other payment gateways (Xendit, Stripe)
5. **Credential History:** Track changes to payment credentials
6. **Automated Testing:** Validate credentials against Midtrans API on save

## Related Documentation

- [Environment Configuration](ENVIRONMENT.md)
- [Guest QRIS Ordering Feature](../specs/003-guest-qris-ordering/spec.md)
- [Payment Webhook Handling](../specs/003-guest-qris-ordering/contracts/payment-webhook.yaml)
- [Midtrans Integration Guide](https://docs.midtrans.com/)

## Support

For issues or questions:
1. Check logs in `/tmp/*.log`
2. Review this documentation
3. Check Midtrans dashboard for transaction status
4. Contact technical team with tenant_id and order_reference
