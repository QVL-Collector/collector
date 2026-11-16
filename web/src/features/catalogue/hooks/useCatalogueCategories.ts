import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { catalogueQueryRoot } from '../api/catalogueQueryKeys';
import { useCatalogueClient } from '../api/useCatalogueClient';
import type { ArticleCategory } from '../domain/catalogue';

const categoriesStaleTimeMs = 5 * 60_000;

const catalogueCategoriesQueryKey = [...catalogueQueryRoot, 'categories'] as const;

export function useCatalogueCategories(): UseQueryResult<readonly ArticleCategory[]> {
  const client = useCatalogueClient();
  return useQuery({
    queryKey: catalogueCategoriesQueryKey,
    queryFn: ({ signal }) => client.listCategories(signal),
    staleTime: categoriesStaleTimeMs,
  });
}
