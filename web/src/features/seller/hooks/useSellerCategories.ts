import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { sellerQueryKeys } from '../api/sellerQueryKeys';
import { useSellerClient } from '../api/useSellerClient';
import type { SellerCategory } from '../domain/sellerArticle';

const categoriesStaleTimeMs = 5 * 60_000;

export function useSellerCategories(): UseQueryResult<readonly SellerCategory[]> {
  const client = useSellerClient();
  return useQuery({
    queryKey: sellerQueryKeys.categories(),
    queryFn: ({ signal }) => client.listCategories(signal),
    staleTime: categoriesStaleTimeMs,
  });
}
