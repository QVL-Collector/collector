import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { useSellerClient } from '../api/useSellerClient';
import { acceptsSubmission, type SellerArticle } from '../domain/sellerArticle';
import { toArticleFailure } from './articleFailure';
import type { LockNoticeHandlers } from './useLockNotice';
import { useStoreSellerArticle } from './useStoreSellerArticle';

export type SubmissionFailure = 'conflict' | 'unavailable';

export type SubmitForControlModel = {
  readonly canSubmit: boolean;
  readonly isSubmitting: boolean;
  readonly failure: SubmissionFailure | null;
  readonly submit: () => void;
};

type SubmissionRequest = {
  readonly id: string;
  readonly revision: number;
};

export function useSubmitForControl(
  article: SellerArticle,
  fetchLatest: () => Promise<SellerArticle | null>,
  lockNotice: LockNoticeHandlers,
): SubmitForControlModel {
  const client = useSellerClient();
  const storeArticle = useStoreSellerArticle();
  const [failure, setFailure] = useState<SubmissionFailure | null>(null);
  const mutation = useMutation({
    mutationFn: ({ id, revision }: SubmissionRequest) => client.submitForControl(id, revision),
    onSuccess: storeArticle,
  });
  const showFailure = (error: unknown) => {
    const { kind } = toArticleFailure(error);
    switch (kind) {
      case 'reserved':
        lockNotice.show('reserved');
        return;
      case 'locked':
        lockNotice.show('not-submittable');
        void fetchLatest();
        return;
      case 'conflict':
        setFailure('conflict');
        void fetchLatest();
        return;
      case 'invalid':
      case 'not-found':
      case 'unavailable':
        setFailure('unavailable');
    }
  };
  const submit = () => {
    if (mutation.isPending) {
      return;
    }
    setFailure(null);
    lockNotice.clear();
    mutation.mutate({ id: article.id, revision: article.revision }, { onError: showFailure });
  };
  return {
    canSubmit: acceptsSubmission(article.state),
    isSubmitting: mutation.isPending,
    failure,
    submit,
  };
}
