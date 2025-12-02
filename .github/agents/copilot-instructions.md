# point-of-sale-system Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-11-22

## Active Technologies
- Node.js 18+ (backend), React 18+ (frontend), Go 1.21+ (microservices) + Express.js/Echo, i18next/react-i18next, bcrypt, jsonwebtoken, PostgreSQL driver (001-auth-multitenancy)
- PostgreSQL 14+ with shared schema multi-tenancy (tenant_id column) (001-auth-multitenancy)
- Go 1.23.0 (backend services), Node.js 18+ / Next.js 16 / React 19 (frontend) + Echo v4 (HTTP framework), lib/pq (PostgreSQL driver), Redis v9, Kafka (event streaming), Axios (HTTP client) (001-product-inventory)
- PostgreSQL 14+ with Row-Level Security (RLS) for multi-tenant isolation (001-product-inventory)

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
- 001-product-inventory: Added Go 1.23.0 (backend services), Node.js 18+ / Next.js 16 / React 19 (frontend) + Echo v4 (HTTP framework), lib/pq (PostgreSQL driver), Redis v9, Kafka (event streaming), Axios (HTTP client)
- 001-auth-multitenancy: Added Node.js 18+ (backend), React 18+ (frontend), Go 1.21+ (microservices) + Express.js/Echo, i18next/react-i18next, bcrypt, jsonwebtoken, PostgreSQL driver

- 001-auth-multitenancy: Added

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
