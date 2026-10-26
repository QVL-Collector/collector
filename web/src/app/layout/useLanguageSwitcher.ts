import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { SupportedLanguage } from '../../i18n/languages';
import { useLanguage } from '../../i18n/useLanguage';
import type { LanguageOption } from '../../ui/components/LanguageSwitcher';

export type LanguageSwitcherModel = {
  readonly label: string;
  readonly language: SupportedLanguage;
  readonly options: readonly LanguageOption<SupportedLanguage>[];
  readonly changeLanguage: (language: SupportedLanguage) => void;
};

export function useLanguageSwitcher(): LanguageSwitcherModel {
  const { t } = useTranslation();
  const { language, changeLanguage } = useLanguage();
  const options = useMemo<readonly LanguageOption<SupportedLanguage>[]>(
    () => [
      { value: 'fr', shortLabel: t('app.language.frShort'), fullLabel: t('app.language.frFull') },
      { value: 'en', shortLabel: t('app.language.enShort'), fullLabel: t('app.language.enFull') },
    ],
    [t],
  );
  return { label: t('app.language.label'), language, options, changeLanguage };
}
