import { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router';
import type { AuthState } from '../../auth/AuthSession';
import type { CollectorRole } from '../../auth/collectorRoles';
import { useAuthSession } from '../../auth/useAuthSession';
import { useAuthState } from '../../auth/useAuthState';
import { usePageRestored } from '../../ui/hooks/usePageRestored';

export type AuthenticationGateStatus = 'granted' | 'forbidden' | 'redirecting' | 'unavailable';

export type AuthenticationGateModel = {
  readonly status: AuthenticationGateStatus;
  readonly retry: () => void;
};

function statusOf(
  state: AuthState,
  requiredRole: CollectorRole | undefined,
  unavailable: boolean,
): AuthenticationGateStatus {
  if (state.status === 'anonymous') {
    return unavailable ? 'unavailable' : 'redirecting';
  }
  if (requiredRole !== undefined && !state.user.roles.includes(requiredRole)) {
    return 'forbidden';
  }
  return 'granted';
}

export function useAuthenticationGate(requiredRole?: CollectorRole): AuthenticationGateModel {
  const session = useAuthSession();
  const state = useAuthState();
  const { pathname, search } = useLocation();
  const [unavailable, setUnavailable] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const anonymous = state.status === 'anonymous';

  useEffect(() => {
    if (!anonymous) {
      return undefined;
    }
    let active = true;
    session.signIn(`${pathname}${search}`).catch(() => {
      if (active) {
        setUnavailable(true);
      }
    });
    return () => {
      active = false;
    };
  }, [anonymous, session, pathname, search, attempt]);

  const retry = useCallback(() => {
    setUnavailable(false);
    setAttempt((previous) => previous + 1);
  }, []);
  usePageRestored(retry);

  return { status: statusOf(state, requiredRole, unavailable), retry };
}
