import { acceptsChanges, type SellerArticle } from '../domain/sellerArticle';
import { useArticleEditor, type ArticleEditorModel } from './useArticleEditor';
import { useArticleStatus, type ArticleStatus } from './useArticleStatus';
import { useCategoryOptions, type CategoryOptions } from './useCategoryOptions';
import { useSubmitForControl, type SubmitForControlModel } from './useSubmitForControl';

export type ArticleWorkspaceModel = {
  readonly article: SellerArticle;
  readonly status: ArticleStatus;
  readonly editor: ArticleEditorModel;
  readonly submission: SubmitForControlModel;
  readonly categories: CategoryOptions;
  readonly isEditable: boolean;
  readonly isSubmissionBlocked: boolean;
};

export function useArticleWorkspace(
  article: SellerArticle,
  fetchLatest: () => Promise<SellerArticle | null>,
): ArticleWorkspaceModel {
  const status = useArticleStatus(article);
  const editor = useArticleEditor({ article, fetchLatest });
  const submission = useSubmitForControl(article, fetchLatest);
  const categories = useCategoryOptions(article.category);
  return {
    article,
    status,
    editor,
    submission,
    categories,
    isEditable: acceptsChanges(article.state),
    isSubmissionBlocked: editor.isDirty || editor.isSaving,
  };
}
