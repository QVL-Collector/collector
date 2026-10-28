import { useTranslation } from 'react-i18next';
import type { ArticlePageView } from './articlePageView';

export type ArticlePageText = {
  readonly title: string;
  readonly subtitle: string | undefined;
  readonly documentTitle: string;
  readonly announcement: string;
};

export function useArticlePageText(view: ArticlePageView): ArticlePageText {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return {
        title: t('article.pendingTitle'),
        subtitle: undefined,
        documentTitle: t('article.pendingDocumentTitle'),
        announcement: t('article.loading'),
      };
    case 'error':
      return {
        title: t('article.pendingTitle'),
        subtitle: undefined,
        documentTitle: t('article.pendingDocumentTitle'),
        announcement: '',
      };
    case 'not-found':
      return {
        title: t('article.notFound.title'),
        subtitle: t('article.notFound.description'),
        documentTitle: t('article.notFound.documentTitle'),
        announcement: t('article.notFound.title'),
      };
    case 'ready':
      return {
        title: view.article.title,
        subtitle: undefined,
        documentTitle: t('article.documentTitle', { title: view.article.title }),
        announcement: t('article.loaded', { title: view.article.title }),
      };
  }
}
