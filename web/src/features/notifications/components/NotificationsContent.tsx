import LoginRounded from '@mui/icons-material/LoginRounded';
import NotificationsNoneRounded from '@mui/icons-material/NotificationsNoneRounded';
import RefreshRounded from '@mui/icons-material/RefreshRounded';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { EmptyState } from '../../../ui/components/EmptyState';
import { ErrorState } from '../../../ui/components/ErrorState';
import { PendingState } from '../../../ui/components/PendingState';
import type { NotificationsView } from '../hooks/notificationsView';
import { NotificationList } from './NotificationList';

type NotificationsContentProps = {
  readonly view: NotificationsView;
  readonly sessionExpiredDescription: string;
  readonly retry: () => void;
  readonly reconnect: () => void;
};

export function NotificationsContent({
  view,
  sessionExpiredDescription,
  retry,
  reconnect,
}: NotificationsContentProps) {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return <PendingState message={t('notifications.loading')} />;
    case 'unauthenticated':
      return (
        <ErrorState
          title={t('notifications.unauthenticated.title')}
          description={sessionExpiredDescription}
          action={
            <Button
              variant="outlined"
              color="error"
              startIcon={<LoginRounded />}
              onClick={reconnect}
            >
              {t('notifications.unauthenticated.signIn')}
            </Button>
          }
        />
      );
    case 'forbidden':
      return (
        <ErrorState
          title={t('notifications.forbidden.title')}
          description={t('notifications.forbidden.description')}
        />
      );
    case 'error':
      return (
        <ErrorState
          title={t('notifications.error.title')}
          description={t('notifications.error.description')}
          action={
            <Button variant="outlined" color="error" startIcon={<RefreshRounded />} onClick={retry}>
              {t('notifications.error.retry')}
            </Button>
          }
        />
      );
    case 'empty':
      return (
        <EmptyState
          icon={<NotificationsNoneRounded fontSize="inherit" />}
          title={t('notifications.empty.title')}
          description={t('notifications.empty.description')}
        />
      );
    case 'ready':
      return <NotificationList notifications={view.notifications} />;
  }
}
