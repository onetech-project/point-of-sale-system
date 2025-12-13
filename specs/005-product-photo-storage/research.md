# Research: Product Photo Storage in Object Storage

**Feature**: 005-product-photo-storage  
**Date**: December 12, 2025  
**Status**: Complete

## Executive Summary

This document consolidates research findings for implementing object storage-based product photo management in the POS system. Key decisions cover object storage selection, Go SDK integration, multi-tenancy isolation strategies, image optimization approaches, and error handling patterns.

---

## Decision 1: Object Storage Solution

**Decision**: Use MinIO for development/staging, AWS S3 for production (S3-compatible API)

**Rationale**:
- **S3 Compatibility**: MinIO implements S3 API, enabling seamless migration between environments
- **Development Experience**: MinIO runs locally in Docker, eliminating cloud costs during development
- **Production Reliability**: AWS S3 provides 99.999999999% durability, multi-region replication, mature tooling
- **Cost Efficiency**: MinIO self-hosted for dev/test, S3 pay-per-use in production scales with actual usage
- **Mature Ecosystem**: Both solutions have extensive Go SDK support (aws-sdk-go-v2 and minio-go)

**Alternatives Considered**:
1. **Local Filesystem Only** - Rejected: Not scalable, no replication, complicates multi-server deployments, difficult tenant isolation
2. **Google Cloud Storage** - Rejected: Requires GCP account, less common in microservices deployments, similar features to S3
3. **Azure Blob Storage** - Rejected: Similar to GCS, less S3-compatible API support
4. **Ceph/Rook** - Rejected: Over-engineered for current scale, complex operational overhead

**Implementation Notes**:
- Use MinIO Go SDK (`github.com/minio/minio-go/v7`) for S3-compatible operations
- Configure storage backend via environment variables (`S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`)
- For production S3: Use IAM roles and bucket policies for access control

---

## Decision 2: Multi-Tenant Isolation Strategy

**Decision**: Single bucket with tenant-prefixed object keys (`photos/{tenant_id}/{product_id}/{filename}`)

**Rationale**:
- **Simplicity**: Single bucket reduces operational complexity (backup, monitoring, quotas)
- **Cost Efficiency**: S3 charges per request; single bucket minimizes management overhead
- **Scalability**: S3 buckets support unlimited objects; key prefixes scale indefinitely
- **Access Control**: IAM/bucket policies can restrict access by prefix patterns
- **Migration-Friendly**: Easy to split into per-tenant buckets later if needed

**Alternatives Considered**:
1. **Bucket per Tenant** - Rejected: AWS account limits (100 buckets default, 1000 with request), operational complexity scales linearly with tenants
2. **Shared Keys (no tenant prefix)** - Rejected: Cannot enforce tenant isolation, risk of cross-tenant access bugs
3. **Database-Only Storage** - Rejected: Bloats database, poor performance for binary data, no CDN integration

**Key Structure**:
```
photos/{tenant_id}/{product_id}/{photo_id}_{timestamp}.{ext}

Example:
photos/550e8400-e29b-41d4-a716-446655440000/7c9e6679-7425-40de-944b-e07fc1f90ae7/abc123_1734048000.jpg
```

**Security Implementation**:
- Validate tenant_id from JWT claims matches object key prefix before operations
- Generate presigned URLs with 24-hour expiration for authenticated access
- Apply bucket policy: deny access if object key doesn't match authenticated tenant ID

---

## Decision 3: Go SDK and Library Selection

**Decision**: Use `github.com/minio/minio-go/v7` for object storage operations

**Rationale**:
- **S3 Compatibility**: Works with both MinIO and AWS S3 without code changes
- **Active Maintenance**: Regularly updated, good community support
- **Feature Complete**: Supports multipart uploads, presigned URLs, bucket policies, streaming
- **Simple API**: Clean abstractions for common operations (PutObject, GetObject, RemoveObject)
- **Existing Use**: Already used in similar Go microservices projects

**Best Practices**:
- Initialize client once at startup, reuse across requests (connection pooling)
- Use `PutObject` with `ContentType` and `UserMetadata` for searchability
- Implement exponential backoff for retries on transient failures
- Use `context.Context` for request timeouts and cancellation
- Stream large files rather than loading into memory

**Code Pattern**:
```go
// Initialize client
minioClient, err := minio.New(endpoint, &minio.Options{
    Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
    Secure: useSSL,
})

// Upload with metadata
_, err = minioClient.PutObject(ctx, bucketName, objectKey, reader, size,
    minio.PutObjectOptions{
        ContentType: "image/jpeg",
        UserMetadata: map[string]string{
            "tenant-id":   tenantID,
            "product-id":  productID,
            "uploaded-by": userID,
        },
    })
```

---

## Decision 4: Image Optimization and Validation

**Decision**: Use `github.com/disintegration/imaging` (already in dependencies) for image processing

**Rationale**:
- **Already Integrated**: Package exists in go.mod, no new dependency
- **Feature Rich**: Resize, crop, rotate, format conversion, quality adjustment
- **Performance**: Fast, native Go implementation
- **Simplicity**: Clean API, easy to use

**Validation Rules**:
1. **File Size**: Reject files >10MB before upload starts (read first N bytes)
2. **Dimensions**: Decode image header, check width/height <4096px
3. **MIME Type**: Validate Content-Type header matches actual file signature (magic bytes)
4. **Format**: Accept JPEG, PNG, WebP, GIF only

**Optimization Strategy**:
- **Resize Large Images**: If width or height >2048px, resize maintaining aspect ratio
- **Quality Compression**: JPEG quality 85, PNG optimized compression
- **Format Conversion**: Optional - convert PNG to JPEG if no transparency detected (future enhancement)

**Code Pattern**:
```go
// Decode and validate
img, err := imaging.Decode(file)
if err != nil {
    return errors.New("invalid image format")
}

bounds := img.Bounds()
if bounds.Dx() > 4096 || bounds.Dy() > 4096 {
    return errors.New("image dimensions exceed limit")
}

// Resize if needed
if bounds.Dx() > 2048 || bounds.Dy() > 2048 {
    img = imaging.Fit(img, 2048, 2048, imaging.Lanczos)
}

// Encode optimized version
var buf bytes.Buffer
imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(85))
```

---

## Decision 5: Database Schema for Photo Metadata

**Decision**: Create `product_photos` table with foreign key to products

**Rationale**:
- **Relational Integrity**: Enforce product-photo relationship at database level
- **Queryability**: Efficient queries for "all photos for product X"
- **Ordering**: Store display_order for photo carousel functionality
- **Audit Trail**: Track upload timestamps, file sizes for analytics
- **Separation of Concerns**: Products table stays lean, photos are separate concern

**Schema Design**:
```sql
CREATE TABLE product_photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    storage_key TEXT NOT NULL, -- S3 object key
    original_filename TEXT NOT NULL,
    file_size_bytes INTEGER NOT NULL,
    mime_type TEXT NOT NULL,
    width_px INTEGER,
    height_px INTEGER,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT unique_display_order UNIQUE (product_id, display_order)
);

CREATE INDEX idx_product_photos_product_id ON product_photos(product_id);
CREATE INDEX idx_product_photos_tenant_id ON product_photos(tenant_id);
```

**Alternatives Considered**:
1. **JSON Column in Products** - Rejected: Difficult to query, no relational integrity, bloats products table
2. **Separate Photos Service** - Rejected: Over-engineering, adds network latency, complicates transactions
3. **NoSQL Document Store** - Rejected: Unnecessary complexity, PostgreSQL handles this scale easily

---

## Decision 6: Error Handling and Resilience

**Decision**: Implement graceful degradation with fallback placeholders and circuit breaker pattern

**Rationale**:
- **User Experience**: System remains functional even if S3 is temporarily unavailable
- **Reliability**: Prevent cascading failures from storage service outages
- **Observability**: Clear error logging and metrics for debugging

**Error Handling Strategy**:

1. **Upload Failures**:
   - Retry with exponential backoff (3 attempts: 1s, 2s, 4s)
   - If all retries fail, return 503 Service Unavailable with retry-after header
   - Log error with context (tenant, product, file size) for investigation
   - Rollback database transaction if metadata already saved

2. **Retrieval Failures**:
   - Return placeholder image URL if photo not found or S3 unreachable
   - Cache placeholder URLs to reduce load
   - Log warning but don't block page render
   - Emit metric for monitoring

3. **Deletion Failures**:
   - Mark photo as "pending_deletion" in database
   - Background job retries deletion every 5 minutes
   - Alert if deletion pending >24 hours
   - Prevents orphaned storage but doesn't block product deletion

**Circuit Breaker Pattern**:
```go
// After 5 consecutive failures, open circuit for 60s
// During open circuit, immediately return error without attempting S3 call
// After 60s, allow one test request (half-open state)
// If test succeeds, close circuit; if fails, reopen

type CircuitBreaker struct {
    failures      int
    lastFailTime  time.Time
    state         string // closed, open, half-open
}
```

**Placeholder Image**:
- Serve default product image from CDN or local static assets
- Return URL: `/assets/placeholder-product.png`
- Different placeholders for different categories (optional enhancement)

---

## Decision 7: Storage Quota Management

**Decision**: Track storage usage per tenant in database, enforce quota at upload time

**Rationale**:
- **Cost Control**: Prevent runaway storage costs from malicious or accidental overuse
- **Fair Usage**: Ensure no single tenant monopolizes resources
- **Business Model**: Enable tiered pricing based on storage usage

**Implementation**:

1. **Tenant Storage Tracking**:
```sql
ALTER TABLE tenants ADD COLUMN storage_used_bytes BIGINT NOT NULL DEFAULT 0;
ALTER TABLE tenants ADD COLUMN storage_quota_bytes BIGINT NOT NULL DEFAULT 5368709120; -- 5GB default
CREATE INDEX idx_tenants_storage ON tenants(tenant_id, storage_used_bytes);
```

2. **Quota Check Logic**:
```go
// Before upload
currentUsage := getTenantStorageUsage(tenantID)
quota := getTenantQuota(tenantID)
if currentUsage + fileSize > quota {
    return errors.New("storage quota exceeded")
}

// After successful upload
incrementTenantStorageUsage(tenantID, fileSize)

// After deletion
decrementTenantStorageUsage(tenantID, fileSize)
```

3. **Quota Enforcement Points**:
   - Single photo upload
   - Bulk photo upload
   - Photo replacement (check: new size - old size + current usage)

4. **Quota Monitoring**:
   - Emit metrics: `tenant_storage_usage_bytes{tenant_id}`
   - Alert when tenant reaches 80%, 90%, 95% of quota
   - Admin API endpoint to view/modify tenant quotas

**Alternatives Considered**:
1. **S3 Bucket Quotas** - Rejected: Not supported by MinIO, difficult to track per-tenant in shared bucket
2. **Calculate on Demand** - Rejected: Expensive query (sum all photo sizes per tenant), slow
3. **No Quotas** - Rejected: Risk of abuse and uncontrolled costs

---

## Decision 8: Photo URL Generation and Access Control

**Decision**: Generate presigned URLs for authenticated access, public URLs for public products

**Rationale**:
- **Security**: Prevents unauthorized access to private product photos
- **Flexibility**: Supports both public catalog and authenticated admin access
- **Performance**: Direct S3 access bypasses application server (reduced load)
- **CDN-Ready**: Presigned URLs can be used with CloudFront or similar CDN

**Access Control Logic**:

1. **Public Products** (guest ordering, public catalog):
   - Generate presigned URL with 7-day expiration
   - Cache URL in Redis with 6-day TTL
   - Frontend uses cached URL directly
   - No authentication required

2. **Private Products** (admin-only, draft products):
   - Generate presigned URL with 24-hour expiration
   - Include tenant_id validation in URL generation
   - No caching (URLs expire quickly)
   - Requires valid JWT token

**URL Generation Pattern**:
```go
// Public product photo
func (s *StorageService) GetPhotoURL(ctx context.Context, photoID uuid.UUID) (string, error) {
    photo := s.repo.GetPhotoByID(photoID)
    
    // Check cache first
    if cachedURL, found := s.cache.Get(photoID); found {
        return cachedURL, nil
    }
    
    // Generate presigned URL
    expiry := time.Hour * 24 * 7 // 7 days for public
    if !photo.Product.IsPublic {
        expiry = time.Hour * 24 // 24 hours for private
    }
    
    url, err := s.minioClient.PresignedGetObject(ctx, s.bucket, photo.StorageKey, expiry, nil)
    if err != nil {
        return "", err
    }
    
    // Cache public URLs
    if photo.Product.IsPublic {
        s.cache.Set(photoID, url, time.Hour * 24 * 6)
    }
    
    return url, nil
}
```

**CDN Integration (Future Enhancement)**:
- Configure CloudFront or similar CDN in front of S3 bucket
- Update URL generation to return CDN URLs instead of direct S3
- Invalidate CDN cache on photo deletion/update

---

## Decision 9: Migration Strategy for Existing Photos

**Decision**: Provide manual migration script, NOT automated within feature

**Rationale**:
- **Scope Control**: Feature focuses on new photo uploads, migration is separate concern
- **Risk Management**: Migration involves moving production data, requires careful planning
- **Flexibility**: Different deployments may have different migration needs
- **Rollback**: Clear separation allows rolling back feature without affecting existing photos

**Migration Approach** (documented, not implemented):

1. **Pre-Migration**:
   - Audit existing photos in `./uploads` directory
   - Calculate total size to validate storage quota
   - Generate migration plan report

2. **Migration Script** (provided as separate tool):
```bash
# scripts/migrate-photos-to-s3.sh
# - Reads photos from local uploads/ directory
# - Uploads to S3 with correct tenant prefixes
# - Updates database photo_path to storage_key
# - Validates upload success before deleting local file
# - Generates migration report (success/failure counts)
```

3. **Post-Migration**:
   - Verify all photos accessible via new URLs
   - Run integration tests against migrated data
   - Keep local backups for 30 days before deletion

**Note**: This feature implements dual-mode support:
- If `photo_path` starts with `photos/`, treat as S3 key (new)
- If `photo_path` is local path, serve from filesystem (legacy)
- Allows gradual migration without breaking existing functionality

---

## Research Summary and Next Steps

### Resolved Clarifications

All "NEEDS CLARIFICATION" items from Technical Context have been resolved:

1. ✅ **Object Storage Solution**: MinIO (dev) + S3 (prod)
2. ✅ **Go SDK**: minio-go v7
3. ✅ **Multi-Tenancy**: Tenant-prefixed keys in single bucket
4. ✅ **Image Processing**: disintegration/imaging (existing dependency)
5. ✅ **Database Schema**: product_photos table with metadata
6. ✅ **Error Handling**: Circuit breaker + graceful degradation
7. ✅ **Quota Management**: Database-tracked usage with enforcement
8. ✅ **URL Access**: Presigned URLs with caching
9. ✅ **Migration**: Manual script, not in feature scope

### Technology Stack Confirmed

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| Object Storage (Dev) | MinIO | Latest | Local S3-compatible storage |
| Object Storage (Prod) | AWS S3 | N/A | Production photo storage |
| Go SDK | minio-go | v7 | S3 client operations |
| Image Processing | disintegration/imaging | v1.6.2 | Resize, optimize, validate |
| Database | PostgreSQL | 14.19 | Photo metadata storage |
| Cache | Redis | 8.0.5 | URL caching |

### Dependencies to Add

```go
// go.mod additions
require (
    github.com/minio/minio-go/v7 v7.0.63
)
```

### Configuration Required

```env
# .env additions
S3_ENDPOINT=minio:9000           # MinIO for dev, s3.amazonaws.com for prod
S3_BUCKET=pos-product-photos
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_USE_SSL=false                 # true for production S3
S3_REGION=us-east-1
STORAGE_QUOTA_BYTES=5368709120   # 5GB default
PHOTO_URL_CACHE_TTL=518400       # 6 days in seconds
```

### Ready for Phase 1

All research complete. Proceed to Phase 1: Data Model and Contracts design.
