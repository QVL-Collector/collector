import { act, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import axe from 'axe-core';
import { beforeEach, expect, vi } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { sellerA, signInAs } from '../US-008/identityProviderHarness';

export type ArticleState = 'brouillon' | 'en_controle' | 'publie' | 'refuse' | 'a_verifier';

export type ControlReason =
  | 'annonce_incomplete'
  | 'coordonnees_detectees'
  | 'compte_non_autorise'
  | 'prix_a_confirmer'
  | 'vendeur_a_verifier'
  | 'categorie_inconnue';

type Amount = { readonly montantCentimes: number; readonly devise: 'EUR' };
type Category = { readonly id: string; readonly libelle: string };

export type StoredArticle = {
  id: string;
  etat: ArticleState;
  revision: number;
  titre: string;
  description: string;
  categorie: Category | null;
  prix: Amount | null;
  fraisDePort: Amount | null;
  motif: ControlReason | null;
  creeLe: string;
  modifieLe: string;
};

export type ApiRequest = {
  readonly method: string;
  readonly path: string;
  readonly authorization: string | null;
  readonly body: unknown;
};

export type FieldError = { readonly champ: string; readonly code: string };

export type ScriptedAnswer = { readonly status: number; readonly body: unknown };

export type FakeMarketplace = {
  readonly articles: Map<string, StoredArticle>;
  readonly requests: ApiRequest[];
  categories: Category[];
  categoriesAvailable: boolean;
  readonly scriptedWrites: ScriptedAnswer[];
  readonly scriptedReads: ScriptedAnswer[];
};

export const categories: Category[] = [
  { id: 'cartes', libelle: 'Cartes' },
  { id: 'monnaies', libelle: 'Monnaies' },
  { id: 'timbres', libelle: 'Timbres' },
];

const vendorArticlesPath = '/marche/v1/vendeur/articles';
const categoriesPath = '/marche/v1/categories';
const statesReviewedOnEdit: ReadonlySet<ArticleState> = new Set([
  'publie',
  'en_controle',
  'a_verifier',
]);
let sequence = 0;

vi.setConfig({ testTimeout: 30000 });
let currentMarketplace: FakeMarketplace | null = null;

export function articleId(rank: number): string {
  return `00000000-0000-4000-8000-${rank.toString().padStart(12, '0')}`;
}

function timestamp(minute: number): string {
  return new Date(Date.UTC(2026, 9, 27, 10, minute)).toISOString();
}

export function storedArticle(rank: number, overrides: Partial<StoredArticle> = {}): StoredArticle {
  return {
    id: articleId(rank),
    etat: 'brouillon',
    revision: 1,
    titre: `Lot de cartes ${String(rank)}`,
    description: 'Cartes anciennes en bon état',
    categorie: { id: 'cartes', libelle: 'Cartes' },
    prix: { montantCentimes: 1250, devise: 'EUR' },
    fraisDePort: { montantCentimes: 300, devise: 'EUR' },
    motif: null,
    creeLe: timestamp(rank),
    modifieLe: timestamp(rank),
    ...overrides,
  };
}

function problem(status: number, type: string, erreurs?: FieldError[]): ScriptedAnswer {
  return {
    status,
    body: {
      type,
      title: 'Problème',
      status,
      ...(erreurs === undefined ? {} : { erreurs }),
    },
  };
}

export function invalidInput(erreurs: FieldError[]): ScriptedAnswer {
  return problem(400, 'urn:collector:probleme:saisie-invalide', erreurs);
}

export function staleRevision(): ScriptedAnswer {
  return problem(409, 'urn:collector:probleme:revision-perimee');
}

export function notModifiable(): ScriptedAnswer {
  return problem(409, 'urn:collector:probleme:article-non-modifiable');
}

export function notSubmittable(): ScriptedAnswer {
  return problem(409, 'urn:collector:probleme:article-non-soumettable');
}

function json(answer: ScriptedAnswer): Response {
  const isProblem = answer.status >= 400;
  return new Response(JSON.stringify(answer.body), {
    status: answer.status,
    headers: {
      'Content-Type': isProblem ? 'application/problem+json' : 'application/json',
    },
  });
}

async function readRequest(input: RequestInfo | URL, init?: RequestInit): Promise<ApiRequest> {
  const source = input instanceof Request ? input : null;
  const href = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
  const url = new URL(href, window.location.origin);
  const rawBody = init?.body ?? (source === null ? null : await source.clone().text());
  const text = typeof rawBody === 'string' ? rawBody : '';
  const headers = new Headers(init?.headers ?? source?.headers);
  return {
    method: (init?.method ?? source?.method ?? 'GET').toUpperCase(),
    path: `${url.pathname}${url.search}`,
    authorization: headers.get('Authorization'),
    body: text === '' ? null : (JSON.parse(text) as unknown),
  };
}

type Content = {
  readonly titre?: string;
  readonly description?: string;
  readonly categorieId?: string | null;
  readonly prix?: Amount | null;
  readonly fraisDePort?: Amount | null;
  readonly revision?: number;
};

function categoryOf(marketplace: FakeMarketplace, id: string | null | undefined): Category | null {
  return marketplace.categories.find((category) => category.id === id) ?? null;
}

function validationErrors(marketplace: FakeMarketplace, content: Content): FieldError[] {
  const errors: FieldError[] = [];
  if ((content.titre ?? '').length > 200) {
    errors.push({ champ: 'titre', code: 'trop_long' });
  }
  if ((content.prix?.montantCentimes ?? 0) < 0) {
    errors.push({ champ: 'prix.montantCentimes', code: 'negatif' });
  }
  if ((content.prix?.montantCentimes ?? 0) > 100000000) {
    errors.push({ champ: 'prix.montantCentimes', code: 'au_dela_du_plafond' });
  }
  const categoryId = content.categorieId ?? null;
  if (categoryId !== null && categoryOf(marketplace, categoryId) === null) {
    errors.push({ champ: 'categorieId', code: 'categorie_inconnue' });
  }
  return errors;
}

function withContent(
  marketplace: FakeMarketplace,
  article: StoredArticle,
  content: Content,
): StoredArticle {
  sequence += 1;
  return {
    ...article,
    titre: content.titre ?? '',
    description: content.description ?? '',
    categorie: categoryOf(marketplace, content.categorieId),
    prix: content.prix ?? null,
    fraisDePort: content.fraisDePort ?? null,
    modifieLe: timestamp(30 + sequence),
  };
}

function create(marketplace: FakeMarketplace, content: Content): ScriptedAnswer {
  const errors = validationErrors(marketplace, content);
  if (errors.length > 0) {
    return invalidInput(errors);
  }
  const id = articleId(100 + marketplace.articles.size);
  const article = withContent(marketplace, storedArticle(0, { id }), content);
  const created = { ...article, creeLe: article.modifieLe };
  marketplace.articles.set(id, created);
  return { status: 201, body: created };
}

function update(
  marketplace: FakeMarketplace,
  article: StoredArticle,
  content: Content,
): ScriptedAnswer {
  if (content.revision !== article.revision) {
    return staleRevision();
  }
  const errors = validationErrors(marketplace, content);
  if (errors.length > 0) {
    return invalidInput(errors);
  }
  const edited = { ...withContent(marketplace, article, content), revision: article.revision + 1 };
  const updated: StoredArticle = statesReviewedOnEdit.has(article.etat)
    ? { ...edited, etat: 'en_controle', motif: null }
    : edited;
  marketplace.articles.set(article.id, updated);
  return { status: 200, body: updated };
}

function submit(
  marketplace: FakeMarketplace,
  article: StoredArticle,
  content: Content,
): ScriptedAnswer {
  if (content.revision !== article.revision) {
    return staleRevision();
  }
  if (article.etat !== 'brouillon' && article.etat !== 'refuse') {
    return problem(409, 'urn:collector:probleme:article-non-soumettable');
  }
  const submitted: StoredArticle = {
    ...article,
    etat: 'en_controle',
    motif: null,
    revision: article.revision + 1,
  };
  marketplace.articles.set(article.id, submitted);
  return { status: 202, body: submitted };
}

function list(marketplace: FakeMarketplace, url: URL): ScriptedAnswer {
  const page = Number(url.searchParams.get('page') ?? '1');
  const size = Number(url.searchParams.get('taille') ?? '20');
  const sorted = [...marketplace.articles.values()].sort((left, right) =>
    right.modifieLe.localeCompare(left.modifieLe),
  );
  return {
    status: 200,
    body: {
      elements: sorted.slice((page - 1) * size, page * size),
      page,
      taille: size,
      total: sorted.length,
    },
  };
}

function route(marketplace: FakeMarketplace, request: ApiRequest): ScriptedAnswer {
  const url = new URL(request.path, window.location.origin);
  if (url.pathname === categoriesPath) {
    return marketplace.categoriesAvailable
      ? { status: 200, body: { elements: marketplace.categories } }
      : problem(500, 'about:blank');
  }
  const isWrite = request.method !== 'GET';
  const scripted = isWrite ? marketplace.scriptedWrites.shift() : marketplace.scriptedReads.shift();
  if (scripted !== undefined) {
    return scripted;
  }
  const content = (request.body ?? {}) as Content;
  if (url.pathname === vendorArticlesPath) {
    return request.method === 'POST' ? create(marketplace, content) : list(marketplace, url);
  }
  const [id, action] = url.pathname.slice(vendorArticlesPath.length + 1).split('/');
  const article = marketplace.articles.get(id ?? '');
  if (article === undefined) {
    return problem(404, 'about:blank');
  }
  if (action === 'soumission') {
    return submit(marketplace, article, content);
  }
  return request.method === 'PUT'
    ? update(marketplace, article, content)
    : { status: 200, body: article };
}

function isSellerApi(input: RequestInfo | URL): boolean {
  const href = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
  const { pathname } = new URL(href, window.location.origin);
  return pathname.startsWith(vendorArticlesPath) || pathname === categoriesPath;
}

beforeEach(() => {
  const marketplace: FakeMarketplace = {
    articles: new Map(),
    requests: [],
    categories: [...categories],
    categoriesAvailable: true,
    scriptedWrites: [],
    scriptedReads: [],
  };
  currentMarketplace = marketplace;
  const upstream = globalThis.fetch;
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (!isSellerApi(input)) {
        return upstream(input, init);
      }
      const request = await readRequest(input, init);
      marketplace.requests.push(request);
      return json(route(marketplace, request));
    }),
  );
});

export function marketplace(): FakeMarketplace {
  if (currentMarketplace === null) {
    throw new Error('Place de marché simulée non initialisée');
  }
  return currentMarketplace;
}

export function givenArticles(...articles: StoredArticle[]): void {
  for (const article of articles) {
    marketplace().articles.set(article.id, article);
  }
}

export function concludeControl(id: string, etat: ArticleState, motif: ControlReason | null): void {
  const article = marketplace().articles.get(id);
  if (article === undefined) {
    throw new Error(`Article ${id} absent`);
  }
  marketplace().articles.set(id, { ...article, etat, motif, revision: article.revision + 1 });
}

export function writesTo(method: string, pathSuffix = ''): ApiRequest[] {
  return marketplace().requests.filter(
    (request) => request.method === method && request.path.endsWith(pathSuffix),
  );
}

export function readsOf(id: string): ApiRequest[] {
  return marketplace().requests.filter(
    (request) => request.method === 'GET' && request.path === `${vendorArticlesPath}/${id}`,
  );
}

export async function openAsSeller(
  path: string,
  language: SupportedLanguage = 'fr',
): Promise<void> {
  await signInAs(sellerA, path, language);
}

export async function type(element: HTMLElement, text: string): Promise<void> {
  await userEvent.clear(element);
  if (text !== '') {
    await userEvent.type(element, text);
  }
}

export async function click(element: HTMLElement): Promise<void> {
  await userEvent.click(element);
}

export async function chooseOption(fieldName: string, optionName: string): Promise<void> {
  const field = screen.getByRole('combobox', { name: fieldName });
  const option = await screen.findByRole('option', { name: optionName });
  await userEvent.selectOptions(field, option);
}

export async function waitAtLeast(milliseconds: number): Promise<void> {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, milliseconds));
  });
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

export function statusRegionsText(): string {
  return screen
    .queryAllByRole('status')
    .map((region) => region.textContent)
    .join(' | ');
}

export async function expectAnnounced(text: string, timeout = 5000): Promise<void> {
  await vi.waitFor(
    () => {
      expect(statusRegionsText()).toContain(text);
    },
    { timeout, interval: 100 },
  );
}

export const axeTimeout = 30000;
export const controlTimeout = 20000;
