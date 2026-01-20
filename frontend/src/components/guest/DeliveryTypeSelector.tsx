import React from 'react';

interface DeliveryType {
  value: string;
  label: string;
  description: string;
  icon: string;
}

interface DeliveryTypeSelectorProps {
  enabledTypes: string[];
  selectedType: string;
  onChange: (type: string) => void;
}

const deliveryTypes: Record<string, DeliveryType> = {
  pickup: {
    value: 'pickup',
    label: 'Pickup',
    description: 'Pick up your order at the store',
    icon: '🏪',
  },
  delivery: {
    value: 'delivery',
    label: 'Delivery',
    description: 'Get your order delivered to your address',
    icon: '🚚',
  },
  dine_in: {
    value: 'dine_in',
    label: 'Dine In',
    description: 'Eat at the restaurant',
    icon: '🍽️',
  },
};

export const DeliveryTypeSelector: React.FC<DeliveryTypeSelectorProps> = ({
  enabledTypes,
  selectedType,
  onChange,
}) => {
  const availableTypes = enabledTypes
    .map((type) => deliveryTypes[type])
    .filter((type) => type !== undefined);

  if (availableTypes.length === 0) {
    return (
      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
        <p className="text-yellow-800">No delivery types available for this merchant.</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <label className="block text-sm font-medium text-gray-700">
        Delivery Type <span className="text-red-500">*</span>
      </label>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        {availableTypes.map((type) => (
          <button
            data-testid={`delivery-type-${type.value}`}
            key={type.value}
            type="button"
            onClick={() => onChange(type.value)}
            className={`p-4 border-2 rounded-lg text-left transition-all ${
              selectedType === type.value
                ? 'border-blue-500 bg-blue-50'
                : 'border-gray-200 hover:border-gray-300 bg-white'
            }`}
          >
            <div className="flex items-start gap-3">
              <span className="text-3xl">{type.icon}</span>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-900">{type.label}</h3>
                <p className="text-sm text-gray-600 mt-1">{type.description}</p>
              </div>
              {selectedType === type.value && (
                <span className="text-blue-600">✓</span>
              )}
            </div>
          </button>
        ))}
      </div>
    </div>
  );
};

export default DeliveryTypeSelector;
