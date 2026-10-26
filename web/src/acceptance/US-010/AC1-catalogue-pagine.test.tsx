import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import {
  catalogueContaining,
  catalogueFailing,
  catalogueNeverAnswering,
  currentSearch,
  fakeCatalogue,
  formattedPrice,
  pageOf,
  publishedArticle,
  publishedArticles,
  renderCatalogue,
} from './catalogueHarness';

const thirtyArticles = publishedArticles(30);

async function findArticleList(): Promise<HTMLElement> {
  return screen.findByRole('list', { name: 'Articles publiés' });
}

async function findPagination(): Promise<HTMLElement> {
  return screen.findByRole('navigation', { name: 'Pagination du catalogue' });
}

describe('US-010-AC1 catalogue public paginé', () => {
  it('Given des articles publiés, When un visiteur ouvre le catalogue, Then il voit le titre, le prix et la catégorie de chaque article de la première page', async () => {
    const catalogue = catalogueContaining(thirtyArticles);
    renderCatalogue(catalogue);

    const items = within(await findArticleList()).getAllByRole('listitem');

    expect(items).toHaveLength(12);
    const first = publishedArticle(1);
    expect(items[0]).toHaveTextContent(first.title);
    expect(items[0]).toHaveTextContent(formattedPrice(first.price.amountInCents, 'fr-FR'));
    expect(items[0]).toHaveTextContent(first.category.label);
    expect(catalogue.listPublishedArticles).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1 }),
      expect.anything(),
    );
  });

  it('Given 30 articles publiés, When le visiteur choisit la page 2, Then la page 2 est demandée, affichée et inscrite dans ?page=', async () => {
    const user = userEvent.setup();
    const catalogue = catalogueContaining(thirtyArticles);
    renderCatalogue(catalogue);

    await user.click(
      within(await findPagination()).getByRole('button', { name: 'Aller à la page 2' }),
    );

    expect(await screen.findByText(publishedArticle(13).title)).toBeInTheDocument();
    expect(screen.queryByText(publishedArticle(1).title)).not.toBeInTheDocument();
    expect(currentSearch().get('page')).toBe('2');
    expect(catalogue.listPublishedArticles).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 2 }),
      expect.anything(),
    );
  });

  it('Given une URL ?page=3, When le visiteur l’ouvre directement, Then la dernière page partielle est affichée', async () => {
    const catalogue = catalogueContaining(thirtyArticles);
    renderCatalogue(catalogue, '/?page=3');

    const items = within(await findArticleList()).getAllByRole('listitem');

    expect(items).toHaveLength(6);
    expect(items[0]).toHaveTextContent(publishedArticle(25).title);
    expect(
      within(await findPagination()).getByRole('button', { name: 'Page 3, page actuelle' }),
    ).toHaveAttribute('aria-current', 'page');
  });

  it('Given le visiteur sur la page 2, When il revient en arrière dans l’historique, Then la page 1 est réaffichée', async () => {
    const user = userEvent.setup();
    renderCatalogue(catalogueContaining(thirtyArticles));
    await user.click(within(await findPagination()).getByRole('button', { name: 'Page suivante' }));
    await screen.findByText(publishedArticle(13).title);

    window.history.back();

    expect(await screen.findByText(publishedArticle(1).title)).toBeInTheDocument();
  });

  it('Given exactement 12 articles, When le catalogue s’affiche, Then aucune pagination n’est proposée au-delà d’une page', async () => {
    renderCatalogue(catalogueContaining(publishedArticles(12)));

    expect(within(await findArticleList()).getAllByRole('listitem')).toHaveLength(12);
    expect(screen.queryByRole('button', { name: 'Aller à la page 2' })).not.toBeInTheDocument();
  });

  it.each(['abc', '0', '-2', '1.5'])(
    'Given une URL ?page=%s invalide, When le visiteur l’ouvre, Then la première page est affichée',
    async (page) => {
      const catalogue = catalogueContaining(thirtyArticles);
      renderCatalogue(catalogue, `/?page=${page}`);

      expect(await screen.findByText(publishedArticle(1).title)).toBeInTheDocument();
      expect(catalogue.listPublishedArticles).not.toHaveBeenCalledWith(
        expect.not.objectContaining({ page: 1 }),
        expect.anything(),
      );
    },
  );

  it('Given une URL ?page=99 au-delà du catalogue, When le visiteur l’ouvre, Then un état hors limites propose de revenir à la première page', async () => {
    const user = userEvent.setup();
    renderCatalogue(catalogueContaining(thirtyArticles), '/?page=99');

    expect(
      await screen.findByRole('heading', { name: 'Cette page du catalogue est vide' }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Revenir à la première page' }));

    expect(await screen.findByText(publishedArticle(1).title)).toBeInTheDocument();
    expect(currentSearch().get('page') ?? '1').toBe('1');
  });

  it('Given une réponse en attente, When le catalogue s’ouvre, Then un état de chargement est annoncé', async () => {
    renderCatalogue(catalogueNeverAnswering());

    expect(await screen.findByRole('status')).toHaveTextContent('Chargement du catalogue en cours');
    expect(screen.queryByRole('list', { name: 'Articles publiés' })).not.toBeInTheDocument();
  });

  it('Given aucun article publié, When le catalogue s’ouvre, Then un état vide est affiché sans liste ni pagination', async () => {
    renderCatalogue(catalogueContaining([]));

    expect(
      await screen.findByRole('heading', { name: 'Aucun article publié pour le moment' }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('list', { name: 'Articles publiés' })).not.toBeInTheDocument();
    expect(
      screen.queryByRole('navigation', { name: 'Pagination du catalogue' }),
    ).not.toBeInTheDocument();
  });

  it('Given l’API en erreur, When le catalogue s’ouvre, Then un état d’erreur avec « Réessayer » est affiché', async () => {
    renderCatalogue(catalogueFailing());

    expect(
      await screen.findByRole(
        'heading',
        { name: 'Le catalogue est momentanément indisponible' },
        { timeout: 15000 },
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Réessayer' })).toBeEnabled();
  }, 20000);

  it('Given l’API rétablie après une erreur, When le visiteur clique sur « Réessayer », Then la liste s’affiche', async () => {
    const user = userEvent.setup();
    let available = false;
    const catalogue = fakeCatalogue((query) =>
      available
        ? Promise.resolve(pageOf(thirtyArticles, query))
        : Promise.reject(new Error('indisponible')),
    );
    renderCatalogue(catalogue);
    const retry = await screen.findByRole('button', { name: 'Réessayer' }, { timeout: 15000 });
    const callsBeforeRetry = catalogue.listPublishedArticles.mock.calls.length;

    available = true;
    await user.click(retry);

    expect(await screen.findByText(publishedArticle(1).title)).toBeInTheDocument();
    await waitFor(() => {
      expect(catalogue.listPublishedArticles.mock.calls.length).toBeGreaterThan(callsBeforeRetry);
    });
  }, 20000);
});
