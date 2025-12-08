# point-of-sale-system Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-11-22

## Active Technologies
- Node.js 18+ (backend), React 18+ (frontend), Go 1.21+ (microservices) + Express.js/Echo, i18next/react-i18next, bcrypt, jsonwebtoken, PostgreSQL driver (001-auth-multitenancy)
- PostgreSQL 14+ with shared schema multi-tenancy (tenant_id column) (001-auth-multitenancy)
- Go 1.23.0 (backend services), Node.js 18+ / Next.js 16 / React 19 (frontend) + Echo v4 (HTTP framework), lib/pq (PostgreSQL driver), Redis v9, Kafka (event streaming), Axios (HTTP client) (001-product-inventory)
- PostgreSQL 14+ with Row-Level Security (RLS) for multi-tenant isolation (001-product-inventory)
- Go 1.23.0 (backend), TypeScript/Next.js 16 (frontend) + Echo v4 (REST API), PostgreSQL (persistence), Redis (caching/sessions), Midtrans Go SDK (payment), Google Maps Geocoding API (address validation) (001-guest-qris-ordering)
- PostgreSQL with schema per service pattern, Redis for session/cart data (001-guest-qris-ordering)
- Go 1.23.0 (backend microservices), Next.js 16 with TypeScript (frontend) + Echo v4 (API framework), PostgreSQL (persistent data), Redis (session/cart/cache), Midtrans Snap API (payment), Google Maps Geocoding API (address validation) (001-guest-qris-ordering)
- PostgreSQL for orders/inventory/tenants (6 new tables), Redis for cart sessions and geocoding cache (001-guest-qris-ordering)

- (001-auth-multitenancy)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for 

## Code Style

: Follow standard conventions

## Recent Changes
- 001-guest-qris-ordering: Added Go 1.23.0 (backend microservices), Next.js 16 with TypeScript (frontend) + Echo v4 (API framework), PostgreSQL (persistent data), Redis (session/cart/cache), Midtrans Snap API (payment), Google Maps Geocoding API (address validation)
- 001-guest-qris-ordering: Added Go 1.23.0 (backend), TypeScript/Next.js 16 (frontend) + Echo v4 (REST API), PostgreSQL (persistence), Redis (caching/sessions), Midtrans Go SDK (payment), Google Maps Geocoding API (address validation)
- 001-product-inventory: Added Go 1.23.0 (backend services), Node.js 18+ / Next.js 16 / React 19 (frontend) + Echo v4 (HTTP framework), lib/pq (PostgreSQL driver), Redis v9, Kafka (event streaming), Axios (HTTP client)


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
