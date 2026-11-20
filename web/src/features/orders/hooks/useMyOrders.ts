import { keepPreviousData, useQuery, type UseQueryResult } from '@tanstack/react-query';
import { ordersQueryKeys } from '../api/ordersQueryKeys';
import { useOrdersClient } from '../api/useOrdersClient';
import { ordersPageSize, type OrderPage } from '../domain/order';
import { pollUnlessAccessRefused } from './myOrdersPolling';

export function useMyOrders(page: number): UseQueryResult<OrderPage> {
  const client = useOrdersClient();
  return useQuery({
    queryKey: ordersQueryKeys.list(page, ordersPageSize),
    queryFn: ({ signal }) => client.listMyOrders({ page, pageSize: ordersPageSize }, signal),
    placeholderData: keepPreviousData,
    refetchInterval: (query) => pollUnlessAccessRefused(query.state.error),
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: 'always',
  });
}
