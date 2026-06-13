import type { InventoryNumber, StockMovementType, UOM } from '@/types/inventory';

export const UOM_CATEGORIES = ['weight', 'volume', 'count', 'packaging'] as const;

export const toNumber = (value: InventoryNumber | null | undefined): number => {
  if (value === null || value === undefined || value === '') {
    return 0;
  }

  const parsed = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
};

export const formatQuantity = (value: InventoryNumber | null | undefined, decimals = 2): string => {
  const parsed = toNumber(value);
  const maximumFractionDigits = Number.isInteger(parsed) ? 0 : decimals;

  return parsed.toLocaleString('en-US', {
    minimumFractionDigits: 0,
    maximumFractionDigits,
  });
};

export const formatDateTime = (value: string): string =>
  new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));

export const getUOMLabel = (uoms: UOM[], id?: string | null): string => {
  const uom = uoms.find(item => item.id === id);
  if (!uom) {
    return '-';
  }
  return `${uom.name} (${uom.code})`;
};

export const getUOMCode = (uoms: UOM[], id?: string | null): string => {
  const uom = uoms.find(item => item.id === id);
  return uom?.code || '-';
};

export const getStockStatus = (
  current: InventoryNumber,
  minimum: InventoryNumber,
  isActive: boolean
): { label: string; className: string } => {
  if (!isActive) {
    return { label: 'Inactive', className: 'bg-gray-100 text-gray-700' };
  }

  const currentValue = toNumber(current);
  const minimumValue = toNumber(minimum);

  if (currentValue <= 0) {
    return { label: 'Out', className: 'bg-red-100 text-red-700' };
  }

  if (minimumValue > 0 && currentValue <= minimumValue) {
    return { label: 'Low', className: 'bg-yellow-100 text-yellow-800' };
  }

  return { label: 'OK', className: 'bg-green-100 text-green-700' };
};

export const movementTypeLabel = (movementType: StockMovementType): string => {
  switch (movementType) {
    case 'INITIAL_STOCK':
      return 'Initial stock';
    case 'PURCHASE_IN':
      return 'Purchase';
    case 'MANUAL_ADJUSTMENT':
      return 'Adjustment';
    case 'WASTE':
      return 'Waste';
    default:
      return movementType
        .toLowerCase()
        .split('_')
        .map(part => part.charAt(0).toUpperCase() + part.slice(1))
        .join(' ');
  }
};

export const movementBadgeClass = (movementType: StockMovementType): string => {
  switch (movementType) {
    case 'INITIAL_STOCK':
      return 'bg-blue-100 text-blue-700';
    case 'PURCHASE_IN':
      return 'bg-green-100 text-green-700';
    case 'MANUAL_ADJUSTMENT':
      return 'bg-purple-100 text-purple-700';
    case 'WASTE':
      return 'bg-red-100 text-red-700';
    default:
      return 'bg-gray-100 text-gray-700';
  }
};

export const getApiErrorMessage = (error: unknown, fallback: string): string => {
  const response = (error as { response?: { data?: Record<string, unknown> } })?.response;
  const data = response?.data;
  const message = typeof data?.message === 'string' ? data.message : undefined;
  const details = typeof data?.details === 'string' ? data.details : undefined;
  const apiError = typeof data?.error === 'string' ? data.error : undefined;

  if (message && details && message !== details) {
    return `${message}: ${details}`;
  }

  return details || message || apiError || fallback;
};

export const decimalInputValue = (value: InventoryNumber | null | undefined): string => {
  const parsed = toNumber(value);
  return parsed === 0 ? '0' : String(parsed);
};
