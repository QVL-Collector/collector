import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { defaultLanguage, toSupportedLanguage, type SupportedLanguage } from './languages';

export type LanguageState = {
  readonly language: SupportedLanguage;
  readonly changeLanguage: (language: SupportedLanguage) => void;
};

export function useLanguage(): LanguageState {
  const { i18n } = useTranslation();
  const language = toSupportedLanguage(i18n.resolvedLanguage) ?? defaultLanguage;
  const changeLanguage = useCallback(
    (next: SupportedLanguage) => {
      void i18n.changeLanguage(next);
    },
    [i18n],
  );
  return { language, changeLanguage };
}
