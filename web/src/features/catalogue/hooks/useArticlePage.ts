import { useCallback, type RefObject } from 'react';
import { useLocation, useParams, type Location } from 'react-router';
import { usePageIdentity } from '../../../navigation/usePageIdentity';
import { catalogueReturnPath } from '../catalogueReturn';
import { toArticlePageView, type ArticlePageView } from './articlePageView';
import { useArticlePageText } from './useArticlePageText';
import { usePublishedArticle } from './usePublishedArticle';

export type ArticlePageModel = {
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly title: string;
  readonly subtitle: string | undefined;
  readonly announcement: string;
  readonly backPath: string;
  readonly view: ArticlePageView;
  readonly retry: () => void;
};

export function useArticlePage(): ArticlePageModel {
  const { articleId = '' } = useParams();
  const location: Location<unknown> = useLocation();
  const query = usePublishedArticle(articleId);
  const view = toArticlePageView(query);
  const { documentTitle, ...text } = useArticlePageText(view);
  const headingRef = usePageIdentity(documentTitle);
  const { refetch } = query;
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  return {
    headingRef,
    ...text,
    backPath: catalogueReturnPath(location.state),
    view,
    retry,
  };
}
