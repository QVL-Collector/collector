import { screen, waitFor, within } from '@testing-library/react';
import userEvent, { type UserEvent } from '@testing-library/user-event';
import axe from 'axe-core';
import { describe, expect, it } from 'vitest';
import {
  catalogueContaining,
  catalogueFailing,
  catalogueNeverAnswering,
  publishedArticle,
  publishedArticles,
  renderCatalogue,
  switchLanguage,
  type FakeCatalogue,
} from './catalogueHarness';

const blockingImpacts = new Set(['serious', 'critical']);

async function blockingViolations(): Promise<string[]> {
  const results = await axe.run(document, {
    resultTypes: ['violations'],
    rules: { 'color-contrast': { enabled: false } },
  });
  return results.violations
    .filter((violation) => blockingImpacts.has(violation.impact ?? ''))
    .map(
      (violation) =>
        `${violation.id} (${violation.impact ?? ''}) : ${violation.nodes.map((node) => node.target.join(' ')).join(', ')}`,
    );
}

async function tabUntilFocused(user: UserEvent, target: HTMLElement, maxTabs = 40): Promise<void> {
  for (let tabs = 0; tabs < maxTabs && document.activeElement !== target; tabs += 1) {
    await user.tab();
  }
  expect(target).toHaveFocus();
}

const states: readonly (readonly [string, () => FakeCatalogue, string, string])[] = [
  [
    'liste paginée',
    () => catalogueContaining(publishedArticles(30)),
    '/',
    publishedArticle(1).title,
  ],
  ['vide', () => catalogueContaining([]), '/', 'Aucun article publié pour le moment'],
  ['erreur', catalogueFailing, '/', 'Le catalogue est momentanément indisponible'],
  [
    'hors limites',
    () => catalogueContaining(publishedArticles(30)),
    '/?page=99',
    'Cette page du catalogue est vide',
  ],
];

describe('US-010-AC6 accessibilité du catalogue', () => {
  it.each(states)(
    'Given l’état %s, When axe-core analyse la page, Then aucune violation serious ou critical',
    async (_, catalogue, url, landmark) => {
      renderCatalogue(catalogue(), url);
      await screen.findAllByText(landmark, {}, { timeout: 15000 });

      expect(await blockingViolations()).toEqual([]);
    },
    20000,
  );

  it('Given l’état de chargement, When axe-core analyse la page, Then aucune violation serious ou critical', async () => {
    renderCatalogue(catalogueNeverAnswering());
    await screen.findAllByText('Chargement du catalogue en cours');

    expect(await blockingViolations()).toEqual([]);
  });

  it('Given la page en anglais, When axe-core analyse la page, Then aucune violation serious ou critical', async () => {
    await switchLanguage('en');
    renderCatalogue(catalogueContaining(publishedArticles(30)));
    await screen.findByRole('list', { name: 'Published items' });

    expect(await blockingViolations()).toEqual([]);
  });

  it('Given la page du catalogue, When elle s’affiche, Then elle a un titre de document, un seul h1, une langue et une zone principale', async () => {
    renderCatalogue(catalogueContaining(publishedArticles(3)));
    await screen.findByRole('list', { name: 'Articles publiés' });

    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Catalogue');
    expect(screen.getByRole('main')).toBeInTheDocument();
    await waitFor(() => {
      expect(document.title).toBe('Catalogue · Collector');
    });
    expect(document.documentElement).toHaveAttribute('lang', 'fr');
  });

  it('Given la page du catalogue, When le visiteur appuie sur Tab, Then le premier arrêt est le lien d’évitement vers le contenu principal', async () => {
    const user = userEvent.setup();
    renderCatalogue(catalogueContaining(publishedArticles(3)));
    await screen.findByRole('list', { name: 'Articles publiés' });

    await user.tab();

    const skipLink = screen.getByRole('link', { name: 'Aller au contenu principal' });
    expect(skipLink).toHaveFocus();
    expect(skipLink.getAttribute('href')).toBe(`#${screen.getByRole('main').id}`);
  });

  it('Given la pagination, When le visiteur l’atteint au clavier et valide la page 2, Then la page 2 s’affiche et le focus passe sur le h1', async () => {
    const user = userEvent.setup();
    renderCatalogue(catalogueContaining(publishedArticles(30)));
    const pagination = await screen.findByRole('navigation', { name: 'Pagination du catalogue' });
    const pageTwo = within(pagination).getByRole('button', { name: 'Aller à la page 2' });

    await tabUntilFocused(user, pageTwo);
    await user.keyboard('{Enter}');

    expect(await screen.findByText(publishedArticle(13).title)).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1 })).toHaveFocus();
    });
  });

  it('Given la page 2, When le visiteur revient à la page 1 par la pagination, Then la page courante est signalée par aria-current et annoncée', async () => {
    const user = userEvent.setup();
    renderCatalogue(catalogueContaining(publishedArticles(30)), '/?page=2');
    const pagination = await screen.findByRole('navigation', { name: 'Pagination du catalogue' });

    await user.click(within(pagination).getByRole('button', { name: 'Page précédente' }));

    expect(
      await within(pagination).findByRole('button', { name: 'Page 1, page actuelle' }),
    ).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('status')).toHaveTextContent('30 articles publiés, page 1 sur 3');
  });

  it('Given l’état d’erreur, When le visiteur atteint « Réessayer » au clavier, Then le bouton est focalisable', async () => {
    const user = userEvent.setup();
    renderCatalogue(catalogueFailing());
    const retry = await screen.findByRole('button', { name: 'Réessayer' }, { timeout: 15000 });

    await tabUntilFocused(user, retry);
  }, 20000);
});

describe('US-010 catalogue bilingue FR/EN', () => {
  it('Given la langue anglaise, When le catalogue s’affiche, Then libellés, prix et annonce sont en anglais', async () => {
    await switchLanguage('en');
    renderCatalogue(catalogueContaining(publishedArticles(30)));

    const items = within(await screen.findByRole('list', { name: 'Published items' })).getAllByRole(
      'listitem',
    );

    expect(items[0]).toHaveTextContent('€11.25');
    expect(screen.getByRole('navigation', { name: 'Catalogue pagination' })).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent('30 published items, page 1 of 3');
    expect(document.documentElement).toHaveAttribute('lang', 'en');
  });

  it('Given le catalogue en français, When le visiteur choisit English, Then l’interface passe en anglais sans rechargement', async () => {
    const user = userEvent.setup();
    const catalogue = catalogueContaining(publishedArticles(30));
    renderCatalogue(catalogue);
    await screen.findByRole('list', { name: 'Articles publiés' });

    await user.click(screen.getByRole('button', { name: 'English' }));

    expect(await screen.findByRole('list', { name: 'Published items' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'English' })).toHaveAttribute('aria-pressed', 'true');
    expect(document.documentElement).toHaveAttribute('lang', 'en');
    expect(screen.getByText(publishedArticle(1).title)).toBeInTheDocument();
  });

  it('Given la langue anglaise et l’API en erreur, When le catalogue s’ouvre, Then le bouton s’intitule « Try again »', async () => {
    await switchLanguage('en');
    renderCatalogue(catalogueFailing());

    expect(
      await screen.findByRole('button', { name: 'Try again' }, { timeout: 15000 }),
    ).toBeEnabled();
  }, 20000);
});
