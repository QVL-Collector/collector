import type { UseQueryResult } from '@tanstack/react-query';
import type { PublishedArticle } from '../domain/catalogue';
import { isArticleNotFound } from '../domain/catalogueProblems';

export type ArticlePageView =
  | { readonly status: 'loading' }
  | { readonly status: 'not-found' }
  | { readonly status: 'error' }
  | { readonly status: 'ready'; readonly article: PublishedArticle };

export function toArticlePageView(query: UseQueryResult<PublishedArticle>): ArticlePageView {
  if (query.data !== undefined) {
    return { status: 'ready', article: query.data };
  }
  if (isArticleNotFound(query.error)) {
    return { status: 'not-found' };
  }
  return query.isError ? { status: 'error' } : { status: 'loading' };
}
