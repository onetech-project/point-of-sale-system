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
import consentEn from './locales/en/consent.json';
import consentId from './locales/id/consent.json';
import privacyEn from './locales/en/privacy.json';
import privacyId from './locales/id/privacy.json';
import guestDataEn from './locales/en/guest_data.json';
import guestDataId from './locales/id/guest_data.json';
import privacySettingsEn from './locales/en/privacy_settings.json';
import privacySettingsId from './locales/id/privacy_settings.json';
import AuditEn from './locales/en/audit.json';
import AuditId from './locales/id/audit.json';

const resources = {
  en: {
    common: commonEn,
    auth: authEn,
    products: productsEn,
    notifications: notificationsEn,
    consent: consentEn,
    privacy: privacyEn,
    guest_data: guestDataEn,
    privacy_settings: privacySettingsEn,
    audit: AuditEn,
  },
  id: {
    common: commonId,
    auth: authId,
    products: productsId,
    notifications: notificationsId,
    consent: consentId,
    privacy: privacyId,
    guest_data: guestDataId,
    privacy_settings: privacySettingsId,
    audit: AuditId,
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
