import Inventory2Outlined from '@mui/icons-material/Inventory2Outlined';
import RefreshRounded from '@mui/icons-material/RefreshRounded';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { EmptyState } from '../../../ui/components/EmptyState';
import { ErrorState } from '../../../ui/components/ErrorState';
import { PendingState } from '../../../ui/components/PendingState';
import type { MyArticlesView } from '../hooks/myArticlesView';
import { MyArticlesResults } from './MyArticlesResults';

type MyArticlesContentProps = {
  readonly view: MyArticlesView;
  readonly retry: () => void;
  readonly goToPage: (page: number) => void;
};

export function MyArticlesContent({ view, retry, goToPage }: MyArticlesContentProps) {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return <PendingState message={t('seller.list.loading')} />;
    case 'error':
      return (
        <ErrorState
          title={t('seller.list.error.title')}
          description={t('seller.list.error.description')}
          action={
            <Button variant="outlined" color="error" startIcon={<RefreshRounded />} onClick={retry}>
              {t('seller.list.error.retry')}
            </Button>
          }
        />
      );
    case 'empty':
      return (
        <EmptyState
          icon={<Inventory2Outlined fontSize="inherit" />}
          title={t('seller.list.empty.title')}
          description={t('seller.list.empty.description')}
        />
      );
    case 'ready':
      return (
        <MyArticlesResults
          articles={view.articles}
          page={view.page}
          pageCount={view.pageCount}
          isRefreshing={view.isRefreshing}
          onPageChange={goToPage}
        />
      );
  }
}
