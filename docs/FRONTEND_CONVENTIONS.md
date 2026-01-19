# Frontend Development Conventions

**Last Updated:** December 1, 2024  
**Project:** Point of Sale System  
**Framework:** Next.js 14+ with TypeScript

---

## 📁 File Naming Conventions

### Services (`/src/services/`)

```
✅ Correct:   auth.ts, user.ts, product.ts
❌ Incorrect: auth.service.ts, user-service.ts, UserService.ts
```

**Pattern:** `<feature-name>.ts` (lowercase, no suffixes)

**Structure:**

```typescript
// user.ts
import apiClient from './api';

class UserService {
  async getUser(id: string) { ... }
  async updateUser(id: string, data) { ... }
}

export default new UserService();
```

---

### Types (`/src/types/`)

```
✅ Correct:   product.ts, auth.ts, user.ts
❌ Incorrect: product.types.ts, IProduct.ts, ProductTypes.ts
```

**Pattern:** `<feature-name>.ts` (lowercase, no suffixes)

**Structure:**

```typescript
// product.ts
export interface Product {
  id: string
  name: string
  // ...
}

export interface CreateProductRequest {
  name: string
  // ...
}

export interface UpdateProductRequest {
  name?: string
  // ...
}
```

---

### Components (`/src/components/<feature>/`)

```
✅ Correct:   ProductForm.tsx, ProductList.tsx, CategorySelect.tsx
❌ Incorrect: product-form.tsx, productList.tsx, product_form.tsx
```

**Pattern:** `PascalCase.tsx`

**Structure:**

```typescript
// ProductForm.tsx
'use client';

import React from 'react';
import { useTranslation } from '@/i18n/provider';

interface ProductFormProps {
  initialData?: Product;
  onSubmit: (data: CreateProductRequest) => void;
  onCancel: () => void;
}

const ProductForm: React.FC<ProductFormProps> = ({
  initialData,
  onSubmit,
  onCancel
}) => {
  const { t } = useTranslation(['products', 'common']);

  return (
    <form>
      {/* Component JSX */}
    </form>
  );
};

export default ProductForm;
```

---

### Pages (`/app/<feature>/`)

```
app/
├── products/
│   ├── page.tsx              ✅ List page
│   ├── new/
│   │   └── page.tsx          ✅ Create page
│   ├── [id]/
│   │   └── page.tsx          ✅ Detail/Edit page
│   └── categories/
│       └── page.tsx          ✅ Sub-feature page
```

**Pattern:** Next.js App Router conventions

---

## 🌍 i18n (Internationalization)

### Locale Files (`/src/i18n/locales/<lang>/`)

```
locales/
├── en/
│   ├── common.json    ✅ Shared translations
│   ├── auth.json      ✅ Authentication feature
│   ├── products.json  ✅ Products feature
│   └── ...
├── id/
│   ├── common.json
│   ├── auth.json
│   ├── products.json
│   └── ...
```

**Pattern:** `<feature>.json` (lowercase, singular or plural based on context)

### JSON Structure

```json
{
  "<namespace>": {
    "key": "Value",
    "nested": {
      "key": "Nested value"
    },
    "list": {
      "item1": "Item 1",
      "item2": "Item 2"
    }
  }
}
```

**Example - products.json:**

```json
{
  "products": {
    "title": "Products",
    "subtitle": "Manage your product catalog",
    "addProduct": "Add Product",

    "form": {
      "name": "Product Name",
      "namePlaceholder": "Enter product name"
    },

    "messages": {
      "createSuccess": "Product created successfully",
      "createError": "Failed to create product"
    },

    "validation": {
      "nameRequired": "Product name is required",
      "nameTooLong": "Product name is too long"
    }
  }
}
```

---

### Using i18n in Components

#### 1. Import the hook

```typescript
import { useTranslation } from '@/i18n/provider'
```

#### 2. Use in component

```typescript
const ProductsPage = () => {
  // Load multiple namespaces
  const { t } = useTranslation(['products', 'common']);

  return (
    <div>
      <h1>{t('products.title')}</h1>
      <p>{t('products.subtitle')}</p>
      <button>{t('common.save')}</button>

      {/* Nested keys */}
      <label>{t('products.form.name')}</label>

      {/* With fallback */}
      <span>{t('products.optional', 'Default Value')}</span>
    </div>
  );
};
```

#### 3. Dynamic translations

```typescript
// Error messages
const errorMsg = t(`products.validation.${errorKey}`)

// Success messages
const successMsg = t('products.messages.createSuccess')

// Conditional translations
const statusText = archived ? t('products.list.archived') : t('products.list.inStock')
```

---

## 📋 Feature Implementation Checklist

When adding a new feature (e.g., "Orders"), follow these steps:

### 1. Types

```bash
✅ Create: /src/types/order.ts
✅ Export: Order, CreateOrderRequest, UpdateOrderRequest, etc.
```

### 2. Service

```bash
✅ Create: /src/services/order.ts
✅ Export: default new OrderService()
```

### 3. i18n Locale Files

```bash
✅ Create: /src/i18n/locales/en/orders.json
✅ Create: /src/i18n/locales/id/orders.json
✅ Structure: Follow products.json pattern
```

### 4. Components

```bash
✅ Create: /src/components/orders/OrderForm.tsx
✅ Create: /src/components/orders/OrderList.tsx
✅ Create: /src/components/orders/OrderItem.tsx
✅ Use: useTranslation(['orders', 'common'])
```

### 5. Pages

```bash
✅ Create: /app/orders/page.tsx
✅ Create: /app/orders/new/page.tsx
✅ Create: /app/orders/[id]/page.tsx
✅ Wrap: ProtectedRoute with RBAC
✅ Layout: Use DashboardLayout
```

### 6. RBAC (if needed)

```typescript
<ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
  <DashboardLayout>
    {/* Page content */}
  </DashboardLayout>
</ProtectedRoute>
```

### 7. Update Sidebar

```typescript
// /src/components/layout/Sidebar.tsx
{
  name: t('orders.title'),
  href: '/orders',
  roles: [ROLES.OWNER, ROLES.MANAGER, ROLES.CASHIER],
  icon: <OrderIcon />
}
```

---

## 🎨 Code Style Conventions

### Import Order

```typescript
// 1. React and Next.js
import React, { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'

// 2. Third-party libraries
import axios from 'axios'

// 3. Project imports (absolute paths with @/)
import { useAuth } from '@/store/auth'
import DashboardLayout from '@/components/layout/DashboardLayout'
import ProtectedRoute from '@/components/auth/ProtectedRoute'
import ProductForm from '@/components/products/ProductForm'

// 4. Services and types
import productService from '@/services/product'
import { Product, CreateProductRequest } from '@/types/product'

// 5. Constants and utilities
import { ROLES } from '@/constants/roles'
import { useTranslation } from '@/i18n/provider'

// 6. Styles (if any)
import styles from './styles.module.css'
```

### Component Structure

```typescript
'use client'; // If using hooks or client-side features

import React from 'react';
import { useTranslation } from '@/i18n/provider';

interface ComponentProps {
  // Props definition
}

const ComponentName: React.FC<ComponentProps> = ({
  prop1,
  prop2
}) => {
  // 1. Hooks
  const { t } = useTranslation(['feature', 'common']);
  const [state, setState] = useState();

  // 2. Effects
  useEffect(() => {
    // Effect logic
  }, []);

  // 3. Handlers
  const handleAction = () => {
    // Handler logic
  };

  // 4. Render helpers (if needed)
  const renderItem = () => {
    // Render logic
  };

  // 5. Early returns
  if (loading) return <Loading />;
  if (error) return <Error />;

  // 6. Main render
  return (
    <div>
      {/* JSX */}
    </div>
  );
};

export default ComponentName;
```

---

## 🔐 Authentication & RBAC

### Protected Routes

```typescript
<ProtectedRoute allowedRoles={[ROLES.OWNER, ROLES.MANAGER]}>
  <DashboardLayout>
    <PageContent />
  </DashboardLayout>
</ProtectedRoute>
```

### Sidebar Menu Filtering

```typescript
const navigationItems = [
  {
    name: t('products.title'),
    href: '/products',
    roles: [ROLES.OWNER, ROLES.MANAGER], // Only these roles see it
    icon: <ProductIcon />
  }
];
```

---

## 📦 API Service Pattern

```typescript
class FeatureService {
  // GET list with pagination
  async getItems(params?: ListParams): Promise<PaginatedResponse<Item>> {
    const queryParams = new URLSearchParams()
    if (params?.page) queryParams.append('page', params.page.toString())
    if (params?.limit) queryParams.append('limit', params.limit.toString())

    const url = queryParams.toString() ? `${BASE_URL}?${queryParams}` : BASE_URL

    return apiClient.get<PaginatedResponse<Item>>(url)
  }

  // GET single item
  async getItem(id: string): Promise<Item> {
    return apiClient.get<Item>(`${BASE_URL}/${id}`)
  }

  // POST create
  async createItem(data: CreateItemRequest): Promise<Item> {
    return apiClient.post<Item>(BASE_URL, data)
  }

  // PUT update
  async updateItem(id: string, data: UpdateItemRequest): Promise<Item> {
    return apiClient.put<Item>(`${BASE_URL}/${id}`, data)
  }

  // DELETE
  async deleteItem(id: string): Promise<void> {
    return apiClient.delete(`${BASE_URL}/${id}`)
  }
}

export default new FeatureService()
```

---

## 📸 Photo URL Handling

### Backend vs Frontend

- **Backend** returns `photo_path` (relative path like `/uploads/products/123.jpg`)
- **Frontend** must construct full URL to display images

### Correct Implementation

```typescript
// ✅ In service file (product.ts)
class ProductService {
  getPhotoUrl(id: string): string {
    return `${apiClient.getAxiosInstance().defaults.baseURL}/api/v1/products/${id}/photo`;
  }
}

// ✅ In component
<img src={productService.getPhotoUrl(productId)} alt={product.name} />

// ❌ Wrong - won't work
<img src={product.photo_path} alt={product.name} />
```

### Error Handling for Images

```typescript
<img
  src={productService.getPhotoUrl(productId)}
  alt={product.name}
  onError={(e) => {
    const img = e.target as HTMLImageElement;
    img.style.display = 'none'; // Hide broken image
  }}
/>
```

---

## 💰 Number and Currency Formatting

### Utility Functions (`/src/utils/format.ts`)

**Available formatters:**

```typescript
// Format with thousand separators (no currency symbol)
formatNumber(value: number, decimals?: number): string

// Compact format for large numbers (1.2M, 3.5B)
formatCompactNumber(value: number, decimals?: number): string

// Responsive format (full on desktop, compact on mobile)
formatResponsiveNumber(value: number, isMobile?: boolean, decimals?: number): string

// Parse formatted string back to number
parseFormattedNumber(value: string): number
```

### Display Formatting (Read-only)

**For prices and monetary values:**

```typescript
import { formatNumber } from '@/utils/format';

// Display price without decimals (whole numbers)
<span>{formatNumber(product.selling_price, 0)}</span>
// Output: 1,250,000

// Display with 2 decimals if needed
<span>{formatNumber(product.cost, 2)}</span>
// Output: 1,250,000.50
```

**For large inventory values (responsive):**

```typescript
import { formatNumber, formatCompactNumber } from '@/utils/format';

// Desktop: full number, Mobile: compact
<span className="text-3xl font-bold">
  <span className="hidden sm:inline">{formatNumber(totalValue, 0)}</span>
  <span className="sm:hidden">{formatCompactNumber(totalValue)}</span>
</span>
// Desktop: 1,250,000,000
// Mobile: 1.3B
```

### Input Field Formatting (Editable)

**For form inputs with live formatting:**

```typescript
import { formatNumber, parseFormattedNumber } from '@/utils/format';

const [formData, setFormData] = useState({ price: '' });
const [displayPrice, setDisplayPrice] = useState('');

// On input change - allow typing
const handlePriceChange = (value: string) => {
  const cleaned = value.replace(/,/g, ''); // Remove commas for storage
  setFormData({ ...formData, price: cleaned });
  setDisplayPrice(value); // Keep user's input
};

// On blur - format the number
const handlePriceBlur = () => {
  const value = parseFloat(formData.price);
  if (!isNaN(value)) {
    setDisplayPrice(formatNumber(value, 0));
  }
};

// In JSX
<input
  type="text"
  value={displayPrice}
  onChange={(e) => handlePriceChange(e.target.value)}
  onBlur={handlePriceBlur}
  placeholder="0"
/>
```

### Rules & Best Practices

1. **NO currency symbols** - Remove Rp, $, etc. for now
2. **Use thousand separators** - Always format numbers ≥1,000
3. **No decimals for prices** - Indonesian Rupiah doesn't use cents
4. **Responsive large numbers** - Use compact format on mobile (1.2M vs 1,200,000)
5. **Separate storage from display**:
   - Store: raw numbers without formatting
   - Display: formatted with separators
6. **Input fields**: Format on blur, not on every keystroke
7. **Consistency**: Use same decimals across similar fields

### Examples

```typescript
// ✅ Correct
{formatNumber(1234567, 0)}           // "1,234,567"
{formatCompactNumber(1234567)}       // "1.2M"
{formatNumber(totalValue, 0)}        // "1,250,000"

// ❌ Incorrect
{product.price}                      // 1250000 (no separators)
Rp {formatNumber(price, 0)}          // Rp 1,250,000 (no currency yet)
{formatNumber(price, 2)}             // 1,250,000.00 (unnecessary decimals)
```

---

## ✅ Summary Checklist

When implementing a new feature:

- [ ] **Types**: Create `/src/types/<feature>.ts`
- [ ] **Service**: Create `/src/services/<feature>.ts`
- [ ] **i18n**: Create locale files for all languages
- [ ] **Components**: Create in `/src/components/<feature>/`
- [ ] **Pages**: Create in `/app/<feature>/`
- [ ] **RBAC**: Add ProtectedRoute wrappers
- [ ] **Sidebar**: Update navigation with role filtering
- [ ] **Translations**: Use `useTranslation` in all components
- [ ] **Testing**: Test with different user roles
- [ ] **Documentation**: Update this file if conventions change

---

## 🔍 Quick Reference

| Item            | Pattern                        | Example                               |
| --------------- | ------------------------------ | ------------------------------------- |
| Service file    | `<feature>.ts`                 | `product.ts`                          |
| Type file       | `<feature>.ts`                 | `product.ts`                          |
| Component       | `PascalCase.tsx`               | `ProductForm.tsx`                     |
| Locale file     | `<feature>.json`               | `products.json`                       |
| Translation key | `feature.section.key`          | `products.form.name`                  |
| Import path     | `@/<path>`                     | `@/services/product`                  |
| Export service  | `export default new Service()` | `export default new ProductService()` |

---

## 📧 Notification Feature Patterns

### Type Organization

**Pattern:** Separate types from service implementations

```typescript
// /src/types/notification.ts
export interface NotificationHistoryItem {
  id: number
  tenant_id: string
  user_id: string
  type: 'staff' | 'customer'
  status: 'sent' | 'pending' | 'failed' | 'cancelled'
  subject: string
  recipient: string
  sent_at?: string
  failed_at?: string
  error_msg?: string
  retry_count: number
  metadata: {
    order_reference?: string
    transaction_id?: string
    event_type?: string
  }
  created_at: string
}

export interface NotificationHistoryFilters {
  page?: number
  page_size?: number
  order_reference?: string
  status?: string
  type?: string
  start_date?: string
  end_date?: string
}

export interface NotificationHistoryResponse {
  notifications: NotificationHistoryItem[]
  pagination: {
    current_page: number
    page_size: number
    total_items: number
    total_pages: number
  }
}
```

```typescript
// /src/services/notification.ts
import { NotificationHistoryFilters, NotificationHistoryResponse } from '@/types/notification'

class NotificationService {
  async getNotificationHistory(
    filters: NotificationHistoryFilters
  ): Promise<NotificationHistoryResponse> {
    // Implementation
  }
}
```

**Key Points:**

- Define types in separate files under `/src/types/`
- Import types in service files (don't define inline)
- Use union types for constrained strings (`'sent' | 'pending'`)
- Make optional fields explicit with `?`

### Dashboard Component Structure

**Pattern:** Comprehensive state management with filter, pagination, and action handling

```typescript
const NotificationHistory: React.FC = () => {
  const { t } = useTranslation(['notifications', 'common'])

  // Data state
  const [notifications, setNotifications] = useState<NotificationHistoryItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)
  const [totalItems, setTotalItems] = useState(0)

  // Filter state
  const [orderReference, setOrderReference] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  // Action state
  const [resendingId, setResendingId] = useState<number | null>(null)
  const [resendSuccess, setResendSuccess] = useState<string | null>(null)
  const [resendError, setResendError] = useState<string | null>(null)
  const [expandedError, setExpandedError] = useState<number | null>(null)

  // Fetch data
  useEffect(() => {
    fetchNotifications()
  }, [currentPage])

  const fetchNotifications = async () => {
    setLoading(true)
    setError(null)

    try {
      const filters: NotificationHistoryFilters = {
        page: currentPage,
        page_size: 20,
      }

      if (orderReference) filters.order_reference = orderReference
      if (statusFilter) filters.status = statusFilter
      if (typeFilter) filters.type = typeFilter
      if (startDate) filters.start_date = startDate
      if (endDate) filters.end_date = endDate

      const response = await notificationService.getNotificationHistory(filters)
      setNotifications(response.notifications)
      setTotalPages(response.pagination.total_pages)
      setTotalItems(response.pagination.total_items)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load notifications')
    } finally {
      setLoading(false)
    }
  }

  const handleResend = async (notificationId: number) => {
    setResendingId(notificationId)
    setResendSuccess(null)
    setResendError(null)

    try {
      await notificationService.resendNotification(notificationId)
      setResendSuccess(t('notifications.history.resend_success'))
      fetchNotifications() // Refresh list
    } catch (err) {
      if (err instanceof Error) {
        if (err.message.includes('429')) {
          setResendError(t('notifications.history.resend_rate_limit'))
        } else if (err.message.includes('409')) {
          setResendError(t('notifications.history.resend_conflict'))
        } else {
          setResendError(err.message)
        }
      }
    } finally {
      setResendingId(null)
    }
  }

  // ... render JSX
}
```

**Key Points:**

- Separate concerns: data, pagination, filters, actions
- Use descriptive state variable names
- Handle loading and error states
- Provide user feedback for all actions
- Refresh data after state-changing operations

### Status Badge Pattern

**Pattern:** Consistent color-coding for status indicators

```typescript
const getStatusBadgeColor = (status: string) => {
  switch (status) {
    case 'sent':
      return 'bg-green-100 text-green-800';
    case 'failed':
      return 'bg-red-100 text-red-800';
    case 'pending':
      return 'bg-yellow-100 text-yellow-800';
    case 'cancelled':
      return 'bg-gray-100 text-gray-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
};

// In JSX
<span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusBadgeColor(notification.status)}`}>
  {t(`notifications.status.${notification.status}`)}
</span>
```

**Color Standards:**

- **Green**: Success, sent, active
- **Red**: Error, failed, rejected
- **Yellow**: Warning, pending, in-progress
- **Gray**: Neutral, cancelled, disabled
- **Blue**: Info, default

### Filter Controls

**Pattern:** Grouped filters with search button

```typescript
<div className="mb-6 bg-white p-4 rounded-lg shadow">
  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-2">
        {t('notifications.history.filter_order_ref')}
      </label>
      <input
        type="text"
        value={orderReference}
        onChange={(e) => setOrderReference(e.target.value)}
        className="w-full px-3 py-2 border rounded-md"
        placeholder="GO-2024-001"
      />
    </div>

    <div>
      <label className="block text-sm font-medium text-gray-700 mb-2">
        {t('notifications.history.filter_status')}
      </label>
      <select
        value={statusFilter}
        onChange={(e) => setStatusFilter(e.target.value)}
        className="w-full px-3 py-2 border rounded-md"
      >
        <option value="">{t('common.all')}</option>
        <option value="sent">{t('notifications.status.sent')}</option>
        <option value="failed">{t('notifications.status.failed')}</option>
        <option value="pending">{t('notifications.status.pending')}</option>
      </select>
    </div>

    <div>
      <label className="block text-sm font-medium text-gray-700 mb-2">
        {t('notifications.history.filter_type')}
      </label>
      <select
        value={typeFilter}
        onChange={(e) => setTypeFilter(e.target.value)}
        className="w-full px-3 py-2 border rounded-md"
      >
        <option value="">{t('common.all')}</option>
        <option value="staff">{t('notifications.type.staff')}</option>
        <option value="customer">{t('notifications.type.customer')}</option>
      </select>
    </div>
  </div>

  <div className="mt-4 flex justify-end">
    <button
      onClick={() => {
        setCurrentPage(1);
        fetchNotifications();
      }}
      className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
    >
      {t('common.search')}
    </button>
  </div>
</div>
```

**Key Points:**

- Group related filters
- Use responsive grid layout (mobile-first)
- Provide "All" option for dropdowns
- Reset to page 1 when applying filters
- Translate all labels and options

### Pagination Controls

**Pattern:** Standard prev/next with page indicator

```typescript
<div className="flex justify-between items-center mt-6">
  <div className="text-sm text-gray-600">
    {t('common.showing')} {((currentPage - 1) * 20) + 1}-{Math.min(currentPage * 20, totalItems)} {t('common.of')} {totalItems}
  </div>

  <div className="flex gap-2">
    <button
      onClick={() => setCurrentPage(currentPage - 1)}
      disabled={currentPage === 1}
      className="px-4 py-2 border rounded-md disabled:opacity-50 disabled:cursor-not-allowed"
    >
      {t('common.previous')}
    </button>

    <span className="px-4 py-2 border rounded-md bg-gray-50">
      {t('common.page')} {currentPage} {t('common.of')} {totalPages}
    </span>

    <button
      onClick={() => setCurrentPage(currentPage + 1)}
      disabled={currentPage >= totalPages}
      className="px-4 py-2 border rounded-md disabled:opacity-50 disabled:cursor-not-allowed"
    >
      {t('common.next')}
    </button>
  </div>
</div>
```

**Key Points:**

- Show result count with range
- Disable buttons at boundaries
- Display current page / total pages
- Use consistent button styling

### E2E Testing Attributes

**Pattern:** Add data-testid for all interactive elements

```typescript
// In component
<div data-testid="notification-list">
  {notifications.map((notification) => (
    <div key={notification.id} data-testid={`notification-item-${notification.id}`}>
      <span data-testid={`status-badge-${notification.id}`}>
        {notification.status}
      </span>
      <span data-testid={`order-reference-${notification.id}`}>
        {notification.metadata.order_reference}
      </span>
      <button
        data-testid={`resend-button-${notification.id}`}
        onClick={() => handleResend(notification.id)}
      >
        {t('notifications.history.resend')}
      </button>
    </div>
  ))}
</div>

// In E2E test
await page.waitForSelector('[data-testid="notification-list"]');
const firstItem = await page.locator('[data-testid^="notification-item-"]').first();
const statusBadge = await firstItem.locator('[data-testid^="status-badge-"]');
expect(await statusBadge.textContent()).toBe('sent');
```

**Key Points:**

- Use descriptive `data-testid` attributes
- Include IDs for dynamic elements (`notification-item-${id}`)
- Test by test ID, not by class names or text content
- Keep test IDs stable across UI changes

---

**Follow these conventions for consistent, maintainable code across the project!** 🚀
