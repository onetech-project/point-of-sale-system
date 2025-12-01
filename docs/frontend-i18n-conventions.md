# Frontend i18n Conventions

## Overview
This project uses `react-i18next` for internationalization with support for English (en) and Indonesian (id) languages.

## File Structure

```
frontend/src/i18n/
├── config.ts              # i18next configuration
├── provider.tsx           # React provider wrapper
└── locales/
    ├── en/
    │   ├── common.json    # Common UI strings
    │   ├── auth.json      # Authentication strings
    │   └── products.json  # Product feature strings
    └── id/
        ├── common.json
        ├── auth.json
        └── products.json
```

## JSON File Structure

**IMPORTANT:** Each JSON file must wrap translations in a namespace object:

```json
{
  "namespace_name": {
    "key1": "Value 1",
    "key2": "Value 2",
    "nested": {
      "key3": "Value 3"
    }
  }
}
```

**Example:** `locales/en/common.json`
```json
{
  "common": {
    "submit": "Submit",
    "cancel": "Cancel",
    "loading": "Loading...",
    "search": "Search"
  }
}
```

## Usage in Components

### Single Namespace
```typescript
import { useTranslation } from '@/i18n/provider';

function MyComponent() {
  const { t } = useTranslation('common');
  
  return (
    <button>{t('submit')}</button>  // "Submit"
  );
}
```

### Multiple Namespaces

**⚠️ CRITICAL RULE:** When using multiple namespaces, **the FIRST namespace is the default**, all other namespaces **MUST use the `ns` option**.

```typescript
import { useTranslation } from '@/i18n/provider';

function ProductList() {
  const { t } = useTranslation(['products', 'common']);
  
  return (
    <div>
      {/* First namespace (products) - NO ns option needed */}
      <h1>{t('title')}</h1>
      <p>{t('list.inStock')}</p>
      
      {/* Other namespaces - MUST specify ns option */}
      <button>{t('search', { ns: 'common' })}</button>
      <span>{t('loading', { ns: 'common' })}</span>
    </div>
  );
}
```

**Usage Rules:**

```typescript
const { t } = useTranslation(['products', 'common']);

// ✅ CORRECT - First namespace (products) without ns
t('title')                           // From products namespace
t('list.inStock')                    // From products namespace
t('form.name')                       // From products namespace

// ✅ CORRECT - Other namespaces with ns option
t('search', { ns: 'common' })        // From common namespace
t('loading', { ns: 'common' })       // From common namespace
t('submit', { ns: 'common' })        // From common namespace

// ❌ WRONG - Other namespaces without ns option
t('search')                          // Won't find it in products
t('loading')                         // Won't work
t('common.search')                   // Wrong - don't use prefix

// ❌ WRONG - First namespace with ns option (unnecessary)
t('title', { ns: 'products' })       // Works but redundant
```

## Naming Conventions

### Namespace Organization
- **common**: UI elements used across the app (buttons, labels, pagination)
- **auth**: Authentication/authorization related strings
- **[feature]**: Feature-specific strings (products, sales, customers, etc.)

### Key Naming
Use dot notation for nested structures:

```json
{
  "products": {
    "title": "Products",
    "list": {
      "search": "Search products...",
      "inStock": "In Stock",
      "lowStock": "Low Stock"
    },
    "form": {
      "name": "Product Name",
      "price": "Price"
    },
    "messages": {
      "createSuccess": "Product created successfully",
      "loadError": "Failed to load products"
    }
  }
}
```

Access as: `t('products.list.search')`

### Key Categories
Organize keys by category:
- **`title`**: Page/section titles
- **`list.*`**: List view strings
- **`form.*`**: Form field labels and placeholders
- **`messages.*`**: Success/error messages
- **`actions.*`**: Action button labels
- **`validation.*`**: Validation error messages

## Adding New Translations

### 1. Add to English File
```json
// locales/en/feature.json
{
  "feature": {
    "newKey": "New English Text"
  }
}
```

### 2. Add to Indonesian File
```json
// locales/id/feature.json
{
  "feature": {
    "newKey": "Teks Bahasa Indonesia Baru"
  }
}
```

### 3. Register in Config (if new namespace)
```typescript
// src/i18n/config.ts
import featureEn from './locales/en/feature.json';
import featureId from './locales/id/feature.json';

const resources = {
  en: {
    common: commonEn,
    auth: authEn,
    products: productsEn,
    feature: featureEn,  // Add new namespace
  },
  id: {
    common: commonId,
    auth: authId,
    products: productsId,
    feature: featureId,
  },
};

// No need to modify i18n.init() - namespaces are auto-detected from resources
```

## Common Patterns

### Dynamic Values
```typescript
// JSON: "greeting": "Hello, {{name}}!"
t('greeting', { name: 'John' })  // "Hello, John!"

// With namespace option
t('greeting', { name: 'John', ns: 'common' })  // When not first namespace
```

### Pluralization
```typescript
// JSON: "items": "{{count}} item",
//       "items_plural": "{{count}} items"
t('items', { count: 1 })  // "1 item"
t('items', { count: 5 })  // "5 items"

// With namespace option
t('items', { count: 5, ns: 'common' })
```

### Date/Number Formatting
```typescript
// JSON: "price": "Price: {{value, number}}"
t('price', { value: 1234.56 })  // "Price: 1,234.56"
```

### Conditional Text
```typescript
const statusKey = isActive ? 'status.active' : 'status.inactive';
t(statusKey)

// With namespace option
t(statusKey, { ns: 'common' })
```

## Multiple Namespace Pattern Examples

### Example 1: Product List Component
```typescript
function ProductList() {
  // First namespace: products, Others: common
  const { t } = useTranslation(['products', 'common']);
  
  return (
    <div>
      {/* Products namespace (first) - NO ns option */}
      <h1>{t('title')}</h1>
      <p>{t('list.search')}</p>
      <span>{t('list.inStock')}</span>
      
      {/* Common namespace (not first) - REQUIRES ns option */}
      <button>{t('submit', { ns: 'common' })}</button>
      <button>{t('cancel', { ns: 'common' })}</button>
      <span>{t('loading', { ns: 'common' })}</span>
    </div>
  );
}
```

### Example 2: Form with Mixed Namespaces
```typescript
function ProductForm() {
  const { t } = useTranslation(['products', 'common']);
  
  return (
    <form>
      {/* Products namespace */}
      <label>{t('form.name')}</label>
      <label>{t('form.price')}</label>
      
      {/* Common namespace - use ns option */}
      <button type="submit">{t('submit', { ns: 'common' })}</button>
      <button type="button">{t('cancel', { ns: 'common' })}</button>
    </form>
  );
}
```

### Example 3: Dynamic Translation with Namespace
```typescript
function InventoryStatus({ status }: { status: string }) {
  const { t } = useTranslation(['products', 'common']);
  
  // Dynamic key from products (first namespace)
  const statusText = t(`list.status.${status}`);
  
  // Loading state from common (needs ns option)
  if (isLoading) {
    return <span>{t('loading', { ns: 'common' })}</span>;
  }
  
  return <span>{statusText}</span>;
}
```

## Best Practices

### ✅ DO
- Always use `useTranslation` hook for translations
- Put the most-used namespace first in the array
- Use `ns` option for all namespaces except the first one
- Keep translation keys descriptive and hierarchical
- Group related translations together
- Maintain same structure in all language files
- Test translations in both languages

### ❌ DON'T
- Hardcode user-facing text in components
- Forget `ns` option for non-first namespaces
- Use namespace prefix in keys (e.g., `t('common.search')`)
- Use inconsistent key naming
- Leave untranslated strings
- Create deeply nested structures (max 3-4 levels)
- Put feature-specific strings in common namespace

## Language Switching

```typescript
import { useTranslation } from '@/i18n/provider';

function LanguageSwitcher() {
  const { i18n } = useTranslation();
  
  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
    localStorage.setItem('locale', lng);
  };
  
  return (
    <select onChange={(e) => changeLanguage(e.target.value)}>
      <option value="en">English</option>
      <option value="id">Indonesia</option>
    </select>
  );
}
```

## Debugging

### Check Current Language
```typescript
const { i18n } = useTranslation();
console.log('Current language:', i18n.language);
```

### Check If Translation Exists
```typescript
const { t, i18n } = useTranslation('common');
console.log('Translation exists:', i18n.exists('common.submit'));
```

### Fallback Behavior
If a translation key is not found:
1. Falls back to `fallbackLng` (English)
2. If still not found, returns the key itself

## Testing Translations

```typescript
// Test component with specific language
import { I18nextProvider } from 'react-i18next';
import i18n from '@/i18n/config';

beforeEach(() => {
  i18n.changeLanguage('en');
});

test('displays translated text', () => {
  render(
    <I18nextProvider i18n={i18n}>
      <MyComponent />
    </I18nextProvider>
  );
  
  expect(screen.getByText('Submit')).toBeInTheDocument();
});
```

## Common Issues & Solutions

### Issue: Translation not found
**Symptoms:** Key appears instead of translated text
**Solutions:**
1. Check key spelling and case
2. Verify namespace is loaded in `useTranslation(['namespace'])`
3. Check JSON structure (namespace wrapper present?)
4. Ensure you're using correct key path in JSON

### Issue: Non-first namespace not working
**Symptoms:** Second/third namespace keys not translated
**Solutions:**
1. **Add `ns` option**: `t('search', { ns: 'common' })`
2. Check namespace is listed in `useTranslation(['first', 'second'])`
3. Verify JSON files exist for that namespace

**Example Fix:**
```typescript
// ❌ WRONG
const { t } = useTranslation(['products', 'common']);
<button>{t('search')}</button>  // Won't find 'search' in products

// ✅ CORRECT
const { t } = useTranslation(['products', 'common']);
<button>{t('search', { ns: 'common' })}</button>  // Found in common
```

### Issue: Accidentally using namespace prefix
**Symptoms:** Keys like "common.search" not found
**Solutions:**
```typescript
// ❌ WRONG - Don't use prefix with ns option
t('common.search', { ns: 'common' })  // Won't work

// ✅ CORRECT - Use ns option without prefix
t('search', { ns: 'common' })  // Works!
```

### Issue: Language doesn't switch
**Symptoms:** UI stays in same language
**Solutions:**
1. Call `i18n.changeLanguage(lng)`
2. Save to localStorage for persistence
3. Check if translation files exist for target language

## Future Enhancements

Consider adding:
- Context-based translations (formal/informal)
- Right-to-left (RTL) language support
- Translation management UI
- Automated translation coverage reports
- Lazy-loading of namespace files
