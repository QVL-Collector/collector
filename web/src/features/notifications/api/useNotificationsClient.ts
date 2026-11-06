import { use } from 'react';
import type { NotificationsClient } from '../domain/sellerNotification';
import { NotificationsClientContext } from './NotificationsClientContext';

export function useNotificationsClient(): NotificationsClient {
  const client = use(NotificationsClientContext);
  if (client === null) {
    throw new Error('NotificationsClientProvider is missing');
  }
  return client;
}
