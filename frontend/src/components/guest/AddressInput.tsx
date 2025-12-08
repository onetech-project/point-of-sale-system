import React, { useState } from 'react';

interface AddressInputProps {
  value: string;
  onChange: (value: string) => void;
  error?: string;
  required?: boolean;
  validationError?: {
    type: 'geocoding' | 'service_area' | 'general';
    message: string;
  };
}

export const AddressInput: React.FC<AddressInputProps> = ({
  value,
  onChange,
  error,
  required = true,
  validationError,
}) => {
  const [detailedAddress, setDetailedAddress] = useState({
    street: '',
    city: '',
    province: '',
    postalCode: '',
    notes: '',
  });

  const handleFieldChange = (field: string, fieldValue: string) => {
    const updated = { ...detailedAddress, [field]: fieldValue };
    setDetailedAddress(updated);

    // Combine fields into single address string
    const fullAddress = [
      updated.street,
      updated.city,
      updated.province,
      updated.postalCode,
    ]
      .filter((part) => part.trim() !== '')
      .join(', ');

    onChange(fullAddress);
  };

  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Delivery Address {required && <span className="text-red-500">*</span>}
        </label>

        {/* Street Address */}
        <div className="mb-3">
          <input
            type="text"
            placeholder="Street address (e.g., Jl. Sudirman No. 123)"
            value={detailedAddress.street}
            onChange={(e) => handleFieldChange('street', e.target.value)}
            className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent ${error ? 'border-red-500' : 'border-gray-300'
              }`}
            required={required}
          />
        </div>

        {/* City and Province */}
        <div className="grid grid-cols-2 gap-3 mb-3">
          <input
            type="text"
            placeholder="City (e.g., Jakarta)"
            value={detailedAddress.city}
            onChange={(e) => handleFieldChange('city', e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            required={required}
          />
          <input
            type="text"
            placeholder="Province (e.g., DKI Jakarta)"
            value={detailedAddress.province}
            onChange={(e) => handleFieldChange('province', e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            required={required}
          />
        </div>

        {/* Postal Code */}
        <div className="mb-3">
          <input
            type="text"
            placeholder="Postal code (e.g., 12190)"
            value={detailedAddress.postalCode}
            onChange={(e) => handleFieldChange('postalCode', e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            maxLength={5}
          />
        </div>

        {/* Additional Notes */}
        <div>
          <textarea
            placeholder="Additional notes (e.g., building name, floor, landmarks)"
            value={detailedAddress.notes}
            onChange={(e) => handleFieldChange('notes', e.target.value)}
            rows={2}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
          />
        </div>

        {error && (
          <p className="mt-2 text-sm text-red-600 flex items-start gap-2">
            <svg
              className="w-4 h-4 mt-0.5 flex-shrink-0"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                clipRule="evenodd"
              />
            </svg>
            <span>{error}</span>
          </p>
        )}

        {/* T085: Address validation error handling with user-friendly messages */}
        {validationError && (
          <div className="mt-3 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
            <div className="flex items-start gap-2">
              {validationError.type === 'geocoding' && (
                <svg
                  className="w-5 h-5 text-yellow-600 mt-0.5 flex-shrink-0"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M5.05 4.05a7 7 0 119.9 9.9L10 18.9l-4.95-4.95a7 7 0 010-9.9zM10 11a2 2 0 100-4 2 2 0 000 4z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
              {validationError.type === 'service_area' && (
                <svg
                  className="w-5 h-5 text-yellow-600 mt-0.5 flex-shrink-0"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
              <div className="flex-1">
                <p className="text-sm font-medium text-yellow-800">
                  {validationError.type === 'geocoding' && 'Address Not Found'}
                  {validationError.type === 'service_area' &&
                    'Outside Delivery Area'}
                  {validationError.type === 'general' && 'Validation Error'}
                </p>
                <p className="text-sm text-yellow-700 mt-1">
                  {validationError.message}
                </p>
                {validationError.type === 'geocoding' && (
                  <ul className="mt-2 text-xs text-yellow-600 list-disc list-inside space-y-1">
                    <li>Check for typos in street names or building numbers</li>
                    <li>Include city and province for better accuracy</li>
                    <li>Use well-known landmarks if available</li>
                  </ul>
                )}
                {validationError.type === 'service_area' && (
                  <p className="mt-2 text-xs text-yellow-600">
                    Please contact the restaurant for alternative delivery
                    options, or consider pickup instead.
                  </p>
                )}
              </div>
            </div>
          </div>
        )}

        <p className="mt-2 text-xs text-gray-500">
          💡 Tip: Provide detailed address information to ensure accurate delivery
        </p>
      </div>
    </div>
  );
};

export default AddressInput;
