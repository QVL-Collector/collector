import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { ordersQueryKeys } from '../api/ordersQueryKeys';
import { useOrdersClient } from '../api/useOrdersClient';
import type { Order } from '../domain/order';

export function useOrder(id: string): UseQueryResult<Order> {
  const client = useOrdersClient();
  return useQuery({
    queryKey: ordersQueryKeys.detail(id),
    queryFn: ({ signal }) => client.getOrder(id, signal),
  });
}
