import { screen } from '@testing-library/react';
import axe from 'axe-core';
import { vi } from 'vitest';
import { createAppServices } from '../../app/createAppServices';
import { renderApp } from '../US-010/catalogueHarness';

type Amount = { readonly montantCentimes: number; readonly devise: 'EUR' };
type Category = { readonly id: string; readonly libelle: string };

export type MarketplaceArticle = {
  readonly id: string;
  readonly etat: 'brouillon' | 'en_controle' | 'publie' | 'refuse' | 'a_verifier';
  readonly titre: string;
  readonly description: string;
  readonly prix: Amount;
  readonly fraisDePort: Amount | null;
  readonly categorie: Category;
};

export type PublicMarketplace = {
  readonly articles: MarketplaceArticle[];
  readonly categories: Category[];
  readonly requests: URL[];
};

export const figurines: Category = { id: 'figurines', libelle: 'Figurines' };
export const timbres: Category = { id: 'timbres', libelle: 'Timbres' };
export const monnaies: Category = { id: 'monnaies', libelle: 'Monnaies' };

vi.setConfig({ testTimeout: 30000 });

export function articleId(rank: number): string {
  return `00000000-0000-4000-8000-${rank.toString().padStart(12, '0')}`;
}

export function article(
  rank: number,
  categorie: Category,
  overrides: Partial<MarketplaceArticle> = {},
): MarketplaceArticle {
  return {
    id: articleId(rank),
    etat: 'publie',
    titre: `${categorie.libelle} numéro ${String(rank)}`,
    description: `Description de l’objet ${String(rank)}`,
    prix: { montantCentimes: 1000 + rank * 100, devise: 'EUR' },
    fraisDePort: { montantCentimes: 450, devise: 'EUR' },
    categorie,
    ...overrides,
  };
}

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': status >= 400 ? 'application/problem+json' : 'application/json' },
  });
}

function problem(status: number, erreurs?: readonly { champ: string; code: string }[]): Response {
  return json(status, {
    type: status === 400 ? 'urn:collector:probleme:saisie-invalide' : 'about:blank',
    title: 'Problème',
    status,
    ...(erreurs === undefined ? {} : { erreurs }),
  });
}

function entry(item: MarketplaceArticle): object {
  return { id: item.id, titre: item.titre, prix: item.prix, categorie: item.categorie };
}

function listCatalogue(marketplace: PublicMarketplace, url: URL): Response {
  const categoryId = url.searchParams.get('categorie');
  if (categoryId !== null && !marketplace.categories.some((known) => known.id === categoryId)) {
    return problem(400, [{ champ: 'categorie', code: 'categorie_inconnue' }]);
  }
  const page = Number(url.searchParams.get('page') ?? '1');
  const size = Number(url.searchParams.get('taille') ?? '20');
  const published = marketplace.articles.filter(
    (item) => item.etat === 'publie' && (categoryId === null || item.categorie.id === categoryId),
  );
  return json(200, {
    elements: published.slice((page - 1) * size, page * size).map(entry),
    page,
    taille: size,
    total: published.length,
  });
}

function readArticle(marketplace: PublicMarketplace, id: string): Response {
  const found = marketplace.articles.find((item) => item.id === id && item.etat === 'publie');
  if (found === undefined) {
    return problem(404);
  }
  return json(200, {
    id: found.id,
    titre: found.titre,
    prix: found.prix,
    categorie: found.categorie,
    description: found.description,
    fraisDePort: found.fraisDePort,
  });
}

function answer(marketplace: PublicMarketplace, url: URL): Response {
  if (url.pathname === '/marche/v1/categories') {
    return json(200, { elements: marketplace.categories });
  }
  if (url.pathname === '/marche/v1/articles') {
    return listCatalogue(marketplace, url);
  }
  const articlePath = /^\/marche\/v1\/articles\/([^/]+)$/.exec(url.pathname);
  if (articlePath?.[1] !== undefined) {
    return readArticle(marketplace, decodeURIComponent(articlePath[1]));
  }
  return problem(404);
}

function urlOf(input: RequestInfo | URL): URL {
  const href = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
  return new URL(href, window.location.origin);
}

export function publicMarketplace(
  articles: MarketplaceArticle[],
  categories: Category[] = [figurines, monnaies, timbres],
): PublicMarketplace {
  const marketplace: PublicMarketplace = { articles, categories, requests: [] };
  vi.stubGlobal(
    'fetch',
    vi.fn<typeof fetch>((input) => {
      const url = urlOf(input);
      marketplace.requests.push(url);
      return Promise.resolve(answer(marketplace, url));
    }),
  );
  return marketplace;
}

export function catalogueRequests(marketplace: PublicMarketplace): URL[] {
  return marketplace.requests.filter((url) => url.pathname === '/marche/v1/articles');
}

export async function openAt(url: string): Promise<void> {
  const services = await createAppServices({
    marketplaceApiUrl: '/marche',
    catalogue: { source: 'api', demoScenario: 'nominal' },
  });
  renderApp(services, url);
}

export function currentSearch(): URLSearchParams {
  return new URLSearchParams(window.location.search);
}

export async function categoryFilter(name = 'Catégorie'): Promise<HTMLSelectElement> {
  const filter = await screen.findByRole('combobox', { name });
  if (!(filter instanceof HTMLSelectElement)) {
    throw new Error('Le filtre n’est pas une liste native');
  }
  await screen.findByRole('option', { name: figurines.libelle });
  return filter;
}

const blockingImpacts = new Set(['serious', 'critical']);

export async function blockingViolations(): Promise<string[]> {
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
