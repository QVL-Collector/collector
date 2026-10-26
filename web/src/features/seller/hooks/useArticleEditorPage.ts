import type { UseQueryResult } from '@tanstack/react-query';
import { useCallback, useMemo, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router';
import { usePageIdentity } from '../../../navigation/usePageIdentity';
import type { SellerArticle } from '../domain/sellerArticle';
import { isApiNotFound } from './articleFailure';
import { useArticleLabels } from './useArticleLabels';
import { useSellerArticle } from './useSellerArticle';
import { useStateChangeAnnouncement } from './useStateChangeAnnouncement';

export type ArticleEditorPageView =
  | { readonly status: 'loading' }
  | { readonly status: 'not-found' }
  | { readonly status: 'error' }
  | { readonly status: 'ready'; readonly article: SellerArticle };

export type ArticleEditorPageModel = {
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly title: string;
  readonly view: ArticleEditorPageView;
  readonly announcement: string;
  readonly retry: () => void;
  readonly fetchLatest: () => Promise<SellerArticle | null>;
};

function toEditorPageView(query: UseQueryResult<SellerArticle>): ArticleEditorPageView {
  if (query.data !== undefined) {
    return { status: 'ready', article: query.data };
  }
  if (isApiNotFound(query.error)) {
    return { status: 'not-found' };
  }
  return query.isError ? { status: 'error' } : { status: 'loading' };
}

export function useArticleEditorPage(): ArticleEditorPageModel {
  const { t } = useTranslation();
  const { titleOf } = useArticleLabels();
  const { articleId = '' } = useParams();
  const query = useSellerArticle(articleId);
  const { data, refetch } = query;
  const title = data === undefined ? t('seller.title') : titleOf(data);
  const headingRef = usePageIdentity(
    data === undefined ? t('seller.documentTitle') : t('seller.editor.documentTitle', { title }),
  );
  const watchedArticles = useMemo(() => (data === undefined ? undefined : [data]), [data]);
  const announcement = useStateChangeAnnouncement(watchedArticles);
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  const fetchLatest = useCallback(async () => (await refetch()).data ?? null, [refetch]);
  return { headingRef, title, view: toEditorPageView(query), announcement, retry, fetchLatest };
}
