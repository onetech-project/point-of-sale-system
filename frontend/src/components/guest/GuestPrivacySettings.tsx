'use client';

import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import consentService from '@/services/consent';

interface ConsentPurpose {
  purpose_code: string;
  display_name_id: string;
  description_id: string;
  is_required: boolean;
}

interface GuestPrivacySettingsProps {
  orderReference: string;
}

export default function GuestPrivacySettings({ orderReference }: GuestPrivacySettingsProps) {
  const { t } = useTranslation('privacy_settings');
  const [purposes, setPurposes] = useState<ConsentPurpose[]>([]);
  const [consentStatus, setConsentStatus] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(true);
  const [processing, setProcessing] = useState<string | null>(null);

  useEffect(() => {
    loadConsentData();
  }, [orderReference]);

  const loadConsentData = async () => {
    try {
      setLoading(true);
      const [purposesData, statusResponse] = await Promise.all([
        consentService.getConsentPurposes(localStorage.getItem('locale') || 'en', 'guest'),
        consentService.getConsentStatus('guest', orderReference),
      ]);
      
      // Filter to only show optional consents (guests can't change required ones)
      const optionalPurposes = purposesData.filter((p: ConsentPurpose) => !p.is_required);
      setPurposes(optionalPurposes);
      
      const statusMap: Record<string, boolean> = {};
      statusResponse.forEach((consent: { purpose_code: string; granted: boolean }) => {
        statusMap[consent.purpose_code] = consent.granted;
      });
      
      setConsentStatus(statusMap);
    } catch (err) {
      console.error('Failed to load consent data:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleConsent = async (purposeCode: string, currentlyGranted: boolean) => {
    try {
      setProcessing(purposeCode);
      
      if (currentlyGranted) {
        await consentService.revokeConsent(purposeCode, 'guest', orderReference);
        setConsentStatus(prev => ({ ...prev, [purposeCode]: false }));
      } else {
        await consentService.grantConsent([purposeCode], 'guest', orderReference);
        setConsentStatus(prev => ({ ...prev, [purposeCode]: true }));
      }
    } catch (err) {
      console.error('Failed to update consent:', err);
      await loadConsentData();
    } finally {
      setProcessing(null);
    }
  };

  if (loading) {
    return (
      <div className="bg-white rounded-2xl shadow-xl p-8">
        <div className="text-center py-4 text-gray-500">{t('loading')}</div>
      </div>
    );
  }

  if (purposes.length === 0) {
    return null; // No optional consents to show
  }

  return (
    <div className="bg-white rounded-2xl shadow-xl p-8">
      <h2 className="text-xl font-bold text-gray-900 mb-4">{t('consents.title')}</h2>
      <p className="text-gray-600 mb-6">{t('consents.description')}</p>

      <div className="space-y-4">
        {purposes.map((purpose) => {
          const isGranted = consentStatus[purpose.purpose_code] || false;
          const isProcessing = processing === purpose.purpose_code;

          return (
            <div
              key={purpose.purpose_code}
              className="border border-gray-200 rounded-lg p-4"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <h3 className="text-base font-medium text-gray-900 mb-1">
                    {purpose.display_name_id}
                  </h3>
                  <p className="text-sm text-gray-600">{purpose.description_id}</p>
                </div>

                <div className="ml-4 flex-shrink-0">
                  <button
                    type="button"
                    disabled={purpose.is_required || isProcessing}
                    onClick={() => !purpose.is_required && !isProcessing && handleToggleConsent(purpose.purpose_code, isGranted)}
                    className={`
                      relative
                      inline-flex
                      h-6 w-11
                      flex-shrink-0
                      cursor-pointer
                      rounded-full
                      border-2
                      border-transparent
                      transition-colors
                      duration-200
                      ease-in-out
                      focus:outline-none
                      focus:ring-2
                      focus:ring-blue-500
                      focus:ring-offset-2
                      ${isGranted ? 'bg-blue-600' : 'bg-gray-200'}
                      ${purpose.is_required || isProcessing ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'}
                    `}
                  >
                    <span
                      className={`
                        pointer-events-none
                        inline-block h-5 w-5
                        transform rounded-full
                        bg-white shadow ring-0
                        transition duration-200
                        ease-in-out ${isGranted ? 'translate-x-5' : 'translate-x-0'}
                      `}
                    />
                  </button>
                </div>
              </div>

              <div className="mt-3 flex items-center text-xs">
                {isGranted ? (
                  <span className="flex items-center text-green-600">
                    <svg className="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                    </svg>
                    {t('consents.status.granted')}
                  </span>
                ) : (
                  <span className="flex items-center text-gray-500">
                    <svg className="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                    </svg>
                    {t('consents.status.revoked')}
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
