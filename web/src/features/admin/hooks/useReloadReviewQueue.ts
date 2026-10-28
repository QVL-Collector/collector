import { useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { adminQueryKeys } from '../api/adminQueryKeys';

export function useReloadReviewQueue(): () => void {
  const queryClient = useQueryClient();
  return useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: adminQueryKeys.reviewPages() });
  }, [queryClient]);
}
