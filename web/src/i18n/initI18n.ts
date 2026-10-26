import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { defaultLanguage, supportedLanguages, type SupportedLanguage } from './languages';
import { en } from './locales/en';
import { fr } from './locales/fr';

export async function initI18n(language: SupportedLanguage): Promise<void> {
  await i18n.use(initReactI18next).init({
    resources: {
      fr: { translation: fr },
      en: { translation: en },
    },
    lng: language,
    fallbackLng: defaultLanguage,
    supportedLngs: [...supportedLanguages],
    interpolation: { escapeValue: false },
  });
}
