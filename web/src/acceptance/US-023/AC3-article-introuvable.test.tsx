import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import {
  article,
  articleId,
  figurines,
  openAt,
  publicMarketplace,
} from './publicMarketplaceHarness';

const hiddenStates = ['brouillon', 'en_controle', 'refuse', 'a_verifier'] as const;

async function expectNotFoundPage(): Promise<void> {
  const heading = await screen.findByRole(
    'heading',
    { level: 1, name: 'Article introuvable' },
    { timeout: 15000 },
  );
  expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);
  await waitFor(() => {
    expect(document.title).toBe('Article introuvable · Collector');
  });
  expect(heading).toBeInTheDocument();
  expect(screen.getByRole('link', { name: 'Retour au catalogue' })).toBeInTheDocument();
}

describe('US-023-AC3 article non publié introuvable', () => {
  it.each(hiddenStates)(
    'Given un article à l’état %s, When un visiteur ouvre l’adresse de sa fiche, Then l’API répond 404 et la page « Article introuvable » s’affiche',
    async (etat) => {
      publicMarketplace([article(1, figurines, { etat, titre: 'Objet secret' })]);
      await openAt(`/articles/${articleId(1)}`);

      await expectNotFoundPage();
      expect(screen.queryByText('Objet secret')).not.toBeInTheDocument();
      expect(screen.queryByText('Vendeur particulier')).not.toBeInTheDocument();
    },
    20000,
  );

  it('Given un identifiant inexistant, When un visiteur ouvre l’adresse, Then la page « Article introuvable » s’affiche', async () => {
    publicMarketplace([]);
    await openAt(`/articles/${articleId(404)}`);

    await expectNotFoundPage();
  }, 20000);

  it('Given un identifiant mal formé, When un visiteur ouvre l’adresse, Then la page « Article introuvable » s’affiche', async () => {
    publicMarketplace([]);
    await openAt('/articles/pas-un-uuid');

    await expectNotFoundPage();
  }, 20000);

  it('Given la page « Article introuvable », When le visiteur suit « Retour au catalogue », Then le catalogue s’affiche', async () => {
    const user = userEvent.setup();
    publicMarketplace([article(2, figurines)]);
    await openAt(`/articles/${articleId(1)}`);
    await expectNotFoundPage();

    await user.click(screen.getByRole('link', { name: 'Retour au catalogue' }));

    expect(await screen.findByRole('heading', { level: 1, name: 'Catalogue' })).toBeInTheDocument();
    expect(window.location.pathname).toBe('/');
  }, 20000);

  it('Given un article listé puis retiré de la vente, When le visiteur ouvre sa carte depuis le catalogue, Then « Article introuvable » s’affiche et le focus passe sur le h1', async () => {
    const user = userEvent.setup();
    const marketplace = publicMarketplace([
      article(1, figurines, { titre: 'Objet bientôt retiré' }),
    ]);
    await openAt('/');
    const list = await screen.findByRole('list', { name: 'Articles publiés' });
    marketplace.articles[0] = article(1, figurines, { etat: 'refuse' });

    await user.click(within(list).getByRole('link'));

    await expectNotFoundPage();
    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1, name: 'Article introuvable' })).toHaveFocus();
    });
  }, 20000);

  it('Given la langue anglaise, When un visiteur ouvre un article non publié, Then la page s’intitule « Item not found »', async () => {
    publicMarketplace([article(1, figurines, { etat: 'refuse' })]);
    await openAt(`/articles/${articleId(1)}`);
    await screen.findByRole(
      'heading',
      { level: 1, name: 'Article introuvable' },
      { timeout: 15000 },
    );

    await userEvent.setup().click(screen.getByRole('button', { name: 'English' }));

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Item not found' }),
    ).toBeInTheDocument();
  }, 20000);
});
