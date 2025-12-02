# i18n Quick Reference Card

## The Golden Rule
**When using multiple namespaces, ONLY the FIRST namespace works without `ns` option.**

## Single Namespace
```typescript
const { t } = useTranslation('common');
t('submit')   // ✅ Works
t('cancel')   // ✅ Works
```

## Multiple Namespaces
```typescript
const { t } = useTranslation(['products', 'common']);

// First namespace (products) - NO ns option
t('title')           // ✅ Works - from products
t('list.inStock')    // ✅ Works - from products

// Other namespaces - REQUIRES ns option
t('submit', { ns: 'common' })   // ✅ Works - from common
t('loading', { ns: 'common' })  // ✅ Works - from common

// ❌ COMMON MISTAKES
t('submit')                     // ❌ Won't find in products
t('common.submit')              // ❌ Don't use prefix
t('submit', { ns: 'products' }) // ❌ Unnecessary (it's first)
```

## Pattern Template
```typescript
function MyComponent() {
  // List feature namespace FIRST, common SECOND
  const { t } = useTranslation(['myFeature', 'common']);
  
  return (
    <div>
      {/* Feature translations - NO ns */}
      <h1>{t('title')}</h1>
      <p>{t('description')}</p>
      
      {/* Common translations - WITH ns */}
      <button>{t('submit', { ns: 'common' })}</button>
      <button>{t('cancel', { ns: 'common' })}</button>
    </div>
  );
}
```

## Checklist
- [ ] First namespace = most-used namespace
- [ ] Other namespaces use `{ ns: 'namespace' }`
- [ ] No namespace prefix in keys
- [ ] JSON has namespace wrapper: `{ "common": { ... } }`
- [ ] Namespace added to config.ts resources

---
**Remember:** i18next treats the first namespace in the array as the default namespace!
