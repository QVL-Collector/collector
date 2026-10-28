import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useArticleReasonLabel } from '../../../i18n/useArticleReasonLabel';
import type { ArticleState, SellerArticle } from '../domain/sellerArticle';

export type ArticleLabels = {
  readonly stateLabel: (state: ArticleState) => string;
  readonly stateDetail: (state: ArticleState) => string;
  readonly titleOf: (article: SellerArticle) => string;
  readonly reasonOf: (article: SellerArticle) => string | null;
  readonly stateSentence: (article: SellerArticle) => string;
};

export function useArticleLabels(): ArticleLabels {
  const { t } = useTranslation();
  const reasonLabel = useArticleReasonLabel();
  return useMemo(() => {
    const stateLabel = (state: ArticleState) => t(`seller.states.${state}`);
    const titleOf = (article: SellerArticle) =>
      article.title.trim() === '' ? t('seller.untitled') : article.title;
    const reasonOf = (article: SellerArticle) =>
      article.reason === null ? null : t('seller.reason', { reason: reasonLabel(article.reason) });
    const stateSentence = (article: SellerArticle) => {
      const values = { title: titleOf(article), state: stateLabel(article.state) };
      return article.reason === null
        ? t('seller.announcements.stateChanged', values)
        : t('seller.announcements.stateChangedWithReason', {
            ...values,
            reason: reasonLabel(article.reason),
          });
    };
    return {
      stateLabel,
      stateDetail: (state: ArticleState) => t(`seller.stateDetails.${state}`),
      titleOf,
      reasonOf,
      stateSentence,
    };
  }, [t, reasonLabel]);
}
