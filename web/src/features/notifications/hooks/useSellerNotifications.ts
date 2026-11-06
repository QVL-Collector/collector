import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { notificationsQueryKeys } from '../api/notificationsQueryKeys';
import { useNotificationsClient } from '../api/useNotificationsClient';
import type { SellerNotification } from '../domain/sellerNotification';
import { notificationsPollingIntervalMs, pollUnlessAccessRefused } from './notificationsPolling';

export function useSellerNotifications(): UseQueryResult<readonly SellerNotification[]> {
  const client = useNotificationsClient();
  return useQuery({
    queryKey: notificationsQueryKeys.list(),
    queryFn: ({ signal }) => client.listNotifications(signal),
    staleTime: notificationsPollingIntervalMs,
    refetchInterval: (query) => pollUnlessAccessRefused(query.state.error),
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  });
}
