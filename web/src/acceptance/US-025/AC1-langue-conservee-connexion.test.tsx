import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  authorizationEndpoint,
  identityProvider,
  openCollector,
  press,
  returnFromIdentityProvider,
  sellerA,
  waitForNavigationTo,
} from '../US-008/identityProviderHarness';

async function chooseEnglish(): Promise<void> {
  await press(await screen.findByRole('button', { name: 'English' }));
  await screen.findByRole('button', { name: 'Sign in' });
}

async function startSignInInEnglish(from = '/'): Promise<URL> {
  await openCollector(from, 'fr');
  await chooseEnglish();
  await press(screen.getByRole('button', { name: 'Sign in' }));
  return waitForNavigationTo(authorizationEndpoint());
}

async function startSignInInFrench(): Promise<URL> {
  await openCollector('/', 'fr');
  await press(await screen.findByRole('button', { name: 'Se connecter' }));
  return waitForNavigationTo(authorizationEndpoint());
}

describe('US-025-AC1 langue transmise au serveur d’autorisation', () => {
  it('Given un visiteur qui a choisi l’anglais, When il clique « Sign in », Then ui_locales vaut en', async () => {
    const authorize = await startSignInInEnglish();

    expect(authorize.searchParams.get('ui_locales')).toBe('en');
  });

  it('Given un visiteur en français, When il clique « Se connecter », Then ui_locales vaut fr', async () => {
    const authorize = await startSignInInFrench();

    expect(authorize.searchParams.get('ui_locales')).toBe('fr');
  });

  it('Given un visiteur passé à l’anglais puis revenu au français, When il se connecte, Then ui_locales vaut fr', async () => {
    await openCollector('/', 'fr');
    await chooseEnglish();
    await press(screen.getByRole('button', { name: 'Français' }));
    await press(await screen.findByRole('button', { name: 'Se connecter' }));

    const authorize = await waitForNavigationTo(authorizationEndpoint());

    expect(authorize.searchParams.get('ui_locales')).toBe('fr');
  });
});

describe('US-025-AC1 langue conservée au retour de connexion', () => {
  it('Given un navigateur en français et l’anglais choisi, When la connexion réussit, Then l’interface reste en anglais', async () => {
    identityProvider().account = sellerA;
    await startSignInInEnglish();

    await returnFromIdentityProvider(
      (state) => `code=code-fictif-vendeur-a&state=${encodeURIComponent(state)}`,
    );

    expect(await screen.findByText('Alice Vendeur')).toBeInTheDocument();
    expect(screen.getByText('Seller')).toBeInTheDocument();
    expect(
      within(screen.getByRole('banner')).getByRole('button', { name: 'Sign out' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'English' })).toHaveAttribute('aria-pressed', 'true');
    await waitFor(() => {
      expect(document.documentElement).toHaveAttribute('lang', 'en');
    });
  });

  it('Given l’anglais choisi sur la page 2 du catalogue, When la connexion réussit, Then la page 2 s’affiche en anglais', async () => {
    identityProvider().account = sellerA;
    await startSignInInEnglish('/?page=2');

    await returnFromIdentityProvider(
      (state) => `code=code-fictif-vendeur-a&state=${encodeURIComponent(state)}`,
    );

    expect(await screen.findByText('Alice Vendeur')).toBeInTheDocument();
    expect(await screen.findByText('Objet de collection 13')).toBeInTheDocument();
    expect(screen.getByRole('list', { name: 'Published items' })).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent('30 published items, page 2 of 3');
  });

  it('Given un navigateur en français et l’anglais choisi, When la connexion échoue, Then le message d’échec s’affiche en anglais', async () => {
    await startSignInInEnglish();

    await returnFromIdentityProvider(
      (state) => `error=access_denied&state=${encodeURIComponent(state)}`,
    );

    expect(await screen.findByText('Sign-in did not complete')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign in' })).toBeEnabled();
    expect(screen.queryByText('La connexion n’a pas abouti')).not.toBeInTheDocument();
    await waitFor(() => {
      expect(document.documentElement).toHaveAttribute('lang', 'en');
    });
  });

  it('Given l’anglais choisi, When le retour porte un state inconnu, Then le site ne plante pas et propose de se reconnecter', async () => {
    await startSignInInEnglish();

    await returnFromIdentityProvider(() => 'code=code-vole&state=state-forge');

    expect(
      await screen.findByText(/Sign-in did not complete|La connexion n’a pas abouti/),
    ).toBeInTheDocument();
    expect(identityProvider().issuedAccessTokens).toHaveLength(0);
  });

  it('Given le français conservé, When la connexion réussit, Then l’interface reste en français', async () => {
    identityProvider().account = sellerA;
    await startSignInInFrench();

    await returnFromIdentityProvider(
      (state) => `code=code-fictif-vendeur-a&state=${encodeURIComponent(state)}`,
    );

    expect(await screen.findByText('Alice Vendeur')).toBeInTheDocument();
    expect(screen.getByText('Vendeur')).toBeInTheDocument();
    await waitFor(() => {
      expect(document.documentElement).toHaveAttribute('lang', 'fr');
    });
  });

  it('Given un navigateur en anglais et le français choisi, When la connexion réussit, Then l’interface reste en français', async () => {
    identityProvider().account = sellerA;
    await openCollector('/', 'en');
    await press(await screen.findByRole('button', { name: 'Français' }));
    await press(await screen.findByRole('button', { name: 'Se connecter' }));
    const authorize = await waitForNavigationTo(authorizationEndpoint());
    const state = authorize.searchParams.get('state') ?? '';

    await openCollector(
      `/rappel#code=code-fictif-vendeur-a&state=${encodeURIComponent(state)}`,
      'en',
    );

    expect(await screen.findByText('Alice Vendeur')).toBeInTheDocument();
    expect(screen.getByText('Vendeur')).toBeInTheDocument();
    await waitFor(() => {
      expect(document.documentElement).toHaveAttribute('lang', 'fr');
    });
  });
});
