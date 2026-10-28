import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { ReviewArticle } from '../domain/reviewArticle';

export function useReviewArticleTitle(): (article: ReviewArticle) => string {
  const { t } = useTranslation();
  return useCallback(
    (article: ReviewArticle) =>
      article.title.trim() === '' ? t('admin.review.untitled') : article.title,
    [t],
  );
}
