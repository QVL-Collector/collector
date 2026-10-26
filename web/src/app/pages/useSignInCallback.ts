import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router';
import { useAuthSession } from '../../auth/useAuthSession';

export type SignInCallbackStatus = 'pending' | 'failed';

export function useSignInCallback(): SignInCallbackStatus {
  const session = useAuthSession();
  const navigate = useNavigate();
  const [status, setStatus] = useState<SignInCallbackStatus>('pending');

  useEffect(() => {
    let active = true;
    session.completeSignIn().then(
      (returnPath) => {
        if (active) {
          void navigate(returnPath, { replace: true });
        }
      },
      () => {
        if (active) {
          setStatus('failed');
        }
      },
    );
    return () => {
      active = false;
    };
  }, [session, navigate]);

  return status;
}
