import { keepPreviousData, useQuery, type UseQueryResult } from '@tanstack/react-query';
import { sellerQueryKeys } from '../api/sellerQueryKeys';
import { useSellerClient } from '../api/useSellerClient';
import { sellerArticlesPageSize, type SellerArticlePage } from '../domain/sellerArticle';
import { pollWhileUnderControl } from './controlPolling';

export function useMyArticles(page: number): UseQueryResult<SellerArticlePage> {
  const client = useSellerClient();
  return useQuery({
    queryKey: sellerQueryKeys.articlePage(page, sellerArticlesPageSize),
    queryFn: ({ signal }) =>
      client.listMyArticles({ page, pageSize: sellerArticlesPageSize }, signal),
    placeholderData: keepPreviousData,
    refetchInterval: (query) => pollWhileUnderControl(query.state.data?.articles),
  });
}
