import { screen, within } from '@testing-library/react';
import { expect, vi } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';

const panelTitle: Readonly<Record<SupportedLanguage, string>> = {
  fr: 'État de l’annonce',
  en: 'Listing status',
};

export function statePanel(language: SupportedLanguage = 'fr'): HTMLElement {
  return screen.getByRole('region', { name: panelTitle[language] });
}

export async function expectStateShown(
  state: string,
  language: SupportedLanguage = 'fr',
  timeout = 5000,
): Promise<void> {
  await vi.waitFor(
    () => {
      expect(within(statePanel(language)).getByText(state)).toBeInTheDocument();
    },
    { timeout, interval: 100 },
  );
}

export function submitButtonIsUsable(name: string): boolean {
  const button = screen.queryByRole('button', { name });
  return (
    button !== null &&
    !button.hasAttribute('disabled') &&
    button.getAttribute('aria-disabled') !== 'true'
  );
}

export function liveRegionsText(): string {
  return [...screen.queryAllByRole('status'), ...screen.queryAllByRole('alert')]
    .map((region) => region.textContent)
    .join(' | ');
}
