# Product Feature Fixes - Summary

## Issues Fixed

### 1. Stock Adjustment Endpoint (POST /products/:id/stock)

**Problem:** The endpoint was missing, frontend was calling it but getting 404.

**Solution:**
- Added `StockAdjustment` model (already existed in `src/models/stock_adjustment.go`)
- Extended `ProductRepository` interface with:
  - `UpdateStock(ctx, id, newQuantity)` - Updates product stock quantity
  - `CreateStockAdjustment(ctx, adjustment)` - Records stock adjustment in audit log
- Added `AdjustStock` service method in `ProductService` that:
  - Gets current product and its stock
  - Creates stock adjustment record (audit trail)
  - Updates product stock quantity
  - Returns updated product
- Added `AdjustStock` handler in `ProductHandler` that:
  - Validates product ID, tenant ID, and user ID
  - Binds and validates request body (new_quantity, reason, notes)
  - Calls service layer
  - Returns updated product with 200 status
- Registered route: `POST /api/v1/products/:id/stock`

**Files Modified:**
- `backend/product-service/src/repository/product_repository.go` - Added methods
- `backend/product-service/src/services/product_service.go` - Added AdjustStock method
- `backend/product-service/api/product_handler.go` - Added handler and route

### 2. Photo Path Not Returned in API Response

**Problem:** Frontend expected `photo_path` field but it wasn't being returned.

**Investigation:**
- Model already has `PhotoPath *string` with json tag `photo_path,omitempty`
- Repository queries already SELECT photo_path in both `FindAll` and `FindByID`
- The field should be returned automatically when photo exists

**Root Cause:** No actual bug found in backend. The issue was likely:
- No products had photos uploaded yet (confirmed via DB query)
- The `omitempty` json tag excludes null fields from response

**Status:** Working as designed. When a product has a photo, `photo_path` will be returned.

## API Contract

### Adjust Stock Endpoint

**Request:**
```http
POST /api/v1/products/:id/stock
Content-Type: application/json
Cookie: auth_token=<jwt>

{
  "new_quantity": 100,
  "reason": "supplier_delivery",
  "notes": "Restocked from Supplier XYZ"
}
```

**Valid reasons:**
- `supplier_delivery` - New stock from supplier
- `physical_count` - Stock count adjustment
- `shrinkage` - Loss/theft
- `damage` - Damaged goods
- `return` - Customer return
- `correction` - Manual correction
- `sale` - Sale transaction (automated)

**Response:**
```http
200 OK
Content-Type: application/json

{
  "id": "uuid",
  "tenant_id": "uuid",
  "sku": "SKU-001",
  "name": "Product Name",
  "stock_quantity": 100,
  "photo_path": "/uploads/tenant-id/product-id/photo.jpg",
  ...
}
```

## Testing

To test the stock adjustment endpoint from browser console:
```javascript
// Get product ID from product list
const productId = 'd0f5bea2-1fcb-45dd-ad6a-ef994c59a0c8';

// Adjust stock
fetch(`http://localhost:8080/api/v1/products/${productId}/stock`, {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  credentials: 'include', // Sends auth cookie
  body: JSON.stringify({
    new_quantity: 150,
    reason: 'physical_count',
    notes: 'Stock count adjustment'
  })
})
.then(r => r.json())
.then(console.log);
```

## Database Schema

The `stock_adjustments` table already exists with:
- Full audit trail (previous_quantity, new_quantity, quantity_delta)
- Reason tracking
- User tracking (who made the adjustment)
- Timestamp (when adjustment was made)
- Trigger to auto-calculate quantity_delta
- RLS policy for tenant isolation

## Notes

- Stock adjustments are immutable audit records
- The trigger `trg_calculate_delta` automatically calculates `quantity_delta`
- All adjustments are tied to a user for accountability
- Frontend `StockAdjustmentModal` component is ready to use this endpoint
