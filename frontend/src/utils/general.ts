// Public pages that don't require authentication
export const PUBLIC_PAGES = {
  AUTH: [
    '/login',
    '/signup',
    '/forgot-password',
    '/reset-password',
    '/accept-invitation',
    '/verify-email',
  ],
  GUEST_ORDER_PATTERN: /^\/orders\/[A-Z0-9-]+$/, // Matches /orders/{orderReference}
  GUEST_PATHS: ['/menu/', '/checkout/', '/guest/', '/platform'], // Any path containing these segments
  OTHERS: [
    '/',
    '/pricing',
    '/about',
    '/contact',
    '/blog',
    '/terms',
    '/terms-of-service',
    '/privacy-policy',
    '/help',
    '/status',
    '/careers',
    '/features',
    '/integrations',
  ],
};
