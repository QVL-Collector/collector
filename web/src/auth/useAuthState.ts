import { useSyncExternalStore } from 'react';
import type { AuthState } from './AuthSession';
import { useAuthSession } from './useAuthSession';

export function useAuthState(): AuthState {
  const session = useAuthSession();
  return useSyncExternalStore(session.subscribe, session.getState);
}
