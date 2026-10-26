import { enUS, frFR, type Localization } from '@mui/material/locale';
import type { Theme } from '@mui/material/styles';
import { useMemo } from 'react';
import type { SupportedLanguage } from '../i18n/languages';
import { useLanguage } from '../i18n/useLanguage';
import { createAppTheme } from '../ui/theme/createAppTheme';
import { usePrefersReducedMotion } from '../ui/hooks/usePrefersReducedMotion';

const muiLocalizations: Readonly<Record<SupportedLanguage, Localization>> = {
  fr: frFR,
  en: enUS,
};

export function useAppTheme(): Theme {
  const { language } = useLanguage();
  const reducedMotion = usePrefersReducedMotion();
  return useMemo(
    () => createAppTheme({ localization: muiLocalizations[language], reducedMotion }),
    [language, reducedMotion],
  );
}
