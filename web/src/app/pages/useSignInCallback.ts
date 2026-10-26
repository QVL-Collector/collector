import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router';
import type { SignInOutcome } from '../../auth/signInIntent';
import { useAuthSession } from '../../auth/useAuthSession';
import { useLanguage } from '../../i18n/useLanguage';

export type SignInCallbackStatus = 'pending' | 'failed';

const unexpectedFailure: SignInOutcome = { status: 'failed', language: undefined };

export function useSignInCallback(): SignInCallbackStatus {
  const session = useAuthSession();
  const navigate = useNavigate();
  const { changeLanguage } = useLanguage();
  const [status, setStatus] = useState<SignInCallbackStatus>('pending');

  useEffect(() => {
    let active = true;
    void session
      .completeSignIn()
      .catch(() => unexpectedFailure)
      .then((outcome) => {
        if (!active) {
          return;
        }
        if (outcome.language !== undefined) {
          changeLanguage(outcome.language);
        }
        if (outcome.status === 'completed') {
          void navigate(outcome.returnPath, { replace: true });
        } else {
          setStatus('failed');
        }
      });
    return () => {
      active = false;
    };
  }, [session, navigate, changeLanguage]);

  return status;
}
