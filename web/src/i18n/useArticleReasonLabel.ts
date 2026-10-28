import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { ArticleReason } from '../shared/articleReason';

export function useArticleReasonLabel(): (reason: ArticleReason) => string {
  const { t } = useTranslation();
  return useCallback((reason: ArticleReason) => t(`control.reasons.${reason}`), [t]);
}
