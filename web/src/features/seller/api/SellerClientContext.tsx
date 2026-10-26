import { createContext, type ReactNode } from 'react';
import type { SellerClient } from '../domain/sellerArticle';

export const SellerClientContext = createContext<SellerClient | null>(null);

type SellerClientProviderProps = {
  readonly client: SellerClient | null;
  readonly children: ReactNode;
};

export function SellerClientProvider({ client, children }: SellerClientProviderProps) {
  return <SellerClientContext value={client}>{children}</SellerClientContext>;
}
