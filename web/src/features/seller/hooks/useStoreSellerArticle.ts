import { useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { sellerQueryKeys } from '../api/sellerQueryKeys';
import type { SellerArticle } from '../domain/sellerArticle';

export function useStoreSellerArticle(): (article: SellerArticle) => void {
  const queryClient = useQueryClient();
  return useCallback(
    (article: SellerArticle) => {
      queryClient.setQueryData(sellerQueryKeys.article(article.id), article);
      void queryClient.invalidateQueries({ queryKey: sellerQueryKeys.articlePages() });
    },
    [queryClient],
  );
}
