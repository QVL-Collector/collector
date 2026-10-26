import type { User } from 'oidc-client-ts';
import { anonymousState, type AuthState } from './AuthSession';
import { toAuthUser } from './toAuthUser';

export type AuthStateStore = {
  readonly getState: () => AuthState;
  readonly getAccessToken: () => string | null;
  readonly subscribe: (listener: () => void) => () => void;
  readonly publish: (user: User | null) => void;
};

function toAuthState(user: User | null): AuthState {
  return user === null ? anonymousState : { status: 'authenticated', user: toAuthUser(user) };
}

export function createAuthStateStore(): AuthStateStore {
  const listeners = new Set<() => void>();
  let currentUser: User | null = null;
  let state = anonymousState;
  return {
    getState: () => state,
    getAccessToken: () =>
      currentUser === null || currentUser.expired === true ? null : currentUser.access_token,
    subscribe: (listener) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    publish: (user) => {
      currentUser = user;
      state = toAuthState(user);
      listeners.forEach((listener) => {
        listener();
      });
    },
  };
}
