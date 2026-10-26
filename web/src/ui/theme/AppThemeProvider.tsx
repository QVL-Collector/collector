import CssBaseline from '@mui/material/CssBaseline';
import { ThemeProvider, type Theme } from '@mui/material/styles';
import type { ReactNode } from 'react';

type AppThemeProviderProps = {
  readonly theme: Theme;
  readonly children: ReactNode;
};

export function AppThemeProvider({ theme, children }: AppThemeProviderProps) {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      {children}
    </ThemeProvider>
  );
}
