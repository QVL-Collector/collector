import type { UseQueryResult } from '@tanstack/react-query';
import type { SellerArticle, SellerArticlePage } from '../domain/sellerArticle';

export type MyArticlesView =
  | { readonly status: 'loading' }
  | { readonly status: 'error' }
  | { readonly status: 'empty' }
  | {
      readonly status: 'ready';
      readonly articles: readonly SellerArticle[];
      readonly page: number;
      readonly pageCount: number;
      readonly totalItems: number;
      readonly isRefreshing: boolean;
    };

export function toMyArticlesView(
  query: UseQueryResult<SellerArticlePage>,
  requestedPage: number,
): MyArticlesView {
  if (query.data === undefined) {
    return query.isError ? { status: 'error' } : { status: 'loading' };
  }
  const { articles, totalItems, pageSize } = query.data;
  if (totalItems === 0) {
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
