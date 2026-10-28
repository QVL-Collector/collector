import { useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { adminQueryKeys } from '../api/adminQueryKeys';
import type { ReviewArticlePage } from '../domain/reviewArticle';
import { useReloadReviewQueue } from './useReloadReviewQueue';

function withoutArticle(page: ReviewArticlePage, id: string): ReviewArticlePage {
  const articles = page.articles.filter((article) => article.id !== id);
  const removed = page.articles.length - articles.length;
  return { ...page, articles, totalItems: Math.max(0, page.totalItems - removed) };
}

export function useRemoveReviewedArticle(): (id: string) => void {
  const queryClient = useQueryClient();
  const reload = useReloadReviewQueue();
  return useCallback(
    (id: string) => {
      queryClient.setQueriesData<ReviewArticlePage>(
        { queryKey: adminQueryKeys.reviewPages() },
        (page) => (page === undefined ? page : withoutArticle(page, id)),
      );
      reload();
    },
    [queryClient, reload],
  );
}
