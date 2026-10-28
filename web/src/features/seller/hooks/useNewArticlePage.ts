import { useCallback, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { usePageIdentity } from '../../../navigation/usePageIdentity';
import type { SellerArticle } from '../domain/sellerArticle';
import { articlePath } from '../sellerPaths';
import { useArticleEditor, type ArticleEditorModel } from './useArticleEditor';
import { useCategoryOptions, type CategoryOptions } from './useCategoryOptions';

export type NewArticlePageModel = {
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly title: string;
  readonly subtitle: string;
  readonly editor: ArticleEditorModel;
  readonly categories: CategoryOptions;
};

export function useNewArticlePage(): NewArticlePageModel {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const headingRef = usePageIdentity(t('seller.editor.newDocumentTitle'));
  const onCreated = useCallback(
    (article: SellerArticle) => {
      void navigate(articlePath(article.id), { replace: true });
    },
    [navigate],
  );
  const editor = useArticleEditor({ article: null, onCreated });
  const categories = useCategoryOptions(null);
  return {
    headingRef,
    title: t('seller.newArticle'),
    subtitle: t('seller.editor.newSubtitle'),
    editor,
    categories,
  };
}
