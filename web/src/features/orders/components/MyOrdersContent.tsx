import LoginRounded from '@mui/icons-material/LoginRounded';
import RefreshRounded from '@mui/icons-material/RefreshRounded';
import ShoppingBagOutlined from '@mui/icons-material/ShoppingBagOutlined';
import StorefrontOutlined from '@mui/icons-material/StorefrontOutlined';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { EmptyState } from '../../../ui/components/EmptyState';
import { ErrorState } from '../../../ui/components/ErrorState';
import { LinkButton } from '../../../ui/components/LinkButton';
import { PendingState } from '../../../ui/components/PendingState';
import { cataloguePath } from '../../catalogue/cataloguePaths';
import type { MyOrdersView } from '../hooks/myOrdersView';
import { MyOrdersResults } from './MyOrdersResults';

type MyOrdersContentProps = {
  readonly view: MyOrdersView;
  readonly now: Date;
  readonly sessionExpiredDescription: string;
  readonly retry: () => void;
  readonly reconnect: () => void;
  readonly goToPage: (page: number) => void;
};

export function MyOrdersContent({
  view,
  now,
  sessionExpiredDescription,
  retry,
  reconnect,
  goToPage,
}: MyOrdersContentProps) {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return <PendingState message={t('orders.list.loading')} />;
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
          description={t('orders.list.forbiddenDescription')}
        />
      );
    case 'error':
      return (
        <ErrorState
          title={t('orders.list.error.title')}
          description={t('orders.list.error.description')}
          action={
            <Button variant="outlined" color="error" startIcon={<RefreshRounded />} onClick={retry}>
              {t('orders.list.error.retry')}
            </Button>
          }
        />
      );
    case 'empty':
      return (
        <EmptyState
          icon={<ShoppingBagOutlined fontSize="inherit" />}
          title={t('orders.list.empty.title')}
          description={t('orders.list.empty.description')}
          action={
            <LinkButton
              to={cataloguePath}
              label={t('orders.list.empty.action')}
              icon={<StorefrontOutlined />}
              variant="contained"
            />
          }
        />
      );
    case 'ready':
      return (
        <MyOrdersResults
          orders={view.orders}
          now={now}
          page={view.page}
          pageCount={view.pageCount}
          isRefreshing={view.isRefreshing}
          onPageChange={goToPage}
        />
      );
  }
}
