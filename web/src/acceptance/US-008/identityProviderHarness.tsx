import '@testing-library/jest-dom/vitest';
import { act, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import i18n from 'i18next';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeAll, beforeEach, expect, vi } from 'vitest';
import { startApplication } from '../../app/startApplication';
import type { SupportedLanguage } from '../../i18n/languages';

export const authority = 'https://auth.collector.qvl-project.com/realms/collector';
export const clientId = 'collector-web';

export type FictitiousAccount = {
  readonly subject: string;
  readonly name?: string;
  readonly username: string;
  readonly roles: readonly string[];
};

export const sellerA: FictitiousAccount = {
  subject: 'vendeur-a',
  name: 'Alice Vendeur',
  username: 'vendeur.a',
  roles: ['vendeur'],
};

export const buyer: FictitiousAccount = {
  subject: 'acheteur',
  name: 'Bruno Acheteur',
  username: 'acheteur',
  roles: ['acheteur'],
};

export const administrator: FictitiousAccount = {
  subject: 'admin',
  name: 'Claire Admin',
  username: 'admin',
  roles: ['admin'],
};

export type RecordedRequest = {
  readonly url: URL;
  readonly method: string;
  readonly headers: Headers;
  readonly body: string;
};

type TokenAnswer = { readonly status: number; readonly body: unknown };

export type IdentityProvider = {
  readonly requests: RecordedRequest[];
  readonly navigations: URL[];
  account: FictitiousAccount;
  accessTokenLifetimeSeconds: number;
  discoveryAvailable: boolean;
  revocationAvailable: boolean;
  refreshAnswer: TokenAnswer | null;
  issuedAccessTokens: string[];
  issuedRefreshTokens: string[];
  issuedIdTokens: string[];
};

export const defaultConfig = {
  marketplaceApiUrl: '/marche',
  catalogue: { source: 'api', demoScenario: 'nominal' },
  oidc: { authority, clientId, scope: 'openid profile' },
};

const pageSize = 12;
const publishedCount = 30;

function base64Url(value: unknown): string {
  return btoa(JSON.stringify(value)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function unsignedJwt(claims: Record<string, unknown>): string {
  return `${base64Url({ alg: 'RS256', typ: 'JWT', kid: 'test' })}.${base64Url(claims)}.c2lnbmF0dXJl`;
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function catalogueAnswer(url: URL): Response {
  const page = Number(url.searchParams.get('page') ?? '1');
  const start = (page - 1) * pageSize;
  const elements = Array.from(
    { length: Math.max(0, Math.min(pageSize, publishedCount - start)) },
    (_, offset) => ({
      id: `a${String(start + offset + 1)}`,
      titre: `Objet de collection ${String(start + offset + 1)}`,
      prix: { montantCentimes: 1000, devise: 'EUR' },
      categorie: { id: 'c1', libelle: 'Cartes' },
    }),
  );
  return json({ elements, page, taille: pageSize, total: publishedCount });
}

function discoveryFor(issuer: string) {
  return {
    issuer,
    authorization_endpoint: `${issuer}/protocol/openid-connect/auth`,
    token_endpoint: `${issuer}/protocol/openid-connect/token`,
    end_session_endpoint: `${issuer}/protocol/openid-connect/logout`,
    revocation_endpoint: `${issuer}/protocol/openid-connect/revoke`,
    jwks_uri: `${issuer}/protocol/openid-connect/certs`,
    userinfo_endpoint: `${issuer}/protocol/openid-connect/userinfo`,
  };
}

const discovery = discoveryFor(authority);
const discoverySuffix = '/.well-known/openid-configuration';

function issueTokens(idp: IdentityProvider): Response {
  const now = Math.floor(Date.now() / 1000);
  const sequence = idp.issuedAccessTokens.length + 1;
  const accessToken = unsignedJwt({
    iss: authority,
    aud: 'account',
    azp: clientId,
    sub: idp.account.subject,
    exp: now + idp.accessTokenLifetimeSeconds,
    iat: now,
    jti: `access-${String(sequence)}`,
    realm_access: { roles: [...idp.account.roles, 'default-roles-collector'] },
  });
  const idToken = unsignedJwt({
    iss: authority,
    aud: clientId,
    sub: idp.account.subject,
    exp: now + idp.accessTokenLifetimeSeconds,
    iat: now,
    preferred_username: idp.account.username,
    ...(idp.account.name === undefined ? {} : { name: idp.account.name }),
  });
  const refreshToken = `refresh-${String(sequence)}-${idp.account.subject}`;
  idp.issuedAccessTokens.push(accessToken);
  idp.issuedIdTokens.push(idToken);
  idp.issuedRefreshTokens.push(refreshToken);
  return json({
    access_token: accessToken,
    id_token: idToken,
    refresh_token: refreshToken,
    token_type: 'Bearer',
    expires_in: idp.accessTokenLifetimeSeconds,
    refresh_expires_in: 1800,
    scope: 'openid profile',
  });
}

function answerToken(idp: IdentityProvider, body: string): Response {
  const grant = new URLSearchParams(body).get('grant_type');
  if (grant === 'refresh_token' && idp.refreshAnswer !== null) {
    return json(idp.refreshAnswer.body, idp.refreshAnswer.status);
  }
  return issueTokens(idp);
}

function answer(idp: IdentityProvider, config: () => unknown, request: RecordedRequest): Response {
  const { url } = request;
  if (url.pathname === '/config.json') {
    return json(config());
  }
  if (url.pathname.startsWith('/marche/')) {
    return catalogueAnswer(url);
  }
  if (url.pathname.endsWith(discoverySuffix)) {
    const issuer = url.href.slice(0, -discoverySuffix.length);
    return idp.discoveryAvailable ? json(discoveryFor(issuer)) : json({}, 503);
  }
  if (url.pathname.endsWith('/protocol/openid-connect/token')) {
    return answerToken(idp, request.body);
  }
  if (url.pathname.endsWith('/protocol/openid-connect/revoke')) {
    return idp.revocationAvailable ? new Response(null, { status: 200 }) : json({}, 503);
  }
  return json({ error: 'not_found' }, 404);
}

async function recordRequest(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<RecordedRequest> {
  const source = input instanceof Request ? input : null;
  const href = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
  const rawBody = init?.body ?? (source === null ? null : await source.clone().text());
  return {
    url: new URL(href, window.location.origin),
    method: init?.method ?? source?.method ?? 'GET',
    headers: new Headers(init?.headers ?? source?.headers),
    body:
      typeof rawBody === 'string'
        ? rawBody
        : rawBody instanceof URLSearchParams
          ? rawBody.toString()
          : '',
  };
}

class RequestWithoutJsdomSignal extends Request {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    super(input, init === undefined ? undefined : { ...init, signal: null });
  }
}

function stubMatchMedia(matches: (query: string) => boolean = () => false): void {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: (query: string): MediaQueryList => ({
      matches: matches(query),
      media: query,
      onchange: null,
      addListener: () => undefined,
      removeListener: () => undefined,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      dispatchEvent: () => false,
    }),
  });
}

function interceptTopLevelNavigation(navigations: URL[]): () => void {
  const realSelf = window.self;
  const navigate = (url: string | URL): void => {
    navigations.push(new URL(url.toString()));
  };
  const fakeLocation = {
    assign: navigate,
    replace: navigate,
    get href(): string {
      return window.location.href;
    },
  };
  const fakeSelf = new Proxy(realSelf, {
    get: (target, property) => {
      if (property === 'location') {
        return fakeLocation;
      }
      if (property === 'stop') {
        return () => undefined;
      }
      return Reflect.get(target, property) as unknown;
    },
  });
  Object.defineProperty(window, 'self', { configurable: true, writable: true, value: fakeSelf });
  return () => {
    Object.defineProperty(window, 'self', { configurable: true, writable: true, value: realSelf });
  };
}

let mountedRoot: Root | null = null;
let restoreNavigation: () => void = () => undefined;
let currentIdp: IdentityProvider | null = null;
let currentConfig: unknown = defaultConfig;

function newIdentityProvider(): IdentityProvider {
  return {
    requests: [],
    navigations: [],
    account: sellerA,
    accessTokenLifetimeSeconds: 300,
    discoveryAvailable: true,
    revocationAvailable: true,
    refreshAnswer: null,
    issuedAccessTokens: [],
    issuedRefreshTokens: [],
    issuedIdTokens: [],
  };
}

beforeAll(() => {
  stubMatchMedia();
});

export function simulateCompactScreen(): void {
  stubMatchMedia((query) => query.includes('max-width'));
}

beforeEach(() => {
  stubMatchMedia();
  vi.stubGlobal('Request', RequestWithoutJsdomSignal);
  browserSessionStorage().clear();
  browserLocalStorage().clear();
  currentConfig = defaultConfig;
  const idp = newIdentityProvider();
  currentIdp = idp;
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = await recordRequest(input, init);
      idp.requests.push(request);
      return answer(idp, () => currentConfig, request);
    }),
  );
  restoreNavigation = interceptTopLevelNavigation(idp.navigations);
});

afterEach(async () => {
  closeTab();
  restoreNavigation();
  vi.useRealTimers();
  vi.unstubAllGlobals();
  browserSessionStorage().clear();
  browserLocalStorage().clear();
  await i18n.changeLanguage('fr');
  window.history.replaceState(null, '', '/');
});

export function identityProvider(): IdentityProvider {
  if (currentIdp === null) {
    throw new Error('Fournisseur d’identité non initialisé');
  }
  return currentIdp;
}

export function serveConfig(config: unknown): void {
  currentConfig = config;
}

function closeTab(): void {
  const root = mountedRoot;
  mountedRoot = null;
  if (root !== null) {
    act(() => {
      root.unmount();
    });
  }
  document.body.innerHTML = '';
}

export async function openCollector(url = '/', language: SupportedLanguage = 'fr'): Promise<void> {
  closeTab();
  identityProvider().navigations.length = 0;
  window.history.replaceState(null, '', url);
  Object.defineProperty(window.navigator, 'language', { configurable: true, value: language });
  Object.defineProperty(window.navigator, 'languages', { configurable: true, value: [language] });
  const container = document.createElement('div');
  container.id = 'root';
  document.body.appendChild(container);
  const root = createRoot(container);
  mountedRoot = root;
  await act(() => startApplication(root));
}

export function lastNavigation(): URL {
  const navigation = identityProvider().navigations.at(-1);
  if (navigation === undefined) {
    throw new Error('Aucune navigation vers le fournisseur d’identité');
  }
  return navigation;
}

export async function waitForNavigationTo(endpoint: string): Promise<URL> {
  await vi.waitFor(() => {
    expect(identityProvider().navigations.some((url) => url.href.startsWith(endpoint))).toBe(true);
  });
  return lastNavigation();
}

export function authorizationEndpoint(): string {
  return discovery.authorization_endpoint;
}

export function endSessionEndpoint(): string {
  return discovery.end_session_endpoint;
}

export function revocationRequests(): RecordedRequest[] {
  return identityProvider().requests.filter((request) =>
    request.url.pathname.endsWith('/protocol/openid-connect/revoke'),
  );
}

export function marketplaceRequests(): RecordedRequest[] {
  return identityProvider().requests.filter((request) =>
    request.url.pathname.startsWith('/marche/'),
  );
}

export async function returnFromIdentityProvider(
  fragment: (state: string) => string,
): Promise<void> {
  const authorize = lastNavigation();
  const state = authorize.searchParams.get('state') ?? '';
  await openCollector(`/rappel#${fragment(state)}`);
}

export async function signInAs(
  account: FictitiousAccount,
  from = '/',
  language: SupportedLanguage = 'fr',
): Promise<void> {
  identityProvider().account = account;
  await openCollector(from, language);
  const signIn = await screen.findByRole('button', {
    name: language === 'fr' ? 'Se connecter' : 'Sign in',
  });
  await press(signIn);
  await waitForNavigationTo(authorizationEndpoint());
  await returnFromIdentityProvider(
    (state) => `code=code-fictif-${account.subject}&state=${encodeURIComponent(state)}`,
  );
}

export function currentPath(): string {
  return `${window.location.pathname}${window.location.search}`;
}

const browser: Window = window;
const page: Document = document;

export function browserLocalStorage(): Storage {
  return browser.localStorage;
}

export function browserSessionStorage(): Storage {
  return browser.sessionStorage;
}

export function browserCookies(): string {
  return page.cookie;
}

export async function press(element: HTMLElement): Promise<void> {
  await userEvent.click(element);
}

export function browserStorageSnapshot(): string {
  const entries: string[] = [];
  for (const storage of [browserLocalStorage(), browserSessionStorage()]) {
    for (let index = 0; index < storage.length; index += 1) {
      const key = storage.key(index) ?? '';
      entries.push(`${key}=${storage.getItem(key) ?? ''}`);
    }
  }
  return `${entries.join('\n')}\n${browserCookies()}`;
}

export function sessionStorageKeys(): string[] {
  return Array.from(
    { length: browserSessionStorage().length },
    (_, index) => browserSessionStorage().key(index) ?? '',
  );
}

export async function switchLanguage(language: SupportedLanguage): Promise<void> {
  await act(async () => {
    await i18n.changeLanguage(language);
  });
}
