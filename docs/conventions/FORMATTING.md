# Formatting Conventions

This document outlines the formatting conventions used across the POS system frontend.

## Number Formatting

### Price and Currency Display

**Convention:** Always use thousand separators for numeric values. Do NOT display currency symbols unless explicitly required.

**Utility Functions:**

Located in `frontend/src/utils/format.ts`:

```typescript
// Format a number with thousand separators
formatNumber(value: number, decimals: number = 2): string

// Parse a formatted number string to a number
parseFormattedNumber(value: string): number
```

### Usage Examples

#### Display Values (Read-only)

```typescript
import { formatNumber } from '@/utils/format';

// Price display
<span>{formatNumber(product.selling_price)}</span>
// Output: "1,234.56" (not "$1,234.56")

// Value with custom decimals
<span>{formatNumber(product.tax_rate, 1)}</span>
// Output: "10.5"
```

#### Input Fields (Editable)

For editable price fields, use the two-state pattern:

```typescript
const [formData, setFormData] = useState({
  selling_price: '', // Store raw numeric string
  cost_price: '',
});

const [displayPrices, setDisplayPrices] = useState({
  selling_price: '', // Store formatted display string
  cost_price: '',
});

// On change - allow user input
const handlePriceChange = (field: 'selling_price', displayValue: string) => {
  const cleanValue = displayValue.replace(/,/g, '');
  setFormData(prev => ({ ...prev, [field]: cleanValue }));
  setDisplayPrices(prev => ({ ...prev, [field]: displayValue }));
};

// On blur - format the display
const handlePriceBlur = (field: 'selling_price') => {
  const value = parseFloat(formData[field]);
  if (!isNaN(value)) {
    const formatted = value.toLocaleString('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    });
    setDisplayPrices(prev => ({ ...prev, [field]: formatted }));
  }
};

// In JSX
<input
  type="text"
  value={displayPrices.selling_price}
  onChange={(e) => handlePriceChange('selling_price', e.target.value)}
  onBlur={() => handlePriceBlur('selling_price')}
  placeholder="0.00"
/>
```

### Rules

1. **Always format numbers for display** - Use `formatNumber()` for all numeric displays
2. **No currency symbols** - Do not prefix with $ or other currency symbols unless specifically required
3. **Consistent decimal places** - Use 2 decimals for prices, appropriate decimals for other values
4. **Input vs Display** - Use text inputs with formatting for better UX, store clean numeric values
5. **Locale consistency** - Use 'en-US' locale for consistent comma/decimal separator

### Areas Requiring Formatting

- ✅ Product prices (selling_price, cost_price)
- ✅ Inventory values (total_value)
- ✅ Transaction amounts
- ✅ Financial summaries
- ✅ Any monetary or large numeric values

### Migration Checklist

When adding new numeric fields:

- [ ] Import `formatNumber` from `@/utils/format`
- [ ] Use `formatNumber()` for display
- [ ] Use text input with formatting pattern for editable fields
- [ ] Remove any hardcoded currency symbols ($)
- [ ] Test with large values (e.g., 1,000,000.00)

## Date and Time Formatting

**Convention:** Use locale-aware formatting for dates

```typescript
// Date only
new Date(value).toLocaleDateString()
// Output: "12/1/2025"

// Date and time
new Date(value).toLocaleString()
// Output: "12/1/2025, 5:30:00 PM"
```

## Best Practices

1. **Consistency:** All numeric values should follow the same formatting pattern
2. **User Experience:** Format on display, accept flexible input
3. **Validation:** Always validate after parsing formatted numbers
4. **Accessibility:** Ensure formatted numbers are screen-reader friendly
5. **Testing:** Test with edge cases (0, negative numbers, very large numbers)

---

**Last Updated:** December 2025
