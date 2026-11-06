import { act, screen, within } from '@testing-library/react';
import { afterEach, beforeEach, expect, vi } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { sellerA, signInAs, type FictitiousAccount } from '../US-008/identityProviderHarness';
import type { Issue, ReasonCode } from './notificationFixtures';

export { notification } from './notificationFixtures';
export type { Issue, ReasonCode } from './notificationFixtures';

type Answer = { readonly status: number; readonly body: unknown };

export type NotificationRequest = {
  readonly path: string;
  readonly authorization: string | null;
};

export type FakeInteractions = {
  notifications: unknown[];
  scripted: Answer | null;
  pending: Promise<void> | null;
  readonly requests: NotificationRequest[];
};

const notificationsPath = '/interactions/v1/notifications';
export const notificationsScreenPath = '/vendeur/notifications';
export const pollingIntervalMs = 5000;
export const maximumDelayMs = 10000;
export const failureTimeout = { timeout: 15000 };

let current: FakeInteractions | null = null;

vi.setConfig({ testTimeout: 30000 });

function problem(status: number): Answer {
  return { status, body: { type: 'about:blank', title: 'Problème', status } };
}

function hrefOf(input: RequestInfo | URL): string {
  return typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
}

function respond(answer: Answer): Response {
  return new Response(JSON.stringify(answer.body), {
    status: answer.status,
    headers: {
      'Content-Type': answer.status >= 400 ? 'application/problem+json' : 'application/json',
    },
  });
}

function list(interactions: FakeInteractions): Answer {
  return {
    status: 200,
    body: {
      elements: interactions.notifications,
      page: 1,
      taille: 20,
      total: interactions.notifications.length,
    },
  };
}

beforeEach(() => {
  const interactions: FakeInteractions = {
    notifications: [],
    scripted: null,
    pending: null,
    requests: [],
  };
  current = interactions;
  const upstream = globalThis.fetch;
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = new URL(hrefOf(input), window.location.origin);
      if (!url.pathname.startsWith('/interactions/')) {
        return upstream(input, init);
      }
      const source = input instanceof Request ? input : null;
      const headers = new Headers(init?.headers ?? source?.headers);
      interactions.requests.push({
        path: `${url.pathname}${url.search}`,
        authorization: headers.get('Authorization'),
      });
      if (url.pathname !== notificationsPath) {
        return respond(problem(404));
      }
      if (interactions.pending !== null) {
        await interactions.pending;
      }
      return respond(interactions.scripted ?? list(interactions));
    }),
  );
});

afterEach(() => {
  Reflect.deleteProperty(document, 'visibilityState');
  Reflect.deleteProperty(document, 'hidden');
});

export function interactions(): FakeInteractions {
  if (current === null) {
    throw new Error('Interactions simulé non initialisé');
  }
  return current;
}

export function givenNotifications(...notifications: unknown[]): void {
  interactions().notifications = [...notifications];
}

export function addNotification(added: unknown): void {
  interactions().notifications = [added, ...interactions().notifications];
}

export function answerWith(status: number, body?: unknown): void {
  interactions().scripted = { status, body: body ?? problem(status).body };
}

export function answerNormally(): void {
  interactions().scripted = null;
}

export function holdAnswers(): () => void {
  let release: () => void = () => undefined;
  interactions().pending = new Promise((resolve) => {
    release = () => {
      interactions().pending = null;
      resolve();
    };
  });
  return () => {
    release();
  };
}

export function notificationReads(): NotificationRequest[] {
  return interactions().requests.filter((request) => request.path.startsWith(notificationsPath));
}

export type NotificationsWording = {
  readonly link: string;
  readonly title: string;
  readonly listTitle: string;
  readonly loading: string;
  readonly loaded: (count: number) => string;
  readonly issues: Readonly<Record<Issue, string>>;
  readonly empty: string;
  readonly error: string;
  readonly retry: string;
  readonly expired: string;
  readonly signInAgain: string;
  readonly forbidden: string;
  readonly reasons: Readonly<Record<ReasonCode, string>>;
};

export const wording: Readonly<Record<SupportedLanguage, NotificationsWording>> = {
  fr: {
    link: 'Notifications',
    title: 'Notifications',
    listTitle: 'Décisions sur vos articles',
    loading: 'Chargement de vos notifications en cours',
    loaded: (count) => (count === 1 ? '1 notification' : `${String(count)} notifications`),
    issues: {
      publie: 'Votre article a été publié',
      refuse: 'Votre article a été refusé',
      a_verifier: 'Votre article doit être vérifié',
    },
    empty: 'Aucune notification pour le moment',
    error: 'Vos notifications sont momentanément indisponibles',
    retry: 'Réessayer',
    expired: 'Votre session a expiré',
    signInAgain: 'Se reconnecter',
    forbidden: 'Accès réservé',
    reasons: {
      annonce_incomplete: 'Annonce incomplète',
      coordonnees_detectees: 'Coordonnées détectées',
      compte_non_autorise: 'Compte non autorisé à vendre',
      prix_a_confirmer: 'Prix à confirmer',
      vendeur_a_verifier: 'Vendeur à vérifier',
      categorie_inconnue: 'Catégorie inconnue',
      prix_abusif: 'Prix abusif',
      article_interdit: 'Article interdit à la vente',
      vendeur_non_fiable: 'Vendeur non fiable',
    },
  },
  en: {
    link: 'Notifications',
    title: 'Notifications',
    listTitle: 'Decisions on your items',
    loading: 'Loading your notifications',
    loaded: (count) => (count === 1 ? '1 notification' : `${String(count)} notifications`),
    issues: {
      publie: 'Your item has been published',
      refuse: 'Your item has been rejected',
      a_verifier: 'Your item needs to be checked',
    },
    empty: 'No notifications yet',
    error: 'Your notifications are temporarily unavailable',
    retry: 'Try again',
    expired: 'Your session has expired',
    signInAgain: 'Sign in again',
    forbidden: 'Restricted access',
    reasons: {
      annonce_incomplete: 'Incomplete listing',
      coordonnees_detectees: 'Contact details detected',
      compte_non_autorise: 'Account not allowed to sell',
      prix_a_confirmer: 'Price to be confirmed',
      vendeur_a_verifier: 'Seller to be reviewed',
      categorie_inconnue: 'Unknown category',
      prix_abusif: 'Excessive price',
      article_interdit: 'Item not allowed for sale',
      vendeur_non_fiable: 'Unreliable seller',
    },
  },
};

export async function openNotificationsAs(
  account: FictitiousAccount = sellerA,
  language: SupportedLanguage = 'fr',
): Promise<void> {
  await signInAs(account, notificationsScreenPath, language);
  await screen.findByRole('heading', { level: 1, name: wording[language].title });
}

export function notificationsList(language: SupportedLanguage = 'fr'): HTMLElement {
  const region = screen.getByRole('region', { name: wording[language].listTitle });
  return within(region).getByRole('list');
}

export function cards(language: SupportedLanguage = 'fr'): HTMLElement[] {
  return within(notificationsList(language)).getAllByRole('listitem');
}

export function cardHeadings(language: SupportedLanguage = 'fr'): string[] {
  return within(notificationsList(language))
    .getAllByRole('heading', { level: 3 })
    .map((heading) => heading.textContent);
}

export async function expectCardCount(
  count: number,
  language: SupportedLanguage = 'fr',
  timeout = 5000,
): Promise<void> {
  await vi.waitFor(
    () => {
      expect(cards(language)).toHaveLength(count);
    },
    { timeout, interval: 50 },
  );
}

export async function waitUntilSettled(language: SupportedLanguage = 'fr'): Promise<void> {
  await vi.waitFor(() => {
    expect(notificationReads().length).toBeGreaterThan(0);
    expect(screen.queryByText(wording[language].loading)).not.toBeInTheDocument();
  }, failureTimeout);
}

export function announcements(): string {
  return screen
    .queryAllByRole('status')
    .map((region) => region.textContent)
    .join(' | ');
}

export function hideTab(hidden: boolean): void {
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    get: () => (hidden ? 'hidden' : 'visible'),
  });
  Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden });
  act(() => {
    document.dispatchEvent(new Event('visibilitychange'));
    window.dispatchEvent(new Event('visibilitychange'));
  });
}

export async function elapse(milliseconds: number): Promise<void> {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(milliseconds);
  });
}
