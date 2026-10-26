import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  authorizationEndpoint,
  buyer,
  currentPath,
  marketplaceRequests,
  openCollector,
  returnFromIdentityProvider,
  sellerA,
  signInAs,
  waitForNavigationTo,
} from './identityProviderHarness';

describe('US-008-AC1 accès à l’espace vendeur après connexion', () => {
  it('Given un visiteur, When il ouvre /vendeur, Then il est redirigé vers le serveur d’autorisation', async () => {
    await openCollector('/vendeur');

    const authorize = await waitForNavigationTo(authorizationEndpoint());

    expect(authorize.searchParams.get('code_challenge_method')).toBe('S256');
    expect(screen.queryByRole('heading', { name: 'Espace vendeur' })).not.toBeInTheDocument();
  });

  it('Given un visiteur redirigé depuis /vendeur, When il revient authentifié comme vendeur A, Then il arrive dans l’espace vendeur', async () => {
    await openCollector('/vendeur');
    await waitForNavigationTo(authorizationEndpoint());

    await returnFromIdentityProvider(
      (state) => `code=code-fictif-vendeur-a&state=${encodeURIComponent(state)}`,
    );

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Espace vendeur' }),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(currentPath()).toBe('/vendeur');
    });
  });

  it('Given le vendeur A connecté, When il ouvre /vendeur, Then l’espace vendeur l’accueille par son nom', async () => {
    await signInAs(sellerA, '/vendeur');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Espace vendeur' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Bienvenue, Alice Vendeur. Vos annonces apparaîtront ici.'),
    ).toBeInTheDocument();
  });

  it('Given un acheteur connecté sans rôle vendeur, When il ouvre /vendeur, Then une page d’accès refusé s’affiche sans appel d’API vendeur', async () => {
    await signInAs(buyer, '/vendeur');

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Accès réservé' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Votre compte n’a pas le rôle requis pour ouvrir cet espace.'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Espace vendeur' })).not.toBeInTheDocument();
    expect(
      marketplaceRequests().filter((request) => !request.url.pathname.includes('/articles')),
    ).toEqual([]);
  });
});
