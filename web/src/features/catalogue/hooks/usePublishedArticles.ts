import { keepPreviousData, useQuery, type UseQueryResult } from '@tanstack/react-query';
import { useCatalogueClient } from '../api/useCatalogueClient';
import { cataloguePageSize, type CataloguePage } from '../domain/catalogue';

const publishedArticlesQueryKey = (page: number, categoryId: string | null) =>
  ['catalogue', 'published-articles', categoryId, page, cataloguePageSize] as const;

export function usePublishedArticles(
  page: number,
  categoryId: string | null,
): UseQueryResult<CataloguePage> {
  const client = useCatalogueClient();
  return useQuery({
    queryKey: publishedArticlesQueryKey(page, categoryId),
    queryFn: ({ signal }) =>
      client.listPublishedArticles({ page, pageSize: cataloguePageSize, categoryId }, signal),
    placeholderData: keepPreviousData,
  });
}
