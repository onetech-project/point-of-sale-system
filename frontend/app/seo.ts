import type { Metadata } from 'next';

export const SITE_NAME = 'Posku';
export const DEFAULT_SITE_URL = 'https://posku.web.id';
export const DEFAULT_TITLE = 'Posku | Software POS Indonesia untuk Cafe, Restoran, dan Retail';
export const DEFAULT_DESCRIPTION =
  'Posku adalah software POS online untuk bisnis Indonesia, dengan online ordering, pembayaran QRIS, inventory, laporan penjualan, dan manajemen tim.';

export function getSiteUrl() {
  const rawUrl = process.env.NEXT_PUBLIC_SITE_URL || DEFAULT_SITE_URL;

  try {
    return new URL(rawUrl).origin;
  } catch {
    return DEFAULT_SITE_URL;
  }
}

export function absoluteUrl(path = '/') {
  return new URL(path, `${getSiteUrl()}/`).toString();
}

export const defaultRobots: NonNullable<Metadata['robots']> = {
  index: true,
  follow: true,
  googleBot: {
    index: true,
    follow: true,
    'max-video-preview': -1,
    'max-image-preview': 'large',
    'max-snippet': -1,
  },
};

export const noIndexMetadata: Metadata = {
  robots: {
    index: false,
    follow: false,
    googleBot: {
      index: false,
      follow: false,
      noimageindex: true,
    },
  },
};
