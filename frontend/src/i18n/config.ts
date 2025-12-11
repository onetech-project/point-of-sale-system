import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import commonEn from './locales/en/common.json';
import commonId from './locales/id/common.json';
import authEn from './locales/en/auth.json';
import authId from './locales/id/auth.json';
import productsEn from './locales/en/products.json';
import productsId from './locales/id/products.json';
import notificationsEn from './locales/en/notifications.json';
import notificationsId from './locales/id/notifications.json';

const resources = {
  en: {
    common: commonEn,
    auth: authEn,
    products: productsEn,
    notifications: notificationsEn,
  },
  id: {
    common: commonId,
    auth: authId,
    products: productsId,
    notifications: notificationsId,
  },
};

i18n.use(initReactI18next).init({
  resources,
  lng: typeof window !== 'undefined' ? localStorage.getItem('locale') || 'en' : 'en',
  fallbackLng: 'en',
  interpolation: {
    escapeValue: false,
  },
  react: {
    useSuspense: false,
  },
});

export default i18n;
