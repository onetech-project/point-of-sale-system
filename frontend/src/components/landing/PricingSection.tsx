'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';

// Fallback values — kept in sync with tenant-service env defaults.
// Once the billing service is deployed, GET /api/v1/public/plans
// becomes the authoritative source and these are never shown.
const MONTHLY_PRICE = 299000;
const ANNUAL_DISCOUNT_PCT = 20;
const TRIAL_DAYS = 7;

interface PlanData {
  monthly_price_idr: number;
  annual_discount_pct: number;
  trial_days: number;
}

function CheckIcon() {
  return (
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
  );
}

const ALL_FEATURES = [
  'Online ordering with shareable menu link',
  'Customer QRIS & card payments via Midtrans',
  'Offline order management & installment plans',
  'Dashboard analytics & revenue reports',
  'Inventory management with low-stock alerts',
  'Team management with role-based access',
  'Email notifications for orders & invoices',
  'Data privacy & audit log (UU PDP compliant)',
  'Up to 2 GB storage per workspace',
];

export default function PricingSection() {
  const [isAnnual, setIsAnnual] = useState(false);
  const [plan, setPlan] = useState<PlanData>({
    monthly_price_idr: MONTHLY_PRICE,
    annual_discount_pct: ANNUAL_DISCOUNT_PCT,
    trial_days: TRIAL_DAYS,
  });

  useEffect(() => {
    fetch('/api/v1/public/plans')
      .then(r => (r.ok ? r.json() : null))
      .then(data => {
        if (data?.monthly_price_idr) setPlan(data);
      })
      .catch(() => {
        /* keep fallback */
      });
  }, []);

  const monthlyPrice = plan.monthly_price_idr;
  const annualTotal = Math.round(monthlyPrice * 12 * (1 - plan.annual_discount_pct / 100));
  const annualMonthly = Math.round(annualTotal / 12);
  const annualSavings = monthlyPrice * 12 - annualTotal;

  const formatRp = (n: number) => `Rp\u00a0${n.toLocaleString('id-ID')}`;

  return (
    <section
      id="pricing"
      className="py-20 px-4 sm:px-6 lg:px-8 bg-gradient-to-b from-gray-50 to-white"
    >
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="text-center mb-12">
          <h2 className="text-4xl md:text-5xl font-bold text-gray-900 mb-4">
            Simple, Transparent Pricing
          </h2>
          <p className="text-xl text-gray-600 max-w-xl mx-auto">
            One plan. All features. Start free — no credit card required.
          </p>
        </div>

        {/* Billing toggle */}
        <div className="flex justify-center mb-10">
          <div className="inline-flex items-center bg-gray-100 rounded-full p-1 gap-1">
            <button
              onClick={() => setIsAnnual(false)}
              className={`px-6 py-2 rounded-full text-sm font-semibold transition-all ${
                !isAnnual ? 'bg-white text-primary-600 shadow' : 'text-gray-500'
              }`}
            >
              Monthly
            </button>
            <button
              onClick={() => setIsAnnual(true)}
              className={`px-6 py-2 rounded-full text-sm font-semibold transition-all ${
                isAnnual ? 'bg-white text-primary-600 shadow' : 'text-gray-500'
              }`}
            >
              Annual
              {!isAnnual && (
                <span className="ml-2 bg-green-100 text-green-700 text-xs px-2 py-0.5 rounded-full">
                  Save {plan.annual_discount_pct}%
                </span>
              )}
            </button>
          </div>
        </div>

        {/* Single plan card */}
        <div className="relative bg-white border-2 border-primary-600 rounded-3xl shadow-2xl overflow-hidden">
          {/* Top accent */}
          <div className="h-2 bg-gradient-to-r from-primary-500 to-primary-700" />

          <div className="p-6 md:p-12">
            <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-8">
              {/* Left: price block */}
              <div className="flex-shrink-0">
                <div className="inline-flex items-center bg-primary-50 text-primary-700 text-sm font-semibold px-3 py-1 rounded-full mb-4">
                  {plan.trial_days}-day free trial
                </div>

                {isAnnual ? (
                  <>
                    <div className="text-5xl font-extrabold text-gray-900 leading-none">
                      {formatRp(annualMonthly)}
                    </div>
                    <div className="text-gray-500 text-sm mt-1">per month, billed annually</div>
                    <div className="mt-2 text-gray-400 text-sm">
                      <span className="line-through">{formatRp(monthlyPrice * 12)}</span> &rarr;{' '}
                      <span className="text-green-600 font-semibold">
                        {formatRp(annualTotal)}/year
                      </span>
                    </div>
                    <div className="mt-1 text-green-600 text-sm font-medium">
                      You save {formatRp(annualSavings)} per year
                    </div>
                  </>
                ) : (
                  <>
                    <div className="text-5xl font-extrabold text-gray-900 leading-none">
                      {formatRp(monthlyPrice)}
                    </div>
                    <div className="text-gray-500 text-sm mt-1">per month</div>
                    <div className="mt-2 text-gray-400 text-sm">
                      Switch to annual &amp; save{' '}
                      <span className="text-green-600 font-semibold">
                        {plan.annual_discount_pct}%
                      </span>
                    </div>
                  </>
                )}

                <Link
                  href="/signup"
                  className="mt-6 block w-full py-3 px-4 rounded-xl bg-primary-600 text-white font-bold text-center hover:bg-primary-700 transition-colors"
                >
                  Start Free Trial
                </Link>
                <p className="text-center text-xs text-gray-400 mt-2">No credit card required</p>
              </div>

              {/* Divider */}
              <div className="hidden md:block w-px bg-gray-200 self-stretch" />

              {/* Right: features */}
              <div className="flex-1">
                <p className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-4">
                  Everything included
                </p>
                <ul className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  {ALL_FEATURES.map((feature, i) => (
                    <li key={i} className="flex items-start">
                      <CheckIcon />
                      <span className="text-gray-700 text-sm">{feature}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        </div>

        {/* Trial note */}
        <p className="text-center text-gray-500 text-sm mt-8">
          After your {TRIAL_DAYS}-day free trial, choose monthly or annual billing. Cancel anytime —
          no lock-in.
        </p>
      </div>
    </section>
  );
}
