import { useTranslation } from 'react-i18next';
import type { NotificationsView } from './notificationsView';

export function useNotificationsAnnouncement(view: NotificationsView): string {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
    case 'unauthenticated':
    case 'forbidden':
    case 'error':
      return '';
    case 'empty':
      return t('notifications.empty.title');
    case 'ready':
      return t('notifications.loaded', { count: view.notifications.length });
  }
}
