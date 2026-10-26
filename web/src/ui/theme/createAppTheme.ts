import type { Localization } from '@mui/material/locale';
import { createTheme, type Theme, type ThemeOptions } from '@mui/material/styles';
import { themeVariables } from './themeVariables';

export type AppThemeSettings = {
  readonly localization: Localization;
  readonly reducedMotion: boolean;
};

const breakpointsReference = createTheme();
const fromDesktop = breakpointsReference.breakpoints.up('md');

const palette: ThemeOptions['palette'] = {
  mode: 'light',
  primary: {
    main: themeVariables.brandPrimary,
    dark: themeVariables.brandPrimaryDark,
    light: themeVariables.brandPrimaryLight,
    contrastText: themeVariables.onBrand,
  },
  secondary: {
    main: themeVariables.brandAccent,
    dark: themeVariables.brandAccentDark,
    light: themeVariables.brandAccentLight,
    contrastText: themeVariables.onBrand,
  },
  error: { main: themeVariables.error, contrastText: themeVariables.onBrand },
  background: {
    default: themeVariables.surfaceBackground,
    paper: themeVariables.surfacePaper,
  },
  text: {
    primary: themeVariables.textPrimary,
    secondary: themeVariables.textSecondary,
  },
  divider: themeVariables.divider,
};

const typography: ThemeOptions['typography'] = {
  fontFamily: 'Roboto, system-ui, -apple-system, "Segoe UI", sans-serif',
  h1: {
    fontSize: '2rem',
    fontWeight: 700,
    lineHeight: 1.15,
    letterSpacing: '-0.02em',
    [fromDesktop]: { fontSize: '2.75rem' },
  },
  h2: { fontSize: '1.5rem', fontWeight: 700, lineHeight: 1.25 },
  h6: { fontSize: '1.0625rem', fontWeight: 500, lineHeight: 1.35 },
  subtitle1: { fontSize: '1.125rem', fontWeight: 700, lineHeight: 1.3 },
  body1: { fontSize: '1rem', lineHeight: 1.6 },
  button: { textTransform: 'none', fontWeight: 500 },
};

function components(reducedMotion: boolean): ThemeOptions['components'] {
  return {
    MuiCssBaseline: {
      styleOverrides: {
        ':focus-visible': {
          outline: `0.1875rem solid ${themeVariables.focusRing}`,
          outlineOffset: '0.125rem',
        },
      },
    },
    MuiButtonBase: {
      defaultProps: { disableRipple: reducedMotion },
    },
    MuiButton: {
      defaultProps: { disableElevation: true },
      styleOverrides: { root: { borderRadius: '62.5rem', paddingInline: '1.25rem' } },
    },
    MuiCard: {
      styleOverrides: { root: { borderRadius: '1rem' } },
    },
    MuiChip: {
      styleOverrides: { root: { fontWeight: 500 } },
    },
    MuiAlert: {
      styleOverrides: { root: { borderRadius: '1rem' } },
    },
    MuiSkeleton: {
      defaultProps: { animation: reducedMotion ? false : 'wave' },
    },
  };
}

function transitions(reducedMotion: boolean): ThemeOptions['transitions'] {
  return reducedMotion ? { create: () => 'none' } : {};
}

export function createAppTheme({ localization, reducedMotion }: AppThemeSettings): Theme {
  return createTheme(
    {
      palette,
      typography,
      components: components(reducedMotion),
      transitions: transitions(reducedMotion),
    },
    localization,
  );
}
