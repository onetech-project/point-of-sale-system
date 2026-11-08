import {
  getTenantUnavailableMessage,
  isTenantUnavailableError,
  PAYMENT_NOT_CONFIGURED_MESSAGE,
  TENANT_UNAVAILABLE_MESSAGE,
} from './tenantAvailability';

const forbidden = (data: Record<string, unknown>) => ({ response: { status: 403, data } });

it.each(['midtrans_not_configured', 'manjo_not_configured'])(
  'treats %s as a payment gateway configuration error',
  reason => {
    const error = forbidden({ reason });
    expect(isTenantUnavailableError(error)).toBe(true);
    expect(getTenantUnavailableMessage(error, 'fallback')).toBe(PAYMENT_NOT_CONFIGURED_MESSAGE);
  }
);

it('keeps the generic unavailable message for suspended tenants', () => {
  const error = forbidden({ error: 'Tenant currently unavailable' });
  expect(getTenantUnavailableMessage(error, 'fallback')).toBe(TENANT_UNAVAILABLE_MESSAGE);
});

it('falls back to the API message for other errors', () => {
  expect(
    getTenantUnavailableMessage({ response: { status: 500, data: { error: 'boom' } } }, 'x')
  ).toBe('boom');
});
