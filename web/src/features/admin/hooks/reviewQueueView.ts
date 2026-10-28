import type { UseQueryResult } from '@tanstack/react-query';
import { isApiError } from '../../../api/ApiError';
import { adminHttpStatuses } from '../domain/adminProblems';
import type { ReviewArticle, ReviewArticlePage } from '../domain/reviewArticle';

export type ReviewQueueView =
  | { readonly status: 'loading' }
  | { readonly status: 'forbidden' }
  | { readonly status: 'error' }
  | { readonly status: 'empty' }
  | {
      readonly status: 'ready';
      readonly articles: readonly ReviewArticle[];
      readonly page: number;
      readonly pageCount: number;
      readonly totalItems: number;
      readonly isRefreshing: boolean;
    };

function isForbidden(error: unknown): boolean {
  return isApiError(error) && error.status === adminHttpStatuses.forbidden;
}

export function toReviewQueueView(
  query: UseQueryResult<ReviewArticlePage>,
  requestedPage: number,
): ReviewQueueView {
  if (query.data === undefined) {
    if (!query.isError) {
      return { status: 'loading' };
    }
    return isForbidden(query.error) ? { status: 'forbidden' } : { status: 'error' };
  }
  const { articles, totalItems, pageSize } = query.data;
  if (totalItems === 0 || articles.length === 0) {
    return { status: 'empty' };
  }
  return {
    status: 'ready',
    articles,
    page: requestedPage,
    pageCount: Math.max(1, Math.ceil(totalItems / pageSize)),
    totalItems,
    isRefreshing: query.isPlaceholderData,
  };
}
