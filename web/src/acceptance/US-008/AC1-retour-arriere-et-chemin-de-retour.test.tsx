import { act, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  authorizationEndpoint,
  currentPath,
  identityProvider,
  openCollector,
  press,
  returnFromIdentityProvider,
  waitForNavigationTo,
} from './identityProviderHarness';

function authorizationNavigations(): URL[] {
  return identityProvider().navigations.filter((url) =>
    url.href.startsWith(authorizationEndpoint()),
  );
}

function restorePageFromBackForwardCache(): void {
  const event = new Event('pageshow');
  Object.defineProperty(event, 'persisted', { value: true });
  act(() => {
    window.dispatchEvent(event);
  });
}

async function clickSignIn(): Promise<void> {
  await press(await screen.findByRole('button', { name: 'Se connecter' }));
}

describe('US-008-AC1 retour arrière depuis le serveur d’autorisation', () => {
  it('Given un visiteur revenu par « Précédent » depuis la page de connexion, When il clique à nouveau « Se connecter », Then la redirection vers le serveur d’autorisation est relancée', async () => {
    await openCollector('/');
    await clickSignIn();
    await waitForNavigationTo(authorizationEndpoint());

    restorePageFromBackForwardCache();
    await clickSignIn();

    await waitFor(() => {
      expect(authorizationNavigations()).toHaveLength(2);
    });
  });

  it('Given un visiteur redirigé depuis /vendeur puis revenu par « Précédent », When la page est restaurée, Then la garde relance la redirection vers le serveur d’autorisation', async () => {
    await openCollector('/vendeur');
    await waitForNavigationTo(authorizationEndpoint());

    restorePageFromBackForwardCache();

    await waitFor(() => {
      expect(authorizationNavigations()).toHaveLength(2);
    });
  });
});

describe('US-008-AC1 chemin de retour après connexion limité au site', () => {
  it.each(['/a/..//evil.com', '//evil.com', '/\\evil.com'])(
    'Given une connexion démarrée depuis le chemin %s, When le visiteur revient du serveur d’autorisation, Then il est ramené à l’accueil du site',
    async (startPath) => {
      await openCollector(`${window.location.origin}${startPath}`);
      await clickSignIn();
      await waitForNavigationTo(authorizationEndpoint());

      await returnFromIdentityProvider(
        (state) => `code=code-fictif-vendeur-a&state=${encodeURIComponent(state)}`,
      );

      expect(await screen.findByText('Alice Vendeur')).toBeInTheDocument();
      await waitFor(() => {
        expect(currentPath()).toBe('/');
      });
      expect(window.location.origin).toBe('http://localhost:3000');
    },
  );
});
