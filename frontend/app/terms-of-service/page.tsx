import type { Metadata } from 'next';
import Link from 'next/link';
import PublicLayout from '@/components/layout/PublicLayout';

const TERMS_VERSION = '1.0.0';
const description =
  'Syarat layanan Posku untuk trial, subscription, grace period, retensi data, dan penggunaan software POS.';

export const metadata: Metadata = {
  title: 'Syarat Layanan',
  description,
  alternates: {
    canonical: '/terms-of-service',
  },
  openGraph: {
    title: 'Syarat Layanan Posku',
    description,
    url: '/terms-of-service',
    type: 'website',
    locale: 'id_ID',
  },
  twitter: {
    card: 'summary',
    title: 'Syarat Layanan Posku',
    description,
  },
};

export default function TermsOfServicePage() {
  return (
    <PublicLayout>
      <main className="bg-white">
        <section className="mx-auto max-w-3xl px-4 py-12 sm:px-6 lg:px-8">
          <div className="mb-8">
            <p className="text-sm font-medium text-primary-600">Version {TERMS_VERSION}</p>
            <h1 className="mt-2 text-3xl font-bold text-gray-900">Terms of Service</h1>
            <p className="mt-3 text-sm text-gray-600">
              These terms explain the subscription, data retention, and compliance rules for using
              this point of sale platform.
            </p>
          </div>

          <div className="space-y-8 text-sm leading-6 text-gray-700">
            <section>
              <h2 className="text-lg font-semibold text-gray-900">Trial and Subscription</h2>
              <p className="mt-2">
                New tenants receive a 7-day free trial. After the trial or paid subscription
                expires, tenants receive a 7-day access grace period to complete subscription
                payment and avoid account suspension.
              </p>
            </section>

            <section>
              <h2 className="text-lg font-semibold text-gray-900">Grace Period and Access</h2>
              <p className="mt-2">
                Tenants in grace period may continue using the application with warning banners.
                Once the grace period ends, access is limited to subscription recovery, invoices,
                payment, and logout until payment is completed.
              </p>
            </section>

            <section>
              <h2 className="text-lg font-semibold text-gray-900">Operational Data Retention</h2>
              <p className="mt-2">
                The 30-day operational-data retention window starts when the tenant enters grace
                period. If payment is not completed before this window ends, operational workspace
                data such as products, categories, inventory state, tenant settings, team invites,
                notification settings, carts, and product photo metadata may be deleted or
                anonymized.
              </p>
              <p className="mt-2">
                If a tenant pays after retention cleanup has already run, the account can be
                reactivated, but deleted or anonymized operational data will not be restored.
              </p>
            </section>

            <section>
              <h2 className="text-lg font-semibold text-gray-900">Historical Records</h2>
              <p className="mt-2">
                Billing invoices, payment attempts, Terms acceptance records, consent records, audit
                events, and financial order records required for legal or tax retention are
                preserved as historical and compliance records. Customer personally identifiable
                information in preserved order records may be anonymized.
              </p>
            </section>

            <section>
              <h2 className="text-lg font-semibold text-gray-900">Acceptance</h2>
              <p className="mt-2">
                By creating an account, the tenant owner accepts this Terms version on behalf of the
                business. The platform records the accepted version, timestamp, IP address, and user
                agent for audit purposes.
              </p>
            </section>
          </div>

          <div className="mt-10 border-t border-gray-200 pt-6">
            <Link
              href="/signup"
              className="text-sm font-medium text-primary-600 hover:text-primary-500"
            >
              Back to signup
            </Link>
          </div>
        </section>
      </main>
    </PublicLayout>
  );
}
