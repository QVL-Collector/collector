import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { sellerQueryKeys } from '../api/sellerQueryKeys';
import { useSellerClient } from '../api/useSellerClient';
import type { SellerArticle } from '../domain/sellerArticle';
import { pollWhileUnderControl } from './controlPolling';

export function useSellerArticle(id: string): UseQueryResult<SellerArticle> {
  const client = useSellerClient();
  return useQuery({
    queryKey: sellerQueryKeys.article(id),
    queryFn: ({ signal }) => client.getMyArticle(id, signal),
    refetchInterval: (query) =>
      pollWhileUnderControl(query.state.data === undefined ? undefined : [query.state.data]),
  });
}
