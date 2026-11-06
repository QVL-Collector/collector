import { useTranslation } from 'react-i18next';
import type { ReviewQueueView } from './reviewQueueView';

export function useReviewQueueAnnouncement(view: ReviewQueueView): string {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
    case 'forbidden':
    case 'error':
      return '';
    case 'empty':
      return t('admin.review.empty.title');
    case 'ready':
      return view.isRefreshing
        ? t('admin.review.loading')
        : t('admin.review.loaded', { count: view.totalItems });
  }
}
