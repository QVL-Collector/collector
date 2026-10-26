import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  buyer,
  identityProvider,
  openCollector,
  press,
  sellerA,
  signInAs,
} from '../US-008/identityProviderHarness';

async function chooseEnglish(): Promise<void> {
  await press(await screen.findByRole('button', { name: 'English' }));
}

function configRequests(): number {
  return identityProvider().requests.filter((request) => request.url.pathname === '/config.json')
    .length;
}

describe('US-025-AC1 bascule en anglais sans rechargement', () => {
  it('Given la page introuvable en français, When l’utilisateur choisit l’anglais, Then le message d’erreur passe en anglais sans rechargement', async () => {
    await openCollector('/adresse-inexistante', 'fr');
    await screen.findByRole('heading', { level: 1, name: 'Page introuvable' });
    const loadsBefore = configRequests();

    await chooseEnglish();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Page not found' }),
    ).toBeInTheDocument();
    expect(screen.getByText('The requested address does not match any page.')).toBeInTheDocument();
    expect(configRequests()).toBe(loadsBefore);
    await waitFor(() => {
      expect(document.title).toBe('Page not found · Collector');
    });
  });

  it('Given la page d’accès refusé en français, When l’utilisateur choisit l’anglais, Then le message passe en anglais', async () => {
    await signInAs(buyer, '/vendeur');
    await screen.findByRole('heading', { level: 1, name: 'Accès réservé' });

    await chooseEnglish();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Restricted access' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Buyer')).toBeInTheDocument();
  });

  it('Given l’espace vendeur en français, When le vendeur choisit l’anglais, Then l’espace vendeur et l’en-tête passent en anglais', async () => {
    await signInAs(sellerA, '/vendeur');
    await screen.findByRole('heading', { level: 1, name: 'Espace vendeur' });

    await chooseEnglish();

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Seller area' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Seller')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument();
  });

  it('Given l’anglais choisi sur une page d’erreur, When l’utilisateur revient au catalogue par le site, Then le catalogue reste en anglais', async () => {
    await openCollector('/adresse-inexistante', 'fr');
    await chooseEnglish();

    await press(await screen.findByRole('link', { name: 'Back to the catalogue' }));

    expect(await screen.findByRole('list', { name: 'Published items' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'English' })).toHaveAttribute('aria-pressed', 'true');
  });
});
