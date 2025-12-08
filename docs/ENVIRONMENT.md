# Environment Configuration

This document describes the environment variables used across all services in the POS system.

## Quick Start

1. Copy `.env.example` to `.env` in the root directory:
   ```bash
   cp .env.example .env
   ```

2. Copy `.env.example` to `.env` in each service directory:
   ```bash
   # API Gateway
   cp api-gateway/.env.example api-gateway/.env
   
   # Auth Service
   cp backend/auth-service/.env.example backend/auth-service/.env
   
   # User Service
   cp backend/user-service/.env.example backend/user-service/.env
   
   # Tenant Service
   cp backend/tenant-service/.env.example backend/tenant-service/.env
   
   # Notification Service
   cp backend/notification-service/.env.example backend/notification-service/.env
   
   # Frontend
   cp frontend/.env.example frontend/.env.local
   ```

3. Update the `.env` files with your configuration (see details below)

## Environment Files Structure

```
point-of-sale-system/
├── .env                              # Root configuration (docker-compose)
├── .env.example                      # Root template
├── api-gateway/
│   ├── .env                          # API Gateway config
│   └── .env.example                  # API Gateway template
├── backend/
│   ├── auth-service/
│   │   ├── .env                      # Auth Service config
│   │   └── .env.example              # Auth Service template
│   ├── user-service/
│   │   ├── .env                      # User Service config
│   │   └── .env.example              # User Service template
│   ├── tenant-service/
│   │   ├── .env                      # Tenant Service config
│   │   └── .env.example              # Tenant Service template
│   └── notification-service/
│       ├── .env                      # Notification Service config
│       └── .env.example              # Notification Service template
└── frontend/
    ├── .env.local                    # Frontend config (gitignored)
    └── .env.example                  # Frontend template
```

## Configuration Details

### Root Configuration (.env)

Used by docker-compose and as a reference for all services.

**Required Variables:**
- `POSTGRES_DB` - PostgreSQL database name
- `POSTGRES_USER` - PostgreSQL username
- `POSTGRES_PASSWORD` - PostgreSQL password
- `JWT_SECRET` - Shared JWT secret (MUST be the same across all services)

**Service Ports:**
- `API_GATEWAY_PORT=8080` - API Gateway port
- `AUTH_SERVICE_PORT=8082` - Auth Service port
- `USER_SERVICE_PORT=8083` - User Service port
- `TENANT_SERVICE_PORT=8084` - Tenant Service port
- `NOTIFICATION_SERVICE_PORT=8085` - Notification Service port
- `FRONTEND_PORT=3000` - Frontend port

### API Gateway (.env)

**Required Variables:**
- `PORT` - Server port (default: 8080)
- `JWT_SECRET` - JWT secret for token validation (MUST match auth service)
- `AUTH_SERVICE_URL` - Auth service URL
- `USER_SERVICE_URL` - User service URL

**Optional Variables:**
- `TENANT_SERVICE_URL` - Tenant service URL
- `ALLOWED_ORIGINS` - CORS allowed origins (comma-separated)

### Auth Service (.env)

**Required Variables:**
- `PORT` - Server port (default: 8082)
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_HOST` - Redis host (for sessions)
- `JWT_SECRET` - JWT secret (MUST match API gateway)

**Optional Variables:**
- `JWT_EXPIRATION_MINUTES` - JWT token expiration (default: 15)
- `SESSION_TTL_MINUTES` - Session TTL in Redis (default: 15)
- `RATE_LIMIT_LOGIN_MAX` - Max login attempts (default: 5)
- `RATE_LIMIT_LOGIN_WINDOW` - Rate limit window in seconds (default: 900)

### User Service (.env)

**Required Variables:**
- `PORT` - Server port (default: 8083)
- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - JWT secret for token validation

**Email Configuration (for invitations):**
- `SMTP_HOST` - SMTP server host
- `SMTP_PORT` - SMTP server port
- `SMTP_USER` - SMTP username
- `SMTP_PASSWORD` - SMTP password
- `SMTP_FROM` - From email address

### Tenant Service (.env)

**Required Variables:**
- `PORT` - Server port (default: 8084)
- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - JWT secret for token validation

**Optional Variables:**
- `ENABLE_TENANT_ISOLATION` - Enable tenant isolation (default: true)
- `DEFAULT_TENANT_PLAN` - Default plan for new tenants (default: free)

**Note on Midtrans Configuration:**
- Midtrans credentials (server_key, client_key, merchant_id) are stored **per-tenant** in the database
- Each tenant configures their own payment gateway through the admin UI at `/settings/payment`
- No global Midtrans environment variables needed in tenant-service
- See Order Service configuration for fallback/testing credentials

### Notification Service (.env)

**Required Variables:**
- `PORT` - Server port (default: 8085)
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_HOST` - Redis host
- `KAFKA_BROKERS` - Kafka broker addresses

**Email Configuration:**
- Same as User Service

**Optional Variables:**
- `SMS_PROVIDER` - SMS provider (twilio, etc.)
- `FIREBASE_CREDENTIALS_PATH` - Firebase credentials for push notifications

### Order Service (.env)

**Required Variables:**
- `PORT` - Server port (default: 8087)
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string for cart and caching
- `TENANT_SERVICE_URL` - Tenant service URL to fetch payment configs

**Midtrans Configuration (Fallback/Testing):**
- `MIDTRANS_SERVER_KEY` - Fallback Midtrans server key (optional, tenant-specific keys preferred)
- `MIDTRANS_CLIENT_KEY` - Fallback Midtrans client key (optional)
- `MIDTRANS_ENVIRONMENT` - sandbox or production (default: sandbox)
- `MIDTRANS_MERCHANT_ID` - Merchant ID (optional)

**Google Maps API:**
- `GOOGLE_MAPS_API_KEY` - For geocoding and delivery fee calculation

**Important:** Order service now fetches tenant-specific Midtrans credentials from tenant-service at runtime. The environment variables above are only used as fallback if tenant hasn't configured their own credentials.

**Cart Configuration:**
- `CART_SESSION_TTL` - Cart expiration in seconds (default: 86400 = 24 hours)
- `GEOCODING_CACHE_TTL` - Address geocoding cache TTL (default: 604800 = 7 days)

### Frontend (.env.local)

**Required Variables:**
- `NEXT_PUBLIC_API_URL` - API Gateway URL (default: http://localhost:8080)

**Optional Variables:**
- `NEXT_PUBLIC_APP_NAME` - Application name
- `NEXT_PUBLIC_ENVIRONMENT` - Environment (development, staging, production)
- `NEXT_PUBLIC_ENABLE_DEBUG` - Enable debug mode
- `NEXT_PUBLIC_DEFAULT_LOCALE` - Default language (en, id)

## Important Security Notes

⚠️ **NEVER commit `.env` files to version control!**

✅ Only commit `.env.example` files as templates

🔒 **Production Checklist:**
1. Change `JWT_SECRET` to a strong random value (minimum 32 characters)
2. Use strong database passwords
3. Enable HTTPS in production (set `Secure` flag for cookies)
4. Update `ALLOWED_ORIGINS` to your production domains
5. Use production-grade SMTP credentials
6. Enable rate limiting in production
7. Set `ENVIRONMENT=production` in all services

## Tenant-Specific Payment Configuration

**New in Version 2.0:** Each tenant can now configure their own Midtrans payment credentials.

### How It Works

1. **Database Storage**: Midtrans credentials are stored in `tenant_configs` table with these fields:
   - `midtrans_server_key` - Server key for backend API calls
   - `midtrans_client_key` - Client key for frontend integration
   - `midtrans_merchant_id` - Merchant identifier
   - `midtrans_environment` - `sandbox` or `production`

2. **Admin Configuration**: Tenant owners configure payment settings at `/settings/payment` in the admin UI

3. **Runtime Fetching**: Order service fetches tenant-specific credentials when processing payments:
   ```go
   // Order service dynamically retrieves credentials per tenant
   snapClient := GetSnapClientForTenant(ctx, tenantID)
   ```

4. **Fallback Mechanism**: If tenant hasn't configured credentials, order service uses fallback from `.env`:
   - Development/Testing: Use shared sandbox credentials
   - Production: Require tenant-specific credentials (no fallback)

### Setting Up Tenant Payment Credentials

**Via Admin UI (Recommended):**
1. Login as tenant owner
2. Navigate to Settings → Payment Settings
3. Enter Midtrans credentials
4. Select environment (sandbox/production)
5. Save configuration

**Via Database (Development):**
```sql
UPDATE tenant_configs 
SET 
  midtrans_server_key = 'SB-Mid-server-xxx',
  midtrans_client_key = 'SB-Mid-client-xxx',
  midtrans_merchant_id = 'G123456789',
  midtrans_environment = 'sandbox'
WHERE tenant_id = 'your-tenant-uuid';
```

### Security Considerations

- **Encryption**: In production, consider encrypting `midtrans_server_key` at rest
- **Access Control**: Only tenant owners can view/update payment credentials
- **Audit Logging**: All credential updates are logged with user and timestamp
- **API Isolation**: Each tenant's transactions use only their credentials (multi-tenant isolation)

## Running Services with Environment Variables

### Using .env files directly:

```bash
# API Gateway
cd api-gateway
go run main.go

# Auth Service
cd backend/auth-service
go run main.go

# Frontend
cd frontend
npm run dev
```

The services will automatically load `.env` files from their directories.

### Using docker-compose:

```bash
docker-compose up -d
```

Docker Compose will use the root `.env` file for all services.

## Troubleshooting

### Issue: Services can't connect to database
- Check `DATABASE_URL` in each service's `.env`
- Verify PostgreSQL is running: `docker ps | grep postgres`
- Test connection: `psql postgresql://pos_user:pos_password@localhost:5432/pos_db`

### Issue: JWT token validation fails
- Ensure `JWT_SECRET` is **exactly the same** in:
  - Root `.env`
  - `api-gateway/.env`
  - `backend/auth-service/.env`
  - Any service that validates tokens

### Issue: Frontend can't reach API
- Check `NEXT_PUBLIC_API_URL` in `frontend/.env.local`
- Verify API Gateway is running on the correct port
- Check CORS settings in API Gateway

### Issue: Sessions not persisting
- Check Redis connection in `REDIS_HOST`
- Verify Redis is running: `docker ps | grep redis`
- Test Redis: `redis-cli ping`

## Environment Variables Priority

Services load environment variables in this order (highest priority first):

1. System environment variables
2. `.env` file in service directory
3. Default values in code

## Getting Help

If you encounter issues with environment configuration:

1. Check this README
2. Verify all `.env` files exist and have correct values
3. Check service logs for specific error messages
4. Compare your `.env` with `.env.example`

## Related Documentation

- [Docker Setup Guide](../docs/docker-setup.md)
- [Development Guide](../docs/development.md)
- [Production Deployment](../docs/production.md)
