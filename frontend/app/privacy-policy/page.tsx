import type { Metadata } from 'next';
import PrivacyPolicyClient from './PrivacyPolicyClient';

const description =
  'Kebijakan privasi Posku untuk pengelolaan data akun, tenant, transaksi, consent, dan layanan software POS.';

export const metadata: Metadata = {
  title: 'Kebijakan Privasi',
  description,
  alternates: {
    canonical: '/privacy-policy',
  },
  openGraph: {
    title: 'Kebijakan Privasi Posku',
    description,
    url: '/privacy-policy',
    type: 'website',
    locale: 'id_ID',
  },
  twitter: {
    card: 'summary',
    title: 'Kebijakan Privasi Posku',
    description,
  },
};

export default function PrivacyPolicyPage() {
  return <PrivacyPolicyClient />;
}
