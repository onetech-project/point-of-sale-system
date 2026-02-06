/**
 * Format a number with thousand separators
 * @param value - The number to format
 * @param decimals - Number of decimal places (default: 2)
 * @returns Formatted string with thousand separators
 */
export const formatNumber = (value: number, decimals: number = 0): string => {
  if (isNaN(value)) return '0';
  return value.toLocaleString('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  });
};

/**
 * Format large numbers in compact form (K, M, B)
 * @param value - The number to format
 * @param decimals - Number of decimal places (default: 1)
 * @returns Formatted string (e.g., "1.2M", "3.5B")
 */
export const formatCompactNumber = (value: number, decimals: number = 2): string => {
  if (value >= 1_000_000_000) {
    return (value / 1_000_000_000).toFixed(decimals) + 'B';
  }
  if (value >= 1_000_000) {
    return (value / 1_000_000).toFixed(decimals) + 'M';
  }
  if (value >= 1_000) {
    return (value / 1_000).toFixed(decimals) + 'K';
  }
  return value.toFixed(decimals);
};

/**
 * Format number with responsive display based on screen size
 * Returns full format for desktop, compact for mobile
 * @param value - The number to format
 * @param isMobile - Whether to use compact format
 * @param decimals - Number of decimal places
 * @returns Formatted string
 */
export const formatResponsiveNumber = (
  value: number,
  isMobile: boolean = false,
  decimals: number = 2
): string => {
  if (isMobile && value >= 1_000_000) {
    return formatCompactNumber(value, 1);
  }
  return formatNumber(value, decimals);
};

/**
 * Parse a formatted number string to a number
 * @param value - The formatted string
 * @returns Parsed number
 */
export const parseFormattedNumber = (value: string): number => {
  const cleaned = value.replace(/,/g, '');
  return parseFloat(cleaned) || 0;
};

/**
 * Format a price in Indonesian Rupiah (IDR)
 * @param price - The price to format
 * @returns Formatted currency string
 */
export const formatPrice = (price: number, compact: boolean = false, decimals: number = 2): string => {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
    notation: compact ? 'compact' : 'standard',
    compactDisplay: 'short'
  }).format(price);
};

/**
 * Format currency (alias for formatPrice)
 * @param amount - The amount to format
 * @returns Formatted currency string
 */
export const formatCurrency = (amount: number, compact: boolean = false, decimals: number = 2): string => {
  if (isNaN(amount)) return formatPrice(0);
  return formatPrice(amount, compact, decimals);
};

/**
 * Format file size for display
 * @param bytes - The file size in bytes
 * @returns Formatted file size string (e.g., "1.5 MB")
 */
export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
};

export const formatDate = (dateString: string) => {
  const locale = localStorage.getItem('locale') === 'id' ? 'id-ID' : 'en-US';
  return new Date(dateString).toLocaleDateString(locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
};