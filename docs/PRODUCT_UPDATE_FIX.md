# Product Update Response Fix

**Date:** December 1, 2024  
**Issue:** PUT /products/{id} returns incorrect `category_name` and `created_at` values

---

## 🐛 Issues Fixed

### 1. Missing `category_name` in Update Response
**Problem:** After updating a product, the response didn't include the category name.

**Root Cause:** The LEFT JOIN with the `categories` table was blocked by Row-Level Security (RLS) because it didn't include an explicit tenant filter.

**Fix:** Added tenant_id to the JOIN condition:
```go
LEFT JOIN categories c ON p.category_id = c.id AND c.tenant_id = p.tenant_id
```

### 2. Wrong `created_at` Value (Zero Time)
**Problem:** The `created_at` field showed `"0001-01-01T00:00:00Z"` instead of the actual creation timestamp.

**Root Cause:** Same as above - the JOIN failure caused the entire row to not be properly retrieved.

**Fix:** Same as above - properly filtering the JOIN allows all fields to be retrieved correctly.

---

## 📝 Changes Made

### File: `backend/product-service/src/repository/product_repository.go`

#### 1. Updated `FindByID` method (Line 124)
```go
// Before
LEFT JOIN categories c ON p.category_id = c.id

// After  
LEFT JOIN categories c ON p.category_id = c.id AND c.tenant_id = p.tenant_id
```

#### 2. Updated `FindAll` method (Line 56)
```go
// Before
LEFT JOIN categories c ON p.category_id = c.id

// After
LEFT JOIN categories c ON p.category_id = c.id AND c.tenant_id = p.tenant_id
```

---

## ✅ Test Results

### Before Fix
```json
{
    "id": "d0f5bea2-1fcb-45dd-ad6a-ef994c59a0c8",
    "category_id": "2cd45ca3-d21a-4e6d-b065-048ca3ea9793",
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "2025-12-01T16:39:11.210819Z"
}
```

### After Fix
```json
{
    "id": "d0f5bea2-1fcb-45dd-ad6a-ef994c59a0c8",
    "category_id": "2cd45ca3-d21a-4e6d-b065-048ca3ea9793",
    "category_name": "Drink",
    "created_at": "2025-12-01T15:51:08.993169Z",
    "updated_at": "2025-12-01T16:41:55.417786Z"
}
```

---

## 📚 Documentation Updated

Added new section to `BACKEND_CONVENTIONS.md`:

**Section:** "⚠️ CRITICAL: JOINs with RLS Tables"

**Key Points:**
1. Always add tenant filter to JOIN conditions when joining RLS-enabled tables
2. RLS policies apply to JOINed tables, not just the main table
3. LEFT JOINs may silently return NULL without proper filtering
4. Pattern: `LEFT JOIN table t ON main.id = t.id AND t.tenant_id = main.tenant_id`

---

## 🎯 Impact

- ✅ Product updates now return complete and correct data
- ✅ Category names properly displayed in UI
- ✅ Creation timestamps accurate
- ✅ Consistent with GET /products endpoint behavior
- ✅ Future JOINs will follow documented pattern

---

## 🔄 Related Files

1. `backend/product-service/src/repository/product_repository.go` - Fixed
2. `docs/BACKEND_CONVENTIONS.md` - Updated with JOIN pattern
3. This document - New fix documentation

---

## 💡 Lessons Learned

1. **RLS affects JOINs:** Row-Level Security policies apply to all tables in a query, including joined ones
2. **Silent failures:** LEFT JOINs can return NULL without errors when RLS blocks access
3. **Explicit is better:** Always add tenant_id filters to JOINs even with SET LOCAL RLS context
4. **Test thoroughly:** Compare responses across different endpoints (GET vs PUT) to catch inconsistencies
