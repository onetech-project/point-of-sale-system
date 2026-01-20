'use client';

import React from 'react';
import Link from 'next/link';
import { useTranslation } from '@/i18n/provider';

export default function Footer() {
  const { t } = useTranslation(['common']);
  const currentYear = new Date().getFullYear();

  return (
    <footer className="bg-gray-50 border-t border-gray-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
          <div className="col-span-1 md:col-span-2">
            <div className="flex items-center space-x-2 mb-4">
              <div className="w-8 h-8 bg-primary-600 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-xl">P</span>
              </div>
              <span className="text-xl font-bold text-gray-900">{t('common.appName')}</span>
            </div>
            <p className="text-gray-600 text-sm max-w-md">{t('common.footer.description')}</p>
          </div>

          <div>
            <h3 className="text-sm font-semibold text-gray-900 uppercase tracking-wider mb-4">
              {t('common.footer.product')}
            </h3>
            <ul className="space-y-2">
              <li>
                <Link
                  href="/features"
                  className="text-gray-600 hover:text-primary-600 text-sm transition-colors"
                >
                  {t('common.footer.features')}
                </Link>
              </li>
              <li>
                <Link
                  href="/pricing"
                  className="text-gray-600 hover:text-primary-600 text-sm transition-colors"
                >
                  {t('common.footer.pricing')}
                </Link>
              </li>
              <li>
                <Link
                  href="/docs"
                  className="text-gray-600 hover:text-primary-600 text-sm transition-colors"
                >
                  {t('common.footer.documentation')}
                </Link>
              </li>
            </ul>
          </div>

          <div>
            <h3 className="text-sm font-semibold text-gray-900 uppercase tracking-wider mb-4">
              {t('common.footer.support')}
            </h3>
            <ul className="space-y-2">
              <li>
                <Link
                  href="/help"
                  className="text-gray-600 hover:text-primary-600 text-sm transition-colors"
                >
                  {t('common.footer.helpCenter')}
                </Link>
              </li>
              <li>
                <Link
                  href="/contact"
                  className="text-gray-600 hover:text-primary-600 text-sm transition-colors"
                >
                  {t('common.footer.contact')}
                </Link>
              </li>
              <li>
                <Link
                  href="/privacy-policy"
                  className="text-gray-600 hover:text-primary-600 text-sm transition-colors"
                >
                  {t('common.footer.privacy')}
                </Link>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-8 pt-8 border-t border-gray-200">
          <p className="text-gray-500 text-sm text-center">
            &copy; {currentYear} {t('common.appName')}. {t('common.footer.rights')}
          </p>
        </div>
      </div>
    </footer>
  );
}
