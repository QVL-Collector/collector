import { useTranslation } from 'react-i18next';
import type { CatalogueView } from './catalogueView';

type ReadyView = Extract<CatalogueView, { readonly status: 'ready' }>;

export function useCatalogueAnnouncement(
  view: CatalogueView,
  categoryLabel: string | null,
): string {
  const { t } = useTranslation();
  const loaded = (ready: ReadyView) => {
    const counts = { count: ready.totalItems, page: ready.page, pageCount: ready.pageCount };
    return categoryLabel === null
      ? t('catalogue.loaded', counts)
      : t('catalogue.loadedInCategory', { ...counts, category: categoryLabel });
  };
  switch (view.status) {
    case 'loading':
      return t('catalogue.loading');
    case 'error':
      return '';
    case 'unknown-category':
      return t('catalogue.unknownCategory.title');
    case 'empty':
      return view.filtered ? t('catalogue.emptyCategory.title') : t('catalogue.empty.title');
    case 'out-of-range':
      return t('catalogue.outOfRange.title');
    case 'ready':
      return view.isRefreshing ? t('catalogue.loading') : loaded(view);
  }
}
