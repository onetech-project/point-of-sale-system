'use client';

import React from 'react';
import Link from 'next/link';
import { useTranslation } from '@/i18n/provider';

const TRIAL_DAYS = 7;

export default function CtaSection() {
  const { t } = useTranslation(['landing']);

  return (
    <section className="py-20 px-4 sm:px-6 lg:px-8 bg-gradient-to-r from-primary-600 to-primary-800">
      <div className="max-w-4xl mx-auto text-center">
        <h2 className="text-4xl md:text-5xl font-bold text-white mb-4">
          {t('landing.cta.title')}
        </h2>
        <p className="text-xl text-primary-100 mb-8 max-w-2xl mx-auto">
          {t('landing.cta.description', { days: TRIAL_DAYS })}
        </p>

        <Link
          href="/signup"
          className="inline-block px-10 py-4 bg-white text-primary-600 font-bold text-lg rounded-lg hover:bg-primary-50 transition-colors shadow-lg hover:shadow-xl"
        >
          {t('landing.cta.primaryCta')}
        </Link>

        <p className="text-primary-100 text-sm mt-6">
          {t('landing.cta.loginPrompt')}{' '}
          <Link href="/login" className="text-white font-semibold hover:underline">
            {t('landing.cta.loginLink')}
          </Link>
        </p>
      </div>
    </section>
  );
}
