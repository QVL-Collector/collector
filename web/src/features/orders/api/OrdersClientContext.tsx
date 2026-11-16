import { createContext, type ReactNode } from 'react';
import type { OrdersClient } from '../domain/order';

export const OrdersClientContext = createContext<OrdersClient | null>(null);

type OrdersClientProviderProps = {
  readonly client: OrdersClient | null;
  readonly children: ReactNode;
};

export function OrdersClientProvider({ client, children }: OrdersClientProviderProps) {
  return <OrdersClientContext value={client}>{children}</OrdersClientContext>;
}
