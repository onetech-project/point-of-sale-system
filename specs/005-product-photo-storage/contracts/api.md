# API Contracts: Product Photo Storage

**Feature**: 005-product-photo-storage  
**Service**: product-service  
**Base URL**: `/api/v1/products`  
**Date**: December 12, 2025

## Overview

This document defines the REST API contracts for product photo management. All endpoints require authentication and enforce tenant isolation via `X-Tenant-ID` header or JWT claims.

---

## Authentication & Authorization

**Authentication**: Bearer JWT token in `Authorization` header  
**Tenant Isolation**: `X-Tenant-ID` header must match JWT `tenant_id` claim  
**Rate Limiting**: 100 requests per minute per tenant

**Common Headers**:
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
Content-Type: multipart/form-data (for uploads)
Content-Type: application/json (for other requests)
```

---

## Endpoints

### 1. Upload Product Photo

Upload a new photo for a product.

**Endpoint**: `POST /api/v1/products/{product_id}/photos`

**Path Parameters**:
- `product_id` (UUID, required): Product identifier

**Request Headers**:
```
Authorization: Bearer <jwt_token>
X-Tenant-ID: <tenant_uuid>
Content-Type: multipart/form-data
```

**Request Body** (multipart/form-data):
```
photo: <file>              // Image file (JPEG, PNG, WebP, GIF)
display_order: <integer>   // Optional, 0-4, default: next available
is_primary: <boolean>      // Optional, default: false
```

**Example Request**:
```bash
curl -X POST "http://localhost:8086/api/v1/products/7c9e6679-7425-40de-944b-e07fc1f90ae7/photos" \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -F "photo=@product-image.jpg" \
  -F "display_order=0" \
  -F "is_primary=true"
```

**Success Response** (201 Created):
```json
{
  "status": "success",
  "data": {
    "id": "a1b2c3d4-5678-90ef-ghij-klmnopqrstuv",
    "product_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "original_filename": "product-image.jpg",
    "file_size_bytes": 2458320,
    "mime_type": "image/jpeg",
    "width_px": 1920,
    "height_px": 1080,
    "display_order": 0,
    "is_primary": true,
    "photo_url": "https://s3.amazonaws.com/pos-photos/photos/550e8400.../presigned-url",
    "created_at": "2025-12-12T10:30:00Z",
    "updated_at": "2025-12-12T10:30:00Z"
  }
}
```

**Error Responses**:

**400 Bad Request** - Invalid file or validation error:
```json
{
  "status": "error",
  "error": {
    "code": "INVALID_FILE",
    "message": "File size exceeds 10MB limit",
    "details": {
      "max_size_bytes": 10485760,
      "uploaded_size_bytes": 15728640
    }
  }
}
```

**403 Forbidden** - Quota exceeded:
```json
{
  "status": "error",
  "error": {
    "code": "QUOTA_EXCEEDED",
    "message": "Storage quota exceeded",
    "details": {
      "current_usage_bytes": 5120000000,
      "quota_bytes": 5368709120,
      "required_bytes": 2458320,
      "available_bytes": 248709120
    }
  }
}
```

**404 Not Found** - Product not found:
```json
{
  "status": "error",
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "Product not found or access denied"
  }
}
```

**409 Conflict** - Maximum photos reached:
```json
{
  "status": "error",
  "error": {
    "code": "MAX_PHOTOS_REACHED",
    "message": "Product already has maximum number of photos",
    "details": {
      "current_photo_count": 5,
      "max_allowed": 5
    }
  }
}
```

**413 Payload Too Large** - File too large:
```json
{
  "status": "error",
  "error": {
    "code": "FILE_TOO_LARGE",
    "message": "File exceeds maximum size of 10MB"
  }
}
```

**415 Unsupported Media Type** - Invalid file type:
```json
{
  "status": "error",
  "error": {
    "code": "UNSUPPORTED_FILE_TYPE",
    "message": "File type not supported",
    "details": {
      "uploaded_mime_type": "image/bmp",
      "supported_types": ["image/jpeg", "image/png", "image/webp", "image/gif"]
    }
  }
}
```

**503 Service Unavailable** - Storage service error:
```json
{
  "status": "error",
  "error": {
    "code": "STORAGE_UNAVAILABLE",
    "message": "Unable to upload photo, please try again later",
    "retry_after": 60
  }
}
```

---

### 2. List Product Photos

Get all photos for a product.

**Endpoint**: `GET /api/v1/products/{product_id}/photos`

**Path Parameters**:
- `product_id` (UUID, required): Product identifier

**Query Parameters**:
- None

**Example Request**:
```bash
curl -X GET "http://localhost:8086/api/v1/products/7c9e6679-7425-40de-944b-e07fc1f90ae7/photos" \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000"
```

**Success Response** (200 OK):
```json
{
  "status": "success",
  "data": [
    {
      "id": "a1b2c3d4-5678-90ef-ghij-klmnopqrstuv",
      "product_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "original_filename": "product-front.jpg",
      "file_size_bytes": 2458320,
      "mime_type": "image/jpeg",
      "width_px": 1920,
      "height_px": 1080,
      "display_order": 0,
      "is_primary": true,
      "photo_url": "https://s3.amazonaws.com/pos-photos/photos/550e8400.../presigned-url",
      "created_at": "2025-12-12T10:30:00Z",
      "updated_at": "2025-12-12T10:30:00Z"
    },
    {
      "id": "b2c3d4e5-6789-01fa-bcde-fghijklmnopq",
      "product_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "original_filename": "product-side.jpg",
      "file_size_bytes": 1987654,
      "mime_type": "image/jpeg",
      "width_px": 1920,
      "height_px": 1080,
      "display_order": 1,
      "is_primary": false,
      "photo_url": "https://s3.amazonaws.com/pos-photos/photos/550e8400.../presigned-url",
      "created_at": "2025-12-12T10:31:00Z",
      "updated_at": "2025-12-12T10:31:00Z"
    }
  ],
  "meta": {
    "total": 2,
    "product_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
  }
}
```

**Error Responses**:

**404 Not Found**:
```json
{
  "status": "error",
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "Product not found or access denied"
  }
}
```

---

### 3. Get Single Photo

Get details of a specific photo.

**Endpoint**: `GET /api/v1/products/{product_id}/photos/{photo_id}`

**Path Parameters**:
- `product_id` (UUID, required): Product identifier
- `photo_id` (UUID, required): Photo identifier

**Example Request**:
```bash
curl -X GET "http://localhost:8086/api/v1/products/7c9e6679-7425-40de-944b-e07fc1f90ae7/photos/a1b2c3d4-5678-90ef-ghij-klmnopqrstuv" \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000"
```

**Success Response** (200 OK):
```json
{
  "status": "success",
  "data": {
    "id": "a1b2c3d4-5678-90ef-ghij-klmnopqrstuv",
    "product_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "original_filename": "product-front.jpg",
    "file_size_bytes": 2458320,
    "mime_type": "image/jpeg",
    "width_px": 1920,
    "height_px": 1080,
    "display_order": 0,
    "is_primary": true,
    "photo_url": "https://s3.amazonaws.com/pos-photos/photos/550e8400.../presigned-url",
    "created_at": "2025-12-12T10:30:00Z",
    "updated_at": "2025-12-12T10:30:00Z"
  }
}
```

**Error Responses**:

**404 Not Found**:
```json
{
  "status": "error",
  "error": {
    "code": "PHOTO_NOT_FOUND",
    "message": "Photo not found or access denied"
  }
}
```

---

### 4. Update Photo Metadata

Update photo display order or primary flag.

**Endpoint**: `PATCH /api/v1/products/{product_id}/photos/{photo_id}`

**Path Parameters**:
- `product_id` (UUID, required): Product identifier
- `photo_id` (UUID, required): Photo identifier

**Request Body** (application/json):
```json
{
  "display_order": 2,
  "is_primary": false
}
```

**Example Request**:
```bash
curl -X PATCH "http://localhost:8086/api/v1/products/7c9e6679-7425-40de-944b-e07fc1f90ae7/photos/a1b2c3d4-5678-90ef-ghij-klmnopqrstuv" \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"display_order": 2, "is_primary": false}'
```

**Success Response** (200 OK):
```json
{
  "status": "success",
  "data": {
    "id": "a1b2c3d4-5678-90ef-ghij-klmnopqrstuv",
    "product_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "original_filename": "product-front.jpg",
    "file_size_bytes": 2458320,
    "mime_type": "image/jpeg",
    "width_px": 1920,
    "height_px": 1080,
    "display_order": 2,
    "is_primary": false,
    "photo_url": "https://s3.amazonaws.com/pos-photos/photos/550e8400.../presigned-url",
    "created_at": "2025-12-12T10:30:00Z",
    "updated_at": "2025-12-12T10:45:00Z"
  }
}
```

**Error Responses**:

**400 Bad Request** - Invalid display order:
```json
{
  "status": "error",
  "error": {
    "code": "INVALID_DISPLAY_ORDER",
    "message": "Display order must be between 0 and 4"
  }
}
```

**409 Conflict** - Display order already used:
```json
{
  "status": "error",
  "error": {
    "code": "DISPLAY_ORDER_CONFLICT",
    "message": "Another photo already uses display order 2"
  }
}
```

---

### 5. Delete Product Photo

Delete a photo from a product.

**Endpoint**: `DELETE /api/v1/products/{product_id}/photos/{photo_id}`

**Path Parameters**:
- `product_id` (UUID, required): Product identifier
- `photo_id` (UUID, required): Photo identifier

**Example Request**:
```bash
curl -X DELETE "http://localhost:8086/api/v1/products/7c9e6679-7425-40de-944b-e07fc1f90ae7/photos/a1b2c3d4-5678-90ef-ghij-klmnopqrstuv" \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000"
```

**Success Response** (204 No Content):
- No response body

**Error Responses**:

**404 Not Found**:
```json
{
  "status": "error",
  "error": {
    "code": "PHOTO_NOT_FOUND",
    "message": "Photo not found or access denied"
  }
}
```

**503 Service Unavailable** - Storage deletion failed:
```json
{
  "status": "error",
  "error": {
    "code": "STORAGE_UNAVAILABLE",
    "message": "Photo marked for deletion but storage removal failed. Will retry automatically.",
    "retry_after": 300
  }
}
```

---

### 6. Reorder Photos

Bulk update display order for multiple photos.

**Endpoint**: `PUT /api/v1/products/{product_id}/photos/reorder`

**Path Parameters**:
- `product_id` (UUID, required): Product identifier

**Request Body** (application/json):
```json
{
  "photos": [
    {"id": "photo-uuid-1", "display_order": 0},
    {"id": "photo-uuid-2", "display_order": 1},
    {"id": "photo-uuid-3", "display_order": 2}
  ]
}
```

**Example Request**:
```bash
curl -X PUT "http://localhost:8086/api/v1/products/7c9e6679-7425-40de-944b-e07fc1f90ae7/photos/reorder" \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"photos": [{"id": "a1b2c3d4-5678-90ef-ghij-klmnopqrstuv", "display_order": 0}]}'
```

**Success Response** (200 OK):
```json
{
  "status": "success",
  "data": {
    "updated_count": 3,
    "photos": [
      {
        "id": "a1b2c3d4-5678-90ef-ghij-klmnopqrstuv",
        "display_order": 0
      },
      {
        "id": "b2c3d4e5-6789-01fa-bcde-fghijklmnopq",
        "display_order": 1
      },
      {
        "id": "c3d4e5f6-789a-12fb-cdef-ghijklmnopqr",
        "display_order": 2
      }
    ]
  }
}
```

**Error Responses**:

**400 Bad Request** - Invalid photo IDs or orders:
```json
{
  "status": "error",
  "error": {
    "code": "INVALID_PHOTO_IDS",
    "message": "One or more photo IDs do not belong to this product"
  }
}
```

---

### 7. Get Tenant Storage Quota

Get current storage usage and quota for the authenticated tenant.

**Endpoint**: `GET /api/v1/products/storage-quota`

**Example Request**:
```bash
curl -X GET "http://localhost:8086/api/v1/products/storage-quota" \
  -H "Authorization: Bearer eyJhbGc..." \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000"
```

**Success Response** (200 OK):
```json
{
  "status": "success",
  "data": {
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "used_bytes": 2458320000,
    "quota_bytes": 5368709120,
    "available_bytes": 2910389120,
    "usage_percent": 45.78,
    "photo_count": 127
  }
}
```

---

### 8. Enhanced Product Endpoints

Existing product endpoints are enhanced to include photo information.

#### Get Product with Photos

**Endpoint**: `GET /api/v1/products/{product_id}?include_photos=true`

**Query Parameters**:
- `include_photos` (boolean, optional): Include photo list in response

**Success Response** (200 OK):
```json
{
  "status": "success",
  "data": {
    "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
    "sku": "PROD-001",
    "name": "Premium Coffee Beans",
    "description": "Organic Arabica beans from Colombia",
    "selling_price": 25.99,
    "cost_price": 12.50,
    "stock_quantity": 150,
    "photos": [
      {
        "id": "a1b2c3d4-5678-90ef-ghij-klmnopqrstuv",
        "original_filename": "coffee-front.jpg",
        "display_order": 0,
        "is_primary": true,
        "photo_url": "https://s3.amazonaws.com/.../presigned-url"
      },
      {
        "id": "b2c3d4e5-6789-01fa-bcde-fghijklmnopq",
        "original_filename": "coffee-side.jpg",
        "display_order": 1,
        "is_primary": false,
        "photo_url": "https://s3.amazonaws.com/.../presigned-url"
      }
    ],
    "created_at": "2025-12-01T10:00:00Z",
    "updated_at": "2025-12-12T10:45:00Z"
  }
}
```

#### List Products with Primary Photo

**Endpoint**: `GET /api/v1/products?include_primary_photo=true`

**Query Parameters**:
- `include_primary_photo` (boolean, optional): Include primary photo URL in response

**Success Response** (200 OK):
```json
{
  "status": "success",
  "data": [
    {
      "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
      "sku": "PROD-001",
      "name": "Premium Coffee Beans",
      "selling_price": 25.99,
      "stock_quantity": 150,
      "primary_photo_url": "https://s3.amazonaws.com/.../presigned-url",
      "photo_count": 2
    }
  ],
  "meta": {
    "total": 1,
    "page": 1,
    "per_page": 20
  }
}
```

---

## Error Codes Reference

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_FILE` | 400 | File validation failed (size, type, dimensions) |
| `INVALID_DISPLAY_ORDER` | 400 | Display order out of range (0-4) |
| `INVALID_PHOTO_IDS` | 400 | Photo IDs don't match product |
| `QUOTA_EXCEEDED` | 403 | Tenant storage quota exceeded |
| `PRODUCT_NOT_FOUND` | 404 | Product not found or access denied |
| `PHOTO_NOT_FOUND` | 404 | Photo not found or access denied |
| `MAX_PHOTOS_REACHED` | 409 | Product has maximum 5 photos |
| `DISPLAY_ORDER_CONFLICT` | 409 | Display order already used |
| `FILE_TOO_LARGE` | 413 | File exceeds 10MB limit |
| `UNSUPPORTED_FILE_TYPE` | 415 | Invalid MIME type |
| `STORAGE_UNAVAILABLE` | 503 | Object storage service error |

---

## Rate Limiting

**Limits**:
- 100 requests per minute per tenant (all endpoints)
- 10 photo uploads per minute per tenant (upload endpoint only)

**Response Headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1734048600
```

**Rate Limit Exceeded** (429 Too Many Requests):
```json
{
  "status": "error",
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Please try again later.",
    "retry_after": 60
  }
}
```

---

## Versioning

**Current Version**: v1  
**Base Path**: `/api/v1/products`

Future breaking changes will increment the version (v2, v3, etc.). Non-breaking changes (new optional fields, new endpoints) don't require version bump.

---

## Testing

**Contract Tests**: Validate request/response schemas match this specification  
**Integration Tests**: Test with real MinIO instance  
**Security Tests**: Verify tenant isolation, unauthorized access blocked  
**Performance Tests**: Validate upload/retrieval times meet SLAs

---

## Summary

This API contract provides:
- ✅ Full CRUD operations for product photos
- ✅ Tenant isolation and authentication
- ✅ Comprehensive error handling
- ✅ Storage quota management
- ✅ Bulk operations (reorder)
- ✅ Enhanced product endpoints with photo data
- ✅ Rate limiting and security controls

Next: Create quickstart.md for developer onboarding.
