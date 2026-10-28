import type { SellerArticle } from '../domain/sellerArticle';
import { useArticleEditor, type ArticleEditorModel } from './useArticleEditor';
import { useArticleStatus, type ArticleStatus } from './useArticleStatus';
import { useCategoryOptions, type CategoryOptions } from './useCategoryOptions';
import { useLockNotice } from './useLockNotice';
import { useSubmitForControl, type SubmitForControlModel } from './useSubmitForControl';

export type ArticleWorkspaceModel = {
  readonly article: SellerArticle;
  readonly status: ArticleStatus;
  readonly editor: ArticleEditorModel;
  readonly submission: SubmitForControlModel;
  readonly categories: CategoryOptions;
  readonly lockMessage: string | null;
  readonly isSubmissionBlocked: boolean;
};

export function useArticleWorkspace(
  article: SellerArticle,
  fetchLatest: () => Promise<SellerArticle | null>,
): ArticleWorkspaceModel {
  const status = useArticleStatus(article);
  const lockNotice = useLockNotice();
  const editor = useArticleEditor({ article, fetchLatest, lockNotice: lockNotice.handlers });
  const submission = useSubmitForControl(article, fetchLatest, lockNotice.handlers);
  const categories = useCategoryOptions(article.category);
  return {
    article,
    status,
    editor,
    submission,
    categories,
    lockMessage: lockNotice.message,
    isSubmissionBlocked: editor.isDirty || editor.isSaving,
  };
}
