'use client';

import React, { useEffect, useRef, useState } from 'react';
import { Onborda, OnbordaProvider as BaseOnbordaProvider, useOnborda } from 'onborda';
import type { CardComponentProps } from 'onborda';
import { usePathname } from 'next/navigation';
import { useAuth } from '@/store/auth';
import { ONBOARDING_TOUR_VERSION, getTourKeyForRole, roleOnboardingTours } from './tours';
import { onboardingService } from '@/services/onboarding';

interface OnboardingServiceClient {
  getProgress: typeof onboardingService.getProgress;
  complete: typeof onboardingService.complete;
}

export function OnboardingProvider({ children }: { children: React.ReactNode }) {
  return (
    <BaseOnbordaProvider>
      <Onborda
        steps={roleOnboardingTours}
        cardComponent={OnboardingCard}
        shadowRgb="17, 24, 39"
        shadowOpacity="0.35"
      >
        <OnboardingController />
        {children}
      </Onborda>
    </BaseOnbordaProvider>
  );
}

export function OnboardingController({
  service = onboardingService,
}: {
  service?: Pick<OnboardingServiceClient, 'getProgress'>;
}) {
  const pathname = usePathname();
  const { user, isAuthenticated, isLoading } = useAuth();
  const { startOnborda } = useOnborda();
  const startedTourRef = useRef<string | null>(null);

  useEffect(() => {
    if (isLoading || !isAuthenticated || pathname !== '/dashboard') return;

    const tourKey = getTourKeyForRole(user?.role);
    if (!tourKey || startedTourRef.current === tourKey) return;

    let cancelled = false;
    void service
      .getProgress(tourKey, ONBOARDING_TOUR_VERSION)
      .then(progress => {
        if (cancelled || progress.completed) return;
        startedTourRef.current = tourKey;
        startOnborda(tourKey);
      })
      .catch(error => {
        console.error('Failed to load onboarding progress:', error);
      });

    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, isLoading, pathname, service, startOnborda, user?.role]);

  return null;
}

function OnboardingCard({
  step,
  currentStep,
  totalSteps,
  nextStep,
  prevStep,
  arrow,
}: CardComponentProps) {
  const { closeOnborda, currentTour } = useOnborda();
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const isLastStep = currentStep >= totalSteps - 1;

  const completeAndClose = async () => {
    if (!currentTour) return;
    setIsSaving(true);
    setError(null);
    try {
      await onboardingService.complete(currentTour, ONBOARDING_TOUR_VERSION);
      closeOnborda();
    } catch (err) {
      console.error('Failed to complete onboarding:', err);
      setError('Could not save onboarding progress. Please try again.');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="relative w-[min(20rem,calc(100vw-2rem))] rounded-lg border border-gray-200 bg-white p-4 shadow-xl">
      {arrow}
      <div className="space-y-3">
        <div>
          <p className="text-xs font-medium uppercase text-primary-600">
            Step {currentStep + 1} of {totalSteps}
          </p>
          <h2 className="mt-1 text-base font-semibold text-gray-900">{step.title}</h2>
        </div>
        <div className="text-sm leading-6 text-gray-600">{step.content}</div>
        {error && (
          <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {error}
          </p>
        )}
        <div className="flex items-center justify-between gap-2">
          <button
            type="button"
            onClick={completeAndClose}
            disabled={isSaving}
            className="text-sm font-medium text-gray-500 hover:text-gray-700 disabled:opacity-50"
          >
            Skip
          </button>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={prevStep}
              disabled={currentStep === 0 || isSaving}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Back
            </button>
            <button
              type="button"
              onClick={isLastStep ? completeAndClose : nextStep}
              disabled={isSaving}
              className="rounded-md bg-primary-600 px-3 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isSaving ? 'Saving...' : isLastStep ? 'Finish' : 'Next'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
