import { useCallback } from 'react';
import { useLocation } from 'react-router';
import { toCatalogueReturnState, type CatalogueReturnState } from '../catalogueReturn';
import { publishedArticlePath } from '../cataloguePaths';

export type ArticleCardLink = {
  readonly to: string;
  readonly state: CatalogueReturnState;
};

export function useArticleCardLink(): (articleId: string) => ArticleCardLink {
  const { search } = useLocation();
  return useCallback(
    (articleId: string) => ({
      to: publishedArticlePath(articleId),
      state: toCatalogueReturnState(search),
    }),
    [search],
  );
}
