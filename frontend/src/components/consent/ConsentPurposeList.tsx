import { useEffect, useState } from 'react';
import { useTranslation } from 'next-i18next';
import ConsentCheckbox from './ConsentCheckbox';

interface ConsentPurpose {
  purpose_code: string;
  display_name_id: string;
  description_id: string;
  is_required: boolean;
  display_order: number;
}

interface ConsentPurposeListProps {
  onConsentChange: (consents: { [key: string]: boolean }) => void;
  initialConsents?: { [key: string]: boolean };
  showError?: boolean;
  errorMessage?: string;
}

export default function ConsentPurposeList({
  onConsentChange,
  initialConsents = {},
  showError = false,
  errorMessage,
}: ConsentPurposeListProps) {
  const { t } = useTranslation('consent');
  const [purposes, setPurposes] = useState<ConsentPurpose[]>([]);
  const [consents, setConsents] = useState<{ [key: string]: boolean }>(initialConsents);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchConsentPurposes();
  }, []);

  const fetchConsentPurposes = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/v1/consent/purposes', {
        headers: {
          'Accept-Language': 'id', // Indonesian
        },
      });

      if (!response.ok) {
        throw new Error('Failed to fetch consent purposes');
      }

      const data = await response.json();
      const sortedPurposes = data.data.sort((a: ConsentPurpose, b: ConsentPurpose) => 
        a.display_order - b.display_order
      );
      setPurposes(sortedPurposes);

      // Initialize consents for required purposes
      const initialState: { [key: string]: boolean } = {};
      sortedPurposes.forEach((purpose: ConsentPurpose) => {
        if (purpose.is_required) {
          initialState[purpose.purpose_code] = true;
        } else if (initialConsents[purpose.purpose_code] !== undefined) {
          initialState[purpose.purpose_code] = initialConsents[purpose.purpose_code];
        } else {
          initialState[purpose.purpose_code] = false;
        }
      });
      setConsents(initialState);
      onConsentChange(initialState);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load consent purposes');
    } finally {
      setLoading(false);
    }
  };

  const handleConsentChange = (purposeCode: string, checked: boolean) => {
    const updatedConsents = {
      ...consents,
      [purposeCode]: checked,
    };
    setConsents(updatedConsents);
    onConsentChange(updatedConsents);
  };

  if (loading) {
    return (
      <div className="animate-pulse space-y-3">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="h-16 bg-gray-200 rounded"></div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-md p-4">
        <p className="text-sm text-red-600">{error}</p>
        <button
          onClick={fetchConsentPurposes}
          className="mt-2 text-sm text-red-700 underline hover:no-underline"
        >
          {t('retry')}
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
        {purposes.map((purpose) => (
          <ConsentCheckbox
            key={purpose.purpose_code}
            purposeCode={purpose.purpose_code}
            displayName={t(purpose.display_name_id)}
            description={t(purpose.description_id)}
            isRequired={purpose.is_required}
            checked={consents[purpose.purpose_code] || false}
            onChange={handleConsentChange}
          />
        ))}
      </div>

      {showError && errorMessage && (
        <div className="mt-2 p-3 bg-red-50 border border-red-200 rounded-md">
          <p className="text-sm text-red-600">{errorMessage}</p>
        </div>
      )}

      <div className="mt-4 text-sm text-gray-600">
        <p>
          {t('privacy_policy_notice')}{' '}
          <a
            href="/privacy-policy"
            target="_blank"
            rel="noopener noreferrer"
            className="text-blue-600 hover:text-blue-700 underline"
          >
            {t('privacy_policy_link')}
          </a>
        </p>
      </div>
    </div>
  );
}
