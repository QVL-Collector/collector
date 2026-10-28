import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import {
  article,
  articleId,
  currentSearch,
  figurines,
  openAt,
  publicMarketplace,
  timbres,
} from './publicMarketplaceHarness';

const figurineAsterix = article(7, figurines, {
  titre: 'Figurine Astérix 1985',
  description: 'Peinture d’origine.\nBoîte un peu abîmée.',
  prix: { montantCentimes: 15000, devise: 'EUR' },
  fraisDePort: { montantCentimes: 690, devise: 'EUR' },
});

const sansFrais = article(8, timbres, { titre: 'Timbre Marianne', fraisDePort: null });

const piege = article(9, timbres, {
  titre: '<b>Lot</b> piégé',
  description:
    '<script>window.__pirate = true</script><img src="x" onerror="window.__pirate = true">',
});

function normalized(text: string | null): string {
  return (text ?? '').replace(/\s+/g, ' ');
}

async function openSheet(rank: number): Promise<HTMLElement> {
  await openAt(`/articles/${articleId(rank)}`);
  return screen.findByRole('main');
}

describe('US-023-AC2 fiche d’un article publié', () => {
  it('Given un article publié, When le visiteur ouvre sa fiche, Then il voit titre, prix, frais de port, catégorie, description et « Vendeur particulier »', async () => {
    publicMarketplace([figurineAsterix]);
    const main = await openSheet(7);

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Figurine Astérix 1985' }),
    ).toBeInTheDocument();
    const content = normalized(main.textContent);
    expect(content).toContain('150,00 €');
    expect(content).toContain('Frais de port');
    expect(content).toContain('6,90 €');
    expect(content).toContain('Figurines');
    expect(content).toContain('Peinture d’origine.');
    expect(content).toContain('Boîte un peu abîmée.');
    expect(within(main).getByText('Vendeur particulier')).toBeInTheDocument();
  });

  it('Given un article publié, When la fiche s’affiche, Then l’API publique est appelée sans authentification sur /v1/articles/{id}', async () => {
    const marketplace = publicMarketplace([figurineAsterix]);
    await openSheet(7);
    await screen.findByRole('heading', { level: 1, name: 'Figurine Astérix 1985' });

    expect(marketplace.requests.map((url) => url.pathname)).toContain(
      `/marche/v1/articles/${articleId(7)}`,
    );
  });

  it('Given une description sur plusieurs lignes, When la fiche s’affiche, Then les sauts de ligne sont conservés', async () => {
    publicMarketplace([figurineAsterix]);
    await openSheet(7);

    const paragraph = await screen.findByText(/Peinture d’origine\./);
    expect(paragraph.textContent).toContain('\n');
    expect(getComputedStyle(paragraph).whiteSpace).toMatch(/pre/);
  });

  it('Given un article sans frais de port, When la fiche s’affiche, Then les frais de port sont « Non renseignés »', async () => {
    publicMarketplace([sansFrais]);
    const main = await openSheet(8);
    await screen.findByRole('heading', { level: 1, name: 'Timbre Marianne' });

    expect(normalized(main.textContent)).toMatch(/Frais de port.*Non renseignés/);
  });

  it('Given une description et un titre contenant du balisage, When la fiche s’affiche, Then ils sont affichés comme du texte sans être interprétés', async () => {
    publicMarketplace([piege]);
    const main = await openSheet(9);

    expect(
      await screen.findByRole('heading', { level: 1, name: '<b>Lot</b> piégé' }),
    ).toBeInTheDocument();
    expect(main.textContent).toContain('<script>window.__pirate = true</script>');
    expect(main.textContent).toContain('<img src="x" onerror="window.__pirate = true">');
    expect(main.querySelector('script')).toBeNull();
    expect(main.querySelector('img[onerror]')).toBeNull();
    expect(main.querySelector('b')).toBeNull();
    expect((window as unknown as { __pirate?: boolean }).__pirate).toBeUndefined();
  });

  it('Given la fiche d’un article, When le visiteur suit le lien de la catégorie, Then le catalogue filtré sur cette catégorie s’affiche', async () => {
    const user = userEvent.setup();
    publicMarketplace([figurineAsterix, sansFrais]);
    await openSheet(7);

    await user.click(await screen.findByRole('link', { name: /Figurines/ }));

    await waitFor(() => {
      expect(currentSearch().get('categorie')).toBe('figurines');
    });
    expect(window.location.pathname).toBe('/');
    const list = await screen.findByRole('list', { name: 'Articles publiés' });
    expect(within(list).getAllByRole('listitem')).toHaveLength(1);
  });

  it('Given le catalogue filtré en page 2, When le visiteur ouvre une fiche puis « Retour au catalogue », Then le filtre et la page d’origine sont restaurés', async () => {
    const user = userEvent.setup();
    publicMarketplace(
      Array.from({ length: 14 }, (_, position) => article(position + 1, figurines)),
    );
    await openAt('/?categorie=figurines&page=2');
    const list = await screen.findByRole('list', { name: 'Articles publiés' });

    await user.click(within(list).getByRole('link', { name: /Figurines numéro 13/ }));
    await screen.findByText('Vendeur particulier');
    await user.click(screen.getByRole('link', { name: 'Retour au catalogue' }));

    await waitFor(() => {
      expect(currentSearch().get('categorie')).toBe('figurines');
    });
    expect(currentSearch().get('page')).toBe('2');
  });

  it('Given le catalogue, When le visiteur ouvre la fiche d’un article depuis sa carte, Then le titre du document reprend le titre de l’article et le focus passe sur le h1', async () => {
    const user = userEvent.setup();
    publicMarketplace([figurineAsterix]);
    await openAt('/');
    const list = await screen.findByRole('list', { name: 'Articles publiés' });

    await user.click(within(list).getByRole('link'));

    const heading = await screen.findByRole('heading', {
      level: 1,
      name: 'Figurine Astérix 1985',
    });
    await waitFor(() => {
      expect(document.title).toBe('Figurine Astérix 1985 · Collector');
    });
    await waitFor(() => {
      expect(heading).toHaveFocus();
    });
    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);
  });
});
