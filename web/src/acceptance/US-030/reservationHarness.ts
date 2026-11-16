import { screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, vi } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import {
  buyer,
  openCollector,
  signInAs,
  type FictitiousAccount,
} from '../US-008/identityProviderHarness';

type Amount = { readonly montantCentimes: number; readonly devise: 'EUR' };

export type PublishedArticle = {
  readonly id: string;
  readonly titre: string;
  readonly prix: Amount;
  readonly fraisDePort: Amount | null;
  readonly categorie: { readonly id: string; readonly libelle: string };
  readonly description: string;
  reserve: boolean;
};

export type OrderWire = {
  readonly id: string;
  readonly articleId: string;
  readonly statut: 'en_attente_paiement';
  readonly prix: Amount;
  readonly fraisDePort: Amount;
  readonly totalPaye: Amount;
  readonly commission: Amount;
  readonly montantVendeur: Amount;
  readonly echeance: string;
  readonly creeLe: string;
};

export type TransactionsRequest = {
  readonly method: string;
  readonly path: string;
  readonly authorization: string | null;
  readonly contentType: string | null;
  readonly body: string;
};

type Answer = { readonly status: number; readonly body: unknown };

export type FakeBackends = {
  articles: PublishedArticle[];
  orders: OrderWire[];
  nextOrder: OrderWire | null;
  reservationAnswer: Answer | null;
  orderReadAnswer: Answer | null;
  reserveOnSuccess: boolean;
  creationStatus: 200 | 201;
  readonly transactions: TransactionsRequest[];
};

const ordersPath = '/transactions/v1/commandes';

vi.setConfig({ testTimeout: 30000 });

export const failureTimeout = { timeout: 15000 };

let current: FakeBackends | null = null;

export function articleId(rank: number): string {
  return `00000000-0000-4000-8000-${rank.toString().padStart(12, '0')}`;
}

export function orderId(rank: number): string {
  return `00000000-0000-4000-b000-${rank.toString().padStart(12, '0')}`;
}

export function euros(cents: number): Amount {
  return { montantCentimes: cents, devise: 'EUR' };
}

export function publishedArticle(
  rank: number,
  overrides: Partial<PublishedArticle> = {},
): PublishedArticle {
  return {
    id: articleId(rank),
    titre: `Figurine de collection ${String(rank)}`,
    prix: euros(12000),
    fraisDePort: euros(590),
    categorie: { id: 'figurines', libelle: 'Figurines' },
    description: `Description de la figurine ${String(rank)}`,
    reserve: false,
    ...overrides,
  };
}

export const deadline = '2026-11-16T10:25:00.123456Z';
export const createdAt = '2026-11-16T10:15:00.234567Z';

export function order(
  rank: number,
  article: string,
  amounts: { prix: number; frais: number; total: number; commission: number; vendeur: number },
): OrderWire {
  return {
    id: orderId(rank),
    articleId: article,
    statut: 'en_attente_paiement',
    prix: euros(amounts.prix),
    fraisDePort: euros(amounts.frais),
    totalPaye: euros(amounts.total),
    commission: euros(amounts.commission),
    montantVendeur: euros(amounts.vendeur),
    echeance: deadline,
    creeLe: createdAt,
  };
}

function json(status: number, body: unknown, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      'Content-Type': status >= 400 ? 'application/problem+json' : 'application/json',
      ...headers,
    },
  });
}

export function problem(status: number, type = 'about:blank'): Answer {
  return { status, body: { type, title: 'Problème', status } };
}

function catalogueEntry(item: PublishedArticle): object {
  return {
    id: item.id,
    titre: item.titre,
    prix: item.prix,
    categorie: item.categorie,
    reserve: item.reserve,
  };
}

function answerMarketplace(backends: FakeBackends, url: URL): Response {
  if (url.pathname === '/marche/v1/categories') {
    return json(200, { elements: [{ id: 'figurines', libelle: 'Figurines' }] });
  }
  if (url.pathname === '/marche/v1/articles') {
    const page = Number(url.searchParams.get('page') ?? '1');
    const size = Number(url.searchParams.get('taille') ?? '12');
    return json(200, {
      elements: backends.articles.slice((page - 1) * size, page * size).map(catalogueEntry),
      page,
      taille: size,
      total: backends.articles.length,
    });
  }
  const match = /^\/marche\/v1\/articles\/([^/]+)$/.exec(url.pathname);
  const found = backends.articles.find((item) => item.id === decodeURIComponent(match?.[1] ?? ''));
  if (found === undefined) {
    return json(404, problem(404).body);
  }
  return json(200, {
    ...catalogueEntry(found),
    description: found.description,
    fraisDePort: found.fraisDePort,
  });
}

function createOrder(backends: FakeBackends, body: string): Response {
  if (backends.reservationAnswer !== null) {
    return json(backends.reservationAnswer.status, backends.reservationAnswer.body);
  }
  const created = backends.nextOrder;
  if (created === null) {
    return json(503, problem(503).body);
  }
  backends.orders.push(created);
  if (backends.reserveOnSuccess) {
    const requested = (JSON.parse(body) as { articleId?: string }).articleId;
    const reserved = backends.articles.find((item) => item.id === requested);
    if (reserved !== undefined) {
      reserved.reserve = true;
    }
  }
  return backends.creationStatus === 200
    ? json(200, created)
    : json(201, created, { Location: `${ordersPath}/${created.id}` });
}

function readOrder(backends: FakeBackends, path: string): Response {
  if (backends.orderReadAnswer !== null) {
    return json(backends.orderReadAnswer.status, backends.orderReadAnswer.body);
  }
  const id = decodeURIComponent(path.slice(ordersPath.length + 1));
  const found = backends.orders.find((item) => item.id === id);
  return found === undefined ? json(404, problem(404).body) : json(200, found);
}

function answerTransactions(backends: FakeBackends, request: TransactionsRequest): Response {
  if (request.method === 'POST' && request.path === ordersPath) {
    return createOrder(backends, request.body);
  }
  if (request.method === 'GET' && request.path.startsWith(`${ordersPath}/`)) {
    return readOrder(backends, request.path);
  }
  return json(404, problem(404).body);
}

function hrefOf(input: RequestInfo | URL): string {
  return typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
}

async function bodyOf(input: RequestInfo | URL, init?: RequestInit): Promise<string> {
  if (typeof init?.body === 'string') {
    return init.body;
  }
  if (input instanceof Request) {
    return input.clone().text();
  }
  return '';
}

beforeEach(() => {
  const backends: FakeBackends = {
    articles: [],
    orders: [],
    nextOrder: null,
    reservationAnswer: null,
    orderReadAnswer: null,
    reserveOnSuccess: true,
    creationStatus: 201,
    transactions: [],
  };
  current = backends;
  const upstream = globalThis.fetch;
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(hrefOf(input), window.location.origin);
      if (url.pathname.startsWith('/marche/')) {
        return answerMarketplace(backends, url);
      }
      if (!url.pathname.startsWith('/transactions/')) {
        return upstream(input, init);
      }
      const source = input instanceof Request ? input : null;
      const headers = new Headers(init?.headers ?? source?.headers);
      const request: TransactionsRequest = {
        method: (init?.method ?? source?.method ?? 'GET').toUpperCase(),
        path: url.pathname,
        authorization: headers.get('Authorization'),
        contentType: headers.get('Content-Type'),
        body: await bodyOf(input, init),
      };
      backends.transactions.push(request);
      return answerTransactions(backends, request);
    }),
  );
});

export function backends(): FakeBackends {
  if (current === null) {
    throw new Error('Services simulés non initialisés');
  }
  return current;
}

export function givenArticles(...articles: PublishedArticle[]): void {
  backends().articles = articles;
}

export function givenNextOrder(created: OrderWire): void {
  backends().nextOrder = created;
}

export function givenExistingOrder(existing: OrderWire): void {
  backends().orders.push(existing);
}

export function givenOrderAlreadyCreated(): void {
  backends().creationStatus = 200;
}

export function reservationAnswers(status: number, type?: string): void {
  backends().reservationAnswer = problem(status, type);
}

export function orderReadAnswers(status: number): void {
  backends().orderReadAnswer = problem(status);
}

export function reservationRequests(): TransactionsRequest[] {
  return backends().transactions.filter(
    (request) => request.method === 'POST' && request.path === ordersPath,
  );
}

export function articlePath(id: string): string {
  return `/articles/${id}`;
}

export function confirmationPath(id: string): string {
  return `/acheteur/commandes/${id}`;
}

export async function openArticleAs(
  account: FictitiousAccount | null,
  id: string,
  language: SupportedLanguage = 'fr',
): Promise<void> {
  if (account === null) {
    await openCollector(articlePath(id), language);
  } else {
    await signInAs(account, articlePath(id), language);
  }
  await screen.findByRole('heading', { level: 1 }, failureTimeout);
}

export async function openOrderAs(
  id: string,
  language: SupportedLanguage = 'fr',
  account: FictitiousAccount = buyer,
): Promise<void> {
  await signInAs(account, confirmationPath(id), language);
}

export function formatted(cents: number, language: SupportedLanguage): string {
  return new Intl.NumberFormat(language === 'fr' ? 'fr-FR' : 'en-GB', {
    style: 'currency',
    currency: 'EUR',
  })
    .format(cents / 100)
    .replace(/\s+/g, ' ');
}

export async function expectNoReservationSent(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 300));
  expect(reservationRequests()).toHaveLength(0);
}

export async function waitForReservationRequest(): Promise<TransactionsRequest> {
  await waitFor(() => {
    expect(reservationRequests().length).toBeGreaterThan(0);
  }, failureTimeout);
  const sent = reservationRequests().at(0);
  if (sent === undefined) {
    throw new Error('Aucune réservation envoyée');
  }
  return sent;
}

export type ReservationWording = {
  readonly reserve: string;
  readonly reserved: string;
  readonly reservedReason: string;
  readonly signInPrompt: string;
  readonly signIn: string;
  readonly buyerOnly: string;
  readonly title: string;
  readonly status: string;
  readonly paymentDue: string;
  readonly price: string;
  readonly shipping: string;
  readonly total: string;
  readonly commission: string;
  readonly sellerAmount: string;
  readonly notFound: string;
  readonly failures: {
    readonly alreadyReserved: string;
    readonly reservationAlreadyActive: string;
    readonly notReservable: string;
    readonly sessionExpired: string;
    readonly forbidden: string;
    readonly unavailable: string;
    readonly rateLimited: string;
  };
};

export const wording: Readonly<Record<SupportedLanguage, ReservationWording>> = {
  fr: {
    reserve: 'Réserver',
    reserved: 'Réservé',
    reservedReason: 'Cet article est déjà réservé : il n’est plus réservable.',
    signInPrompt: 'Connectez-vous avec un compte acheteur pour réserver cet article.',
    signIn: 'Connexion pour réserver',
    buyerOnly: 'Seul un compte acheteur peut réserver un article.',
    title: 'Article réservé',
    status: 'En attente de paiement',
    paymentDue: 'À régler avant le',
    price: 'Prix de référence',
    shipping: 'Frais de port',
    total: 'Total payé',
    commission: 'Commission Collector (5 % du total)',
    sellerAmount: 'Montant dû au vendeur',
    notFound: 'Commande introuvable',
    failures: {
      alreadyReserved: 'Cet article est déjà réservé par un autre acheteur.',
      reservationAlreadyActive:
        'Vous avez déjà une réservation en cours sur un autre article. Réglez-la ou attendez son échéance avant d’en faire une nouvelle.',
      notReservable: 'Cet article n’est plus réservable.',
      sessionExpired: 'Votre session a expiré. Reconnectez-vous, puis réessayez.',
      forbidden: 'Votre compte n’a pas le rôle acheteur : la réservation n’a pas été faite.',
      unavailable: 'La réservation a échoué. Vérifiez votre connexion, puis réessayez.',
      rateLimited: 'Trop de tentatives en peu de temps. Patientez un instant, puis réessayez.',
    },
  },
  en: {
    reserve: 'Reserve',
    reserved: 'Reserved',
    reservedReason: 'This item is already reserved: it can no longer be reserved.',
    signInPrompt: 'Sign in with a buyer account to reserve this item.',
    signIn: 'Reserve after signing in',
    buyerOnly: 'Only a buyer account can reserve an item.',
    title: 'Item reserved',
    status: 'Awaiting payment',
    paymentDue: 'Pay before',
    price: 'Reference price',
    shipping: 'Shipping',
    total: 'Total paid',
    commission: 'Collector commission (5% of the total)',
    sellerAmount: 'Amount owed to the seller',
    notFound: 'Order not found',
    failures: {
      alreadyReserved: 'This item is already reserved by another buyer.',
      reservationAlreadyActive:
        'You already have an active reservation on another item. Pay for it or wait for its deadline before making a new one.',
      notReservable: 'This item can no longer be reserved.',
      sessionExpired: 'Your session has expired. Sign in again, then try again.',
      forbidden: 'Your account does not have the buyer role: nothing was reserved.',
      unavailable: 'The reservation failed. Check your connection, then try again.',
      rateLimited: 'Too many attempts in a short time. Wait a moment, then try again.',
    },
  },
};

export function amountShownFor(label: string): string {
  const term = screen.getByText(label, { selector: 'dt' });
  return (term.nextElementSibling?.textContent ?? '').replace(/\s+/g, ' ').trim();
}

export function paymentDeadline(language: SupportedLanguage): HTMLTimeElement {
  const caption = screen.getByText(wording[language].paymentDue);
  const time = caption.parentElement?.querySelector('time');
  if (!(time instanceof HTMLTimeElement)) {
    throw new Error('Échéance absente de l’écran de confirmation');
  }
  return time;
}

export function expectNoActionableReservation(language: SupportedLanguage): void {
  const offers = [
    ...screen.queryAllByRole('button', { name: wording[language].reserve }),
    ...screen.queryAllByRole('button', { name: wording[language].signIn }),
  ];
  for (const offer of offers) {
    expect(offer).toBeDisabled();
  }
}
