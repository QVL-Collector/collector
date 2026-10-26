import { expectNonEmptyString, expectRecord, InvalidPayloadError } from '../shared/payload';

export type OidcConfig = {
  readonly authority: string;
  readonly clientId: string;
  readonly scope: string;
};

const clientIdPattern = /^[A-Za-z0-9._-]+$/;
const scopeTokenPattern = /^[\x21\x23-\x5B\x5D-\x7E]+$/;
const loopbackHosts = new Set(['localhost', '127.0.0.1', '[::1]']);
const defaultScope = 'openid';

function toUrl(text: string): URL | null {
  try {
    return new URL(text);
  } catch {
    return null;
  }
}

function isAllowedProtocol(url: URL): boolean {
  return url.protocol === 'https:' || (url.protocol === 'http:' && loopbackHosts.has(url.hostname));
}

function hasOnlyOriginAndPath(url: URL): boolean {
  return url.username === '' && url.password === '' && url.search === '' && url.hash === '';
}

function parseAuthority(value: unknown, path: string): string {
  const url = toUrl(expectNonEmptyString(value, path));
  if (url === null || !isAllowedProtocol(url) || !hasOnlyOriginAndPath(url)) {
    throw new InvalidPayloadError(path);
  }
  return `${url.origin}${url.pathname.replace(/\/+$/, '')}`;
}

function parseClientId(value: unknown, path: string): string {
  const clientId = expectNonEmptyString(value, path);
  if (!clientIdPattern.test(clientId)) {
    throw new InvalidPayloadError(path);
  }
  return clientId;
}

function parseScope(value: unknown, path: string): string {
  if (value === undefined) {
    return defaultScope;
  }
  const tokens = expectNonEmptyString(value, path).split(' ');
  if (!tokens.every((token) => scopeTokenPattern.test(token)) || !tokens.includes(defaultScope)) {
    throw new InvalidPayloadError(path);
  }
  return tokens.join(' ');
}

export function parseOidcConfig(value: unknown): OidcConfig {
  const oidc = expectRecord(value, 'oidc');
  return {
    authority: parseAuthority(oidc.authority, 'oidc.authority'),
    clientId: parseClientId(oidc.clientId, 'oidc.clientId'),
    scope: parseScope(oidc.scope, 'oidc.scope'),
  };
}
