import { act, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import {
  article,
  catalogueRequests,
  categoryFilter,
  currentSearch,
  figurines,
  monnaies,
  openAt,
  publicMarketplace,
  timbres,
  type MarketplaceArticle,
} from './publicMarketplaceHarness';

const hiddenStates = ['brouillon', 'en_controle', 'refuse', 'a_verifier'] as const;

function mixedCatalogue(): MarketplaceArticle[] {
  return [
    ...Array.from({ length: 14 }, (_, position) => article(position + 1, figurines)),
    ...Array.from({ length: 3 }, (_, position) => article(50 + position, timbres)),
    ...hiddenStates.map((etat, position) =>
      article(90 + position, timbres, { etat, titre: `Timbre non publié ${etat}` }),
    ),
  ];
}

async function listedTitles(): Promise<string[]> {
  const list = await screen.findByRole('list', { name: 'Articles publiés' });
  return within(list)
    .getAllByRole('listitem')
    .map((item) => item.textContent);
}

describe('US-023-AC1 filtre du catalogue par catégorie', () => {
  it('Given des articles publiés dans plusieurs catégories, When le visiteur choisit Timbres, Then seuls les trois timbres publiés sont listés', async () => {
    const user = userEvent.setup();
    publicMarketplace(mixedCatalogue());
    await openAt('/');
    const filter = await categoryFilter();

    await user.selectOptions(filter, timbres.id);

    await waitFor(async () => {
      expect(await listedTitles()).toHaveLength(3);
    });
    const titles = await listedTitles();
    expect(titles.every((title) => title.includes('Timbres numéro'))).toBe(true);
    expect(screen.queryByText(/Timbre non publié/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Figurines numéro/)).not.toBeInTheDocument();
  });

  it('Given le filtre, When le visiteur l’ouvre, Then il propose « Toutes les catégories » puis chaque catégorie de l’API', async () => {
    publicMarketplace(mixedCatalogue());
    await openAt('/');
    const filter = await categoryFilter();

    expect(
      within(filter)
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual(['Toutes les catégories', 'Figurines', 'Monnaies', 'Timbres']);
    expect(filter).toHaveValue('');
  });

  it('Given le choix d’une catégorie, When l’API est appelée, Then la requête porte categorie et la page 1', async () => {
    const user = userEvent.setup();
    const marketplace = publicMarketplace(mixedCatalogue());
    await openAt('/');

    await user.selectOptions(await categoryFilter(), timbres.id);

    await waitFor(() => {
      const last = catalogueRequests(marketplace).at(-1);
      expect(last?.searchParams.get('categorie')).toBe('timbres');
      expect(last?.searchParams.get('page')).toBe('1');
    });
  });

  it('Given la page 2 du catalogue, When le visiteur choisit Figurines, Then l’URL porte ?categorie=figurines et revient à la page 1', async () => {
    const user = userEvent.setup();
    publicMarketplace(mixedCatalogue());
    await openAt('/?page=2');

    await user.selectOptions(await categoryFilter(), figurines.id);

    await waitFor(() => {
      expect(currentSearch().get('categorie')).toBe('figurines');
    });
    expect(['1', null]).toContain(currentSearch().get('page'));
    expect(
      await screen.findByText('14 articles publiés dans la catégorie Figurines, page 1 sur 2'),
    ).toBeInTheDocument();
  });

  it('Given une catégorie filtrée sur deux pages, When le visiteur passe à la page 2, Then le filtre est conservé', async () => {
    const user = userEvent.setup();
    publicMarketplace(mixedCatalogue());
    await openAt('/?categorie=figurines');
    const pagination = await screen.findByRole('navigation', { name: 'Pagination du catalogue' });

    await user.click(within(pagination).getByRole('button', { name: 'Aller à la page 2' }));

    await waitFor(() => {
      expect(currentSearch().get('page')).toBe('2');
    });
    expect(currentSearch().get('categorie')).toBe('figurines');
    await waitFor(async () => {
      expect(await listedTitles()).toHaveLength(2);
    });
  });

  it('Given deux choix successifs, When le visiteur revient en arrière, Then le filtre précédent est restauré', async () => {
    const user = userEvent.setup();
    publicMarketplace(mixedCatalogue());
    await openAt('/');
    const filter = await categoryFilter();

    await user.selectOptions(filter, figurines.id);
    await waitFor(() => {
      expect(currentSearch().get('categorie')).toBe('figurines');
    });
    await user.selectOptions(await categoryFilter(), timbres.id);
    await waitFor(() => {
      expect(currentSearch().get('categorie')).toBe('timbres');
    });

    act(() => {
      window.history.back();
    });

    await waitFor(() => {
      expect(currentSearch().get('categorie')).toBe('figurines');
    });
    await waitFor(async () => {
      expect(await categoryFilter()).toHaveValue('figurines');
    });
    await waitFor(async () => {
      expect((await listedTitles()).every((title) => title.includes('Figurines numéro'))).toBe(
        true,
      );
    });
  });

  it('Given une adresse ?categorie=timbres, When le visiteur l’ouvre directement, Then le filtre est présélectionné et seuls les timbres publiés sont listés', async () => {
    publicMarketplace(mixedCatalogue());
    await openAt('/?categorie=timbres');

    expect(await categoryFilter()).toHaveValue('timbres');
    await waitFor(async () => {
      expect(await listedTitles()).toHaveLength(3);
    });
  });

  it('Given une catégorie filtrée, When le visiteur choisit « Toutes les catégories », Then le paramètre categorie disparaît et tous les publiés sont listés', async () => {
    const user = userEvent.setup();
    publicMarketplace(mixedCatalogue());
    await openAt('/?categorie=timbres');

    await user.selectOptions(await categoryFilter(), '');

    await waitFor(() => {
      expect(currentSearch().has('categorie')).toBe(false);
    });
    expect(await screen.findByText('17 articles publiés, page 1 sur 2')).toBeInTheDocument();
  });

  it('Given une catégorie existante sans article publié, When le visiteur la choisit, Then l’état « catégorie vide » propose de voir toutes les catégories', async () => {
    const user = userEvent.setup();
    publicMarketplace(mixedCatalogue());
    await openAt('/');

    await user.selectOptions(await categoryFilter(), monnaies.id);

    expect(
      await screen.findByRole('heading', { name: 'Aucun article publié dans cette catégorie' }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('list', { name: 'Articles publiés' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Voir toutes les catégories' }));

    await waitFor(() => {
      expect(currentSearch().has('categorie')).toBe(false);
    });
    expect(await listedTitles()).toHaveLength(12);
  });

  it('Given une catégorie inconnue de l’API, When le visiteur ouvre ?categorie=inconnue, Then l’API répond 400 categorie_inconnue et l’état « catégorie inconnue » s’affiche avec l’action de retour', async () => {
    const user = userEvent.setup();
    publicMarketplace(mixedCatalogue());
    await openAt('/?categorie=inconnue');

    expect(
      await screen.findByRole(
        'heading',
        { name: 'Cette catégorie n’existe pas' },
        { timeout: 15000 },
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole('list', { name: 'Articles publiés' })).not.toBeInTheDocument();
    expect(
      screen.queryByRole('heading', { name: 'Le catalogue est momentanément indisponible' }),
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Voir toutes les catégories' }));

    await waitFor(() => {
      expect(currentSearch().has('categorie')).toBe(false);
    });
    expect(await listedTitles()).toHaveLength(12);
  }, 20000);

  it('Given un identifiant de catégorie hors du motif du contrat, When le visiteur ouvre l’adresse, Then aucun article d’une autre catégorie n’est présenté comme filtré et l’API ne reçoit pas cet identifiant', async () => {
    const marketplace = publicMarketplace(mixedCatalogue());
    await openAt('/?categorie=%3Cscript%3E');

    await screen.findByRole('list', { name: 'Articles publiés' });

    expect(
      catalogueRequests(marketplace).every(
        (url) => url.searchParams.get('categorie') !== '<script>',
      ),
    ).toBe(true);
  });

  it('Given le filtre Timbres, When la page s’affiche, Then chaque carte mène à la fiche de son article', async () => {
    publicMarketplace(mixedCatalogue());
    await openAt('/?categorie=timbres');

    await waitFor(async () => {
      expect(await listedTitles()).toHaveLength(3);
    });
    const firstLink = within(screen.getByRole('list', { name: 'Articles publiés' })).getAllByRole(
      'link',
    )[0];
    expect(firstLink?.getAttribute('href')).toMatch(
      /^\/articles\/00000000-0000-4000-8000-0000000000(50|51|52)$/,
    );
  });
});
