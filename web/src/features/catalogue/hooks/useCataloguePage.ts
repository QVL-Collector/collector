import { useCallback } from 'react';
import { toCatalogueView, type CatalogueView } from './catalogueView';
import { useCatalogueAnnouncement } from './useCatalogueAnnouncement';
import { useCataloguePageParam } from './useCataloguePageParam';
import { usePublishedArticles } from './usePublishedArticles';

export type CataloguePageModel = {
  readonly view: CatalogueView;
  readonly announcement: string;
  readonly retry: () => void;
  readonly goToPage: (page: number) => void;
  readonly goToFirstPage: () => void;
};

export function useCataloguePage(): CataloguePageModel {
  const { page, setPage } = useCataloguePageParam();
  const query = usePublishedArticles(page);
  const view = toCatalogueView(query, page);
  const announcement = useCatalogueAnnouncement(view);
  const { refetch } = query;
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  const goToFirstPage = useCallback(() => {
    setPage(1);
  }, [setPage]);
  return { view, announcement, retry, goToPage: setPage, goToFirstPage };
}
