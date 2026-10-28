import { useMutation } from '@tanstack/react-query';
import { useCallback } from 'react';
import type { RefusalReason } from '../../../shared/articleReason';
import { useAdminClient } from '../api/useAdminClient';
import type { DecidedState, ReviewArticle, ReviewDecision } from '../domain/reviewArticle';
import { toDecisionFailure, type DecisionFailure } from './decisionFailure';
import { useReloadReviewQueue } from './useReloadReviewQueue';
import { useRemoveReviewedArticle } from './useRemoveReviewedArticle';

export type DecisionListener = (article: ReviewArticle, state: DecidedState) => void;

export type PendingDecision = ReviewDecision['kind'] | null;

export type ReviewDecisionModel = {
  readonly pendingDecision: PendingDecision;
  readonly canDecide: boolean;
  readonly failure: DecisionFailure | null;
  readonly publish: () => void;
  readonly refuse: (reason: RefusalReason) => void;
  readonly reload: () => void;
};

export function useReviewDecision(
  article: ReviewArticle,
  onDecided: DecisionListener,
): ReviewDecisionModel {
  const client = useAdminClient();
  const removeArticle = useRemoveReviewedArticle();
  const reloadQueue = useReloadReviewQueue();
  const mutation = useMutation({
    mutationFn: (decision: ReviewDecision) => client.decide(article.id, article.revision, decision),
    onSuccess: (outcome) => {
      onDecided(article, outcome.state);
      removeArticle(outcome.id);
    },
  });
  const { mutate, reset } = mutation;
  const publish = useCallback(() => {
    mutate({ kind: 'publish' });
  }, [mutate]);
  const refuse = useCallback(
    (reason: RefusalReason) => {
      mutate({ kind: 'refuse', reason });
    },
    [mutate],
  );
  const reload = useCallback(() => {
    reset();
    reloadQueue();
  }, [reset, reloadQueue]);
  const failure = mutation.isError ? toDecisionFailure(mutation.error) : null;
  return {
    pendingDecision: mutation.isPending ? mutation.variables.kind : null,
    canDecide: !mutation.isPending && failure !== 'conflict',
    failure,
    publish,
    refuse,
    reload,
  };
}
