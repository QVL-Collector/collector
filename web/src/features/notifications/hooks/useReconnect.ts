import { useCallback, useState } from 'react';
import { useLocation } from 'react-router';
import { useAuthSession } from '../../../auth/useAuthSession';
import { useLanguage } from '../../../i18n/useLanguage';

export type ReconnectModel = {
  readonly reconnect: () => void;
  readonly reconnectFailed: boolean;
};

export function useReconnect(): ReconnectModel {
  const session = useAuthSession();
  const { pathname, search } = useLocation();
  const { language } = useLanguage();
  const [reconnectFailed, setReconnectFailed] = useState(false);
  const reconnect = useCallback(() => {
    setReconnectFailed(false);
    session.signIn({ returnPath: `${pathname}${search}`, language }).catch(() => {
      setReconnectFailed(true);
    });
  }, [session, pathname, search, language]);
  return { reconnect, reconnectFailed };
}
