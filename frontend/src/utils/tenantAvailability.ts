export const TENANT_UNAVAILABLE_MESSAGE = 'This tenant is currently not available at this moment.';
export const MIDTRANS_NOT_CONFIGURED_MESSAGE =
  'Guest ordering is unavailable until Midtrans is configured.';

export function isTenantUnavailableError(error: any): boolean {
  const data = error?.response?.data;
  return (
    error?.response?.status === 403 &&
    (data?.error === 'Tenant currently unavailable' ||
      data?.message === TENANT_UNAVAILABLE_MESSAGE ||
      data?.reason === 'midtrans_not_configured')
  );
}

export function getTenantUnavailableMessage(error: any, fallback: string): string {
  if (error?.response?.data?.reason === 'midtrans_not_configured') {
    return MIDTRANS_NOT_CONFIGURED_MESSAGE;
  }
  if (isTenantUnavailableError(error)) return TENANT_UNAVAILABLE_MESSAGE;
  return (
    error?.response?.data?.message || error?.response?.data?.error || error?.message || fallback
  );
}
