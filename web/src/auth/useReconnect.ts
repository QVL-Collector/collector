import { useCallback, useState } from 'react';
import { useLocation } from 'react-router';
import { useLanguage } from '../i18n/useLanguage';
import { useAuthSession } from './useAuthSession';

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
