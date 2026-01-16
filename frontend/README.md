# POS System Frontend

**Framework**: Next.js 14 (App Router)  
**Language**: TypeScript 5.3+  
**Styling**: Tailwind CSS 3.4  
**State Management**: React Context + SWR  
**UI Components**: shadcn/ui  
**Internationalization**: next-intl

---

## Table of Contents

1. [Features](#features)
2. [Quick Start](#quick-start)
3. [UU PDP Compliance UI](#uu-pdp-compliance-ui)
4. [Project Structure](#project-structure)
5. [Development](#development)
6. [Testing](#testing)
7. [Deployment](#deployment)
8. [Troubleshooting](#troubleshooting)

---

## Features

### Core Features

- **Multi-tenant SaaS**: Tenant-scoped data access with role-based permissions
- **Authentication**: Secure login with JWT tokens and session management
- **Dashboard**: Real-time business metrics and analytics
- **Product Management**: Full CRUD for product catalog with pricing
- **Order Processing**: Guest orders, QRIS payments, invoice generation
- **Team Management**: Invite users, assign roles (OWNER, ADMIN, STAFF)
- **Notifications**: Real-time email notifications for orders and team events

### UU PDP Compliance Features

- **Consent Management**: Purpose-based consent collection and revocation UI
- **Privacy Policy**: Bilingual privacy policy (Indonesian + English) with SSR
- **Tenant Data Rights**: View all data, export JSON, delete team members
- **Guest Data Rights**: Order lookup and data deletion for guest customers
- **Retention Policies**: Admin UI for configuring data retention periods
- **Audit Transparency**: View audit logs for data access and modifications

---

## Quick Start

### Prerequisites

- **Node.js**: 18.x or 20.x (LTS)
- **npm**: 9.x or higher (or yarn/pnpm)
- **Backend API**: Running on `http://localhost:8080` (see backend/README.md)

### Installation

```bash
# 1. Navigate to frontend directory
cd point-of-sale-system/frontend

# 2. Install dependencies
npm install

# 3. Copy environment variables
cp .env.example .env.local

# 4. Edit .env.local with your configuration
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_APP_NAME="POS System"
```

### Development Server

```bash
# Start development server
npm run dev

# Open browser
open http://localhost:3000
```

### Build for Production

```bash
# Create optimized production build
npm run build

# Test production build locally
npm run start

# Run on different port
PORT=3001 npm run start
```

---

## UU PDP Compliance UI

The frontend implements UI components for Indonesian Personal Data Protection Law (UU PDP No.27/2022) compliance.

### Consent Collection Components

#### Registration Form with Consent

Located in: `app/(auth)/register/page.tsx`

**Features**:

- Displays all consent purposes (operational, analytics, advertising, payment)
- Required purposes pre-checked and disabled
- Optional purposes require explicit opt-in
- Bilingual labels (Indonesian + English via next-intl)
- Form validation prevents submission without required consents

**Usage Example**:

```tsx
import { ConsentCheckbox } from '@/src/components/privacy/ConsentCheckbox';

<ConsentCheckbox
  purposeCode="analytics"
  label="I consent to usage analytics for service improvement"
  isRequired={false}
  checked={consents.analytics}
  onChange={checked => setConsents({ ...consents, analytics: checked })}
/>;
```

#### Guest Checkout Consent

Located in: `app/(guest)/checkout/page.tsx`

Same consent UI for guest orders, consent stored with order reference.

### Privacy Settings Component

Located in: `src/components/privacy/ConsentSettingsSection.tsx`

**Features**:

- View current consent status for all purposes
- Revoke optional consents with toggle switches
- Real-time status updates
- Audit trail display (when consent was granted/revoked)

**Usage Example**:

```tsx
import { ConsentSettingsSection } from '@/src/components/privacy/ConsentSettingsSection';

// In settings page
<ConsentSettingsSection />;
```

**API Integration**:

```tsx
// src/services/consent.ts
export const getConsentStatus = async (): Promise<ConsentStatus[]> => {
  const res = await fetch('/api/v1/consent/status');
  return res.json();
};

export const revokeConsent = async (purposeCode: string): Promise<void> => {
  await fetch('/api/v1/consent/revoke', {
    method: 'POST',
    body: JSON.stringify({ purpose_code: purposeCode }),
  });
};
```

### Tenant Data Rights UI

Located in: `app/(dashboard)/settings/tenant-data/page.tsx`

**Features**:

- View all tenant data (profile, users, configurations, consents)
- Export data to JSON (one-click download)
- Delete team members with 90-day grace period
- OWNER role required (automatic permission check)

**Component Structure**:

```tsx
<DashboardLayout>
  <h1>{t('tenant_data.title')}</h1>

  {/* View Data Section */}
  <TenantDataViewer data={tenantData} />

  {/* Export Section */}
  <Button onClick={handleExport}>
    <Download /> {t('tenant_data.export_button')}
  </Button>

  {/* Delete Team Member Section */}
  <TeamMemberList users={users} onDelete={handleDelete} />
</DashboardLayout>
```

### Guest Data Rights UI

Located in: `app/(guest)/data-lookup/page.tsx`

**Features**:

- Order lookup by reference + (email OR phone)
- View order data without authentication
- Request data deletion (anonymization)
- Bilingual interface

**Lookup Form**:

```tsx
<form onSubmit={handleLookup}>
  <Input name="order_reference" placeholder="ORD-2026-001" required />
  <Input name="email" placeholder="customer@example.com" type="email" />
  <Input name="phone" placeholder="+628123456789" type="tel" />
  <Button type="submit">{t('lookup.submit')}</Button>
</form>
```

**Deletion Confirmation**:

```tsx
<Dialog open={showDeleteDialog}>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>{t('data_deletion.confirm_title')}</DialogTitle>
      <DialogDescription>{t('data_deletion.warning_irreversible')}</DialogDescription>
    </DialogHeader>
    <DialogFooter>
      <Button variant="outline" onClick={() => setShowDeleteDialog(false)}>
        {t('common.cancel')}
      </Button>
      <Button variant="destructive" onClick={handleConfirmDelete}>
        {t('data_deletion.confirm_button')}
      </Button>
    </DialogFooter>
  </DialogContent>
</Dialog>
```

### Retention Policies Admin UI

Located in: `app/(dashboard)/admin/retention-policies/page.tsx`

**Features**:

- View all retention policies with current settings
- Inline editing of retention periods
- Validation alert for legal minimums (cannot set below 7 years for audit logs)
- Preview expired record counts before cleanup
- OWNER role required

**Component Example**:

```tsx
<Table>
  <TableHeader>
    <TableRow>
      <TableHead>{t('retention.table_name')}</TableHead>
      <TableHead>{t('retention.retention_period')}</TableHead>
      <TableHead>{t('retention.legal_minimum')}</TableHead>
      <TableHead>{t('retention.actions')}</TableHead>
    </TableRow>
  </TableHeader>
  <TableBody>
    {policies.map(policy => (
      <TableRow key={policy.id}>
        <TableCell>{policy.table_name}</TableCell>
        <TableCell>
          <Input
            type="number"
            value={editingValues[policy.id] || policy.retention_period_days}
            onChange={e => handleEdit(policy.id, e.target.value)}
            min={policy.legal_minimum_days}
          />
          {editingValues[policy.id] < policy.legal_minimum_days && (
            <Alert variant="destructive">{t('retention.validation.below_legal_minimum')}</Alert>
          )}
        </TableCell>
        <TableCell>{policy.legal_minimum_days} days</TableCell>
        <TableCell>
          <Button onClick={() => handleSave(policy.id)}>{t('common.save')}</Button>
        </TableCell>
      </TableRow>
    ))}
  </TableBody>
</Table>
```

### Privacy Policy Page

Located in: `app/privacy-policy/page.tsx`

**Features**:

- Bilingual content (Indonesian + English)
- Server-Side Rendering (SSR) for SEO
- Accessible from footer on all pages
- No authentication required
- Versioned content (shows latest version)

**Implementation**:

```tsx
// SSR for SEO
export default async function PrivacyPolicyPage() {
  // Fetch privacy policy server-side
  const policy = await getPrivacyPolicy();

  return (
    <div className="max-w-4xl mx-auto py-12 px-6">
      <h1 className="text-4xl font-bold mb-8">{policy.title}</h1>
      <div className="prose prose-lg">
        <Markdown>{policy.content}</Markdown>
      </div>
      <footer className="mt-8 text-sm text-gray-600">
        Version {policy.version} - Effective {policy.effective_date}
      </footer>
    </div>
  );
}
```

---

## Project Structure

```
frontend/
├── app/                          # Next.js App Router
│   ├── (auth)/                   # Authentication routes (layout group)
│   │   ├── login/
│   │   └── register/
│   ├── (dashboard)/              # Authenticated routes
│   │   ├── dashboard/
│   │   ├── products/
│   │   ├── orders/
│   │   ├── team/
│   │   ├── settings/
│   │   │   ├── privacy/          # Consent settings
│   │   │   └── tenant-data/      # Data rights
│   │   └── admin/
│   │       └── retention-policies/
│   ├── (guest)/                  # Public routes
│   │   ├── order-lookup/
│   │   └── data-lookup/          # Guest data rights
│   ├── privacy-policy/
│   └── layout.tsx                # Root layout
├── src/
│   ├── components/
│   │   ├── layout/               # DashboardLayout, Header, Footer
│   │   ├── privacy/              # Consent components
│   │   │   ├── ConsentCheckbox.tsx
│   │   │   ├── ConsentSettingsSection.tsx
│   │   │   └── GuestPrivacySettings.tsx
│   │   └── ui/                   # shadcn/ui components
│   ├── services/                 # API clients
│   │   ├── auth.ts
│   │   ├── consent.ts
│   │   ├── tenantData.ts
│   │   ├── guestData.ts
│   │   └── retention.ts
│   ├── hooks/                    # React hooks
│   │   ├── useAuth.ts
│   │   └── useConsent.ts
│   ├── context/                  # React Context providers
│   │   └── AuthContext.tsx
│   └── lib/                      # Utilities
│       ├── api.ts                # API base client
│       └── utils.ts
├── messages/                     # i18n translations
│   ├── en.json                   # English
│   └── id.json                   # Indonesian (Bahasa)
├── public/
│   └── images/
├── tailwind.config.ts
├── next.config.js
└── package.json
```

---

## Development

### Code Conventions

See: **[/docs/FRONTEND_CONVENTIONS.md](../docs/FRONTEND_CONVENTIONS.md)**

**Key Principles**:

- Use TypeScript for all files (no plain JS)
- Follow Next.js App Router patterns (server/client components)
- Use shadcn/ui for consistent UI components
- Implement bilingual support with next-intl
- Use SWR for data fetching with caching
- Follow accessibility best practices (ARIA labels, semantic HTML)

### Internationalization (i18n)

**Setup**: next-intl with locale files in `messages/`

**Usage in Components**:

```tsx
import { useTranslations } from 'next-intl';

export default function MyComponent() {
  const t = useTranslations('dashboard');

  return (
    <h1>{t('title')}</h1>
    <p>{t('description', { name: 'John' })}</p>
  );
}
```

**Translation Files**:

```json
// messages/en.json
{
  "dashboard": {
    "title": "Dashboard",
    "description": "Welcome, {name}"
  }
}

// messages/id.json
{
  "dashboard": {
    "title": "Dasbor",
    "description": "Selamat datang, {name}"
  }
}
```

**See**: [/docs/i18n-quick-reference.md](../docs/i18n-quick-reference.md)

### Styling with Tailwind CSS

**Usage**:

```tsx
<div className="flex items-center justify-between p-4 bg-white rounded-lg shadow-md">
  <h2 className="text-xl font-bold text-gray-800">Title</h2>
  <Button variant="primary">Action</Button>
</div>
```

**Custom Theme** (tailwind.config.ts):

```ts
theme: {
  extend: {
    colors: {
      primary: '#3b82f6',
      secondary: '#64748b',
      accent: '#f59e0b',
    },
  },
}
```

### API Integration

**Base API Client** (`src/lib/api.ts`):

```ts
export const apiClient = {
  get: async <T>(url: string): Promise<T> => {
    const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}${url}`, {
      headers: {
        Authorization: `Bearer ${getToken()}`,
      },
    });
    if (!res.ok) throw new Error('API Error');
    return res.json();
  },
  post: async <T>(url: string, data: any): Promise<T> => {
    const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}${url}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${getToken()}`,
      },
      body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error('API Error');
    return res.json();
  },
};
```

**Service Layer** (`src/services/consent.ts`):

```ts
export const consentService = {
  getStatus: () => apiClient.get<ConsentStatus[]>('/consent/status'),
  grant: (data: GrantConsentRequest) => apiClient.post('/consent/grant', data),
  revoke: (purposeCode: string) => apiClient.post('/consent/revoke', { purpose_code: purposeCode }),
};
```

**Usage in Components**:

```tsx
import useSWR from 'swr';
import { consentService } from '@/src/services/consent';

export default function ConsentSettings() {
  const { data: consents, error, mutate } = useSWR('/consent/status', consentService.getStatus);

  const handleRevoke = async (purposeCode: string) => {
    await consentService.revoke(purposeCode);
    mutate(); // Revalidate data
  };

  if (error) return <div>Failed to load</div>;
  if (!consents) return <div>Loading...</div>;

  return (
    <ul>
      {consents.map(consent => (
        <li key={consent.purpose_code}>
          {consent.purpose_description}
          {!consent.is_required && (
            <Button onClick={() => handleRevoke(consent.purpose_code)}>Revoke</Button>
          )}
        </li>
      ))}
    </ul>
  );
}
```

---

## Testing

### Unit Tests (Jest + React Testing Library)

```bash
# Run all tests
npm run test

# Watch mode
npm run test:watch

# Coverage report
npm run test:coverage
```

**Example Test**:

```tsx
// src/components/privacy/__tests__/ConsentCheckbox.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { ConsentCheckbox } from '../ConsentCheckbox';

describe('ConsentCheckbox', () => {
  it('renders consent label', () => {
    render(
      <ConsentCheckbox
        purposeCode="analytics"
        label="I consent to analytics"
        isRequired={false}
        checked={false}
        onChange={() => {}}
      />
    );
    expect(screen.getByText('I consent to analytics')).toBeInTheDocument();
  });

  it('calls onChange when clicked', () => {
    const handleChange = jest.fn();
    render(
      <ConsentCheckbox
        purposeCode="analytics"
        label="I consent"
        isRequired={false}
        checked={false}
        onChange={handleChange}
      />
    );
    fireEvent.click(screen.getByRole('checkbox'));
    expect(handleChange).toHaveBeenCalledWith(true);
  });

  it('disables checkbox when required', () => {
    render(
      <ConsentCheckbox
        purposeCode="operational"
        label="Required consent"
        isRequired={true}
        checked={true}
        onChange={() => {}}
      />
    );
    expect(screen.getByRole('checkbox')).toBeDisabled();
  });
});
```

### E2E Tests (Playwright)

```bash
# Install Playwright
npx playwright install

# Run E2E tests
npm run test:e2e

# Run with UI
npm run test:e2e:ui
```

**Example E2E Test**:

```ts
// tests/e2e/consent-flow.spec.ts
import { test, expect } from '@playwright/test';

test('guest can grant consent during checkout', async ({ page }) => {
  await page.goto('/checkout');

  // Fill order details
  await page.fill('input[name="customer_name"]', 'John Doe');
  await page.fill('input[name="customer_email"]', 'john@example.com');

  // Check required consents (should be pre-checked)
  await expect(page.locator('input[name="consent_operational"]')).toBeChecked();
  await expect(page.locator('input[name="consent_operational"]')).toBeDisabled();

  // Opt into analytics
  await page.check('input[name="consent_analytics"]');

  // Submit form
  await page.click('button[type="submit"]');

  // Verify order created with consents
  await expect(page).toHaveURL(/\/order\/ORD-\d+-\d+/);
});
```

---

## Deployment

### Production Build

```bash
# Build for production
npm run build

# Output directory
# .next/ - Contains optimized production build
```

### Environment Variables

**Production .env**:

```bash
NEXT_PUBLIC_API_URL=https://api.yourcompany.com/api/v1
NEXT_PUBLIC_APP_NAME="Your POS System"
NODE_ENV=production
```

### Deployment Platforms

**Vercel** (recommended):

```bash
# Install Vercel CLI
npm i -g vercel

# Deploy
vercel --prod
```

**Docker**:

```dockerfile
# Dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./
EXPOSE 3000
CMD ["npm", "start"]
```

**Nginx**:

```nginx
server {
  listen 80;
  server_name yourcompany.com;

  location / {
    proxy_pass http://localhost:3000;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection 'upgrade';
    proxy_set_header Host $host;
    proxy_cache_bypass $http_upgrade;
  }
}
```

---

## Troubleshooting

### Common Issues

#### 1. API Connection Failed

**Error**: `TypeError: Failed to fetch`

**Solution**:

```bash
# Check backend is running
curl http://localhost:8080/health

# Verify NEXT_PUBLIC_API_URL in .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1

# Restart dev server
npm run dev
```

#### 2. Translations Missing

**Error**: `Translation missing: dashboard.title`

**Solution**:

```bash
# Check translation files exist
ls messages/en.json messages/id.json

# Verify key exists in both files
cat messages/en.json | grep "dashboard"

# Restart dev server (hot reload may not catch changes)
npm run dev
```

#### 3. Build Errors

**Error**: `Module not found: Can't resolve '@/src/components'`

**Solution**:

```json
// Verify tsconfig.json paths
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["./*"]
    }
  }
}
```

#### 4. Tailwind Styles Not Applied

**Solution**:

```bash
# Verify tailwind.config.ts content paths
content: [
  './app/**/*.{js,ts,jsx,tsx,mdx}',
  './src/**/*.{js,ts,jsx,tsx,mdx}',
]

# Rebuild
npm run build
```

### Debug Mode

Enable verbose logging:

```bash
# Development
DEBUG=* npm run dev

# Production
NODE_OPTIONS='--inspect' npm run start
```

---

## Additional Resources

- **Frontend Conventions**: [/docs/FRONTEND_CONVENTIONS.md](../docs/FRONTEND_CONVENTIONS.md)
- **i18n Quick Reference**: [/docs/i18n-quick-reference.md](../docs/i18n-quick-reference.md)
- **API Documentation**: [/docs/API.md](../docs/API.md)
- **UU PDP Compliance Guide**: [/docs/UU_PDP_COMPLIANCE.md](../docs/UU_PDP_COMPLIANCE.md)

---

## Contributing

1. Follow frontend conventions ([FRONTEND_CONVENTIONS.md](../docs/FRONTEND_CONVENTIONS.md))
2. Write tests for new components
3. Ensure accessibility (ARIA labels, keyboard navigation)
4. Add translations for new strings (both English and Indonesian)
5. Test in multiple browsers (Chrome, Firefox, Safari)

---

**Questions?** Contact the development team or create an issue in the repository.
