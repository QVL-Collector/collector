import { keepPreviousData, useQuery, type UseQueryResult } from '@tanstack/react-query';
import { useCatalogueClient } from '../api/useCatalogueClient';
import { cataloguePageSize, type CataloguePage } from '../domain/catalogue';

const publishedArticlesQueryKey = (page: number) =>
  ['catalogue', 'published-articles', page, cataloguePageSize] as const;

export function usePublishedArticles(page: number): UseQueryResult<CataloguePage> {
  const client = useCatalogueClient();
  return useQuery({
    queryKey: publishedArticlesQueryKey(page),
    queryFn: ({ signal }) =>
      client.listPublishedArticles({ page, pageSize: cataloguePageSize }, signal),
    placeholderData: keepPreviousData,
  });
}
