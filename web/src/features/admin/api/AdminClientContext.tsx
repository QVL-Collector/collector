import { createContext, type ReactNode } from 'react';
import type { AdminClient } from '../domain/reviewArticle';

export const AdminClientContext = createContext<AdminClient | null>(null);

type AdminClientProviderProps = {
  readonly client: AdminClient | null;
  readonly children: ReactNode;
};

export function AdminClientProvider({ client, children }: AdminClientProviderProps) {
  return <AdminClientContext value={client}>{children}</AdminClientContext>;
}
