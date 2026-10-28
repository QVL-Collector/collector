import { useTranslation } from 'react-i18next';
import type { MyArticlesView } from './myArticlesView';

export function useMyArticlesAnnouncement(view: MyArticlesView): string {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
    case 'error':
      return '';
    case 'empty':
      return t('seller.list.empty.title');
    case 'ready':
      return view.isRefreshing
        ? t('seller.list.loading')
        : t('seller.list.loaded', { count: view.totalItems });
  }
}
