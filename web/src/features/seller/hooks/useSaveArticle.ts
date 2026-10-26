import { useMutation, type UseMutationResult } from '@tanstack/react-query';
import { useSellerClient } from '../api/useSellerClient';
import type { ArticleDraft, SellerArticle } from '../domain/sellerArticle';
import { useStoreSellerArticle } from './useStoreSellerArticle';

export type SaveTarget =
  | { readonly kind: 'create' }
  | { readonly kind: 'update'; readonly id: string; readonly revision: number };

export type SaveRequest = {
  readonly target: SaveTarget;
  readonly draft: ArticleDraft;
};

export function useSaveArticle(): UseMutationResult<SellerArticle, Error, SaveRequest> {
  const client = useSellerClient();
  const storeArticle = useStoreSellerArticle();
  return useMutation({
    mutationFn: ({ target, draft }: SaveRequest) =>
      target.kind === 'create'
        ? client.createDraft(draft)
        : client.updateDraft(target.id, target.revision, draft),
    onSuccess: storeArticle,
  });
}
