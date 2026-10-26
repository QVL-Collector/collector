import { useTranslation } from 'react-i18next';
import type { CatalogueView } from './catalogueView';

export function useCatalogueAnnouncement(view: CatalogueView): string {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return t('catalogue.loading');
    case 'error':
      return '';
    case 'empty':
      return t('catalogue.empty.title');
    case 'out-of-range':
      return t('catalogue.outOfRange.title');
    case 'ready':
      return view.isRefreshing
        ? t('catalogue.loading')
        : t('catalogue.loaded', {
            count: view.totalItems,
            page: view.page,
            pageCount: view.pageCount,
          });
  }
}
