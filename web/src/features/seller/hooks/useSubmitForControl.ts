import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { useSellerClient } from '../api/useSellerClient';
import { acceptsChanges, type SellerArticle } from '../domain/sellerArticle';
import { toArticleFailure } from './articleFailure';
import { useStoreSellerArticle } from './useStoreSellerArticle';

export type SubmissionFailure = 'conflict' | 'locked' | 'unavailable';

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

function toSubmissionFailure(error: unknown): SubmissionFailure {
  const { kind } = toArticleFailure(error);
  return kind === 'conflict' || kind === 'locked' ? kind : 'unavailable';
}

export function useSubmitForControl(
  article: SellerArticle,
  fetchLatest: () => Promise<SellerArticle | null>,
): SubmitForControlModel {
  const client = useSellerClient();
  const storeArticle = useStoreSellerArticle();
  const [failure, setFailure] = useState<SubmissionFailure | null>(null);
  const mutation = useMutation({
    mutationFn: ({ id, revision }: SubmissionRequest) => client.submitForControl(id, revision),
    onSuccess: storeArticle,
  });
  const submit = () => {
    if (mutation.isPending) {
      return;
    }
    setFailure(null);
    mutation.mutate(
      { id: article.id, revision: article.revision },
      {
        onError: (error) => {
          const submissionFailure = toSubmissionFailure(error);
          setFailure(submissionFailure);
          if (submissionFailure !== 'unavailable') {
            void fetchLatest();
          }
        },
      },
    );
  };
  return {
    canSubmit: acceptsChanges(article.state),
    isSubmitting: mutation.isPending,
    failure,
    submit,
  };
}
