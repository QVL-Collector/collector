import type { UseQueryResult } from '@tanstack/react-query';
import type { CatalogueArticle, CataloguePage } from '../domain/catalogue';
import { isUnknownCategory } from '../domain/catalogueProblems';

export type CatalogueView =
  | { readonly status: 'loading' }
  | { readonly status: 'error' }
  | { readonly status: 'unknown-category' }
  | { readonly status: 'empty'; readonly filtered: boolean }
  | { readonly status: 'out-of-range' }
  | {
      readonly status: 'ready';
      readonly articles: readonly CatalogueArticle[];
      readonly page: number;
      readonly pageCount: number;
      readonly totalItems: number;
      readonly isRefreshing: boolean;
    };

function countPages(totalItems: number, pageSize: number): number {
  return Math.max(1, Math.ceil(totalItems / pageSize));
}

export function toCatalogueView(
  query: UseQueryResult<CataloguePage>,
  requestedPage: number,
  categoryId: string | null,
): CatalogueView {
  if (query.status === 'error') {
    return isUnknownCategory(query.error) ? { status: 'unknown-category' } : { status: 'error' };
  }
  if (query.status === 'pending') {
    return { status: 'loading' };
  }
  const { articles, totalItems, pageSize } = query.data;
  if (totalItems === 0) {
    return { status: 'empty', filtered: categoryId !== null };
  }
  if (articles.length === 0) {
    return { status: 'out-of-range' };
  }
  return {
    status: 'ready',
    articles,
    page: requestedPage,
    pageCount: countPages(totalItems, pageSize),
    totalItems,
    isRefreshing: query.isPlaceholderData,
  };
}
