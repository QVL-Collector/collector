import { screen, waitFor, within } from '@testing-library/react';
import userEvent, { type UserEvent } from '@testing-library/user-event';
import axe from 'axe-core';
import { describe, expect, it } from 'vitest';
import {
  authorizationEndpoint,
  buyer,
  defaultConfig,
  endSessionEndpoint,
  openCollector,
  returnFromIdentityProvider,
  sellerA,
  serveConfig,
  signInAs,
  simulateCompactScreen,
  switchLanguage,
  waitForNavigationTo,
  press,
} from './identityProviderHarness';

const blockingImpacts = new Set(['serious', 'critical']);

async function blockingViolations(): Promise<string[]> {
  const results = await axe.run(document, {
    resultTypes: ['violations'],
    rules: { 'color-contrast': { enabled: false } },
  });
  return results.violations
    .filter((violation) => blockingImpacts.has(violation.impact ?? ''))
    .map(
      (violation) =>
        `${violation.id} (${violation.impact ?? ''}) : ${violation.nodes.map((node) => node.target.join(' ')).join(', ')}`,
    );
}

async function tabUntilFocused(user: UserEvent, target: HTMLElement, maxTabs = 40): Promise<void> {
  for (let tabs = 0; tabs < maxTabs && document.activeElement !== target; tabs += 1) {
    await user.tab();
  }
  expect(target).toHaveFocus();
}

async function failedSignInReturn(): Promise<void> {
  await openCollector('/');
  const signIn = await screen.findByRole('button', { name: 'Se connecter' });
  await press(signIn);
  await waitForNavigationTo(authorizationEndpoint());
  await returnFromIdentityProvider(
    (state) => `error=access_denied&state=${encodeURIComponent(state)}`,
  );
  await screen.findByText('La connexion n’a pas abouti');
}

describe('US-008-AC10 axe-core sur la page d’accueil et le bouton de connexion', () => {
  it('Given la page d’accueil en visiteur avec le bouton « Se connecter », When axe-core l’analyse, Then aucune violation serious ou critical', async () => {
    await openCollector('/');
    await screen.findByRole('button', { name: 'Se connecter' });
    await screen.findByText('Objet de collection 1');

    expect(await blockingViolations()).toEqual([]);
  }, 20000);

  it('Given la page d’accueil du vendeur A connecté, When axe-core l’analyse, Then aucune violation serious ou critical', async () => {
    await signInAs(sellerA);
    await screen.findByText('Alice Vendeur');

    expect(await blockingViolations()).toEqual([]);
  }, 20000);

  it('Given la page d’accueil en anglais, When axe-core l’analyse, Then aucune violation serious ou critical', async () => {
    await openCollector('/');
    await switchLanguage('en');
    await screen.findByRole('button', { name: 'Sign in' });

    expect(await blockingViolations()).toEqual([]);
  }, 20000);

  it('Given un petit écran, When axe-core analyse l’accueil, Then aucune violation et les boutons de compte gardent un nom accessible', async () => {
    simulateCompactScreen();
    await openCollector('/');

    expect(await screen.findByRole('button', { name: 'Se connecter' })).toBeInTheDocument();
    expect(await blockingViolations()).toEqual([]);
  }, 20000);

  it('Given un petit écran et le vendeur A connecté, When l’en-tête s’affiche, Then le bouton « Se déconnecter » garde un nom accessible', async () => {
    simulateCompactScreen();
    await signInAs(sellerA);

    expect(
      await within(screen.getByRole('banner')).findByRole('button', { name: 'Se déconnecter' }),
    ).toBeInTheDocument();
    expect(await blockingViolations()).toEqual([]);
  }, 20000);

  it('Given la page d’accès refusé, When axe-core l’analyse, Then aucune violation serious ou critical', async () => {
    await signInAs(buyer, '/vendeur');
    await screen.findByRole('heading', { level: 1, name: 'Accès réservé' });

    expect(await blockingViolations()).toEqual([]);
  }, 20000);

  it('Given la page d’échec de connexion, When axe-core l’analyse, Then aucune violation serious ou critical', async () => {
    await failedSignInReturn();

    expect(await blockingViolations()).toEqual([]);
  }, 20000);

  it('Given l’échec de démarrage, When axe-core l’analyse, Then aucune violation serious ou critical', async () => {
    serveConfig({ ...defaultConfig, oidc: { ...defaultConfig.oidc, scope: 'profile' } });
    await openCollector('/');
    await screen.findByText('L’application n’a pas pu démarrer');

    expect(await blockingViolations()).toEqual([]);
  });
});

describe('US-008-AC10 connexion et déconnexion au clavier', () => {
  it('Given la page d’accueil, When le visiteur atteint « Se connecter » au clavier et appuie sur Entrée, Then il part vers le serveur d’autorisation', async () => {
    const user = userEvent.setup();
    await openCollector('/');
    const signIn = await screen.findByRole('button', { name: 'Se connecter' });

    await tabUntilFocused(user, signIn);
    await user.keyboard('{Enter}');

    expect((await waitForNavigationTo(authorizationEndpoint())).href).toContain(
      authorizationEndpoint(),
    );
  });

  it('Given le vendeur A connecté, When il atteint « Se déconnecter » au clavier et appuie sur Espace, Then la déconnexion part vers le serveur d’autorisation', async () => {
    const user = userEvent.setup();
    await signInAs(sellerA);
    const signOut = await within(screen.getByRole('banner')).findByRole('button', {
      name: 'Se déconnecter',
    });

    await tabUntilFocused(user, signOut);
    await user.keyboard(' ');

    expect((await waitForNavigationTo(endSessionEndpoint())).href).toContain(endSessionEndpoint());
  });

  it('Given le retour de connexion, When la page d’origine s’affiche, Then le focus est placé sur son titre principal', async () => {
    await signInAs(sellerA, '/vendeur');

    const heading = await screen.findByRole('heading', { level: 1, name: 'Espace vendeur' });
    await waitFor(() => {
      expect(heading).toHaveFocus();
    });
  });

  it('Given l’espace vendeur, When il s’affiche, Then le document porte un titre explicite', async () => {
    await signInAs(sellerA, '/vendeur');
    await screen.findByRole('heading', { level: 1, name: 'Espace vendeur' });

    await waitFor(() => {
      expect(document.title).toBe('Espace vendeur · Collector');
    });
  });
});
