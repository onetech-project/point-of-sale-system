import React from 'react';

interface Feature {
  icon: React.ReactNode;
  title: string;
  description: string;
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

function OfflineOrderIcon() {
  return (
    <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
      <path d="M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96zM19 18H6c-2.21 0-4-1.79-4-4s1.79-4 4-4h.71C7.37 7.69 9.48 6 12 6c3.04 0 5.5 2.46 5.5 5.5v.5H19c1.66 0 3 1.34 3 3s-1.34 3-3 3zm-9-3.82l-2.09-2.09L6.5 13.5 10 17l6.01-6.01-1.42-1.41z" />
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

function InventoryIcon() {
  return (
    <svg className="w-8 h-8" fill="currentColor" viewBox="0 0 24 24">
      <path d="M3 6h18v2H3V6zm0 4h18v2H3v-2zm0 4h18v2H3v-2zm0 4h18v2H3v-2z" />
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
      icon: <OnlineOrderingIcon />,
      title: 'Online Ordering with Menu Catalog',
      description:
        'Share a QR code link — customers browse your digital menu and place orders directly from their phone, no app required.',
    },
    {
      icon: <PaymentIcon />,
      title: 'Customer QRIS Payment',
      description:
        'Accept instant payments via QRIS powered by Midtrans. Works with all major Indonesian e-wallets and bank apps.',
    },
    {
      icon: <OfflineOrderIcon />,
      title: 'Offline Order Management',
      description:
        'Record walk-in, phone, or WhatsApp orders directly in the system. Support cash payments and installment plans for orders taken outside the digital menu.',
    },
    {
      icon: <AnalyticsIcon />,
      title: 'Dashboard Analytics',
      description:
        'Track daily revenue, top-selling products, and customer trends with a clear at-a-glance business dashboard.',
    },
    {
      icon: <InventoryIcon />,
      title: 'Inventory Management',
      description:
        'Real-time stock tracking with automatic low-stock alerts and full audit trail for every adjustment.',
    },
    {
      icon: <TeamIcon />,
      title: 'Team Management',
      description:
        'Invite staff by email and assign roles — Owner, Manager, or Cashier — with granular access control per feature.',
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
