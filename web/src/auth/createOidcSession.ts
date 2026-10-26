import { UserManager } from 'oidc-client-ts';
import type { AuthSession } from './AuthSession';
import { createAuthStateStore } from './createAuthStateStore';
import { createUserManagerSettings } from './createUserManagerSettings';
import type { OidcConfig } from './oidcConfig';
import { toSafeReturnPath } from './returnPath';

function ignoreFailure(): undefined {
  return undefined;
}

export function createOidcSession(
  config: OidcConfig,
  origin = window.location.origin,
): AuthSession {
  const manager = new UserManager(createUserManagerSettings(config, origin));
  const store = createAuthStateStore();
  let pendingRedirect: Promise<void> | null = null;
  let pendingCallback: Promise<string> | null = null;

  manager.events.addUserLoaded(store.publish);
  manager.events.addUserUnloaded(() => {
    store.publish(null);
  });
  manager.events.addAccessTokenExpired(() => {
    manager.removeUser().catch(ignoreFailure);
  });
  manager.clearStaleState().catch(ignoreFailure);
  window.addEventListener('pageshow', (event) => {
    if (event.persisted) {
      pendingRedirect = null;
    }
  });

  function signIn(returnPath: string): Promise<void> {
    if (pendingRedirect !== null) {
      return pendingRedirect;
    }
    const redirect = manager
      .signinRedirect({ state: toSafeReturnPath(returnPath, origin) })
      .finally(() => {
        if (pendingRedirect === redirect) {
          pendingRedirect = null;
        }
      });
    pendingRedirect = redirect;
    return redirect;
  }

  async function signOut(): Promise<void> {
    await manager.revokeTokens().catch(ignoreFailure);
    try {
      await manager.signoutRedirect();
    } catch (error: unknown) {
      await manager.removeUser();
      throw error;
    }
  }

  function completeSignIn(): Promise<string> {
    pendingCallback ??= manager.signinRedirectCallback().then((user) => {
      store.publish(user);
      return toSafeReturnPath(user.state, origin);
    });
    return pendingCallback;
  }

  return {
    getAccessToken: store.getAccessToken,
    getState: store.getState,
    subscribe: store.subscribe,
    signIn,
    signOut,
    completeSignIn,
  };
}
