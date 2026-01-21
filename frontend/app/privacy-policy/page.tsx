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
          <div className="bg-white rounded-lg shadow-md p-8">
            {policy && policy.policy_text_id && (
              <div 
                className="prose prose-lg max-w-none text-gray-700"
                dangerouslySetInnerHTML={{ __html: policy.policy_text_id }}
              />
            )}
            
            {!policy?.policy_text_id && (
              <div className="text-center py-12">
                <p className="text-gray-500">
                  {t('privacy.noContent', 'Privacy policy content not available')}
                </p>
              </div>
            )}
          </div>
        </main>
      </div>
    </PublicLayout>
  );
}
