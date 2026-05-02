'use client';

import React, { useState } from 'react';
import Link from 'next/link';

interface PricingTier {
  name: string;
  monthlyPrice: number | null;
  monthlyPriceDisplay: string;
  annualPrice: number | null;
  annualPriceDisplay: string;
  originalAnnualPrice: number | null;
  description: string;
  features: string[];
  cta: string;
  ctaHref: string;
  highlighted?: boolean;
}

export default function PricingSection() {
  const [isAnnual, setIsAnnual] = useState(false);
  const discountPercent = 20;
  const calculateAnnualPrice = (monthlyPrice: number) =>
    Math.round(monthlyPrice * 12 * (1 - discountPercent / 100));

  const tiers: PricingTier[] = [
    {
      name: 'Starter',
      monthlyPrice: 0,
      monthlyPriceDisplay: 'Free',
      annualPrice: 0,
      annualPriceDisplay: 'Free',
      originalAnnualPrice: 0,
      description: 'Perfect to get started',
      features: [
        'Up to 2 staff members',
        '2 GB storage',
        'Basic inventory tracking',
        'Cash and card payments',
        'Daily sales reports',
      ],
      cta: 'Get Started Free',
      ctaHref: '/signup',
    },
    {
      name: 'Professional',
      monthlyPrice: 299000,
      monthlyPriceDisplay: `Rp ${(299000).toLocaleString('id-ID')}`,
      annualPrice: calculateAnnualPrice(299000),
      annualPriceDisplay: `Rp ${calculateAnnualPrice(299000).toLocaleString('id-ID')}`,
      originalAnnualPrice: 299000 * 12,
      description: 'Best for growing businesses',
      features: [
        'Up to 10 staff members',
        '10 GB storage',
        'Advanced inventory management',
        'Online ordering & QRIS',
        'Analytics dashboard',
        'Priority email support',
      ],
      cta: 'Get Started',
      ctaHref: '/signup',
      highlighted: true,
    },
    {
      name: 'Enterprise',
      monthlyPrice: null,
      monthlyPriceDisplay: 'Custom',
      annualPrice: null,
      annualPriceDisplay: 'Custom',
      originalAnnualPrice: null,
      description: 'For large organizations',
      features: [
        'Unlimited staff members',
        'Unlimited storage',
        'Custom integrations',
        'Dedicated account manager',
        'Advanced analytics',
        '24/7 phone & email support',
      ],
      cta: 'Contact Sales',
      ctaHref: '/contact',
    },
  ];

  return (
    <section className="py-20 px-4 sm:px-6 lg:px-8 bg-gradient-to-b from-gray-50 to-white">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-4xl md:text-5xl font-bold text-gray-900 mb-4">
            Simple, Transparent Pricing
          </h2>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto mb-8">
            Choose the plan that fits your business needs
          </p>

          {/* Billing toggle */}
          <div className="inline-flex items-center bg-gray-200 rounded-full p-1">
            <button
              onClick={() => setIsAnnual(false)}
              className={`px-6 py-2 rounded-full font-medium transition-all ${
                !isAnnual
                  ? 'bg-white text-primary-600 shadow-md'
                  : 'text-gray-600 bg-transparent'
              }`}
            >
              Monthly
            </button>
            <button
              onClick={() => setIsAnnual(true)}
              className={`px-6 py-2 rounded-full font-medium transition-all ${
                isAnnual
                  ? 'bg-white text-primary-600 shadow-md'
                  : 'text-gray-600 bg-transparent'
              }`}
            >
              Annual
            </button>
            {isAnnual && (
              <div className="ml-4 inline-block bg-green-100 text-green-800 px-3 py-1 rounded-full text-sm font-semibold">
                Save {discountPercent}%
              </div>
            )}
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 items-start">
          {tiers.map((tier, index) => (
            <div
              key={index}
              className={`relative rounded-2xl transition-all ${
                tier.highlighted
                  ? 'md:scale-105 bg-gradient-to-b from-primary-50 to-white border-2 border-primary-600 shadow-2xl'
                  : 'bg-white border border-gray-200 hover:border-primary-300 shadow-lg'
              }`}
            >
              {/* Most Popular badge */}
              {tier.highlighted && (
                <div className="absolute -top-4 left-1/2 transform -translate-x-1/2">
                  <div className="bg-primary-600 text-white px-4 py-1 rounded-full text-sm font-bold">
                    Most Popular
                  </div>
                </div>
              )}

              <div className="p-8">
                <h3 className="text-2xl font-bold text-gray-900 mb-2">{tier.name}</h3>
                <p className="text-gray-600 text-sm mb-6">{tier.description}</p>

                {/* Price */}
                <div className="mb-6">
                  <div className="text-4xl font-bold text-gray-900 mb-1">
                    {isAnnual
                      ? tier.annualPriceDisplay
                      : tier.monthlyPriceDisplay}
                  </div>
                  {isAnnual && tier.originalAnnualPrice !== null && tier.originalAnnualPrice > 0 && (
                    <div className="text-sm text-gray-500">
                      <span className="line-through">
                        Rp {tier.originalAnnualPrice.toLocaleString('id-ID')}
                      </span>
                      /year
                    </div>
                  )}
                  {!isAnnual && tier.monthlyPrice !== null && tier.monthlyPrice > 0 && (
                    <div className="text-sm text-gray-500">/month</div>
                  )}
                </div>

                {/* CTA Button */}
                <Link
                  href={tier.ctaHref}
                  className={`block w-full py-3 px-4 rounded-lg font-semibold text-center transition-colors mb-8 ${
                    tier.highlighted
                      ? 'bg-primary-600 text-white hover:bg-primary-700'
                      : 'bg-gray-100 text-gray-900 hover:bg-gray-200'
                  }`}
                >
                  {tier.cta}
                </Link>

                {/* Features */}
                <div className="space-y-4">
                  {tier.features.map((feature, featureIndex) => (
                    <div key={featureIndex} className="flex items-start">
                      <svg
                        className="w-5 h-5 text-primary-600 mr-3 flex-shrink-0 mt-0.5"
                        fill="currentColor"
                        viewBox="0 0 20 20"
                      >
                        <path
                          fillRule="evenodd"
                          d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                          clipRule="evenodd"
                        />
                      </svg>
                      <span className="text-gray-700 text-sm">{feature}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* FAQs or note */}
        <div className="text-center mt-16">
          <p className="text-gray-600">
            All plans include a 7-day free trial. Upgrade or downgrade anytime.
          </p>
        </div>
      </div>
    </section>
  );
}
