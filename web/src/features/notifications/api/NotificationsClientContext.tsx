import { createContext, type ReactNode } from 'react';
import type { NotificationsClient } from '../domain/sellerNotification';

export const NotificationsClientContext = createContext<NotificationsClient | null>(null);

type NotificationsClientProviderProps = {
  readonly client: NotificationsClient | null;
  readonly children: ReactNode;
};

export function NotificationsClientProvider({
  client,
  children,
}: NotificationsClientProviderProps) {
  return <NotificationsClientContext value={client}>{children}</NotificationsClientContext>;
}
