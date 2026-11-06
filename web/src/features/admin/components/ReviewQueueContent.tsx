import RefreshRounded from '@mui/icons-material/RefreshRounded';
import TaskAltRounded from '@mui/icons-material/TaskAltRounded';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { EmptyState } from '../../../ui/components/EmptyState';
import { ErrorState } from '../../../ui/components/ErrorState';
import { PendingState } from '../../../ui/components/PendingState';
import type { ReviewQueueView } from '../hooks/reviewQueueView';
import type { DecisionListener } from '../hooks/useReviewDecision';
import { ReviewQueueResults } from './ReviewQueueResults';

type ReviewQueueContentProps = {
  readonly view: ReviewQueueView;
  readonly retry: () => void;
  readonly goToPage: (page: number) => void;
  readonly onDecided: DecisionListener;
};

export function ReviewQueueContent({ view, retry, goToPage, onDecided }: ReviewQueueContentProps) {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return <PendingState message={t('admin.review.loading')} />;
    case 'forbidden':
      return (
        <ErrorState
          title={t('admin.review.forbidden.title')}
          description={t('admin.review.forbidden.description')}
        />
      );
    case 'error':
      return (
        <ErrorState
          title={t('admin.review.error.title')}
          description={t('admin.review.error.description')}
          action={
            <Button variant="outlined" color="error" startIcon={<RefreshRounded />} onClick={retry}>
              {t('admin.review.error.retry')}
            </Button>
          }
        />
      );
    case 'empty':
      return (
        <EmptyState
          icon={<TaskAltRounded fontSize="inherit" />}
          title={t('admin.review.empty.title')}
          description={t('admin.review.empty.description')}
        />
      );
    case 'ready':
      return (
        <ReviewQueueResults
          articles={view.articles}
          page={view.page}
          pageCount={view.pageCount}
          isRefreshing={view.isRefreshing}
          onPageChange={goToPage}
          onDecided={onDecided}
        />
      );
  }
}
