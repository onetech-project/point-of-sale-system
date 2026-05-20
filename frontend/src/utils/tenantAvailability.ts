export const TENANT_UNAVAILABLE_MESSAGE = 'This tenant is currently not available at this moment.';

export function isTenantUnavailableError(error: any): boolean {
  const data = error?.response?.data;
  return error?.response?.status === 403 && (
    data?.error === 'Tenant currently unavailable' ||
    data?.message === TENANT_UNAVAILABLE_MESSAGE
  );
}

export function getTenantUnavailableMessage(error: any, fallback: string): string {
  if (isTenantUnavailableError(error)) return TENANT_UNAVAILABLE_MESSAGE;
  return error?.response?.data?.message || error?.response?.data?.error || error?.message || fallback;
}
