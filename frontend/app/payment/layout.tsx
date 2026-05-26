import type { Metadata } from 'next';
import type { ReactNode } from 'react';
import { noIndexMetadata } from '@/app/seo';

export const metadata: Metadata = noIndexMetadata;

export default function PaymentLayout({ children }: { children: ReactNode }) {
  return children;
}
