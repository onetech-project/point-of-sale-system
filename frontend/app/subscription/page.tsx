'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import DashboardLayout from '../../src/components/layout/DashboardLayout';
import { billingService, BillingSubscriptionUI, PublicPlans } from '@/services/billing';

type BillingInterval = 'monthly' | 'annual';

function formatCurrencyIDR(amount: number): string {
  return `Rp\u00a0${amount.toLocaleString('id-ID')}`;
}

function formatDateIDR(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });
}

const FEATURES = [
  'Unlimited products & categories',
  'Multi-user support (up to 10 staff)',
  'Real-time analytics & reports',
  'QRIS payment integration',
  'Online ordering system',
  'Customer management',
  'Inventory tracking',
  'Email invoices & notifications',
];

function StatusCard({ subscription }: { subscription: BillingSubscriptionUI }) {
  const statusConfig = {
    trial: {
      label: 'Trial Period',
      bgClass: 'bg-primary-100 text-primary-700',
      borderClass: 'border-primary-200',
    },
    active: {
      label: 'Active',
      bgClass: 'bg-green-100 text-green-700',
      borderClass: 'border-green-200',
    },
    grace_period: {
      label: 'Grace Period',
      bgClass: 'bg-yellow-100 text-yellow-700',
      borderClass: 'border-yellow-200',
    },
    expired: {
      label: 'Suspended',
      bgClass: 'bg-red-100 text-red-700',
      borderClass: 'border-red-200',
    },
  };

  const cfg = statusConfig[subscription.status] || statusConfig.trial;
  const trialTotal = 7;
  const daysRemaining = subscription.trial_days_remaining ?? subscription.days_remaining ?? 0;
  const trialProgress = Math.max(0, Math.min(100, (daysRemaining / trialTotal) * 100));

  return (
    <div className={`border rounded-xl p-6 ${cfg.borderClass} bg-white`}>
      <div className="flex items-center gap-3 mb-4">
        <h2 className="text-lg font-semibold text-gray-900">Current Plan</h2>
        <span className={`px-2.5 py-0.5 rounded-full text-xs font-semibold ${cfg.bgClass}`}>
          {cfg.label}
        </span>
      </div>

      {subscription.status === 'trial' && (
        <div>
          <p className="text-gray-600 mb-3">
            <span className="font-semibold text-gray-900">{daysRemaining}</span> days remaining in
            your free trial.
          </p>
          <div className="w-full bg-gray-200 rounded-full h-2 mb-2">
            <div
              className="bg-primary-600 h-2 rounded-full transition-all"
              style={{ width: `${trialProgress}%` }}
            />
          </div>
          <p className="text-xs text-gray-500">
            {daysRemaining} of {trialTotal} days left
          </p>
        </div>
      )}

      {subscription.status === 'active' && subscription.subscription_ends_at && (
        <p className="text-gray-600">
          Next renewal:{' '}
          <span className="font-semibold text-gray-900">
            {formatDateIDR(subscription.subscription_ends_at)}
          </span>{' '}
          &middot; {subscription.billing_interval === 'monthly' ? 'Monthly' : 'Annual'}
        </p>
      )}

      {subscription.status === 'grace_period' && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-700 font-medium">
            Your trial has expired. Subscribe now to continue using the POS system.
          </p>
        </div>
      )}

      {subscription.status === 'expired' && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-700 font-medium">
            Your account has been suspended. Please subscribe to reactivate.
          </p>
        </div>
      )}
    </div>
  );
}

export default function SubscriptionPage() {
  const router = useRouter();
  const [subscription, setSubscription] = useState<BillingSubscriptionUI | null>(null);
  const [plans, setPlans] = useState<PublicPlans | null>(null);
  const [billingInterval, setBillingInterval] = useState<BillingInterval>('monthly');
  const [loading, setLoading] = useState(true);
  const [upgrading, setUpgrading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([billingService.getSubscription(), billingService.getPublicPlans()])
      .then(([sub, p]) => {
        setSubscription(sub);
        setPlans(p);
        setBillingInterval(sub.billing_interval ?? 'monthly');
      })
      .catch(err => {
        console.error('Failed to load subscription:', err);
        setError('Failed to load subscription information.');
      })
      .finally(() => setLoading(false));
  }, []);

  const annualPriceMonthly = plans
    ? Math.round((plans.monthly_price_idr * 12 * (1 - plans.annual_discount_pct / 100)) / 12)
    : 0;
  const annualTotal = annualPriceMonthly * 12;

  const displayPrice =
    billingInterval === 'monthly' ? (plans?.monthly_price_idr ?? 0) : annualTotal;

  const handleUpgrade = async () => {
    try {
      setUpgrading(true);
      setError(null);
      const result = await billingService.upgradeSubscription(billingInterval);
      if (result.payment_url) {
        window.location.href = result.payment_url;
      }
    } catch (err: any) {
      console.error('Upgrade failed:', err);
      setError(err.response?.data?.message ?? 'Failed to initiate subscription. Please try again.');
    } finally {
      setUpgrading(false);
    }
  };

  const isActive =
    subscription?.status === 'active' || subscription?.subscription_status === 'active';
  const buttonLabel = isActive ? 'Upgrade Now' : 'Start Subscription';

  if (loading) {
    return (
      <ProtectedRoute>
        <DashboardLayout>
          <div className="max-w-2xl mx-auto py-8">
            <div className="animate-pulse space-y-4">
              <div className="h-8 bg-gray-200 rounded w-48" />
              <div className="h-40 bg-gray-200 rounded-xl" />
              <div className="h-48 bg-gray-200 rounded-xl" />
            </div>
          </div>
        </DashboardLayout>
      </ProtectedRoute>
    );
  }

  return (
    <ProtectedRoute>
      <DashboardLayout>
        <div className="max-w-2xl mx-auto py-8 space-y-6">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-bold text-gray-900">Subscription & Billing</h1>
            <a href="/subscription/invoices" className="text-sm text-primary-600 hover:underline">
              View Invoices
            </a>
          </div>

          {error && (
            <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-sm text-red-700">
              {error}
            </div>
          )}

          {subscription && <StatusCard subscription={subscription} />}

          {/* Billing Interval Toggle */}
          <div className="bg-white border border-gray-200 rounded-xl p-6">
            <h3 className="text-base font-semibold text-gray-900 mb-4">Billing</h3>
            <div className="flex gap-3">
              <button
                onClick={() => setBillingInterval('monthly')}
                className={`flex-1 py-3 px-4 rounded-lg border text-sm font-medium transition-colors ${
                  billingInterval === 'monthly'
                    ? 'border-primary-600 bg-primary-50 text-primary-700'
                    : 'border-gray-200 text-gray-600 hover:border-gray-300'
                }`}
              >
                Monthly
              </button>
              <button
                onClick={() => setBillingInterval('annual')}
                className={`flex-1 py-3 px-4 rounded-lg border text-sm font-medium transition-colors relative ${
                  billingInterval === 'annual'
                    ? 'border-primary-600 bg-primary-50 text-primary-700'
                    : 'border-gray-200 text-gray-600 hover:border-gray-300'
                }`}
              >
                Annual
                {plans && (
                  <span className="ml-2 inline-flex items-center px-1.5 py-0.5 rounded text-xs font-semibold bg-green-100 text-green-700">
                    Save {plans.annual_discount_pct}%
                  </span>
                )}
              </button>
            </div>

            <div className="mt-6 flex items-end gap-2">
              <span className="text-3xl font-bold text-gray-900">
                {formatCurrencyIDR(displayPrice)}
              </span>
              <span className="text-gray-500 pb-1">
                /{billingInterval === 'monthly' ? 'month' : 'year'}
              </span>
            </div>
            {billingInterval === 'annual' && plans && (
              <p className="mt-1 text-sm text-green-600">
                {formatCurrencyIDR(annualPriceMonthly)}/month &mdash; you save{' '}
                {formatCurrencyIDR(plans.monthly_price_idr * 12 - annualTotal)} per year
              </p>
            )}

            {!isActive && (
              <button
                onClick={handleUpgrade}
                disabled={upgrading}
                className="mt-6 w-full bg-primary-600 hover:bg-primary-700 disabled:opacity-60 text-white font-semibold py-3 px-6 rounded-lg transition-colors"
              >
                {upgrading ? 'Processing...' : buttonLabel}
              </button>
            )}
          </div>

          {/* Features */}
          <div className="bg-white border border-gray-200 rounded-xl p-6">
            <h3 className="text-base font-semibold text-gray-900 mb-4">Everything included</h3>
            <ul className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {FEATURES.map(feature => (
                <li key={feature} className="flex items-center gap-2 text-sm text-gray-700">
                  <svg
                    className="w-4 h-4 text-green-500 flex-shrink-0"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                  {feature}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </DashboardLayout>
    </ProtectedRoute>
  );
}
