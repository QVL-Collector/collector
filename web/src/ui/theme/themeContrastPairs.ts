import type { themeVariables } from './themeVariables';

export type ThemeColor = keyof typeof themeVariables;

export type ContrastRequirement = 'text' | 'nonText';

export type ThemeContrastPair = {
  readonly foreground: ThemeColor;
  readonly background: ThemeColor;
  readonly requirement: ContrastRequirement;
};

export const minimumContrastRatios: Readonly<Record<ContrastRequirement, number>> = {
  text: 4.5,
  nonText: 3,
};

const surfaces = ['surfaceBackground', 'surfacePaper'] as const satisfies readonly ThemeColor[];

const textOnSurfaces = [
  'textPrimary',
  'textSecondary',
  'brandPrimary',
  'brandPrimaryDark',
  'brandAccent',
  'brandAccentDark',
  'error',
] as const satisfies readonly ThemeColor[];

const filledBackgrounds = [
  'brandPrimary',
  'brandPrimaryDark',
  'brandPrimaryLight',
  'brandAccent',
  'brandAccentDark',
  'error',
] as const satisfies readonly ThemeColor[];

function pair(
  foreground: ThemeColor,
  background: ThemeColor,
  requirement: ContrastRequirement,
): ThemeContrastPair {
  return { foreground, background, requirement };
}

export const themeContrastPairs: readonly ThemeContrastPair[] = [
  ...textOnSurfaces.flatMap((foreground) =>
    surfaces.map((background) => pair(foreground, background, 'text')),
  ),
  ...filledBackgrounds.map((background) => pair('onBrand', background, 'text')),
  ...surfaces.map((background) => pair('focusRing', background, 'nonText')),
];
