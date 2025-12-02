'use client';

import { createContext, useContext, ReactNode } from 'react';
import { useTranslation as useI18nTranslation, I18nextProvider } from 'react-i18next';
import i18n from './config';

const I18nContext = createContext(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  return <I18nextProvider i18n={i18n}>{children}</I18nextProvider>;
}

export function useTranslation(namespace?: string | string[]) {
  return useI18nTranslation(namespace);
}
