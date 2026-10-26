import { useCallback, useState } from 'react';
import { useLocation } from 'react-router';
import { useAuthSession } from '../../auth/useAuthSession';
import { usePageRestored } from '../../ui/hooks/usePageRestored';

export type AuthAction = 'signIn' | 'signOut';

export type AuthActionsModel = {
  readonly busy: boolean;
  readonly failedAction: AuthAction | null;
  readonly failureVisible: boolean;
  readonly signIn: () => void;
  readonly signOut: () => void;
  readonly dismissFailure: () => void;
};

export function useAuthActions(): AuthActionsModel {
  const session = useAuthSession();
  const { pathname, search } = useLocation();
  const [busy, setBusy] = useState(false);
  const [failedAction, setFailedAction] = useState<AuthAction | null>(null);
  const [failureVisible, setFailureVisible] = useState(false);

  const run = useCallback((action: AuthAction, operation: () => Promise<void>) => {
    setBusy(true);
    setFailureVisible(false);
    operation().catch(() => {
      setBusy(false);
      setFailedAction(action);
      setFailureVisible(true);
    });
  }, []);

  const signIn = useCallback(() => {
    run('signIn', () => session.signIn(`${pathname}${search}`));
  }, [run, session, pathname, search]);

  const signOut = useCallback(() => {
    run('signOut', session.signOut);
  }, [run, session]);

  const dismissFailure = useCallback(() => {
    setFailureVisible(false);
  }, []);

  const resetBusy = useCallback(() => {
    setBusy(false);
  }, []);
  usePageRestored(resetBusy);

  return { busy, failedAction, failureVisible, signIn, signOut, dismissFailure };
}
