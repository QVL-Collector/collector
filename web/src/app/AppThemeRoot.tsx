import type { ReactNode } from 'react';
import { useDocumentLanguage } from '../i18n/useDocumentLanguage';
import { AppThemeProvider } from '../ui/theme/AppThemeProvider';
import { useAppTheme } from './useAppTheme';

type AppThemeRootProps = {
  readonly children: ReactNode;
};

export function AppThemeRoot({ children }: AppThemeRootProps) {
  const theme = useAppTheme();
  useDocumentLanguage();
  return <AppThemeProvider theme={theme}>{children}</AppThemeProvider>;
}
