'use client';

import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import consentService from '@/services/consent';

interface ConsentPurpose {
  purpose_code: string;
  display_name_id: string;
  description_id: string;
  is_required: boolean;
  display_order: number;
}

type ConsentStatus = Record<string, boolean>;

export default function ConsentSettingsSection() {
  const { t } = useTranslation('privacy_settings');
  const [purposes, setPurposes] = useState<ConsentPurpose[]>([]);
  const [consentStatus, setConsentStatus] = useState<ConsentStatus>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [processing, setProcessing] = useState<string | null>(null);

  useEffect(() => {
    loadConsentData();
  }, []);

  const loadConsentData = async () => {
    try {
      setLoading(true);
      const [purposesData, statusResponse] = await Promise.all([
        consentService.getConsentPurposes('tenant'),
        consentService.getConsentStatus(),
      ]);
      setPurposes(purposesData);
      
      // Convert array of consents to status map
      const statusMap: ConsentStatus = {};
      if (Array.isArray(statusResponse)) {
        statusResponse.forEach((consent: any) => {
          if (consent.purpose_code) {
            statusMap[consent.purpose_code] = true;
          }
        });
      } else if (typeof statusResponse === 'object') {
        // Handle both formats (array or object)
        Object.assign(statusMap, statusResponse);
      }
      
      setConsentStatus(statusMap);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load consent data');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleConsent = async (purposeCode: string, currentlyGranted: boolean) => {
    try {
      setProcessing(purposeCode);
      
      if (currentlyGranted) {
        // Revoke consent
        await consentService.revokeConsent(purposeCode);
        setConsentStatus(prev => ({ ...prev, [purposeCode]: false }));
      } else {
        // Re-grant consent
        await consentService.grantConsent([purposeCode]);
        setConsentStatus(prev => ({ ...prev, [purposeCode]: true }));
      }
    } catch (err) {
      // Rollback on error
      setError(err instanceof Error ? err.message : 'Failed to update consent');
      // Reload to get correct state
      await loadConsentData();
    } finally {
      setProcessing(null);
    }
  };

  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow p-6">
        <div className="text-center py-8 text-gray-500">{t('loading')}</div>
      </div>
    );
  }

  if (error && purposes.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow p-6">
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">{error}</div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow">
      {error && (
        <div className="border-b border-gray-200 p-4">
          <div className="bg-yellow-50 border border-yellow-300 text-yellow-800 px-4 py-3 rounded">
            {error}
          </div>
        </div>
      )}

      <div className="p-6">
        <h2 className="text-xl font-semibold mb-4">{t('consents.title')}</h2>
        <p className="text-gray-600 mb-6">{t('consents.description')}</p>

        <div className="space-y-4">
          {purposes.map((purpose) => {
            const isGranted = consentStatus[purpose.purpose_code] || false;
            const isProcessing = processing === purpose.purpose_code;

            return (
              <div
                key={purpose.purpose_code}
                className="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="text-base font-medium text-gray-900">
                        {purpose.display_name_id}
                      </h3>
                      {purpose.is_required && (
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">
                          {t('consents.required')}
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-gray-600">{purpose.description_id}</p>

                    {purpose.is_required && (
                      <div className="mt-2 text-xs text-gray-500 italic">
                        {t('consents.cannot_revoke')}
                      </div>
                    )}
                  </div>

                  <div className="ml-4 flex-shrink-0">
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        checked={isGranted}
                        disabled={purpose.is_required || isProcessing}
                        onChange={() => handleToggleConsent(purpose.purpose_code, isGranted)}
                        className="sr-only peer"
                      />
                      <div className={`
                        w-11 h-6 rounded-full peer
                        ${purpose.is_required ? 'bg-gray-300 cursor-not-allowed' : 'bg-gray-200'}
                        peer-focus:ring-4 peer-focus:ring-blue-300
                        peer-checked:after:translate-x-full
                        after:content-[''] after:absolute after:top-0.5 after:left-[2px]
                        after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all
                        ${isGranted && !purpose.is_required ? 'peer-checked:bg-blue-600' : ''}
                        ${isProcessing ? 'opacity-50 cursor-wait' : ''}
                      `}></div>
                    </label>
                  </div>
                </div>

                {/* Status indicator */}
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
    </div>
  );
}
