import { WebStorageStateStore } from 'oidc-client-ts';

export function createOidcStateStore(): WebStorageStateStore {
  return new WebStorageStateStore({ store: window.sessionStorage, prefix: 'collector.oidc.' });
}
