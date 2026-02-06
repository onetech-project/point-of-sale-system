import { useTranslation } from 'next-i18next';

interface ConsentCheckboxProps {
  purposeCode: string;
  displayName: string;
  description: string;
  isRequired: boolean;
  checked: boolean;
  onChange: (purposeCode: string, checked: boolean) => void;
  disabled?: boolean;
}

export default function ConsentCheckbox({
  purposeCode,
  displayName,
  description,
  isRequired,
  checked,
  onChange,
  disabled = false,
}: ConsentCheckboxProps) {
  const { t } = useTranslation('consent');

  return (
    <div className="flex items-start space-x-3 py-3">
      <div className="flex items-center h-5">
        <input
          id={`consent-${purposeCode}`}
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(purposeCode, e.target.checked)}
          disabled={disabled || isRequired}
          className={`w-4 h-4 border-gray-300 rounded focus:ring-2 focus:ring-blue-500 ${
            isRequired ? 'bg-gray-100 cursor-not-allowed' : 'cursor-pointer'
          }`}
        />
      </div>
      <div className="flex-1">
        <label htmlFor={`consent-${purposeCode}`} className="block text-sm font-medium text-gray-900">
          {displayName}
          {isRequired && <span className="ml-2 text-red-600 text-xs">{t('required')}</span>}
          {!isRequired && <span className="ml-2 text-gray-500 text-xs">{t('optional')}</span>}
        </label>
        <p className="text-sm text-gray-600 mt-1">{description}</p>
      </div>
    </div>
  );
}
