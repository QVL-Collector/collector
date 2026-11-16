import LoginRounded from '@mui/icons-material/LoginRounded';
import RefreshRounded from '@mui/icons-material/RefreshRounded';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { ErrorState } from '../../../ui/components/ErrorState';
import { LinkButton } from '../../../ui/components/LinkButton';
import { PendingState } from '../../../ui/components/PendingState';
import { cataloguePath } from '../../catalogue/cataloguePaths';
import type { OrderConfirmationView } from '../hooks/orderConfirmationView';
import { OrderSummaryView } from './OrderSummaryView';

type OrderConfirmationContentProps = {
  readonly view: OrderConfirmationView;
  readonly sessionExpiredDescription: string;
  readonly retry: () => void;
  readonly reconnect: () => void;
};

export function OrderConfirmationContent({
  view,
  sessionExpiredDescription,
  retry,
  reconnect,
}: OrderConfirmationContentProps) {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return <PendingState message={t('orders.loading')} />;
    case 'not-found':
      return (
        <LinkButton to={cataloguePath} label={t('common.backToCatalogue')} variant="contained" />
      );
    case 'unauthenticated':
      return (
        <ErrorState
          title={t('orders.unauthenticated.title')}
          description={sessionExpiredDescription}
          action={
            <Button
              variant="outlined"
              color="error"
              startIcon={<LoginRounded />}
              onClick={reconnect}
            >
              {t('orders.unauthenticated.signIn')}
            </Button>
          }
        />
      );
    case 'forbidden':
      return (
        <ErrorState
          title={t('orders.forbidden.title')}
          description={t('orders.forbidden.description')}
        />
      );
    case 'error':
      return (
        <ErrorState
          title={t('orders.error.title')}
          description={t('orders.error.description')}
          action={
            <Button variant="outlined" color="error" startIcon={<RefreshRounded />} onClick={retry}>
              {t('orders.error.retry')}
            </Button>
          }
        />
      );
    case 'ready':
      return <OrderSummaryView order={view.order} />;
  }
}
