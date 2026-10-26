import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  administrator,
  authorizationEndpoint,
  buyer,
  clientId,
  currentPath,
  identityProvider,
  openCollector,
  returnFromIdentityProvider,
  sellerA,
  signInAs,
  waitForNavigationTo,
  press,
} from './identityProviderHarness';

async function clickSignIn(): Promise<void> {
  const signIn = await screen.findByRole('button', { name: 'Se connecter' });
  await press(signIn);
}

describe('US-008-AC1 connexion par le serveur d’autorisation', () => {
  it('Given un visiteur sur le site, When il clique « Se connecter », Then il est redirigé vers le realm collector du serveur d’autorisation', async () => {
    await openCollector('/');

    await clickSignIn();
    const authorize = await waitForNavigationTo(authorizationEndpoint());

    expect(authorize.origin).toBe('https://auth.collector.qvl-project.com');
    expect(authorize.pathname).toBe('/realms/collector/protocol/openid-connect/auth');
    expect(authorize.searchParams.get('client_id')).toBe(clientId);
    expect(authorize.searchParams.get('redirect_uri')).toBe(`${window.location.origin}/rappel`);
    expect(authorize.searchParams.get('scope')?.split(' ')).toContain('openid');
  });

  it('Given un visiteur, When il revient du serveur d’autorisation authentifié avec le compte vendeur A, Then il voit son nom et son rôle', async () => {
    await signInAs(sellerA);

    expect(await screen.findByText('Alice Vendeur')).toBeInTheDocument();
    expect(screen.getByText('Vendeur')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Se connecter' })).not.toBeInTheDocument();
  });

  it.each([
    ['acheteur', buyer, 'Bruno Acheteur', 'Acheteur'],
    ['admin', administrator, 'Claire Admin', 'Administrateur'],
  ])(
    'Given le compte fictif %s, When il se connecte, Then son nom et son rôle s’affichent',
    async (_, account, name, role) => {
      await signInAs(account);

      expect(await screen.findByText(name)).toBeInTheDocument();
      expect(screen.getByText(role)).toBeInTheDocument();
    },
  );

  it('Given un visiteur sur la page 2 du catalogue, When il se connecte, Then il revient sur la page 2 du catalogue', async () => {
    await signInAs(sellerA, '/?page=2');

    await screen.findByText('Alice Vendeur');
    await waitFor(() => {
      expect(currentPath()).toBe('/?page=2');
    });
    expect(await screen.findByText('Objet de collection 13')).toBeInTheDocument();
  });

  it('Given un retour du serveur d’autorisation, When le rappel est traité, Then le code est échangé une seule fois au point de jeton', async () => {
    await signInAs(sellerA);
    await screen.findByText('Alice Vendeur');

    const exchanges = identityProvider().requests.filter(
      (request) => new URLSearchParams(request.body).get('grant_type') === 'authorization_code',
    );
    expect(exchanges).toHaveLength(1);
    expect(new URLSearchParams(exchanges[0]?.body).get('code')).toBe('code-fictif-vendeur-a');
  });

  it('Given le serveur d’autorisation refuse l’authentification, When le visiteur revient sur le site, Then il reste visiteur et peut reprendre la connexion', async () => {
    await openCollector('/');
    await clickSignIn();
    await waitForNavigationTo(authorizationEndpoint());

    await returnFromIdentityProvider(
      (state) => `error=access_denied&state=${encodeURIComponent(state)}`,
    );

    expect(await screen.findByText('La connexion n’a pas abouti')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Se connecter' })).toBeEnabled();
    expect(screen.queryByText('Alice Vendeur')).not.toBeInTheDocument();
  });

  it('Given un rappel au state inconnu, When le site le traite, Then aucune session n’est ouverte', async () => {
    await openCollector('/');
    await clickSignIn();
    await waitForNavigationTo(authorizationEndpoint());

    await returnFromIdentityProvider(() => 'code=code-vole&state=state-forge');

    expect(await screen.findByText('La connexion n’a pas abouti')).toBeInTheDocument();
    expect(identityProvider().issuedAccessTokens).toHaveLength(0);
  });

  it('Given le serveur d’autorisation injoignable, When le visiteur clique « Se connecter », Then il reste sur le site en visiteur', async () => {
    identityProvider().discoveryAvailable = false;
    await openCollector('/');

    await clickSignIn();

    expect(await screen.findByRole('button', { name: 'Se connecter' })).toBeInTheDocument();
    expect(identityProvider().navigations).toHaveLength(0);
    expect(currentPath()).toBe('/');
  });
});
