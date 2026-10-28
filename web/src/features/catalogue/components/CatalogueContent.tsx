import type { CataloguePageModel } from '../hooks/useCataloguePage';
import { CatalogueEmptyState } from './CatalogueEmptyState';
import { CatalogueErrorState } from './CatalogueErrorState';
import { CatalogueOutOfRange } from './CatalogueOutOfRange';
import { CatalogueResults } from './CatalogueResults';
import { CatalogueSkeleton } from './CatalogueSkeleton';
import { CategoryEmptyState } from './CategoryEmptyState';
import { UnknownCategoryState } from './UnknownCategoryState';

type CatalogueContentProps = Omit<CataloguePageModel, 'announcement' | 'filter'>;

export function CatalogueContent({
  view,
  retry,
  goToPage,
  goToFirstPage,
  showAllCategories,
}: CatalogueContentProps) {
  switch (view.status) {
    case 'loading':
      return <CatalogueSkeleton />;
    case 'error':
      return <CatalogueErrorState onRetry={retry} />;
    case 'unknown-category':
      return <UnknownCategoryState onShowAllCategories={showAllCategories} />;
    case 'empty':
      return view.filtered ? (
        <CategoryEmptyState onShowAllCategories={showAllCategories} />
      ) : (
        <CatalogueEmptyState />
      );
    case 'out-of-range':
      return <CatalogueOutOfRange onBackToFirstPage={goToFirstPage} />;
    case 'ready':
      return (
        <CatalogueResults
          articles={view.articles}
          page={view.page}
          pageCount={view.pageCount}
          isRefreshing={view.isRefreshing}
          onPageChange={goToPage}
        />
      );
  }
}
