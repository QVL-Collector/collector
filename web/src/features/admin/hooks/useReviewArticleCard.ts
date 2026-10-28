import { useTranslation } from 'react-i18next';
import { useArticleReasonLabel } from '../../../i18n/useArticleReasonLabel';
import type { ReviewArticle } from '../domain/reviewArticle';
import { useRefusalDialog, type RefusalDialogModel } from './useRefusalDialog';
import { useReviewArticleFacts, type ReviewFact } from './useReviewArticleFacts';
import { useReviewArticleTitle } from './useReviewArticleTitle';
import {
  useReviewDecision,
  type DecisionListener,
  type ReviewDecisionModel,
} from './useReviewDecision';

export type ReviewArticleCardLabels = {
  readonly title: string;
  readonly controlReason: string;
  readonly revision: string;
  readonly publish: string;
  readonly publishAccessible: string;
  readonly refuse: string;
  readonly refuseAccessible: string;
};

export type ReviewArticleCardModel = {
  readonly labels: ReviewArticleCardLabels;
  readonly facts: readonly ReviewFact[];
  readonly decision: ReviewDecisionModel;
  readonly refusal: RefusalDialogModel;
};

export function useReviewArticleCard(
  article: ReviewArticle,
  onDecided: DecisionListener,
): ReviewArticleCardModel {
  const { t } = useTranslation();
  const reasonLabel = useArticleReasonLabel();
  const titleOf = useReviewArticleTitle();
  const facts = useReviewArticleFacts(article);
  const decision = useReviewDecision(article, onDecided);
  const refusal = useRefusalDialog(decision.refuse);
  const title = titleOf(article);
  return {
    labels: {
      title,
      controlReason: t('admin.review.controlReason', {
        reason: reasonLabel(article.controlReason),
      }),
      revision: t('admin.review.revision', { revision: article.revision }),
      publish: t('admin.review.actions.publish'),
      publishAccessible: t('admin.review.actions.publishLabel', { title }),
      refuse: t('admin.review.actions.refuse'),
      refuseAccessible: t('admin.review.actions.refuseLabel', { title }),
    },
    facts,
    decision,
    refusal,
  };
}
