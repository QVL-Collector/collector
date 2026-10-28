import { useCallback, useState, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { usePageIdentity } from '../../../navigation/usePageIdentity';
import { toReviewQueueView, type ReviewQueueView } from './reviewQueueView';
import { useArticlesToReview } from './useArticlesToReview';
import { useDecisionAnnouncement } from './useDecisionAnnouncement';
import type { DecisionListener } from './useReviewDecision';
import { useReviewQueueAnnouncement } from './useReviewQueueAnnouncement';

export type AdminReviewPageModel = {
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly listHeadingRef: RefObject<HTMLHeadingElement | null>;
  readonly view: ReviewQueueView;
  readonly listAnnouncement: string;
  readonly decisionAnnouncement: string;
  readonly retry: () => void;
  readonly goToPage: (page: number) => void;
  readonly onDecided: DecisionListener;
};

export function useAdminReviewPage(): AdminReviewPageModel {
  const { t } = useTranslation();
  const headingRef = usePageIdentity(t('admin.documentTitle'));
  const [page, setPage] = useState(1);
  const query = useArticlesToReview(page);
  const view = toReviewQueueView(query, page);
  const listAnnouncement = useReviewQueueAnnouncement(view);
  const { announcement, focusTargetRef, announceDecision } = useDecisionAnnouncement();
  const { refetch } = query;
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  return {
    headingRef,
    listHeadingRef: focusTargetRef,
    view,
    listAnnouncement,
    decisionAnnouncement: announcement,
    retry,
    goToPage: setPage,
    onDecided: announceDecision,
  };
}
