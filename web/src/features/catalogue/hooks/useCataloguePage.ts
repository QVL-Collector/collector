import { useCallback } from 'react';
import { toCatalogueView, type CatalogueView } from './catalogueView';
import { useCatalogueAnnouncement } from './useCatalogueAnnouncement';
import { useCatalogueCategoryParam } from './useCatalogueCategoryParam';
import { useCataloguePageParam } from './useCataloguePageParam';
import { useCategoryFilter, type CategoryFilterModel } from './useCategoryFilter';
import { usePublishedArticles } from './usePublishedArticles';

export type CataloguePageModel = {
  readonly view: CatalogueView;
  readonly announcement: string;
  readonly filter: CategoryFilterModel;
  readonly retry: () => void;
  readonly goToPage: (page: number) => void;
  readonly goToFirstPage: () => void;
  readonly showAllCategories: () => void;
};

export function useCataloguePage(): CataloguePageModel {
  const { page, setPage } = useCataloguePageParam();
  const { categoryId, selectCategory, clearCategory } = useCatalogueCategoryParam();
  const query = usePublishedArticles(page, categoryId);
  const filter = useCategoryFilter(categoryId, selectCategory);
  const view = toCatalogueView(query, page, categoryId);
  const announcement = useCatalogueAnnouncement(view, filter.selectedLabel);
  const { refetch } = query;
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  const goToFirstPage = useCallback(() => {
    setPage(1);
  }, [setPage]);
  return {
    view,
    announcement,
    filter,
    retry,
    goToPage: setPage,
    goToFirstPage,
    showAllCategories: clearCategory,
  };
}
