'use client';

import { useState, useEffect } from 'react';
import { useTranslation } from '@/i18n/provider';
import PublicLayout from '@/components/layout/PublicLayout';
import consentService from '@/services/consent';
import Head from 'next/head';

interface PrivacyPolicy {
  version: string;
  policy_text_id: string;
  effective_date: string;
  is_current: boolean;
}

export default function PrivacyPolicyPage() {
  const { t, i18n } = useTranslation(['privacy', 'common']);
  const [policy, setPolicy] = useState<PrivacyPolicy | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchPrivacyPolicy();
  }, []);

  const fetchPrivacyPolicy = async () => {
    try {
      setLoading(true);
      const policyData = await consentService.getPrivacyPolicy();
      setPolicy({ ...policyData, is_current: true });
    } catch (err) {
      console.error('Failed to fetch privacy policy:', err);
      setError(t('privacy.loadError', 'Failed to load privacy policy'));
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <PublicLayout>
        <div className="min-h-screen bg-gray-50 flex items-center justify-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      </PublicLayout>
    );
  }

  if (error) {
    return (
      <PublicLayout>
        <div className="min-h-screen bg-gray-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-lg shadow-md p-8 max-w-md w-full text-center">
            <div className="text-red-500 text-5xl mb-4">⚠️</div>
            <h1 className="text-2xl font-bold text-gray-900 mb-2">{t('common.error', { ns: 'common' })}</h1>
            <p className="text-gray-600 mb-6">{error}</p>
            <button
              onClick={fetchPrivacyPolicy}
              className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              {t('common.tryAgain', { ns: 'common' })}
            </button>
          </div>
        </div>
      </PublicLayout>
    );
  }

  return (
    <PublicLayout>
      <Head>
        <title>{t('privacy.title')} | {t('common.appName', { ns: 'common' })}</title>
        <meta
          name="description"
          content={t('privacy.introduction.text')}
        />
        <meta property="og:title" content={`${t('privacy.title')} | ${t('common.appName', { ns: 'common' })}`} />
        <meta property="og:description" content={t('privacy.introduction.text')} />
        <meta property="og:type" content="website" />
        <link rel="canonical" href={typeof window !== 'undefined' ? window.location.href : ''} />
      </Head>
      <div className="min-h-screen bg-gray-50">
        {/* Header */}
        <div className="bg-white border-b border-gray-200">
          <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <h1 className="text-4xl font-bold text-gray-900 mb-2">
              {t('privacy.title', 'Privacy Policy')}
            </h1>
            {policy && (
              <div className="flex items-center gap-4 text-sm text-gray-600">
                <span>
                  {t('privacy.version', 'Version')}: {policy.version}
                </span>
                <span>•</span>
                <span>
                  {t('privacy.effectiveDate', 'Effective Date')}:{' '}
                  {new Date(policy.effective_date).toLocaleDateString('id-ID', {
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric',
                  })}
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Content */}
        <main className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
          <div className="bg-white rounded-lg shadow-md p-8 space-y-8">
            {/* Introduction */}
            <PolicySection title={t('privacy.introduction.title', 'Introduction')}>
              <p className="text-gray-700 leading-relaxed">
                {t('privacy.introduction.text', 'Welcome to our Point of Sale (POS) System. We are committed to protecting your personal data in accordance with Indonesian Law No. 27 of 2022 on Personal Data Protection (UU PDP). This privacy policy explains how we collect, use, store, and protect your personal information.')}
              </p>
            </PolicySection>

            {/* Data We Collect */}
            <PolicySection title={t('privacy.dataCollected.title', 'Data We Collect')}>
              <div className="space-y-4">
                <DataCategory
                  title={t('privacy.dataCollected.accountData', 'Account Data')}
                  items={[
                    t('privacy.dataCollected.email', 'Email address'),
                    t('privacy.dataCollected.name', 'Full name (first and last name)'),
                    t('privacy.dataCollected.businessName', 'Business name'),
                    t('privacy.dataCollected.phone', 'Phone number'),
                  ]}
                />
                <DataCategory
                  title={t('privacy.dataCollected.orderData', 'Order Information')}
                  items={[
                    t('privacy.dataCollected.customerName', 'Customer name'),
                    t('privacy.dataCollected.customerPhone', 'Customer phone number'),
                    t('privacy.dataCollected.customerEmail', 'Customer email (optional)'),
                    t('privacy.dataCollected.deliveryAddress', 'Delivery address'),
                    t('privacy.dataCollected.ipAddress', 'IP address'),
                  ]}
                />
                <DataCategory
                  title={t('privacy.dataCollected.technicalData', 'Technical Data')}
                  items={[
                    t('privacy.dataCollected.cookies', 'Session cookies'),
                    t('privacy.dataCollected.browserInfo', 'Browser and device information'),
                    t('privacy.dataCollected.usageData', 'Usage patterns and analytics'),
                  ]}
                />
              </div>
            </PolicySection>

            {/* How We Use Your Data */}
            <PolicySection title={t('privacy.dataUse.title', 'How We Use Your Data')}>
              <ul className="list-disc list-inside space-y-2 text-gray-700">
                <li>{t('privacy.dataUse.serviceOperation', 'To provide and maintain our POS services')}</li>
                <li>{t('privacy.dataUse.orderProcessing', 'To process and fulfill orders')}</li>
                <li>{t('privacy.dataUse.communication', 'To communicate with you about your account and orders')}</li>
                <li>{t('privacy.dataUse.analytics', 'To analyze and improve our services')}</li>
                <li>{t('privacy.dataUse.payment', 'To facilitate secure payment processing')}</li>
                <li>{t('privacy.dataUse.compliance', 'To comply with legal obligations')}</li>
              </ul>
            </PolicySection>

            {/* Legal Basis */}
            <PolicySection title={t('privacy.legalBasis.title', 'Legal Basis for Processing')}>
              <p className="text-gray-700 leading-relaxed mb-4">
                {t('privacy.legalBasis.text', 'We process your personal data based on:')}
              </p>
              <ul className="list-disc list-inside space-y-2 text-gray-700">
                <li>
                  {t('privacy.legalBasis.consent', 'Your explicit consent (Article 20, UU PDP)')}
                </li>
                <li>
                  {t('privacy.legalBasis.contract', 'Performance of a contract with you')}
                </li>
                <li>
                  {t('privacy.legalBasis.legal', 'Compliance with legal obligations')}
                </li>
                <li>
                  {t('privacy.legalBasis.legitimate', 'Our legitimate business interests')}
                </li>
              </ul>
            </PolicySection>

            {/* Data Retention */}
            <PolicySection title={t('privacy.retention.title', 'Data Retention')}>
              <div className="space-y-4 text-gray-700">
                <p className="leading-relaxed">
                  {t('privacy.retention.intro', 'We retain your personal data for different periods based on the type of data and legal requirements:')}
                </p>
                <ul className="list-disc list-inside space-y-2">
                  <li>
                    {t('privacy.retention.activeAccounts', 'Active accounts: retained while account is active')}
                  </li>
                  <li>
                    {t('privacy.retention.closedAccounts', 'Closed accounts: 90 days grace period for recovery')}
                  </li>
                  <li>
                    {t('privacy.retention.guestOrders', 'Guest orders: 5 years (Indonesian tax law requirement)')}
                  </li>
                  <li>
                    {t('privacy.retention.auditLogs', 'Audit logs: 7 years for compliance investigations')}
                  </li>
                  <li>
                    {t('privacy.retention.temporary', 'Temporary data (verification tokens): 48 hours')}
                  </li>
                </ul>
              </div>
            </PolicySection>

            {/* Data Security */}
            <PolicySection title={t('privacy.security.title', 'Data Security')}>
              <p className="text-gray-700 leading-relaxed mb-4">
                {t('privacy.security.intro', 'We implement comprehensive security measures to protect your data:')}
              </p>
              <ul className="list-disc list-inside space-y-2 text-gray-700">
                <li>{t('privacy.security.encryption', 'Encryption at rest for all personal data')}</li>
                <li>{t('privacy.security.accessControl', 'Strict access controls and authentication')}</li>
                <li>{t('privacy.security.auditTrail', 'Complete audit trail of all data access')}</li>
                <li>{t('privacy.security.logMasking', 'Automatic masking of sensitive data in logs')}</li>
                <li>{t('privacy.security.regularAudits', 'Regular security audits and updates')}</li>
              </ul>
            </PolicySection>

            {/* Third-Party Sharing */}
            <PolicySection title={t('privacy.thirdParty.title', 'Third-Party Data Sharing')}>
              <p className="text-gray-700 leading-relaxed mb-4">
                {t('privacy.thirdParty.intro', 'We share your payment information with:')}
              </p>
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <h4 className="font-semibold text-blue-900 mb-2">
                  {t('privacy.thirdParty.midtrans', 'Midtrans (Payment Processor)')}
                </h4>
                <p className="text-sm text-blue-800 mb-2">
                  {t('privacy.thirdParty.midtransDesc', 'We use Midtrans to process QRIS and other payment methods securely. Only payment-related information is shared.')}
                </p>
                <a
                  href="https://midtrans.com/privacy"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-blue-600 hover:text-blue-700 underline"
                >
                  {t('privacy.thirdParty.viewPolicy', 'View Midtrans Privacy Policy')} →
                </a>
              </div>
            </PolicySection>

            {/* Your Rights */}
            <PolicySection title={t('privacy.yourRights.title', 'Your Rights')}>
              <p className="text-gray-700 leading-relaxed mb-4">
                {t('privacy.yourRights.intro', 'Under UU PDP, you have the following rights:')}
              </p>
              <ul className="list-disc list-inside space-y-2 text-gray-700">
                <li>
                  <strong>{t('privacy.yourRights.access', 'Right to Access')}</strong>:{' '}
                  {t('privacy.yourRights.accessDesc', 'View all your personal data we hold')}
                </li>
                <li>
                  <strong>{t('privacy.yourRights.correction', 'Right to Correction')}</strong>:{' '}
                  {t('privacy.yourRights.correctionDesc', 'Update incorrect or incomplete data')}
                </li>
                <li>
                  <strong>{t('privacy.yourRights.deletion', 'Right to Deletion')}</strong>:{' '}
                  {t('privacy.yourRights.deletionDesc', 'Request deletion of your personal data')}
                </li>
                <li>
                  <strong>{t('privacy.yourRights.revoke', 'Right to Revoke Consent')}</strong>:{' '}
                  {t('privacy.yourRights.revokeDesc', 'Withdraw consent for optional data processing')}
                </li>
                <li>
                  <strong>{t('privacy.yourRights.complaint', 'Right to Complaint')}</strong>:{' '}
                  {t('privacy.yourRights.complaintDesc', 'File a complaint with data protection authorities')}
                </li>
              </ul>
            </PolicySection>

            {/* Contact Information */}
            <PolicySection title={t('privacy.contact.title', 'Contact Us')}>
              <div className="bg-gray-50 rounded-lg p-6">
                <p className="text-gray-700 mb-4">
                  {t('privacy.contact.intro', 'For questions about this privacy policy or to exercise your rights:')}
                </p>
                <div className="space-y-2 text-gray-700">
                  <p>
                    <strong>{t('privacy.contact.emailLabel', 'Email')}:</strong>{' '}
                    <a
                      href="mailto:privacy@yourcompany.com"
                      className="text-blue-600 hover:text-blue-700 underline"
                    >
                      privacy@yourcompany.com
                    </a>
                  </p>
                  <p>
                    <strong>{t('privacy.contact.responseTime', 'Response Time')}:</strong>{' '}
                    {t('privacy.contact.responseTimeValue', 'Within 14 days as required by UU PDP Article 6')}
                  </p>
                </div>
              </div>
            </PolicySection>

            {/* Policy Updates */}
            <PolicySection title={t('privacy.updates.title', 'Policy Updates')}>
              <p className="text-gray-700 leading-relaxed">
                {t('privacy.updates.text', 'We may update this privacy policy periodically. Material changes will be communicated via email to registered users. The effective date at the top of this page indicates when the policy was last updated.')}
              </p>
            </PolicySection>
          </div>
        </main>
      </div>
    </PublicLayout>
  );
}

// Helper Components
function PolicySection({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section>
      <h2 className="text-2xl font-bold text-gray-900 mb-4">{title}</h2>
      {children}
    </section>
  );
}

function DataCategory({ title, items }: { title: string; items: string[] }) {
  return (
    <div>
      <h4 className="font-semibold text-gray-900 mb-2">{title}</h4>
      <ul className="list-disc list-inside space-y-1 text-gray-700 ml-4">
        {items.map((item, index) => (
          <li key={index}>{item}</li>
        ))}
      </ul>
    </div>
  );
}
