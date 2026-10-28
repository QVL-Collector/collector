import { keepPreviousData, useQuery, type UseQueryResult } from '@tanstack/react-query';
import { adminQueryKeys } from '../api/adminQueryKeys';
import { useAdminClient } from '../api/useAdminClient';
import { reviewPageSize, type ReviewArticlePage } from '../domain/reviewArticle';

export function useArticlesToReview(page: number): UseQueryResult<ReviewArticlePage> {
  const client = useAdminClient();
  return useQuery({
    queryKey: adminQueryKeys.reviewPage(page, reviewPageSize),
    queryFn: ({ signal }) =>
      client.listArticlesToReview({ page, pageSize: reviewPageSize }, signal),
    placeholderData: keepPreviousData,
  });
}
