import { screen, within } from '@testing-library/react';
import userEvent, { type UserEvent } from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { openCollector, sellerA, signInAs } from '../US-008/identityProviderHarness';

type MotionPreference = 'reduce' | 'no-preference';

function preferMotion(preference: MotionPreference): void {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: (query: string): MediaQueryList => ({
      matches: query.includes('prefers-reduced-motion') && preference === 'reduce',
      media: query,
      onchange: null,
      addListener: () => undefined,
      removeListener: () => undefined,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      dispatchEvent: () => false,
    }),
  });
}

async function tabUntilFocused(user: UserEvent, target: HTMLElement, maxTabs = 40): Promise<void> {
  for (let tabs = 0; tabs < maxTabs && document.activeElement !== target; tabs += 1) {
    await user.tab();
  }
  expect(target).toHaveFocus();
}

const visibleOutlineStyles = new Set(['solid', 'dashed', 'dotted', 'double', 'auto']);

function lengthInPixels(token: string): number {
  const match = /^([\d.]+)(rem|px)$/.exec(token);
  if (match === null) {
    return 0;
  }
  return Number(match[1]) * (match[2] === 'rem' ? 16 : 1);
}

function outlineTokens(element: HTMLElement): string[] {
  const style = window.getComputedStyle(element);
  return [style.outline, style.outlineStyle, style.outlineWidth]
    .join(' ')
    .split(/\s+/)
    .filter((token) => token !== '');
}

function expectNoFocusRing(element: HTMLElement): void {
  expect(outlineTokens(element).some((token) => visibleOutlineStyles.has(token))).toBe(false);
}

function expectVisibleFocusRing(element: HTMLElement): void {
  const style = window.getComputedStyle(element);
  const tokens = outlineTokens(element);
  const description = `${element.tagName} outline="${style.outline}"`;

  expect(
    tokens.some((token) => visibleOutlineStyles.has(token)),
    description,
  ).toBe(true);
  expect(Math.max(0, ...tokens.map(lengthInPixels)), description).toBeGreaterThanOrEqual(2);
}

const preferences: readonly MotionPreference[] = ['reduce', 'no-preference'];

describe('US-025-AC5 focus visible au clavier, mouvements réduits compris', () => {
  it.each(preferences)(
    'Given la préférence de mouvement %s, When le visiteur atteint « Se connecter » au clavier, Then un anneau de focus est visible',
    async (preference) => {
      const user = userEvent.setup();
      preferMotion(preference);
      await openCollector('/');
      const signIn = await screen.findByRole('button', { name: 'Se connecter' });
      expectNoFocusRing(signIn);

      await tabUntilFocused(user, signIn);

      expectVisibleFocusRing(signIn);
    },
  );

  it.each(preferences)(
    'Given la préférence de mouvement %s, When le visiteur atteint le bouton de langue « English » au clavier, Then un anneau de focus est visible',
    async (preference) => {
      const user = userEvent.setup();
      preferMotion(preference);
      await openCollector('/');
      const english = await screen.findByRole('button', { name: 'English' });

      await tabUntilFocused(user, english);

      expectVisibleFocusRing(english);
    },
  );

  it('Given les mouvements réduits, When le visiteur atteint le lien d’évitement au clavier, Then un anneau de focus est visible', async () => {
    const user = userEvent.setup();
    preferMotion('reduce');
    await openCollector('/');
    await screen.findByText('Objet de collection 1');

    await user.tab();

    const skipLink = screen.getByRole('link', { name: 'Aller au contenu principal' });
    expect(skipLink).toHaveFocus();
    expectVisibleFocusRing(skipLink);
  });

  it('Given les mouvements réduits, When le visiteur atteint la page 2 de la pagination au clavier, Then un anneau de focus est visible', async () => {
    const user = userEvent.setup();
    preferMotion('reduce');
    await openCollector('/');
    const pagination = await screen.findByRole('navigation', { name: 'Pagination du catalogue' });
    const pageTwo = within(pagination).getByRole('button', { name: 'Aller à la page 2' });

    await tabUntilFocused(user, pageTwo);

    expectVisibleFocusRing(pageTwo);
  });

  it('Given les mouvements réduits et le vendeur A connecté, When il atteint « Se déconnecter » au clavier, Then un anneau de focus est visible', async () => {
    const user = userEvent.setup();
    preferMotion('reduce');
    await signInAs(sellerA);
    const signOut = await within(screen.getByRole('banner')).findByRole('button', {
      name: 'Se déconnecter',
    });

    await tabUntilFocused(user, signOut);

    expectVisibleFocusRing(signOut);
  });
});
