import { screen, waitFor } from '@testing-library/react';
import userEvent, { type UserEvent } from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { switchLanguage } from '../US-010/catalogueHarness';
import {
  article,
  articleId,
  blockingViolations,
  categoryFilter,
  currentSearch,
  figurines,
  openAt,
  publicMarketplace,
  timbres,
  type MarketplaceArticle,
} from './publicMarketplaceHarness';

function catalogueWorld(): MarketplaceArticle[] {
  return [
    ...Array.from({ length: 14 }, (_, position) => article(position + 1, figurines)),
    article(30, timbres, { fraisDePort: null, description: '<img src="x" onerror="alert(1)">' }),
    article(31, timbres, { etat: 'refuse' }),
  ];
}

type Screen = readonly [
  name: string,
  language: SupportedLanguage,
  url: string,
  landmark: { readonly level: 1; readonly name: string },
];

const screens: readonly Screen[] = [
  ['catalogue filtré', 'fr', '/?categorie=figurines', { level: 1, name: 'Catalogue' }],
  ['catalogue filtré', 'en', '/?categorie=figurines', { level: 1, name: 'Catalogue' }],
  ['catégorie vide', 'fr', '/?categorie=monnaies', { level: 1, name: 'Catalogue' }],
  ['catégorie inconnue', 'en', '/?categorie=inconnue', { level: 1, name: 'Catalogue' }],
  ['fiche', 'fr', `/articles/${articleId(30)}`, { level: 1, name: 'Timbres numéro 30' }],
  ['fiche', 'en', `/articles/${articleId(30)}`, { level: 1, name: 'Timbres numéro 30' }],
  [
    'article introuvable',
    'fr',
    `/articles/${articleId(31)}`,
    { level: 1, name: 'Article introuvable' },
  ],
  ['article introuvable', 'en', `/articles/${articleId(31)}`, { level: 1, name: 'Item not found' }],
];

const settledTexts: Readonly<Record<string, RegExp>> = {
  'catalogue filtré': /Figurines numéro 1\b/,
  'catégorie vide': /Aucun article publié dans cette catégorie/,
  'catégorie inconnue': /This category does not exist/,
  fiche: /Vendeur particulier|Private seller/,
  'article introuvable': /introuvable|not found/,
};

async function tabUntilFocused(user: UserEvent, target: HTMLElement, maxTabs = 40): Promise<void> {
  for (let tabs = 0; tabs < maxTabs && document.activeElement !== target; tabs += 1) {
    await user.tab();
  }
  expect(target).toHaveFocus();
}

describe('US-023-AC5 accessibilité du filtre, de la fiche et de la page introuvable', () => {
  it.each(screens)(
    'Given l’écran %s en %s, When axe-core l’analyse, Then aucune violation serious ou critical',
    async (name, language, url, landmark) => {
      await switchLanguage(language);
      publicMarketplace(catalogueWorld());
      await openAt(url);
      await screen.findByRole('heading', landmark, { timeout: 15000 });
      await screen.findAllByText(settledTexts[name] ?? /./, {}, { timeout: 15000 });
      await waitFor(() => {
        expect(screen.queryByRole('progressbar')).not.toBeInTheDocument();
      });

      expect(await blockingViolations()).toEqual([]);
      expect(document.documentElement).toHaveAttribute('lang', language);
    },
    30000,
  );

  it('Given le catalogue, When le visiteur atteint le filtre au clavier et y choisit Timbres, Then le catalogue est filtré et le focus reste sur le filtre', async () => {
    const user = userEvent.setup();
    publicMarketplace(catalogueWorld());
    await openAt('/');
    const filter = await categoryFilter();

    await tabUntilFocused(user, filter);
    await user.selectOptions(filter, timbres.id);

    await waitFor(() => {
      expect(currentSearch().get('categorie')).toBe('timbres');
    });
    expect(await screen.findByText(/Timbres numéro 30/)).toBeInTheDocument();
    const refreshedFilter = await categoryFilter();
    await waitFor(() => {
      expect(refreshedFilter).toHaveFocus();
    });
    expect(screen.getByRole('heading', { level: 1 })).not.toHaveFocus();
  });

  it('Given un changement de filtre, When le résultat arrive, Then il est annoncé dans une unique zone role="status"', async () => {
    const user = userEvent.setup();
    publicMarketplace(catalogueWorld());
    await openAt('/');

    await user.selectOptions(await categoryFilter(), figurines.id);

    await waitFor(() => {
      expect(screen.getAllByRole('status')).toHaveLength(1);
      expect(screen.getByRole('status')).toHaveTextContent(
        '14 articles publiés dans la catégorie Figurines, page 1 sur 2',
      );
    });
  });

  it('Given le filtre, When il s’affiche, Then c’est une liste native au libellé « Catégorie » accessible par son nom', async () => {
    publicMarketplace(catalogueWorld());
    await openAt('/');

    const filter = await categoryFilter();

    expect(filter.tagName).toBe('SELECT');
    expect(filter).toBeEnabled();
    expect(filter.tabIndex).toBeGreaterThanOrEqual(0);
  });

  it('Given la fiche, When le visiteur la parcourt au clavier, Then le lien de catégorie et « Retour au catalogue » sont atteignables', async () => {
    const user = userEvent.setup();
    publicMarketplace(catalogueWorld());
    await openAt(`/articles/${articleId(30)}`);
    await screen.findByText('Vendeur particulier');

    await tabUntilFocused(user, screen.getByRole('link', { name: /Timbres/ }));
    await tabUntilFocused(user, screen.getByRole('link', { name: 'Retour au catalogue' }));

    await user.keyboard('{Enter}');
    expect(await screen.findByRole('heading', { level: 1, name: 'Catalogue' })).toBeInTheDocument();
  });

  it('Given le filtre en anglais, When il s’affiche, Then son libellé est « Category »', async () => {
    await switchLanguage('en');
    publicMarketplace(catalogueWorld());
    await openAt('/');

    const filter = await categoryFilter('Category');

    expect(filter).toHaveValue('');
    expect(screen.getByRole('option', { name: 'All categories' })).toBeInTheDocument();
  });
});
