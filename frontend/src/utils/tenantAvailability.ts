export const TENANT_UNAVAILABLE_MESSAGE = 'This tenant is currently not available at this moment.';
export const PAYMENT_NOT_CONFIGURED_MESSAGE =
  'Guest ordering is unavailable until a payment gateway is configured.';
/** @deprecated kept for existing imports; guest ordering is blocked for any unconfigured gateway */
export const MIDTRANS_NOT_CONFIGURED_MESSAGE = PAYMENT_NOT_CONFIGURED_MESSAGE;

const PAYMENT_NOT_CONFIGURED_REASONS = ['midtrans_not_configured', 'manjo_not_configured'];

export function isPaymentNotConfiguredError(error: any): boolean {
  return PAYMENT_NOT_CONFIGURED_REASONS.includes(error?.response?.data?.reason);
}

export function isTenantUnavailableError(error: any): boolean {
  const data = error?.response?.data;
  return (
    error?.response?.status === 403 &&
    (data?.error === 'Tenant currently unavailable' ||
      data?.message === TENANT_UNAVAILABLE_MESSAGE ||
      isPaymentNotConfiguredError(error))
  );
}

export function getTenantUnavailableMessage(error: any, fallback: string): string {
  if (isPaymentNotConfiguredError(error)) {
    return PAYMENT_NOT_CONFIGURED_MESSAGE;
  }
  if (isTenantUnavailableError(error)) return TENANT_UNAVAILABLE_MESSAGE;
  return (
    error?.response?.data?.message || error?.response?.data?.error || error?.message || fallback
  );
}
