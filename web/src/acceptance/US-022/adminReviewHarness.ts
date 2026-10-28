import { screen } from '@testing-library/react';
import { beforeEach, expect, vi } from 'vitest';
import {
  administrator,
  buyer,
  sellerA,
  signInAs,
  type FictitiousAccount,
} from '../US-008/identityProviderHarness';

export type ControlReason =
  | 'annonce_incomplete'
  | 'coordonnees_detectees'
  | 'compte_non_autorise'
  | 'prix_a_confirmer'
  | 'vendeur_a_verifier'
  | 'categorie_inconnue';

export type RefusalReason =
  | 'annonce_incomplete'
  | 'coordonnees_detectees'
  | 'prix_abusif'
  | 'article_interdit'
  | 'vendeur_non_fiable';

type Amount = { readonly montantCentimes: number; readonly devise: 'EUR' };
type Category = { readonly id: string; readonly libelle: string };

export type ArticleToReview = {
  id: string;
  etat: 'a_verifier' | 'publie' | 'refuse' | 'en_controle';
  revision: number;
  titre: string;
  description: string;
  categorie: Category | null;
  prix: Amount | null;
  fraisDePort: Amount | null;
  motif: ControlReason | RefusalReason | null;
  vendeurId: string;
  creeLe: string;
  modifieLe: string;
};

export type AdminRequest = {
  readonly method: string;
  readonly path: string;
  readonly authorization: string | null;
  readonly body: unknown;
};

type Answer = { readonly status: number; readonly body: unknown };

type Decision = { readonly revision?: number; readonly decision?: string; readonly motif?: string };

export type FakeAdministration = {
  readonly articles: Map<string, ArticleToReview>;
  readonly requests: AdminRequest[];
  readonly scriptedDecisions: Answer[];
};

const adminArticlesPath = '/marche/v1/admin/articles';
export const adminScreenPath = '/admin';
const refusalReasons: readonly string[] = [
  'annonce_incomplete',
  'coordonnees_detectees',
  'prix_abusif',
  'article_interdit',
  'vendeur_non_fiable',
];

let current: FakeAdministration | null = null;

vi.setConfig({ testTimeout: 30000 });

function reviewId(rank: number): string {
  return `00000000-0000-4000-8000-${rank.toString().padStart(12, '0')}`;
}

function timestamp(minute: number): string {
  return new Date(Date.UTC(2026, 9, 28, 9, minute)).toISOString();
}

export function toReview(rank: number, overrides: Partial<ArticleToReview> = {}): ArticleToReview {
  return {
    id: reviewId(rank),
    etat: 'a_verifier',
    revision: 2,
    titre: `Figurine à vérifier ${String(rank)}`,
    description: `Description de la figurine ${String(rank)}`,
    categorie: { id: 'figurines', libelle: 'Figurines' },
    prix: { montantCentimes: 1500, devise: 'EUR' },
    fraisDePort: { montantCentimes: 400, devise: 'EUR' },
    motif: 'categorie_inconnue',
    vendeurId: `vendeur-${String(rank)}`,
    creeLe: timestamp(rank),
    modifieLe: timestamp(rank),
    ...overrides,
  };
}

function problem(status: number, type: string, erreurs?: unknown[]): Answer {
  return {
    status,
    body: { type, title: 'Problème', status, ...(erreurs === undefined ? {} : { erreurs }) },
  };
}

export function staleRevision(): Answer {
  return problem(409, 'urn:collector:probleme:revision-perimee');
}

export function noLongerToReview(): Answer {
  return problem(409, 'urn:collector:probleme:article-non-a-verifier');
}

function list(administration: FakeAdministration, url: URL): Answer {
  const page = Number(url.searchParams.get('page') ?? '1');
  const size = Number(url.searchParams.get('taille') ?? '20');
  const pending = [...administration.articles.values()]
    .filter((article) => article.etat === 'a_verifier')
    .sort((left, right) => left.modifieLe.localeCompare(right.modifieLe));
  const elements = pending
    .slice((page - 1) * size, page * size)
    .map(
      ({
        id,
        revision,
        titre,
        description,
        categorie,
        prix,
        fraisDePort,
        motif,
        vendeurId,
        modifieLe,
      }) => ({
        id,
        revision,
        titre,
        description,
        categorie,
        prix,
        fraisDePort,
        motif,
        vendeurId,
        modifieLe,
      }),
    );
  return { status: 200, body: { elements, page, taille: size, total: pending.length } };
}

function asSellerArticle(article: ArticleToReview): Omit<ArticleToReview, 'vendeurId'> {
  return {
    id: article.id,
    etat: article.etat,
    revision: article.revision,
    titre: article.titre,
    description: article.description,
    categorie: article.categorie,
    prix: article.prix,
    fraisDePort: article.fraisDePort,
    motif: article.motif,
    creeLe: article.creeLe,
    modifieLe: article.modifieLe,
  };
}

function decide(administration: FakeAdministration, id: string, decision: Decision): Answer {
  const scripted = administration.scriptedDecisions.shift();
  if (scripted !== undefined) {
    return scripted;
  }
  const article = administration.articles.get(id);
  if (article === undefined) {
    return problem(404, 'about:blank');
  }
  if (article.etat !== 'a_verifier') {
    return noLongerToReview();
  }
  if (decision.revision !== article.revision) {
    return staleRevision();
  }
  const refusing = decision.decision === 'refuser';
  if (
    refusing !== (decision.motif !== undefined) ||
    (refusing && !refusalReasons.includes(decision.motif ?? ''))
  ) {
    return problem(422, 'urn:collector:probleme:saisie-invalide', [
      { champ: 'motif', code: 'invalide' },
    ]);
  }
  const decided: ArticleToReview = refusing
    ? { ...article, etat: 'refuse', motif: decision.motif as RefusalReason }
    : { ...article, etat: 'publie', motif: null };
  administration.articles.set(id, decided);
  return { status: 200, body: asSellerArticle(decided) };
}

function route(administration: FakeAdministration, request: AdminRequest): Answer {
  const url = new URL(request.path, window.location.origin);
  if (url.pathname === adminArticlesPath && request.method === 'GET') {
    return list(administration, url);
  }
  const [id, action] = url.pathname.slice(adminArticlesPath.length + 1).split('/');
  if (action === 'decision' && request.method === 'POST') {
    return decide(administration, id ?? '', (request.body ?? {}) as Decision);
  }
  return problem(404, 'about:blank');
}

function hrefOf(input: RequestInfo | URL): string {
  return typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
}

async function readRequest(input: RequestInfo | URL, init?: RequestInit): Promise<AdminRequest> {
  const source = input instanceof Request ? input : null;
  const url = new URL(hrefOf(input), window.location.origin);
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

function respond(answer: Answer): Response {
  return new Response(JSON.stringify(answer.body), {
    status: answer.status,
    headers: {
      'Content-Type': answer.status >= 400 ? 'application/problem+json' : 'application/json',
    },
  });
}

beforeEach(() => {
  const administration: FakeAdministration = {
    articles: new Map(),
    requests: [],
    scriptedDecisions: [],
  };
  current = administration;
  const upstream = globalThis.fetch;
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const { pathname } = new URL(hrefOf(input), window.location.origin);
      if (!pathname.startsWith(adminArticlesPath)) {
        return upstream(input, init);
      }
      const request = await readRequest(input, init);
      administration.requests.push(request);
      return respond(route(administration, request));
    }),
  );
});

export function administration(): FakeAdministration {
  if (current === null) {
    throw new Error('Administration simulée non initialisée');
  }
  return current;
}

export function givenToReview(...articles: ArticleToReview[]): void {
  for (const article of articles) {
    administration().articles.set(article.id, article);
  }
}

export function scriptNextDecision(answer: Answer): void {
  administration().scriptedDecisions.push(answer);
}

export function changeRevision(id: string): void {
  const article = administration().articles.get(id);
  if (article === undefined) {
    throw new Error(`Article ${id} absent`);
  }
  administration().articles.set(id, {
    ...article,
    revision: article.revision + 1,
    titre: `${article.titre} (modifiée)`,
  });
}

export function listReads(): AdminRequest[] {
  return administration().requests.filter((request) => request.method === 'GET');
}

export function decisions(): AdminRequest[] {
  return administration().requests.filter((request) => request.method === 'POST');
}

export const accounts: Readonly<Record<'admin' | 'vendeur' | 'acheteur', FictitiousAccount>> = {
  admin: administrator,
  vendeur: sellerA,
  acheteur: buyer,
};

export const wording = {
  title: 'Articles à vérifier',
  loaded: (count: number) =>
    count === 1 ? '1 article à vérifier' : `${String(count)} articles à vérifier`,
  controlReason: (label: string) => `Motif du contrôle : ${label}`,
  revision: (revision: number) => `Révision examinée : ${String(revision)}`,
  publish: (title: string) => `Publier « ${title} »`,
  refuse: (title: string) => `Refuser « ${title} »`,
  refusalTitle: 'Refuser l’annonce',
  reasonLabel: 'Motif du refus',
  missingReason: 'Choisissez un motif de refus.',
  cancel: 'Annuler',
  confirm: 'Confirmer le refus',
  published: (title: string) => `« ${title} » a été publié.`,
  refused: (title: string) => `« ${title} » a été refusé.`,
  conflict: 'L’article a changé, rechargez la liste',
  reload: 'Recharger la liste',
  empty: 'Aucun article à vérifier',
  forbidden: 'Accès réservé',
  controlReasons: {
    annonce_incomplete: 'Annonce incomplète',
    coordonnees_detectees: 'Coordonnées détectées',
    compte_non_autorise: 'Compte non autorisé à vendre',
    prix_a_confirmer: 'Prix à confirmer',
    vendeur_a_verifier: 'Vendeur à vérifier',
    categorie_inconnue: 'Catégorie inconnue',
  },
  refusalReasons: {
    annonce_incomplete: 'Annonce incomplète',
    coordonnees_detectees: 'Coordonnées détectées',
    prix_abusif: 'Prix abusif',
    article_interdit: 'Article interdit à la vente',
    vendeur_non_fiable: 'Vendeur non fiable',
  },
} as const;

export async function openAdminScreenAs(account: FictitiousAccount): Promise<void> {
  await signInAs(account, adminScreenPath);
  await screen.findByRole('heading', { level: 1, name: wording.title });
}

export async function openAdminScreenWith(count: number): Promise<void> {
  await openAdminScreenAs(administrator);
  if (count === 0) {
    await screen.findByRole('heading', { level: 2, name: wording.empty });
    return;
  }
  await vi.waitFor(() => {
    expect(screen.getAllByRole('button', { name: /^Publier « / })).toHaveLength(count);
  });
}
