'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import ProtectedRoute from '@/components/auth/ProtectedRoute';
import Modal from '@/components/ui/Modal';
import DashboardLayout from '../../src/components/layout/DashboardLayout';
import {
  billingService,
  BillingInvoice,
  BillingSubscriptionUI,
  PublicPlans,
} from '@/services/billing';
import { useAuth } from '@/store/auth';
import { useSubscription } from '@/store/subscription';
import { redirectToPayment } from '@/utils/paymentRedirect';
import { ROLES } from '@/constants/roles';

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

function formatOptionalDateIDR(dateStr?: string): string {
  if (!dateStr) return 'the scheduled cleanup date';
  return formatDateIDR(dateStr);
}

function formatDateTimeIDR(dateStr?: string): string {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
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

function dateValue(dateStr?: string): number {
  if (!dateStr) return 0;
  const value = new Date(dateStr).getTime();
  return Number.isNaN(value) ? 0 : value;
}

function isFutureDate(dateStr?: string): boolean {
  const value = dateValue(dateStr);
  return value > Date.now();
}

function getRequestErrorMessage(err: any, fallback: string): string {
  return err?.response?.data?.error ?? err?.response?.data?.message ?? fallback;
}

function PaymentReturnNotice() {
  return (
    <div className="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-700">
      Payment returned from Midtrans. We&apos;re checking your payment status and refreshing your
      billing details.
    </div>
  );
}

function findLatestPendingInvoice(invoices: BillingInvoice[]): BillingInvoice | null {
  return (
    [...invoices]
      .filter(invoice => invoice.status === 'pending')
      .sort(
        (a, b) =>
          dateValue(b.due_at) - dateValue(a.due_at) ||
          dateValue(b.created_at) - dateValue(a.created_at)
      )[0] ?? null
  );
}

function findLatestPaidInvoice(invoices: BillingInvoice[]): BillingInvoice | null {
  return (
    [...invoices]
      .filter(invoice => invoice.status === 'paid')
      .sort(
        (a, b) =>
          dateValue(b.paid_at) - dateValue(a.paid_at) ||
          dateValue(b.created_at) - dateValue(a.created_at)
      )[0] ?? null
  );
}

function formatStatusLabel(status?: string): string {
  if (!status) return '-';
  return status
    .split('_')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

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
    cancelled: {
      label: 'Cancelled',
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
            Your subscription is in grace period. Subscribe now to keep access active and prevent
            operational-data cleanup.
          </p>
          <p className="mt-2 text-sm text-red-600">
            Operational data becomes eligible for cleanup on{' '}
            <span className="font-semibold">
              {formatOptionalDateIDR(subscription.retention_cleanup_at)}
            </span>
            .
          </p>
        </div>
      )}

      {(subscription.status === 'expired' || subscription.status === 'cancelled') && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-red-700 font-medium">
            Your account has been suspended. Please subscribe to reactivate.
          </p>
        </div>
      )}
    </div>
  );
}

function BillingSummary({
  subscription,
  latestPendingInvoice,
  latestPaidInvoice,
  payingInvoiceId,
  canManageSubscription,
  onPayInvoice,
}: {
  subscription: BillingSubscriptionUI | null;
  latestPendingInvoice: BillingInvoice | null;
  latestPaidInvoice: BillingInvoice | null;
  payingInvoiceId: string | null;
  canManageSubscription: boolean;
  onPayInvoice: (invoice: BillingInvoice) => void;
}) {
  const fallbackDueDate =
    subscription?.payment_due_at ??
    subscription?.subscription_ends_at ??
    subscription?.trial_ends_at;

  return (
    <div className="grid gap-4 md:grid-cols-3">
      <div className="min-w-0 rounded-xl border border-gray-200 bg-white p-5">
        <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">
          Subscription status
        </p>
        <p className="mt-3 text-xl font-bold text-gray-900">
          {formatStatusLabel(subscription?.subscription_status)}
        </p>
        <p className="mt-1 text-sm text-gray-500">
          {subscription?.subscription_plan ?? '-'} &middot; {subscription?.billing_cycle ?? '-'}
        </p>
      </div>

      <div className="min-w-0 rounded-xl border border-gray-200 bg-white p-5">
        <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">
          Outstanding payment
        </p>
        {latestPendingInvoice ? (
          <>
            <p className="mt-3 text-lg font-bold text-gray-900">
              {formatCurrencyIDR(latestPendingInvoice.amount_idr)}
            </p>
            <p className="mt-1 text-sm text-gray-500">
              Due {formatDateTimeIDR(latestPendingInvoice.due_at)}
            </p>
            <div className="mt-3 flex min-w-0 flex-col items-start gap-3 sm:flex-row sm:flex-wrap sm:items-center">
              <a
                href={`/subscription/invoices/${latestPendingInvoice.id}`}
                className="block max-w-full min-w-0 break-all font-mono text-sm font-medium text-primary-600 hover:underline"
              >
                {latestPendingInvoice.invoice_number}
              </a>
              {canManageSubscription && (
                <button
                  type="button"
                  onClick={() => onPayInvoice(latestPendingInvoice)}
                  disabled={payingInvoiceId === latestPendingInvoice.id}
                  className="w-full rounded-lg bg-primary-600 px-3 py-1.5 text-sm font-semibold text-white transition-colors hover:bg-primary-700 disabled:opacity-60 sm:w-auto"
                >
                  {payingInvoiceId === latestPendingInvoice.id ? 'Opening...' : 'Pay now'}
                </button>
              )}
            </div>
          </>
        ) : (
          <>
            <p className="mt-3 text-lg font-bold text-gray-900">
              {formatDateTimeIDR(fallbackDueDate)}
            </p>
            <p className="mt-2 text-sm text-gray-500">No outstanding invoice</p>
          </>
        )}
      </div>

      <div className="min-w-0 rounded-xl border border-gray-200 bg-white p-5">
        <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">
          Latest paid payment
        </p>
        {latestPaidInvoice ? (
          <>
            <p className="mt-3 text-lg font-bold text-gray-900">
              {formatCurrencyIDR(latestPaidInvoice.amount_idr)}
            </p>
            <p className="mt-1 text-sm text-gray-500">
              Paid {formatDateTimeIDR(latestPaidInvoice.paid_at ?? latestPaidInvoice.created_at)}
            </p>
            <a
              href={`/subscription/invoices/${latestPaidInvoice.id}`}
              className="mt-2 block max-w-full min-w-0 break-all font-mono text-sm font-medium text-primary-600 hover:underline"
            >
              {latestPaidInvoice.invoice_number}
            </a>
          </>
        ) : (
          <>
            <p className="mt-3 text-lg font-bold text-gray-900">-</p>
            <p className="mt-1 text-sm text-gray-500">No paid invoice yet</p>
          </>
        )}
      </div>
    </div>
  );
}

function YearlySwitchConfirmationModal({
  isOpen,
  annualTotal,
  annualSavings,
  upgrading,
  onClose,
  onConfirm,
}: {
  isOpen: boolean;
  annualTotal: number;
  annualSavings: number;
  upgrading: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  return (
    <Modal
      isOpen={isOpen}
      onClose={upgrading ? () => {} : onClose}
      title="Switch to Yearly"
      size="sm"
      showCloseButton={!upgrading}
    >
      <div className="space-y-4">
        <div>
          <p className="text-sm text-gray-600">
            Your current monthly subscription stays active until the yearly payment succeeds.
          </p>
          <p className="mt-3 text-2xl font-bold text-gray-900">{formatCurrencyIDR(annualTotal)}</p>
          <p className="mt-1 text-sm text-green-600">
            Save {formatCurrencyIDR(Math.max(0, annualSavings))} per year.
          </p>
        </div>

        <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-3 text-sm text-yellow-800">
          Midtrans payment links expire after 15 minutes. If the payment is abandoned, your monthly
          plan remains unchanged.
        </div>

        <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <button
            type="button"
            onClick={onClose}
            disabled={upgrading}
            className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-semibold text-gray-700 hover:bg-gray-50 disabled:opacity-60"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={upgrading}
            className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white hover:bg-primary-700 disabled:opacity-60"
          >
            {upgrading ? 'Processing...' : 'Confirm and Pay'}
          </button>
        </div>
      </div>
    </Modal>
  );
}

function ExpiredSubscriptionRecovery({
  subscription,
  plans,
  billingInterval,
  setBillingInterval,
  displayPrice,
  annualPriceMonthly,
  annualTotal,
  upgrading,
  error,
  showPaymentReturnNotice,
  canManageSubscription,
  onUpgrade,
  onLogout,
}: {
  subscription: BillingSubscriptionUI | null;
  plans: PublicPlans | null;
  billingInterval: BillingInterval;
  setBillingInterval: (value: BillingInterval) => void;
  displayPrice: number;
  annualPriceMonthly: number;
  annualTotal: number;
  upgrading: boolean;
  error: string | null;
  showPaymentReturnNotice: boolean;
  canManageSubscription: boolean;
  onUpgrade: () => void;
  onLogout: () => void;
}) {
  const cleanupDate = formatOptionalDateIDR(subscription?.retention_cleanup_at);
  const anonymizedDate = subscription?.data_anonymized_at
    ? formatDateIDR(subscription.data_anonymized_at)
    : null;
  const isCycleChange =
    Boolean(subscription?.billing_cycle) && subscription?.billing_cycle !== billingInterval;
  const actionLabel = isCycleChange
    ? billingInterval === 'monthly'
      ? 'Switch to Monthly'
      : 'Switch to Yearly'
    : 'Complete Subscription';

  return (
    <div className="min-h-screen bg-gray-50 px-4 py-10 sm:px-6 lg:px-8">
      <div className="mx-auto max-w-2xl space-y-6">
        <div className="rounded-xl border border-red-200 bg-white p-6 shadow-sm">
          <div className="flex items-start justify-between gap-4">
            <div>
              <span className="inline-flex rounded-full bg-red-100 px-3 py-1 text-xs font-semibold uppercase tracking-wide text-red-700">
                Suspended
              </span>
              <h1 className="mt-4 text-2xl font-bold text-gray-900">Subscription Required</h1>
              <p className="mt-2 text-sm text-gray-600">
                Your account is expired. Complete the subscription payment to reactivate access.
              </p>
            </div>
            <button
              type="button"
              onClick={onLogout}
              className="rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              Logout
            </button>
          </div>

          {error && (
            <div className="mt-5 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
              {error}
            </div>
          )}

          {showPaymentReturnNotice && (
            <div className="mt-5">
              <PaymentReturnNotice />
            </div>
          )}

          <div className="mt-6 rounded-lg border border-red-200 bg-red-50 p-4">
            <p className="text-sm font-medium text-red-800">Data retention warning</p>
            {anonymizedDate ? (
              <p className="mt-1 text-sm text-red-700">
                Operational workspace data was anonymized or deleted on {anonymizedDate}. Payment
                can reactivate the account, but cleaned operational data will not be restored.
              </p>
            ) : (
              <p className="mt-1 text-sm text-red-700">
                Operational workspace data is eligible for anonymization or deletion on{' '}
                <span className="font-semibold">{cleanupDate}</span>. Billing, payment, consent, and
                audit records are retained as historical/compliance records.
              </p>
            )}
          </div>
        </div>

        {canManageSubscription ? (
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold text-gray-900">Choose Billing</h2>
              <a
                href="/subscription/invoices"
                className="text-sm font-medium text-primary-600 hover:text-primary-500"
              >
                View invoices
              </a>
            </div>

            <div className="mt-5 grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => setBillingInterval('monthly')}
                className={`rounded-lg border px-4 py-3 text-sm font-medium transition-colors ${
                  billingInterval === 'monthly'
                    ? 'border-primary-600 bg-primary-50 text-primary-700'
                    : 'border-gray-200 text-gray-700 hover:border-gray-300'
                }`}
              >
                Monthly
              </button>
              <button
                type="button"
                onClick={() => setBillingInterval('annual')}
                className={`rounded-lg border px-4 py-3 text-sm font-medium transition-colors ${
                  billingInterval === 'annual'
                    ? 'border-primary-600 bg-primary-50 text-primary-700'
                    : 'border-gray-200 text-gray-700 hover:border-gray-300'
                }`}
              >
                Annual
                {plans && (
                  <span className="ml-2 rounded bg-green-100 px-1.5 py-0.5 text-xs font-semibold text-green-700">
                    Save {plans.annual_discount_pct}%
                  </span>
                )}
              </button>
            </div>

            <div className="mt-6 flex items-end gap-2">
              <span className="text-3xl font-bold text-gray-900">
                {formatCurrencyIDR(displayPrice)}
              </span>
              <span className="pb-1 text-gray-500">
                /{billingInterval === 'monthly' ? 'month' : 'year'}
              </span>
            </div>
            {billingInterval === 'annual' && plans && (
              <p className="mt-1 text-sm text-green-600">
                {formatCurrencyIDR(annualPriceMonthly)}/month - you save{' '}
                {formatCurrencyIDR(plans.monthly_price_idr * 12 - annualTotal)} per year
              </p>
            )}

            <button
              type="button"
              onClick={onUpgrade}
              disabled={upgrading || !plans}
              className="mt-6 w-full rounded-lg bg-primary-600 px-6 py-3 font-semibold text-white transition-colors hover:bg-primary-700 disabled:opacity-60"
            >
              {upgrading ? 'Processing...' : actionLabel}
            </button>
          </div>
        ) : (
          <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
            <h2 className="text-base font-semibold text-gray-900">Subscription Recovery</h2>
            <p className="mt-2 text-sm text-gray-600">
              Subscription payments and plan changes are managed by owners and managers.
            </p>
            <a
              href="/subscription/invoices"
              className="mt-4 inline-flex text-sm font-medium text-primary-600 hover:text-primary-500"
            >
              View invoices
            </a>
          </div>
        )}
      </div>
    </div>
  );
}

export default function SubscriptionPage() {
  const router = useRouter();
  const { logout, user } = useAuth();
  const {
    subscription,
    refreshSubscription,
    invalidateSubscription,
    isLoading: subscriptionLoading,
  } = useSubscription();
  const [expiredReason] = useState(() => {
    if (typeof window === 'undefined') return false;
    return new URLSearchParams(window.location.search).get('reason') === 'expired';
  });
  const [returnedFromMidtrans] = useState(() => {
    if (typeof window === 'undefined') return false;
    return new URLSearchParams(window.location.search).get('payment_return') === 'midtrans';
  });
  const [plans, setPlans] = useState<PublicPlans | null>(null);
  const [invoices, setInvoices] = useState<BillingInvoice[]>([]);
  const [billingInterval, setBillingInterval] = useState<BillingInterval>('monthly');
  const [loading, setLoading] = useState(true);
  const [upgrading, setUpgrading] = useState(false);
  const [payingInvoiceId, setPayingInvoiceId] = useState<string | null>(null);
  const [showYearlyConfirm, setShowYearlyConfirm] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const canManageSubscription = user?.role === ROLES.OWNER || user?.role === ROLES.MANAGER;

  useEffect(() => {
    Promise.all([
      refreshSubscription({ force: true }),
      billingService.getPublicPlans(),
      billingService.getInvoices(),
    ])
      .then(([sub, p, inv]) => {
        setPlans(p);
        setInvoices(inv);
        setBillingInterval(sub?.billing_interval ?? 'monthly');
      })
      .catch(err => {
        console.error('Failed to load subscription:', err);
        setError('Failed to load subscription information.');
      })
      .finally(() => setLoading(false));
  }, [refreshSubscription]);

  const isActive =
    subscription?.status === 'active' || subscription?.subscription_status === 'active';
  const isExpired =
    subscription?.status === 'expired' ||
    subscription?.subscription_status === 'expired' ||
    subscription?.status === 'cancelled' ||
    subscription?.subscription_status === 'cancelled';
  const annualPriceMonthly = plans
    ? Math.round((plans.monthly_price_idr * 12 * (1 - plans.annual_discount_pct / 100)) / 12)
    : 0;
  const annualTotal = annualPriceMonthly * 12;
  const annualSavings = plans ? plans.monthly_price_idr * 12 - annualTotal : 0;

  const displayPrice =
    billingInterval === 'monthly' ? (plans?.monthly_price_idr ?? 0) : annualTotal;
  const currentBillingCycle =
    subscription?.billing_cycle ?? subscription?.billing_interval ?? 'monthly';
  const selectedCycleDiffers = Boolean(subscription) && currentBillingCycle !== billingInterval;
  const isYearlySwitchAction = selectedCycleDiffers && billingInterval === 'annual';
  const monthlyRevertLocked =
    currentBillingCycle === 'annual' && isFutureDate(subscription?.subscription_ends_at);
  const yearlyMonthlyLocked = monthlyRevertLocked && billingInterval === 'monthly';
  const billingActionLabel = selectedCycleDiffers
    ? billingInterval === 'annual'
      ? 'Switch to Yearly'
      : 'Switch to Monthly'
    : isActive
      ? 'Current billing cycle'
      : 'Start Subscription';

  const runSubscriptionPayment = async () => {
    if (!canManageSubscription) {
      setError('Subscription payments and plan changes are managed by owners and managers.');
      return;
    }
    try {
      setUpgrading(true);
      setError(null);
      const shouldSwitchCycle = subscription ? currentBillingCycle !== billingInterval : false;
      const result = shouldSwitchCycle
        ? await billingService.switchBillingCycle(billingInterval)
        : await billingService.upgradeSubscription(billingInterval);
      invalidateSubscription();
      if (result.payment_url) {
        redirectToPayment(result.payment_url);
      }
    } catch (err: any) {
      console.error('Upgrade failed:', err);
      setError(getRequestErrorMessage(err, 'Failed to initiate subscription. Please try again.'));
    } finally {
      setUpgrading(false);
    }
  };

  const handleUpgrade = () => {
    if (!canManageSubscription) {
      setError('Subscription payments and plan changes are managed by owners and managers.');
      return;
    }
    if (isYearlySwitchAction) {
      setShowYearlyConfirm(true);
      return;
    }
    runSubscriptionPayment();
  };

  const handleConfirmYearlySwitch = () => {
    setShowYearlyConfirm(false);
    runSubscriptionPayment();
  };

  const handlePayInvoice = async (invoice: BillingInvoice) => {
    if (!canManageSubscription) {
      setError('Subscription payments are managed by owners and managers.');
      return;
    }
    try {
      setPayingInvoiceId(invoice.id);
      setError(null);
      const result = await billingService.initiatePayment(invoice.id);
      invalidateSubscription();
      if (result.payment_url) {
        redirectToPayment(result.payment_url);
      }
    } catch (err: any) {
      console.error('Payment initiation failed:', err);
      setError(getRequestErrorMessage(err, 'Failed to open payment. Please try again.'));
    } finally {
      setPayingInvoiceId(null);
    }
  };

  const handleLogout = async () => {
    await logout();
    router.replace('/login');
  };

  const latestPendingInvoice = findLatestPendingInvoice(invoices);
  const latestPaidInvoice = findLatestPaidInvoice(invoices);

  if (loading || subscriptionLoading) {
    if (expiredReason) {
      return (
        <ProtectedRoute>
          <div className="min-h-screen flex items-center justify-center bg-gray-50">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600"></div>
          </div>
        </ProtectedRoute>
      );
    }

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

  if (isExpired || (expiredReason && !subscription)) {
    return (
      <ProtectedRoute>
        <>
          <ExpiredSubscriptionRecovery
            subscription={subscription}
            plans={plans}
            billingInterval={billingInterval}
            setBillingInterval={setBillingInterval}
            displayPrice={displayPrice}
            annualPriceMonthly={annualPriceMonthly}
            annualTotal={annualTotal}
            upgrading={upgrading}
            error={error}
            showPaymentReturnNotice={returnedFromMidtrans}
            canManageSubscription={canManageSubscription}
            onUpgrade={handleUpgrade}
            onLogout={handleLogout}
          />
          {canManageSubscription && (
            <YearlySwitchConfirmationModal
              isOpen={showYearlyConfirm}
              annualTotal={annualTotal}
              annualSavings={annualSavings}
              upgrading={upgrading}
              onClose={() => setShowYearlyConfirm(false)}
              onConfirm={handleConfirmYearlySwitch}
            />
          )}
        </>
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

          {returnedFromMidtrans && <PaymentReturnNotice />}

          {subscription && <StatusCard subscription={subscription} />}

          <BillingSummary
            subscription={subscription}
            latestPendingInvoice={latestPendingInvoice}
            latestPaidInvoice={latestPaidInvoice}
            payingInvoiceId={payingInvoiceId}
            canManageSubscription={canManageSubscription}
            onPayInvoice={handlePayInvoice}
          />

          {subscription?.data_anonymized_at && (
            <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4 text-sm text-yellow-800">
              Operational workspace data cleanup ran on{' '}
              <span className="font-semibold">
                {formatDateIDR(subscription.data_anonymized_at)}
              </span>
              . The account can remain active after payment, but anonymized or deleted operational
              data is not restored.
            </div>
          )}

          {canManageSubscription ? (
            <div className="bg-white border border-gray-200 rounded-xl p-6">
              <h3 className="text-base font-semibold text-gray-900 mb-4">Billing</h3>
              <div className="flex gap-3">
                <button
                  onClick={() => {
                    if (!monthlyRevertLocked) setBillingInterval('monthly');
                  }}
                  disabled={monthlyRevertLocked}
                  className={`flex-1 py-3 px-4 rounded-lg border text-sm font-medium transition-colors ${
                    billingInterval === 'monthly'
                      ? 'border-primary-600 bg-primary-50 text-primary-700'
                      : 'border-gray-200 text-gray-600 hover:border-gray-300'
                  } disabled:cursor-not-allowed disabled:border-gray-100 disabled:bg-gray-50 disabled:text-gray-400`}
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
              {monthlyRevertLocked && (
                <p className="mt-3 text-sm text-gray-500">
                  Monthly billing is available after{' '}
                  {subscription?.subscription_ends_at
                    ? formatDateIDR(subscription.subscription_ends_at)
                    : '-'}
                  .
                </p>
              )}

              <button
                onClick={handleUpgrade}
                disabled={
                  upgrading || !plans || yearlyMonthlyLocked || (isActive && !selectedCycleDiffers)
                }
                className="mt-6 w-full bg-primary-600 hover:bg-primary-700 disabled:opacity-60 text-white font-semibold py-3 px-6 rounded-lg transition-colors"
              >
                {upgrading ? 'Processing...' : billingActionLabel}
              </button>
            </div>
          ) : (
            <div className="rounded-xl border border-gray-200 bg-white p-6 text-sm text-gray-600">
              Subscription payments and plan changes are managed by owners and managers.
            </div>
          )}

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
      {canManageSubscription && (
        <YearlySwitchConfirmationModal
          isOpen={showYearlyConfirm}
          annualTotal={annualTotal}
          annualSavings={annualSavings}
          upgrading={upgrading}
          onClose={() => setShowYearlyConfirm(false)}
          onConfirm={handleConfirmYearlySwitch}
        />
      )}
    </ProtectedRoute>
  );
}
