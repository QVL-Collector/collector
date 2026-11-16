import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { useNavigate } from 'react-router';
import { catalogueQueryRoot } from '../../catalogue/api/catalogueQueryKeys';
import { ordersQueryKeys } from '../api/ordersQueryKeys';
import { useOrdersClient } from '../api/useOrdersClient';
import type { Order } from '../domain/order';
import { orderPath } from '../ordersPaths';
import {
  blocksReservation,
  toReservationFailure,
  type ReservationFailure,
} from './reservationFailure';

export type ReserveArticleOptions = {
  readonly articleId: string;
  readonly onStart: () => void;
  readonly onFailure: (failure: ReservationFailure) => void;
};

export type ReserveArticleModel = {
  readonly isReserving: boolean;
  readonly reserve: () => void;
};

export function useReserveArticle({
  articleId,
  onStart,
  onFailure,
}: ReserveArticleOptions): ReserveArticleModel {
  const client = useOrdersClient();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const refreshCatalogue = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: catalogueQueryRoot });
  }, [queryClient]);
  const mutation = useMutation({
    mutationFn: () => client.reserveArticle(articleId),
    onSuccess: (order: Order) => {
      queryClient.setQueryData(ordersQueryKeys.detail(order.id), order);
      void navigate(orderPath(order.id));
      refreshCatalogue();
    },
    onError: (error: unknown) => {
      const failure = toReservationFailure(error);
      onFailure(failure);
      if (blocksReservation(failure)) {
        refreshCatalogue();
      }
    },
  });
  const { isPending, mutate } = mutation;
  const reserve = useCallback(() => {
    if (isPending) {
      return;
    }
    onStart();
    mutate();
  }, [isPending, mutate, onStart]);
  return { isReserving: isPending, reserve };
}
