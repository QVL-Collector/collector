import type { CollectorRole } from './collectorRoles';

export type AuthUser = {
  readonly displayName: string | null;
  readonly roles: readonly CollectorRole[];
};

export type AuthState =
  | { readonly status: 'anonymous' }
  | { readonly status: 'authenticated'; readonly user: AuthUser };

export type AuthSession = {
  readonly getAccessToken: () => string | null;
  readonly getState: () => AuthState;
  readonly subscribe: (listener: () => void) => () => void;
  readonly signIn: (returnPath: string) => Promise<void>;
  readonly signOut: () => Promise<void>;
  readonly completeSignIn: () => Promise<string>;
};

export const anonymousState: AuthState = { status: 'anonymous' };

function unavailable(): Promise<never> {
  return Promise.reject(new Error('Authentication is not configured'));
}

export const anonymousSession: AuthSession = {
  getAccessToken: () => null,
  getState: () => anonymousState,
  subscribe: () => () => undefined,
  signIn: () => unavailable(),
  signOut: () => unavailable(),
  completeSignIn: () => unavailable(),
};
