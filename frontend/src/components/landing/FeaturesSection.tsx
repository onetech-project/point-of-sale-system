import React from 'react';

interface Feature {
  icon: React.ReactNode;
  title: string;
  description: string;
}

function MultiTenantIcon() {
  return (
    <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
      <path d="M3 3h8v8H3V3zm10 0h8v8h-8V3zM3 13h8v8H3v-8zm10 0h8v8h-8v-8z" />
    </svg>
  );
}

function InventoryIcon() {
  return (
    <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
      <path d="M3 6h18v2H3V6zm0 4h18v2H3v-2zm0 4h18v2H3v-2zm0 4h18v2H3v-2z" />
    </svg>
  );
}

function OnlineOrderingIcon() {
  return (
    <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
      <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
    </svg>
  );
}

function PaymentIcon() {
  return (
    <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
      <path d="M20 8H4V6h16m0 10H4v-6h16m0-4H2c-1.11 0-2 .89-2 2v12c0 1.11.89 2 2 2h20c1.1 0 2-.9 2-2V4c0-1.11-.9-2-2-2z" />
    </svg>
  );
}

function AnalyticsIcon() {
  return (
    <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
      <path d="M5 9.2h3V19H5zM10.6 5h2.8v14h-2.8zm5.6 8H19v6h-2.8z" />
    </svg>
  );
}

function TeamIcon() {
  return (
    <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
      <path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5s-3 1.34-3 3 1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5C6.34 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z" />
    </svg>
  );
}

export default function FeaturesSection() {
  const features: Feature[] = [
    {
      icon: <MultiTenantIcon />,
      title: 'Multi-Tenant',
      description: 'Each business gets its own isolated workspace with complete data privacy',
    },
    {
      icon: <InventoryIcon />,
      title: 'Inventory Management',
      description: 'Real-time stock tracking with automated alerts for low inventory',
    },
    {
      icon: <OnlineOrderingIcon />,
      title: 'Online Ordering',
      description: 'Accept orders via QR code and digital menu from your customers',
    },
    {
      icon: <PaymentIcon />,
      title: 'QRIS & Credit Card',
      description: 'Seamless payment processing powered by Midtrans',
    },
    {
      icon: <AnalyticsIcon />,
      title: 'Analytics Dashboard',
      description: 'Sales insights and revenue reports at a glance',
    },
    {
      icon: <TeamIcon />,
      title: 'Team Management',
      description: 'Invite staff with role-based access control and permissions',
    },
  ];

  return (
    <section id="features" className="py-20 px-4 sm:px-6 lg:px-8 bg-white">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-4xl md:text-5xl font-bold text-gray-900 mb-4">
            Powerful Features for Your Business
          </h2>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto">
            Everything you need to manage your POS operations efficiently and grow your business
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
          {features.map((feature, index) => (
            <div
              key={index}
              className="p-8 rounded-xl border border-gray-200 hover:border-primary-600 hover:shadow-lg transition-all group bg-white"
            >
              <div className="text-primary-600 mb-4 group-hover:scale-110 transition-transform">
                {feature.icon}
              </div>
              <h3 className="text-xl font-bold text-gray-900 mb-2">{feature.title}</h3>
              <p className="text-gray-600">{feature.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
