import type { CataloguePageModel } from '../hooks/useCataloguePage';
import { CatalogueEmptyState } from './CatalogueEmptyState';
import { CatalogueErrorState } from './CatalogueErrorState';
import { CatalogueOutOfRange } from './CatalogueOutOfRange';
import { CatalogueResults } from './CatalogueResults';
import { CatalogueSkeleton } from './CatalogueSkeleton';

type CatalogueContentProps = Omit<CataloguePageModel, 'announcement'>;

export function CatalogueContent({ view, retry, goToPage, goToFirstPage }: CatalogueContentProps) {
  switch (view.status) {
    case 'loading':
      return <CatalogueSkeleton />;
    case 'error':
      return <CatalogueErrorState onRetry={retry} />;
    case 'empty':
      return <CatalogueEmptyState />;
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
