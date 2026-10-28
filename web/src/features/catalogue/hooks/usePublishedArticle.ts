import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { useCatalogueClient } from '../api/useCatalogueClient';
import type { PublishedArticle } from '../domain/catalogue';

const publishedArticleQueryKey = (id: string) => ['catalogue', 'published-article', id] as const;

export function usePublishedArticle(id: string): UseQueryResult<PublishedArticle> {
  const client = useCatalogueClient();
  return useQuery({
    queryKey: publishedArticleQueryKey(id),
    queryFn: ({ signal }) => client.getPublishedArticle(id, signal),
  });
}
