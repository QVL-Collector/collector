import { InMemoryWebStorage, WebStorageStateStore, type UserManagerSettings } from 'oidc-client-ts';
import { homePath, signInCallbackPath } from './authPaths';
import { createOidcStateStore } from './createOidcStateStore';
import type { OidcConfig } from './oidcConfig';

export function createUserManagerSettings(config: OidcConfig, origin: string): UserManagerSettings {
  return {
    authority: config.authority,
    client_id: config.clientId,
    scope: config.scope,
    redirect_uri: `${origin}${signInCallbackPath}`,
    post_logout_redirect_uri: `${origin}${homePath}`,
    response_type: 'code',
    response_mode: 'fragment',
    userStore: new WebStorageStateStore({ store: new InMemoryWebStorage() }),
    stateStore: createOidcStateStore(),
    automaticSilentRenew: true,
    monitorSession: false,
    loadUserInfo: false,
    revokeTokensOnSignout: false,
  };
}
