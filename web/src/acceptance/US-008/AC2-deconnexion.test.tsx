import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  currentPath,
  endSessionEndpoint,
  identityProvider,
  openCollector,
  revocationRequests,
  sellerA,
  signInAs,
  waitForNavigationTo,
  press,
} from './identityProviderHarness';

async function clickSignOut(): Promise<void> {
  const signOut = within(screen.getByRole('banner')).getByRole('button', {
    name: 'Se déconnecter',
  });
  await press(signOut);
}

async function signedInSeller(): Promise<void> {
  await signInAs(sellerA);
  await screen.findByText('Alice Vendeur');
}

describe('US-008-AC2 déconnexion', () => {
  it('Given le vendeur A connecté, When il clique « Se déconnecter », Then ses jetons sont révoqués auprès du serveur d’autorisation', async () => {
    await signedInSeller();

    await clickSignOut();
    await waitForNavigationTo(endSessionEndpoint());

    const revokedTokens = revocationRequests().map((request) =>
      new URLSearchParams(request.body).get('token'),
    );
    expect(revokedTokens).toContain(identityProvider().issuedRefreshTokens[0]);
  });

  it('Given le vendeur A connecté, When il clique « Se déconnecter », Then la session du serveur d’autorisation est fermée avec retour à l’accueil du site', async () => {
    await signedInSeller();

    await clickSignOut();
    const endSession = await waitForNavigationTo(endSessionEndpoint());

    expect(endSession.searchParams.get('post_logout_redirect_uri')).toBe(
      `${window.location.origin}/`,
    );
    expect(endSession.searchParams.get('id_token_hint')).toBe(identityProvider().issuedIdTokens[0]);
  });

  it('Given la déconnexion confirmée par le serveur d’autorisation, When le site s’affiche à nouveau, Then il est en état visiteur sur l’accueil', async () => {
    await signedInSeller();
    await clickSignOut();
    await waitForNavigationTo(endSessionEndpoint());

    await openCollector('/');

    expect(await screen.findByRole('button', { name: 'Se connecter' })).toBeInTheDocument();
    expect(screen.queryByText('Alice Vendeur')).not.toBeInTheDocument();
    expect(screen.queryByText('Vendeur')).not.toBeInTheDocument();
    expect(currentPath()).toBe('/');
  });

  it('Given le point de révocation indisponible, When le vendeur se déconnecte, Then la session du serveur d’autorisation est tout de même fermée', async () => {
    await signedInSeller();
    identityProvider().revocationAvailable = false;

    await clickSignOut();

    expect((await waitForNavigationTo(endSessionEndpoint())).href).toContain(endSessionEndpoint());
  });
});
