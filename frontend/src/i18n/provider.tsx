'use client';

import { createContext, useEffect, ReactNode } from 'react';
import { useTranslation as useI18nTranslation, I18nextProvider } from 'react-i18next';
import i18n from './config';

const I18nContext = createContext(null);
const supportedLocales = ['en', 'id'];

export function I18nProvider({ children }: { children: ReactNode }) {
  useEffect(() => {
    const storedLocale = localStorage.getItem('locale');

    if (storedLocale && supportedLocales.includes(storedLocale) && storedLocale !== i18n.language) {
      i18n.changeLanguage(storedLocale);
    }
  }, []);

  return <I18nextProvider i18n={i18n}>{children}</I18nextProvider>;
}

export function useTranslation(namespace?: string | string[]) {
  return useI18nTranslation(namespace);
}
