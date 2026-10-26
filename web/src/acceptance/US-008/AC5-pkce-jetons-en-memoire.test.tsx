import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
  authorizationEndpoint,
  browserCookies,
  browserLocalStorage,
  browserSessionStorage,
  browserStorageSnapshot,
  identityProvider,
  marketplaceRequests,
  openCollector,
  sellerA,
  sessionStorageKeys,
  signInAs,
  waitForNavigationTo,
  press,
} from './identityProviderHarness';

const base64UrlSha256Length = 43;

async function startSignIn(): Promise<URL> {
  await openCollector('/');
  const signIn = await screen.findByRole('button', { name: 'Se connecter' });
  await press(signIn);
  return waitForNavigationTo(authorizationEndpoint());
}

async function signedInSeller(): Promise<void> {
  await signInAs(sellerA);
  await screen.findByText('Alice Vendeur');
}

function issuedSecrets(): string[] {
  const idp = identityProvider();
  return [...idp.issuedAccessTokens, ...idp.issuedRefreshTokens, ...idp.issuedIdTokens];
}

async function goToCataloguePage(page: number): Promise<void> {
  const pagination = await screen.findByRole('navigation', { name: 'Pagination du catalogue' });
  const target = within(pagination).getByRole('button', {
    name: `Aller à la page ${String(page)}`,
  });
  await press(target);
  await screen.findByText(`Objet de collection ${String((page - 1) * 12 + 1)}`);
}

describe('US-008-AC5 flux code d’autorisation avec PKCE S256', () => {
  it('Given le client OIDC du front, When la connexion démarre, Then la requête d’autorisation demande un code avec PKCE S256', async () => {
    const authorize = await startSignIn();

    expect(authorize.searchParams.get('response_type')).toBe('code');
    expect(authorize.searchParams.get('code_challenge_method')).toBe('S256');
    expect(authorize.searchParams.get('code_challenge')).toMatch(/^[A-Za-z0-9_-]+$/);
    expect(authorize.searchParams.get('code_challenge')).toHaveLength(base64UrlSha256Length);
    expect(authorize.searchParams.get('state')).toBeTruthy();
    expect(authorize.searchParams.has('client_secret')).toBe(false);
  });

  it('Given deux connexions successives, When elles démarrent, Then chaque requête porte un défi PKCE et un state différents', async () => {
    const first = await startSignIn();
    const second = await startSignIn();

    expect(second.searchParams.get('code_challenge')).not.toBe(
      first.searchParams.get('code_challenge'),
    );
    expect(second.searchParams.get('state')).not.toBe(first.searchParams.get('state'));
  });

  it('Given le retour du serveur d’autorisation, When le code est échangé, Then le vérificateur PKCE accompagne l’échange', async () => {
    await signedInSeller();

    const exchange = identityProvider().requests.find(
      (request) => new URLSearchParams(request.body).get('grant_type') === 'authorization_code',
    );
    const verifier = new URLSearchParams(exchange?.body).get('code_verifier') ?? '';
    expect(verifier.length).toBeGreaterThanOrEqual(43);
    expect(verifier.length).toBeLessThanOrEqual(128);
  });
});

describe('US-008-AC5 jeton d’accès en mémoire uniquement', () => {
  it('Given la connexion en cours, When le visiteur part vers le serveur d’autorisation, Then seul l’état temporaire de la requête est conservé en sessionStorage', async () => {
    await startSignIn();

    expect(browserLocalStorage()).toHaveLength(0);
    expect(sessionStorageKeys().length).toBeGreaterThan(0);
    expect(browserCookies()).toBe('');
  });

  it('Given le vendeur A connecté, When le stockage du navigateur est relu, Then aucun jeton n’est présent dans localStorage, sessionStorage ni les cookies', async () => {
    await signedInSeller();

    const snapshot = browserStorageSnapshot();
    for (const secret of issuedSecrets()) {
      expect(snapshot).not.toContain(secret);
    }
    expect(browserLocalStorage()).toHaveLength(0);
    expect(browserCookies()).toBe('');
  });

  it('Given le retour du serveur d’autorisation, When le rappel est traité, Then l’état temporaire de la requête est supprimé de sessionStorage', async () => {
    await signedInSeller();

    await waitFor(() => {
      expect(browserSessionStorage()).toHaveLength(0);
    });
  });

  it('Given le vendeur A connecté, When la page est rechargée, Then la mémoire est vide et le site revient à l’état visiteur', async () => {
    await signedInSeller();

    await openCollector('/');

    expect(await screen.findByRole('button', { name: 'Se connecter' })).toBeInTheDocument();
    expect(screen.queryByText('Alice Vendeur')).not.toBeInTheDocument();
  });

  it('Given un jeton d’accès de 5 minutes expiré et un renouvellement refusé, When le catalogue est rechargé, Then le jeton expiré n’est jamais envoyé à l’API', async () => {
    identityProvider().refreshAnswer = { status: 400, body: { error: 'invalid_grant' } };
    await signedInSeller();
    const [expiredToken] = identityProvider().issuedAccessTokens;
    await waitFor(() => {
      expect(
        marketplaceRequests().some(
          (request) => request.headers.get('Authorization') === `Bearer ${expiredToken ?? ''}`,
        ),
      ).toBe(true);
    });
    const requestsBeforeExpiry = marketplaceRequests().length;

    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(Date.now() + 6 * 60 * 1000);
    await goToCataloguePage(2);

    const afterExpiry = marketplaceRequests().slice(requestsBeforeExpiry);
    expect(afterExpiry.length).toBeGreaterThan(0);
    for (const request of afterExpiry) {
      expect(request.headers.get('Authorization') ?? '').not.toContain(expiredToken ?? '');
    }
  });

  it('Given un jeton d’accès expiré, When le renouvellement est refusé par le serveur d’autorisation, Then le site revient à l’état visiteur', async () => {
    identityProvider().refreshAnswer = { status: 400, body: { error: 'invalid_grant' } };
    identityProvider().accessTokenLifetimeSeconds = 2;
    await signedInSeller();

    expect(
      await screen.findByRole('button', { name: 'Se connecter' }, { timeout: 8000 }),
    ).toBeInTheDocument();
    expect(screen.queryByText('Alice Vendeur')).not.toBeInTheDocument();
  }, 15000);
});
