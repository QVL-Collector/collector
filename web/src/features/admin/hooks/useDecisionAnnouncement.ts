import { useCallback, useRef, useState, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import type { DecidedState, ReviewArticle } from '../domain/reviewArticle';
import { useReviewArticleTitle } from './useReviewArticleTitle';

export type DecisionAnnouncementModel = {
  readonly announcement: string;
  readonly focusTargetRef: RefObject<HTMLHeadingElement | null>;
  readonly announceDecision: (article: ReviewArticle, state: DecidedState) => void;
};

export function useDecisionAnnouncement(): DecisionAnnouncementModel {
  const { t } = useTranslation();
  const titleOf = useReviewArticleTitle();
  const focusTargetRef = useRef<HTMLHeadingElement>(null);
  const [announcement, setAnnouncement] = useState('');
  const announceDecision = useCallback(
    (article: ReviewArticle, state: DecidedState) => {
      setAnnouncement(t(`admin.review.decided.${state}`, { title: titleOf(article) }));
      focusTargetRef.current?.focus();
    },
    [t, titleOf],
  );
  return { announcement, focusTargetRef, announceDecision };
}
