import { describe, expect, it } from 'vitest';
import {
  minimumContrastRatios,
  themeContrastPairs,
  type ThemeContrastPair,
} from '../../ui/theme/themeContrastPairs';
import { themeVariables } from '../../ui/theme/themeVariables';

const wcagTextMinimum = 4.5;
const wcagNonTextMinimum = 3;

function channelToLinear(channel: number): number {
  const srgb = channel / 255;
  return srgb <= 0.04045 ? srgb / 12.92 : ((srgb + 0.055) / 1.055) ** 2.4;
}

function hexToChannels(hex: string): [number, number, number] {
  const match = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex);
  if (match === null) {
    throw new Error(`Couleur non hexadécimale : ${hex}`);
  }
  return [
    Number.parseInt(match[1] ?? '', 16),
    Number.parseInt(match[2] ?? '', 16),
    Number.parseInt(match[3] ?? '', 16),
  ];
}

function luminance(hex: string): number {
  const [red, green, blue] = hexToChannels(hex).map(channelToLinear);
  return 0.2126 * (red ?? 0) + 0.7152 * (green ?? 0) + 0.0722 * (blue ?? 0);
}

function referenceContrastRatio(first: string, second: string): number {
  const [lighter, darker] = [luminance(first), luminance(second)].sort((a, b) => b - a);
  return ((lighter ?? 0) + 0.05) / ((darker ?? 0) + 0.05);
}

function wcagMinimum(pair: ThemeContrastPair): number {
  return pair.requirement === 'text' ? wcagTextMinimum : wcagNonTextMinimum;
}

function describePair(pair: ThemeContrastPair): string {
  return `${pair.foreground} sur ${pair.background} (${pair.requirement})`;
}

describe('US-025-AC4 implémentation de référence du rapport de contraste WCAG', () => {
  it.each([
    ['#000000', '#FFFFFF', 21],
    ['#FFFFFF', '#FFFFFF', 1],
    ['#777777', '#FFFFFF', 4.48],
    ['#767676', '#FFFFFF', 4.54],
  ])(
    'Given %s et %s, When le rapport est calculé, Then il vaut %s',
    (foreground, background, expected) => {
      expect(referenceContrastRatio(foreground, background)).toBeCloseTo(expected, 2);
    },
  );

  it('Given deux couleurs, When l’ordre est inversé, Then le rapport est identique', () => {
    expect(referenceContrastRatio('#1F4E5F', '#F7F4EF')).toBe(
      referenceContrastRatio('#F7F4EF', '#1F4E5F'),
    );
  });
});

describe('US-025-AC4 contrastes des couples de couleurs du thème', () => {
  it('Given les seuils déclarés, When ils sont comparés à WCAG 2.1 AA, Then texte 4,5:1 et non-texte 3:1', () => {
    expect(minimumContrastRatios.text).toBe(wcagTextMinimum);
    expect(minimumContrastRatios.nonText).toBe(wcagNonTextMinimum);
  });

  it('Given la liste des couples, When elle est lue, Then elle couvre le texte, le texte sur fond de marque et l’anneau de focus', () => {
    expect(themeContrastPairs.length).toBeGreaterThan(0);
    expect(themeContrastPairs.some((pair) => pair.requirement === 'text')).toBe(true);
    expect(themeContrastPairs.some((pair) => pair.foreground === 'focusRing')).toBe(true);
    expect(themeContrastPairs.some((pair) => pair.foreground === 'onBrand')).toBe(true);
  });

  it.each(themeContrastPairs.map((pair) => [describePair(pair), pair] as const))(
    'Given le couple %s, When son contraste est calculé, Then il atteint le seuil WCAG',
    (_, pair) => {
      const ratio = referenceContrastRatio(
        themeVariables[pair.foreground],
        themeVariables[pair.background],
      );

      expect(ratio).toBeGreaterThanOrEqual(wcagMinimum(pair));
    },
  );
});
