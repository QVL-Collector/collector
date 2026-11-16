import { use } from 'react';
import type { OrdersClient } from '../domain/order';
import { OrdersClientContext } from './OrdersClientContext';

export function useOrdersClient(): OrdersClient {
  const client = use(OrdersClientContext);
  if (client === null) {
    throw new Error('OrdersClientProvider is missing');
  }
  return client;
}
