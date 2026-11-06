import { useCallback, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { usePageIdentity } from '../../../navigation/usePageIdentity';
import { toNotificationsView, type NotificationsView } from './notificationsView';
import { useNotificationsAnnouncement } from './useNotificationsAnnouncement';
import { useReconnect } from './useReconnect';
import { useSellerNotifications } from './useSellerNotifications';

export type NotificationsPageModel = {
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly view: NotificationsView;
  readonly announcement: string;
  readonly sessionExpiredDescription: string;
  readonly retry: () => void;
  readonly reconnect: () => void;
};

export function useNotificationsPage(): NotificationsPageModel {
  const { t } = useTranslation();
  const headingRef = usePageIdentity(t('notifications.documentTitle'));
  const query = useSellerNotifications();
  const view = toNotificationsView(query);
  const announcement = useNotificationsAnnouncement(view);
  const { reconnect, reconnectFailed } = useReconnect();
  const { refetch } = query;
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  return {
    headingRef,
    view,
    announcement,
    sessionExpiredDescription: reconnectFailed
      ? t('auth.signInFailed')
      : t('notifications.unauthenticated.description'),
    retry,
    reconnect,
  };
}
